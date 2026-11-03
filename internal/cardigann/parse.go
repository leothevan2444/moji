package cardigann

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/encoding/unicode"
)

var optionalResultFields = map[string]bool{"description": true, "imdbid": true, "tmdbid": true, "rageid": true, "tvdbid": true, "tvmazeid": true, "traktid": true, "doubanid": true, "genre": true, "year": true, "poster": true, "author": true, "booktitle": true, "publisher": true, "album": true, "artist": true, "label": true, "track": true, "minimumratio": true, "minimumseedtime": true, "downloadvolumefactor": true, "uploadvolumefactor": true}

func (e *Execution) parseSearchResponse(ctx context.Context, response HTTPResponse, format Response, variables Variables) (SearchResult, error) {
	var result SearchResult
	text, err := e.responseText(response)
	if err != nil {
		return result, err
	}
	fc := FilterContext{Variables: variables, Now: e.now, Encoding: e.plan.Definition.Encoding}
	text, err = ApplyFilters(ctx, text, e.plan.Definition.Search.PreprocessingFilters, fc)
	if err != nil {
		return result, err
	}
	if format.NoResultsMessage != "" && strings.Contains(text, format.NoResultsMessage) {
		return result, nil
	}
	base, err := url.Parse(response.URL)
	if err != nil {
		return result, fmt.Errorf("invalid response location")
	}
	rows := e.plan.Definition.Search.Rows
	query, err := e.render(ctx, rows.Selector.Selector, variables, nil)
	if err != nil {
		return result, err
	}
	if format.Type == "json" {
		var root any
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		if err := decoder.Decode(&root); err != nil {
			return result, fmt.Errorf("invalid JSON response")
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			return result, fmt.Errorf("trailing JSON response content")
		}
		remaining := 50000
		if err := checkJSONBudget(root, 0, &remaining); err != nil {
			return result, err
		}
		// Upstream uses count only for JSON, before selecting rows. Missing,
		// invalid and failed count selectors do not invalidate otherwise usable
		// rows; a valid Int32 below one is an explicit empty-result signal.
		if rows.Count != nil {
			value, found, countErr := JSONField(ctx, root, *rows.Count, fc)
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if countErr == nil && found {
				if count, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil {
					if count >= 0 {
						result.ReportedCount = &count
					}
					if count < 1 {
						return result, nil
					}
				}
			}
		}
		selector, err := parseJSONSelector(query)
		if err != nil {
			return result, err
		}
		selected, err := selector.selectValues(root)
		if err != nil {
			return result, err
		}
		if len(selected) == 0 {
			if rows.Optional || rows.MissingAttributeEqualsNoResults {
				return result, nil
			}
			return result, fmt.Errorf("required JSON rows selector did not match")
		}
		var items []any
		for _, value := range selected {
			if array, ok := value.([]any); ok {
				items = append(items, array...)
			} else {
				items = append(items, value)
			}
		}
		for _, item := range items {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			value := item
			if rows.Attribute != "" {
				attribute, err := parseJSONSelector(rows.Attribute)
				if err != nil {
					return result, err
				}
				values, err := attribute.selectValues(item)
				if err != nil {
					return result, err
				}
				if len(values) == 0 {
					if rows.MissingAttributeEqualsNoResults {
						continue
					}
					return result, fmt.Errorf("required JSON row attribute did not match")
				}
				value = values[0]
			}
			values := []any{value}
			if rows.Multiple {
				values = nil
				switch typed := value.(type) {
				case []any:
					values = typed
				case map[string]any:
					for _, child := range typed {
						values = append(values, child)
					}
				default:
					return result, fmt.Errorf("multiple JSON row requires an array or object")
				}
			}
			for _, value := range values {
				release, issues, ok := e.parseFields(ctx, variables, base, func(selector Selector, fc FilterContext) (string, bool, error) {
					parent := value
					if strings.HasPrefix(selector.Selector, "..") {
						parent = item
					}
					return JSONField(ctx, parent, selector, fc)
				})
				result.Issues = append(result.Issues, issues...)
				if ok {
					appendSearchResult(&result, release)
				}
			}
		}
		return result, nil
	}
	doc, err := parseMarkup(text, format.Type == "xml")
	if err != nil {
		return result, err
	}
	if err := e.checkErrors(ctx, doc, e.plan.Definition.Search.Error, variables); err != nil {
		return result, err
	}
	selected, err := selectHTML(doc.Selection, query, true)
	if err != nil {
		return result, err
	}
	if rows.After < 0 {
		return result, fmt.Errorf("negative row merge count")
	}
	var elements []*goquery.Selection
	selected.Each(func(_ int, row *goquery.Selection) { elements = append(elements, row) })
	for i := 0; i < len(elements); i += rows.After + 1 {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		row := elements[i]
		if i+rows.After >= len(elements) {
			result.Issues = append(result.Issues, RowIssue{Field: "rows", Code: "INCOMPLETE_ROW_GROUP"})
			break
		}
		for j := 1; j <= rows.After; j++ {
			for _, node := range elements[i+j].Nodes {
				for node.FirstChild != nil {
					child := node.FirstChild
					node.RemoveChild(child)
					row.Nodes[0].AppendChild(child)
				}
			}
		}
		release, issues, ok := e.parseFields(ctx, variables, base, func(selector Selector, fc FilterContext) (string, bool, error) {
			return HTMLField(ctx, row, selector, fc)
		})
		result.Issues = append(result.Issues, issues...)
		if ok && release.PublishedAt == nil && rows.DateHeaders != nil {
			for previous, steps := previousElement(row), 0; previous.Length() > 0 && steps < 1000; previous, steps = previousElement(previous), steps+1 {
				date, found, err := HTMLField(ctx, previous, *rows.DateHeaders, fc)
				if err != nil || !found {
					continue
				}
				parsed, err := parseFuzzyDate(date, e.now)
				if err != nil {
					result.Issues = append(result.Issues, RowIssue{Field: "dateheaders", Code: "FIELD_VALUE_INVALID"})
					ok = false
				} else {
					release.PublishedAt = &parsed
				}
				// The nearest matching header owns the date. An invalid value
				// must not borrow a valid date from an earlier group.
				break
			}
			if ok && release.PublishedAt == nil && !rows.DateHeaders.Optional {
				result.Issues = append(result.Issues, RowIssue{Field: "dateheaders", Code: "REQUIRED_FIELD_MISSING"})
				ok = false
			}
		}
		if ok {
			appendSearchResult(&result, release)
		}
	}
	return result, nil
}

func appendSearchResult(result *SearchResult, release Result) {
	if len(result.Results) >= MaxSearchResults {
		result.Truncated = true
		return
	}
	result.Results = append(result.Results, release)
}

func (e *Execution) parseFields(ctx context.Context, variables Variables, base *url.URL, selectField func(Selector, FilterContext) (string, bool, error)) (Result, []RowIssue, bool) {
	result := Result{Fields: map[string]string{}}
	local := variables.Clone()
	var issues []RowIssue
	for _, field := range e.plan.Definition.Search.Fields {
		parts := strings.Split(field.Name, "|")
		name := parts[0]
		selector := field.Selector
		if optionalResultFields[name] {
			selector.Optional = true
		}
		for _, modifier := range parts[1:] {
			if modifier == "optional" {
				selector.Optional = true
			}
		}
		value, found, err := selectField(selector, FilterContext{Variables: local, Now: e.now, Encoding: e.plan.Definition.Encoding})
		if err != nil || !found {
			local[".Result."+name] = TemplateValue{}
			if !selector.Optional {
				issues = append(issues, RowIssue{Field: name, Code: "FIELD_PARSE_FAILED"})
			}
			continue
		}
		value, err = e.applyField(&result, name, value, parts[1:], base)
		if err != nil {
			local[".Result."+name] = TemplateValue{}
			if !selector.Optional {
				issues = append(issues, RowIssue{Field: name, Code: "FIELD_VALUE_INVALID"})
			}
			continue
		}
		local[".Result."+name] = TemplateValue{Text: value}
	}
	if result.Title == "" || result.DownloadURL == "" && result.MagnetURI == "" && result.InfoHash == "" {
		return result, issues, false
	}
	if matches, err := e.matchesRowFilters(result, local); err != nil {
		issues = append(issues, RowIssue{Field: "rows", Code: "ROW_FILTER_FAILED"})
		return result, issues, false
	} else if !matches {
		return result, issues, false
	}
	if result.MagnetURI == "" && result.InfoHash != "" && e.plan.Definition.Type != "private" {
		result.MagnetURI = generatedMagnetURI(result.InfoHash, result.Title)
	}
	if result.InfoHash == "" && result.MagnetURI != "" {
		if magnet, err := url.Parse(result.MagnetURI); err == nil {
			xt := magnet.Query().Get("xt")
			if strings.HasPrefix(strings.ToLower(xt), "urn:btih:") {
				result.InfoHash = xt[len("urn:btih:"):]
			}
		}
	}
	return result, issues, true
}

func (e *Execution) responseText(response HTTPResponse) (string, error) {
	codec, err := declaredEncoding(e.plan.Definition.Encoding)
	if err != nil {
		return "", fmt.Errorf("unknown response encoding")
	}
	if codec == nil {
		codec = unicode.UTF8
	}
	reader := codec.NewDecoder().Reader(bytes.NewReader(response.Body))
	data, err := io.ReadAll(io.LimitReader(reader, MaxPageBytes+1))
	if err != nil || len(data) > MaxPageBytes {
		return "", fmt.Errorf("decoded response exceeds byte budget")
	}
	return string(data), nil
}

func parseMarkup(text string, isXML bool) (*goquery.Document, error) {
	if !isXML {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
		if err != nil {
			return nil, err
		}
		remaining := 50000
		for _, node := range doc.Nodes {
			if err := checkMarkupBudget(node, 0, &remaining); err != nil {
				return nil, err
			}
		}
		return doc, nil
	}
	decoder := xml.NewDecoder(strings.NewReader(text))
	// responseText has already decoded the response bytes. An XML declaration
	// describes those original bytes and must not reinterpret this UTF-8 text.
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	root := &html.Node{Type: html.DocumentNode}
	stack := []*html.Node{root}
	count := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid XML response")
		}
		count++
		if count > 100000 || len(stack) > 128 {
			return nil, fmt.Errorf("XML node/depth budget exceeded")
		}
		switch value := token.(type) {
		case xml.StartElement:
			node := &html.Node{Type: html.ElementNode, Data: value.Name.Local, DataAtom: atom.Lookup([]byte(value.Name.Local))}
			for _, attribute := range value.Attr {
				node.Attr = append(node.Attr, html.Attribute{Key: attribute.Name.Local, Val: attribute.Value})
			}
			stack[len(stack)-1].AppendChild(node)
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) < 2 {
				return nil, fmt.Errorf("invalid XML hierarchy")
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			stack[len(stack)-1].AppendChild(&html.Node{Type: html.TextNode, Data: string(value)})
		}
	}
	return goquery.NewDocumentFromNode(root), nil
}

func checkMarkupBudget(node *html.Node, depth int, remaining *int) error {
	*remaining--
	if depth > 128 || *remaining < 0 {
		return fmt.Errorf("HTML node/depth budget exceeded")
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := checkMarkupBudget(child, depth+1, remaining); err != nil {
			return err
		}
	}
	return nil
}

func previousElement(row *goquery.Selection) *goquery.Selection {
	if previous := row.Prev(); previous.Length() > 0 {
		return previous
	}
	// Cardigann checks the parent sibling once; it does not search arbitrary
	// ancestors, which can expose a header belonging to another page section.
	return row.Parent().Prev()
}

func checkJSONBudget(value any, depth int, remaining *int) error {
	*remaining--
	if depth > 64 || *remaining < 0 {
		return fmt.Errorf("JSON node/depth budget exceeded")
	}
	switch typed := value.(type) {
	case []any:
		for _, child := range typed {
			if err := checkJSONBudget(child, depth+1, remaining); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, child := range typed {
			if err := checkJSONBudget(child, depth+1, remaining); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Execution) checkErrors(ctx context.Context, doc *goquery.Document, blocks []ErrorBlock, variables Variables) error {
	for _, block := range blocks {
		if block.Selector == "" {
			continue
		}
		query, err := e.render(ctx, block.Selector, variables, nil)
		if err != nil {
			return err
		}
		selected, err := selectHTML(doc.Selection, query, false)
		if err != nil {
			return err
		}
		if selected.Length() > 0 {
			return fmt.Errorf("site reported an error condition")
		}
	}
	return nil
}
