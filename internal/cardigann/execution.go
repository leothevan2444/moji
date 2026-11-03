package cardigann

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const MaxPageBytes = 16 << 20
const MaxTorrentBytes = 8 << 20
const MaxSearchResults = 1000

type Request struct {
	URL            string
	Method         string
	Headers        http.Header
	Body           []byte
	FollowRedirect bool
	Torrent        bool
}

type HTTPResponse struct {
	URL     string
	Status  int
	Headers http.Header
	Body    []byte
}

// Session owns HTTP transport, cookie isolation, rate limiting and redirect
// validation. The interpreter cannot bypass those policies by opening sockets.
type Session interface {
	Do(context.Context, Request) (HTTPResponse, error)
	SetCookieHeader(origin, header string) error
}

type Query struct {
	Mode       string
	Keywords   string
	Categories []int
	Parameters map[string]string // Named fields; raw Season/Ep derive the Episode search token.
	Limit      *int              // nil leaves the template value unset; an explicit zero remains "0".
	Offset     *int
}

var ErrInvalidQuery = errors.New("invalid Cardigann query")

type queryError string

func (e queryError) Error() string      { return string(e) }
func (queryError) Is(target error) bool { return target == ErrInvalidQuery }

type Execution struct {
	plan         *Plan
	session      Session
	base         *url.URL
	variables    Variables
	now          time.Time
	requests     int
	maxRequests  int
	pendingLogin *loginSubmission
}

func (p *Plan) NewExecution(session Session, baseURL string, settings map[string]Scalar, now time.Time) (*Execution, error) {
	if session == nil {
		return nil, fmt.Errorf("site session is required")
	}
	if baseURL == "" {
		baseURL = p.Definition.Links[0]
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" || base.User != nil || (base.Scheme != "http" && base.Scheme != "https") {
		return nil, fmt.Errorf("invalid site base URL")
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	variables := Variables{".Config.sitelink": {Text: base.String()}, ".True": {Text: "True"}, ".False": {}, ".Today.Year": {Text: strconv.Itoa(now.Year())}}
	known := map[string]bool{}
	for _, setting := range p.Definition.Settings {
		known[setting.Name] = true
		value := setting.Default
		if configured, ok := settings[setting.Name]; ok {
			value = configured
		}
		switch setting.Type {
		case "checkbox":
			text := ""
			if strings.EqualFold(string(value), "true") {
				text = "True"
			} else if value != "" && !strings.EqualFold(string(value), "false") {
				return nil, fmt.Errorf("setting %s must be boolean", setting.Name)
			}
			variables[".Config."+setting.Name] = TemplateValue{Text: text}
		case "select":
			// An absent default or explicit empty selection stays empty. Picking
			// a sorted option changes the rule's requests without user intent.
			if _, ok := setting.Options[string(value)]; value != "" && !ok {
				return nil, fmt.Errorf("setting %s has an invalid selection", setting.Name)
			}
			variables[".Config."+setting.Name] = TemplateValue{Text: string(value)}
		case "text", "password":
			variables[".Config."+setting.Name] = TemplateValue{Text: string(value)}
		}
	}
	for key := range settings {
		if !known[key] {
			return nil, fmt.Errorf("unknown rule setting %s", key)
		}
	}
	return &Execution{plan: p, session: session, base: base, variables: variables, now: now, maxRequests: 50}, nil
}

func (e *Execution) render(ctx context.Context, text string, variables Variables, modifier func(string) string) (string, error) {
	template := e.plan.templates[text]
	if template == nil {
		var err error
		template, err = CompileTemplate(text)
		if err != nil {
			return "", err
		}
	}
	return template.RenderWithModifier(ctx, variables, modifier)
}

func resolveURL(base *url.URL, text string, allowMagnet bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(text))
	if err != nil {
		return "", fmt.Errorf("invalid resource URL")
	}
	resolved := base.ResolveReference(parsed)
	if allowMagnet && resolved.Scheme == "magnet" && resolved.User == nil {
		return resolved.String(), nil
	}
	if (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" || resolved.User != nil {
		return "", fmt.Errorf("unsupported resource URL")
	}
	return resolved.String(), nil
}

func (e *Execution) request(ctx context.Context, request Request) (HTTPResponse, error) {
	if err := ctx.Err(); err != nil {
		return HTTPResponse{}, err
	}
	if e.requests >= e.maxRequests {
		return HTTPResponse{}, fmt.Errorf("site request budget exceeded")
	}
	e.requests++
	response, err := e.session.Do(ctx, request)
	if err != nil {
		return HTTPResponse{}, err
	}
	limit := MaxPageBytes
	if request.Torrent {
		limit = MaxTorrentBytes
	}
	if len(response.Body) > limit {
		return HTTPResponse{}, fmt.Errorf("site response exceeds byte budget")
	}
	if response.URL == "" {
		response.URL = request.URL
	}
	if _, err := resolveURL(e.base, response.URL, request.Torrent); err != nil {
		return HTTPResponse{}, err
	}
	if response.Status < 200 || response.Status >= 400 {
		if e.plan.Definition.Login != nil && (response.Status == http.StatusUnauthorized || response.Status == http.StatusForbidden) {
			return response, ErrAuthentication
		}
		return response, fmt.Errorf("site HTTP status %d", response.Status)
	}
	return response, nil
}

func (e *Execution) headers(ctx context.Context, headers map[string][]string, variables Variables) (http.Header, error) {
	out := http.Header{}
	for key, values := range headers {
		for _, source := range values {
			text, err := e.render(ctx, source, variables, nil)
			if err != nil {
				return nil, err
			}
			if strings.ContainsAny(text, "\r\n") {
				return nil, fmt.Errorf("invalid request header value")
			}
			out.Add(key, text)
		}
	}
	return out, nil
}

// requestFrom renders variables separately from path literals so a slash or
// ampersand in a search term cannot alter the shape of the request.
func (e *Execution) requestFrom(ctx context.Context, request RequestBlock, variables Variables, headers map[string][]string, base *url.URL, allowEmpty, encodePath bool) (Request, error) {
	return e.requestFromInputGroups(ctx, request, variables, headers, base, allowEmpty, encodePath, []requestInputGroup{{values: request.Inputs}})
}

type requestInputGroup struct {
	values    ScalarFields
	expandRaw bool
}

func (e *Execution) requestFromInputGroups(ctx context.Context, request RequestBlock, variables Variables, headers map[string][]string, base *url.URL, allowEmpty, encodePath bool, groups []requestInputGroup) (Request, error) {
	var modifier func(string) string
	if encodePath {
		modifier = func(text string) string { return strings.ReplaceAll(encodeFormBytes(text), "+", "%20") }
	}
	path, err := e.render(ctx, request.Path, variables, modifier)
	if err != nil {
		return Request{}, err
	}
	target, err := resolveURL(base, path, false)
	if err != nil {
		return Request{}, err
	}
	method := strings.ToUpper(request.Method)
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodPost {
		return Request{}, fmt.Errorf("unsupported declarative HTTP method")
	}
	var pairs []string
	for _, group := range groups {
		for _, input := range group.values {
			key := input.Name
			if key == "$raw" && group.expandRaw {
				text, err := e.render(ctx, string(input.Value), variables, encodeFormBytes)
				if err != nil {
					return Request{}, err
				}
				// Raw syntax is split before the usual value encoding. The
				// template modifier has already escaped substituted values.
				for _, part := range strings.Split(text, "&") {
					name, value, _ := strings.Cut(part, "=")
					if name == "" {
						continue
					}
					encoded, err := encodeQueryValue(value, e.plan.Definition.Encoding)
					if err != nil {
						return Request{}, err
					}
					pairs = append(pairs, name+"="+encoded)
				}
				continue
			}
			value, err := e.render(ctx, string(input.Value), variables, nil)
			if err != nil {
				return Request{}, err
			}
			if strings.TrimSpace(value) != "" || allowEmpty {
				encoded, err := encodeQueryValue(value, e.plan.Definition.Encoding)
				if err != nil {
					return Request{}, err
				}
				// Cardigann input names are already query syntax (for example
				// category[]). Prowlarr preserves them for GET and POST alike.
				pairs = append(pairs, key+"="+encoded)
			}
		}
	}
	separator := request.QuerySeparator
	if separator == "" {
		separator = "&"
	}
	encoded := strings.Join(pairs, separator)
	outHeaders, err := e.headers(ctx, headers, variables)
	if err != nil {
		return Request{}, err
	}
	out := Request{URL: target, Method: method, Headers: outHeaders}
	if method == http.MethodPost {
		out.Body = []byte(encoded)
		out.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if encoded != "" {
		delimiter := "?"
		if strings.Contains(target, "?") {
			delimiter = "&"
		}
		out.URL += delimiter + encoded
	}
	return out, nil
}
