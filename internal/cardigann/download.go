package cardigann

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/leothevan2444/moji/internal/torrentmeta"
)

type Artifact struct {
	TorrentBytes []byte
	Magnet       string
}

var ErrNotTorrent = errors.New("download response is not a torrent")

func (e *Execution) Fetch(ctx context.Context, result Result) (Artifact, error) {
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	target := result.DownloadURL
	if target == "" {
		if result.MagnetURI != "" {
			return magnetArtifact(result.MagnetURI)
		}
		if result.InfoHash != "" && e.plan.Definition.Type != "private" {
			return magnetArtifact(generatedMagnetURI(result.InfoHash, result.Title))
		}
		return Artifact{}, fmt.Errorf("candidate has no downloadable resource")
	}
	target, err := resolveURL(e.base, target, true)
	if err != nil {
		return Artifact{}, err
	}
	if strings.HasPrefix(target, "magnet:") {
		return magnetArtifact(target)
	}
	link, _ := url.Parse(target)
	variables := e.variables.Clone()
	for key, value := range result.Fields {
		variables[".Result."+key] = TemplateValue{Text: value}
	}
	addDownloadVariables(variables, link)
	headers := e.plan.Definition.Search.Headers
	method := http.MethodGet
	download := e.plan.Definition.Download
	if download != nil && download.Headers != nil {
		headers = download.Headers
	}
	var beforeResponse *HTTPResponse
	if download != nil {
		if strings.EqualFold(download.Method, "post") {
			method = http.MethodPost
		} else if download.Method != "" && !strings.EqualFold(download.Method, "get") {
			return Artifact{}, fmt.Errorf("unsupported download method")
		}
		if download.Before != nil {
			block := download.Before.RequestBlock
			if download.Before.PathSelector != nil {
				response, err := e.downloadPage(ctx, target, headers, variables)
				if err != nil {
					return Artifact{}, err
				}
				path, found, err := e.downloadField(ctx, response, *download.Before.PathSelector, variables)
				if err != nil {
					return Artifact{}, err
				}
				if !found || path == "" {
					return Artifact{}, fmt.Errorf("download before path selector did not match")
				}
				block.Path = path
			}
			request, err := e.requestFrom(ctx, block, variables, headers, e.base, true, false)
			if err != nil {
				return Artifact{}, err
			}
			request.Headers.Set("Referer", target)
			request.FollowRedirect = true
			response, err := e.request(ctx, request)
			if err != nil {
				return Artifact{}, err
			}
			beforeResponse = &response
		}
		// The original page is fetched once and shared by fallback selectors.
		// A successfully fetched test torrent is returned directly, avoiding
		// the upstream pattern of testing and then grabbing it a second time.
		var pageResponse *HTTPResponse
		selectResponse := func(useBefore bool) (HTTPResponse, error) {
			if useBefore && beforeResponse != nil {
				return *beforeResponse, nil
			}
			if pageResponse == nil {
				response, err := e.downloadPage(ctx, target, headers, variables)
				if err != nil {
					return HTTPResponse{}, err
				}
				pageResponse = &response
			}
			return *pageResponse, nil
		}
		if download.InfoHash != nil {
			response, err := selectResponse(download.InfoHash.UseBeforeResponse)
			if err != nil {
				return Artifact{}, err
			}
			hash, found, err := e.downloadField(ctx, response, download.InfoHash.Hash, variables)
			if err != nil {
				return Artifact{}, fmt.Errorf("download infohash selector failed: %w", err)
			}
			if !found {
				return Artifact{}, fmt.Errorf("download infohash selector did not match")
			}
			title, found, err := e.downloadField(ctx, response, download.InfoHash.Title, variables)
			if err != nil {
				return Artifact{}, fmt.Errorf("download title selector failed: %w", err)
			}
			if !found {
				return Artifact{}, fmt.Errorf("download title selector did not match")
			}
			return magnetArtifact(generatedMagnetURI(hash, title))
		}
		if len(download.Selectors) > 0 {
			for _, selector := range download.Selectors {
				response, err := selectResponse(selector.UseBeforeResponse)
				if err != nil {
					return Artifact{}, err
				}
				href, found, err := e.downloadField(ctx, response, selector, variables)
				if err != nil && ctx.Err() != nil {
					return Artifact{}, ctx.Err()
				}
				if err != nil || !found || href == "" {
					continue
				}
				resolved, err := resolveURL(link, href, true)
				if err != nil {
					continue
				}
				artifact, err := e.fetchArtifact(ctx, resolved, method, target, headers, variables)
				if err == nil {
					return artifact, nil
				}
				// A timeout or HTTP failure is not evidence that a grab did
				// not happen. Only a confirmed non-torrent response permits
				// trying a different declared selector in this operation.
				if !errors.Is(err, ErrNotTorrent) {
					return Artifact{}, err
				}
				if e.plan.Definition.TestLinkTorrent != nil && !*e.plan.Definition.TestLinkTorrent {
					return Artifact{}, err
				}
				if ctx.Err() != nil {
					return Artifact{}, ctx.Err()
				}
			}
			return Artifact{}, fmt.Errorf("download selectors did not yield a torrent or magnet")
		}
	}
	return e.fetchArtifact(ctx, target, method, result.DetailsURL, headers, variables)
}

func (e *Execution) downloadPage(ctx context.Context, target string, headers map[string][]string, variables Variables) (HTTPResponse, error) {
	rendered, err := e.headers(ctx, headers, variables)
	if err != nil {
		return HTTPResponse{}, err
	}
	return e.request(ctx, Request{URL: target, Method: http.MethodGet, Headers: rendered, FollowRedirect: true})
}

func (e *Execution) downloadField(ctx context.Context, response HTTPResponse, selector DownloadSelector, variables Variables) (string, bool, error) {
	text, err := e.responseText(response)
	if err != nil {
		return "", false, err
	}
	doc, err := parseMarkup(text, false)
	if err != nil {
		return "", false, err
	}
	selected := selector.Selector
	selected.Optional = true
	return HTMLField(ctx, doc.Selection, selected, FilterContext{Variables: variables, Now: e.now, Encoding: e.plan.Definition.Encoding})
}

func (e *Execution) fetchArtifact(ctx context.Context, target, method, referer string, headers map[string][]string, variables Variables) (Artifact, error) {
	for redirects := 0; redirects <= 5; redirects++ {
		if strings.HasPrefix(target, "magnet:") {
			return magnetArtifact(target)
		}
		rendered, err := e.headers(ctx, headers, variables)
		if err != nil {
			return Artifact{}, err
		}
		if referer != "" {
			rendered.Set("Referer", referer)
		} else if rendered.Get("Referer") == "" {
			rendered.Set("Referer", target)
		}
		response, err := e.request(ctx, Request{URL: target, Method: method, Headers: rendered, Torrent: true})
		if err != nil {
			return Artifact{}, err
		}
		if response.Status >= 300 && response.Status < 400 {
			location := response.Headers.Get("Location")
			if location == "" {
				return Artifact{}, fmt.Errorf("download redirect has no location")
			}
			base, err := url.Parse(response.URL)
			if err != nil {
				return Artifact{}, fmt.Errorf("invalid download redirect")
			}
			target, err = resolveURL(base, location, true)
			if err != nil {
				return Artifact{}, err
			}
			if response.Status == http.StatusSeeOther {
				method = http.MethodGet
			}
			continue
		}
		if strings.Contains(strings.ToLower(response.Headers.Get("Content-Type")), "text/html") || len(response.Body) == 0 || response.Body[0] != 'd' {
			return Artifact{}, ErrNotTorrent
		}
		return Artifact{TorrentBytes: append([]byte(nil), response.Body...)}, nil
	}
	return Artifact{}, fmt.Errorf("download redirect budget exceeded")
}

func magnetArtifact(value string) (Artifact, error) {
	magnet, err := torrentmeta.ParseMagnet(value)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Magnet: magnet.URI}, nil
}

func addDownloadVariables(variables Variables, link *url.URL) {
	port := link.Port()
	if port == "" {
		port = "443"
		if link.Scheme == "http" {
			port = "80"
		}
	}
	query := ""
	if link.RawQuery != "" {
		query = "?" + link.RawQuery
	}
	values := map[string]string{"AbsoluteUri": link.String(), "AbsolutePath": link.EscapedPath(), "Scheme": link.Scheme, "Host": link.Hostname(), "Port": port, "PathAndQuery": link.EscapedPath() + query, "Query": query}
	for key, value := range values {
		variables[".DownloadUri."+key] = TemplateValue{Text: value}
	}
	for key, value := range link.Query() {
		if len(value) > 0 {
			variables[".DownloadUri.Query."+key] = TemplateValue{Text: value[0]}
		}
	}
}
