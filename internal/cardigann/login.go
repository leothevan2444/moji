package cardigann

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Challenge struct {
	Type        string
	Prompt      string
	Image       []byte
	ContentType string
}

var ErrAuthentication = errors.New("site authentication is required")

// CheckSearchAuthentication mirrors the declared login test on HTML results.
// A login redirect/page must not become a successful empty search.
func (e *Execution) checkSearchAuthentication(ctx context.Context, response HTTPResponse) error {
	if e.plan.Definition.Login == nil {
		return nil
	}
	if response.Status == http.StatusUnauthorized || response.Status == http.StatusForbidden || response.Status >= 300 && response.Status < 400 {
		return ErrAuthentication
	}
	test := e.plan.Definition.Login.Test
	if test == nil || test.Selector == "" || response.Headers.Get("Content-Type") != "" && !strings.Contains(strings.ToLower(response.Headers.Get("Content-Type")), "html") {
		return nil
	}
	text, err := e.responseText(response)
	if err != nil {
		return err
	}
	document, err := parseMarkup(text, false)
	if err != nil {
		return err
	}
	query, err := e.render(ctx, test.Selector, e.variables, nil)
	if err != nil {
		return err
	}
	selected, err := selectHTML(document.Selection, query, false)
	if err != nil {
		return err
	}
	if selected.Length() == 0 {
		return ErrAuthentication
	}
	return nil
}

type loginSubmission struct {
	challenge   Challenge
	request     Request
	fields      *formFields
	answerField string
}

// Authenticate executes declarative login. Interactive challenges are returned
// as bounded presentation data; their pending request stays server-side.
func (e *Execution) Authenticate(ctx context.Context) (*Challenge, error) {
	login := e.plan.Definition.Login
	if login == nil {
		return nil, nil
	}
	if e.pendingLogin != nil {
		challenge := e.pendingLogin.challenge
		return &challenge, nil
	}
	headers := login.Headers
	if headers == nil {
		headers = e.plan.Definition.Search.Headers
	}
	if login.Method == "cookie" {
		if err := e.session.SetCookieHeader(e.base.String(), e.variables[".Config.cookie"].Text); err != nil {
			return nil, err
		}
		return nil, e.testLogin(ctx, headers)
	}
	for _, raw := range login.Cookies {
		value, err := e.render(ctx, raw, e.variables, nil)
		if err != nil {
			return nil, err
		}
		if err := e.session.SetCookieHeader(e.base.String(), value); err != nil {
			return nil, err
		}
	}
	mode := login.Method
	if mode == "get" || mode == "oneurl" || mode == "" {
		block := RequestBlock{Path: login.Path, Method: http.MethodGet, Inputs: login.Inputs}
		if mode == "oneurl" {
			path, err := e.render(ctx, login.Path, e.variables, nil)
			if err != nil {
				return nil, err
			}
			input, _ := login.Inputs.Lookup("oneurl")
			tail, err := e.render(ctx, string(input), e.variables, nil)
			if err != nil {
				return nil, err
			}
			block.Path = path + tail
			block.Inputs = nil
		}
		request, err := e.requestFrom(ctx, block, e.variables, headers, e.base, true, false)
		if err != nil {
			return nil, err
		}
		request.Headers.Set("Referer", e.base.String())
		request.FollowRedirect = true
		response, err := e.request(ctx, request)
		if err != nil {
			return nil, err
		}
		if err := e.checkLoginResponse(ctx, response); err != nil {
			return nil, err
		}
		return nil, e.testLogin(ctx, headers)
	}
	if mode != "form" && mode != "post" {
		return nil, fmt.Errorf("unsupported login method")
	}
	fields := new(formFields)
	landingURL := e.base.String()
	var doc *goquery.Document
	if mode == "form" || len(login.SelectorInputs) > 0 || len(login.GetSelectorInputs) > 0 || login.Captcha != nil {
		request, err := e.requestFrom(ctx, RequestBlock{Path: login.Path, Method: http.MethodGet}, e.variables, headers, e.base, false, false)
		if err != nil {
			return nil, err
		}
		request.Headers.Set("Referer", e.base.String())
		request.FollowRedirect = true
		response, err := e.request(ctx, request)
		if err != nil {
			return nil, err
		}
		landingURL = response.URL
		text, err := e.responseText(response)
		if err != nil {
			return nil, err
		}
		doc, err = parseMarkup(text, false)
		if err != nil {
			return nil, err
		}
	}
	action := login.Path
	if mode == "form" {
		formQuery := login.Form
		if formQuery == "" {
			formQuery = "form"
		}
		form, err := selectHTML(doc.Selection, formQuery, false)
		if err != nil {
			return nil, err
		}
		if form.Length() == 0 {
			return nil, fmt.Errorf("login form did not match")
		}
		action = form.AttrOr("action", landingURL)
		form.Find("input[name], textarea[name], select[name]").Each(func(_ int, input *goquery.Selection) {
			if _, disabled := input.Attr("disabled"); disabled {
				return
			}
			typeName := strings.ToLower(input.AttrOr("type", ""))
			if typeName == "checkbox" || typeName == "radio" {
				if _, checked := input.Attr("checked"); !checked {
					return
				}
			}
			name := input.AttrOr("name", "")
			if name == "" {
				return
			}
			switch goquery.NodeName(input) {
			case "textarea":
				fields.Set(name, input.Text())
			case "select":
				options := input.Find("option[selected]")
				if options.Length() == 0 {
					options = input.Find("option").First()
				}
				options.Each(func(_ int, option *goquery.Selection) { fields.Set(name, option.AttrOr("value", option.Text())) })
			default:
				fields.Set(name, input.AttrOr("value", ""))
			}
		})
	}
	fc := FilterContext{Variables: e.variables, Now: e.now, Encoding: e.plan.Definition.Encoding}
	for _, field := range login.Inputs {
		key, source := field.Name, field.Value
		name := key
		if login.Selectors {
			if doc == nil {
				return nil, fmt.Errorf("login input selector requires landing page")
			}
			input, err := selectHTML(doc.Selection, key, false)
			if err != nil {
				return nil, err
			}
			name = input.AttrOr("name", "")
			if name == "" {
				return nil, fmt.Errorf("login input selector did not match named input")
			}
		}
		value, err := e.render(ctx, string(source), e.variables, nil)
		if err != nil {
			return nil, err
		}
		fields.Set(name, value)
	}
	for _, field := range login.SelectorInputs {
		name, selector := field.Name, field.Selector
		value, found, err := HTMLField(ctx, doc.Selection, selector, fc)
		if err != nil {
			return nil, err
		}
		if found {
			fields.Set(name, value)
		}
	}
	if login.SubmitPath != "" {
		action = login.SubmitPath
	}
	action, err := e.render(ctx, action, e.variables, nil)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(landingURL)
	if err != nil {
		return nil, fmt.Errorf("invalid login location")
	}
	target, err := resolveURL(base, action, false)
	if err != nil {
		return nil, err
	}
	if len(login.GetSelectorInputs) > 0 {
		query := formFields{}
		for _, field := range login.GetSelectorInputs {
			name, selector := field.Name, field.Selector
			value, found, err := HTMLField(ctx, doc.Selection, selector, fc)
			if err != nil {
				return nil, err
			}
			if found {
				query.Set(name, value)
			}
		}
		target, err = query.withQuery(target)
		if err != nil {
			return nil, err
		}
	}
	rendered, err := e.headers(ctx, headers, e.variables)
	if err != nil {
		return nil, err
	}
	rendered.Set("Referer", landingURL)
	challengeHeaders := rendered.Clone()
	rendered.Set("Content-Type", "application/x-www-form-urlencoded")
	submission := loginSubmission{request: Request{URL: target, Method: http.MethodPost, Headers: rendered, FollowRedirect: true}, fields: fields}
	if doc != nil {
		if doc.Find(`script[src*="simpleCaptcha"]`).Length() > 0 {
			url, err := resolveURL(e.base, "simpleCaptcha.php?numImages=1", false)
			if err != nil {
				return nil, err
			}
			response, err := e.request(ctx, Request{URL: url, Method: http.MethodGet, Headers: challengeHeaders.Clone(), FollowRedirect: true})
			if err != nil {
				return nil, err
			}
			var captcha struct {
				Images []struct {
					Hash string `json:"hash"`
				} `json:"images"`
			}
			if err := json.Unmarshal(response.Body, &captcha); err != nil || len(captcha.Images) == 0 || captcha.Images[0].Hash == "" {
				return nil, fmt.Errorf("invalid simpleCaptcha challenge")
			}
			fields.Set("captchaSelection", captcha.Images[0].Hash)
			fields.Set("submitme", "X")
		}
		if login.Captcha != nil {
			query, err := e.render(ctx, login.Captcha.Selector, e.variables, nil)
			if err != nil {
				return nil, err
			}
			challengeElement, err := selectHTML(doc.Selection, query, false)
			if err != nil {
				return nil, err
			}
			if challengeElement.Length() > 0 {
				challenge := Challenge{Type: login.Captcha.Type}
				if challenge.Type == "text" {
					challenge.Prompt = strings.TrimSpace(challengeElement.Text())
					if len(challenge.Prompt) > 4096 {
						return nil, fmt.Errorf("captcha prompt exceeds budget")
					}
				} else {
					image, err := resolveURL(base, challengeElement.AttrOr("src", ""), false)
					if err != nil {
						return nil, err
					}
					response, err := e.request(ctx, Request{URL: image, Method: http.MethodGet, Headers: challengeHeaders.Clone(), FollowRedirect: true})
					if err != nil {
						return nil, err
					}
					if len(response.Body) > 1<<20 || !strings.HasPrefix(strings.ToLower(response.Headers.Get("Content-Type")), "image/") {
						return nil, fmt.Errorf("invalid captcha image")
					}
					challenge.Image = append([]byte(nil), response.Body...)
					challenge.ContentType = response.Headers.Get("Content-Type")
				}
				name := login.Captcha.Input
				if login.Selectors {
					input, err := selectHTML(doc.Selection, name, false)
					if err != nil {
						return nil, err
					}
					name = input.AttrOr("name", "")
					if name == "" {
						return nil, fmt.Errorf("captcha input selector did not match")
					}
				}
				submission.challenge = challenge
				submission.answerField = name
				e.pendingLogin = &submission
				return &challenge, nil
			}
		}
	}
	return nil, e.submitLogin(ctx, submission)
}

func (e *Execution) AnswerChallenge(ctx context.Context, answer string) error {
	if e.pendingLogin == nil {
		return fmt.Errorf("no pending login challenge")
	}
	if strings.TrimSpace(answer) == "" || len(answer) > 4096 {
		return fmt.Errorf("invalid challenge answer")
	}
	submission := *e.pendingLogin
	e.pendingLogin = nil
	submission.fields.Set(submission.answerField, answer)
	return e.submitLogin(ctx, submission)
}

func (e *Execution) submitLogin(ctx context.Context, submission loginSubmission) error {
	body, err := submission.fields.Encode(e.plan.Definition.Encoding)
	if err != nil {
		return err
	}
	submission.request.Body = []byte(body)
	response, err := e.request(ctx, submission.request)
	if err != nil {
		return err
	}
	if err := e.checkLoginResponse(ctx, response); err != nil {
		return err
	}
	headers := e.plan.Definition.Login.Headers
	if headers == nil {
		headers = e.plan.Definition.Search.Headers
	}
	return e.testLogin(ctx, headers)
}

func (e *Execution) checkLoginResponse(ctx context.Context, response HTTPResponse) error {
	text, err := e.responseText(response)
	if err != nil {
		return err
	}
	doc, err := parseMarkup(text, false)
	if err != nil {
		return err
	}
	return e.checkErrors(ctx, doc, e.plan.Definition.Login.Error, e.variables)
}

func (e *Execution) testLogin(ctx context.Context, headers map[string][]string) error {
	test := e.plan.Definition.Login.Test
	if test == nil {
		return nil
	}
	request, err := e.requestFrom(ctx, RequestBlock{Path: test.Path}, e.variables, headers, e.base, false, false)
	if err != nil {
		return err
	}
	request.FollowRedirect = true
	response, err := e.request(ctx, request)
	if err != nil {
		return err
	}
	if err := e.checkLoginResponse(ctx, response); err != nil {
		return err
	}
	if test.Selector != "" {
		text, err := e.responseText(response)
		if err != nil {
			return err
		}
		doc, err := parseMarkup(text, false)
		if err != nil {
			return err
		}
		query, err := e.render(ctx, test.Selector, e.variables, nil)
		if err != nil {
			return err
		}
		selected, err := selectHTML(doc.Selection, query, false)
		if err != nil {
			return err
		}
		if selected.Length() == 0 {
			return fmt.Errorf("login verification selector did not match")
		}
	}
	return nil
}
