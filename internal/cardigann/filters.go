package cardigann

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ohler55/ojg/jp"

	"golang.org/x/text/unicode/norm"
)

type FilterContext struct {
	Variables Variables
	Now       time.Time
	Encoding  string
}

var filterArities = map[string][2]int{
	"querystring": {1, 1}, "timeparse": {1, 1}, "dateparse": {1, 1}, "regexp": {1, 1},
	"re_replace": {2, 2}, "split": {2, 2}, "replace": {2, 2}, "trim": {0, 1}, "prepend": {1, 1}, "append": {1, 1},
	"tolower": {0, 0}, "toupper": {0, 0}, "urldecode": {0, 0}, "urlencode": {0, 0}, "htmldecode": {0, 0}, "htmlencode": {0, 0},
	"timeago": {0, 0}, "reltime": {0, 0}, "fuzzytime": {0, 0}, "validfilename": {0, 0}, "diacritics": {1, 1},
	"jsonjoinarray": {2, 2}, "hexdump": {0, 0}, "strdump": {0, 1}, "validate": {1, 1},
}

func ValidateFilter(filter Filter) error {
	arity, ok := filterArities[filter.Name]
	if !ok {
		return fmt.Errorf("unsupported v11 filter %q", filter.Name)
	}
	if len(filter.Args) < arity[0] || len(filter.Args) > arity[1] {
		return fmt.Errorf("filter %s requires %d..%d arguments", filter.Name, arity[0], arity[1])
	}
	if filter.Name == "regexp" || filter.Name == "re_replace" {
		if _, err := boundedRegex(string(filter.Args[0])); err != nil {
			return err
		}
	}
	if filter.Name == "split" {
		if string(filter.Args[0]) == "" {
			return fmt.Errorf("split separator is empty")
		}
		if _, err := strconv.Atoi(string(filter.Args[1])); err != nil {
			return fmt.Errorf("split position must be integer")
		}
	}
	if filter.Name == "diacritics" && filter.Args[0] != "replace" {
		return fmt.Errorf("unsupported diacritics operation")
	}
	if filter.Name == "jsonjoinarray" {
		if _, err := jp.ParseString(string(filter.Args[0])); err != nil {
			return fmt.Errorf("invalid jsonjoinarray path: %w", err)
		}
	}
	return nil
}

func ApplyFilters(ctx context.Context, data string, filters []Filter, fc FilterContext) (string, error) {
	for _, filter := range filters {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if len(data) > MaxTemplateBytes {
			return "", fmt.Errorf("filter input exceeds byte budget")
		}
		if err := ValidateFilter(filter); err != nil {
			return "", err
		}
		arg := func(i int) string {
			if i >= len(filter.Args) {
				return ""
			}
			return string(filter.Args[i])
		}
		renderArg := func(i int) (string, error) { return renderText(ctx, arg(i), fc.Variables) }
		switch filter.Name {
		case "querystring":
			data = queryStringFilter(data, arg(0))
		case "regexp":
			re, err := boundedRegex(arg(0))
			if err != nil {
				return "", err
			}
			m, err := re.FindStringMatch(data)
			if err != nil {
				return "", fmt.Errorf("regexp execution: %w", err)
			}
			data = ""
			if m != nil && m.GroupCount() > 1 {
				data = m.GroupByNumber(1).String()
			}
		case "re_replace":
			re, err := boundedRegex(arg(0))
			if err != nil {
				return "", err
			}
			replacement, err := renderArg(1)
			if err != nil {
				return "", err
			}
			data, err = re.Replace(data, replacement, -1, -1)
			if err != nil {
				return "", fmt.Errorf("re_replace execution: %w", err)
			}
		case "split":
			index, _ := strconv.Atoi(arg(1))
			var err error
			data, err = splitFilter(data, arg(0), index)
			if err != nil {
				return "", err
			}
		case "replace":
			if arg(0) == "" {
				return "", fmt.Errorf("replace source is empty")
			}
			replacement, err := renderArg(1)
			if err != nil {
				return "", err
			}
			data = strings.ReplaceAll(data, arg(0), replacement)
		case "trim":
			if len(filter.Args) == 0 {
				data = strings.TrimSpace(data)
			} else {
				var err error
				data, err = trimFilter(data, arg(0))
				if err != nil {
					return "", err
				}
			}
		case "prepend", "append":
			text, err := renderArg(0)
			if err != nil {
				return "", err
			}
			if filter.Name == "prepend" {
				data = text + data
			} else {
				data += text
			}
		case "tolower":
			data = filterLower(data)
		case "toupper":
			data = filterUpper(data)
		case "htmlencode":
			data = htmlEncodeFilter(data)
		case "htmldecode":
			data = html.UnescapeString(data)
		case "urlencode":
			encoded, err := encodeQueryValue(data, fc.Encoding)
			if err != nil {
				return "", err
			}
			data = encoded
		case "urldecode":
			decoded, err := url.QueryUnescape(data)
			if err != nil {
				return "", fmt.Errorf("URL decoding failed")
			}
			data = decoded
			enc, err := declaredEncoding(fc.Encoding)
			if err != nil {
				return "", err
			}
			if enc != nil {
				data, err = enc.NewDecoder().String(data)
				if err != nil {
					return "", fmt.Errorf("URL character decoding failed")
				}
			}
		case "dateparse", "timeparse":
			// The upstream filter leaves data unchanged when a date does not
			// parse. Final result conversion decides how to represent unknown.
			if parsed, err := parseRuleDate(data, arg(0), fc.Now); err == nil {
				data = formatFilterDate(parsed)
			}
		case "timeago", "reltime", "fuzzytime":
			parsed, err := parseRelativeDate(data, fc.Now)
			if err != nil && filter.Name == "fuzzytime" {
				parsed, err = parseRuleDate(data, "", fc.Now)
			}
			if err != nil {
				return "", err
			}
			data = formatFilterDate(parsed)
		case "validfilename":
			data = validFilenameFilter(data)
		case "diacritics":
			data = norm.NFC.String(strings.Map(func(r rune) rune {
				if unicode.Is(unicode.Mn, r) {
					return -1
				}
				return r
			}, norm.NFD.String(data)))
		case "jsonjoinarray":
			var value any
			if err := json.Unmarshal([]byte(data), &value); err != nil {
				return "", fmt.Errorf("jsonjoinarray requires JSON")
			}
			path, err := jp.ParseString(arg(0))
			if err != nil {
				return "", err
			}
			var parts []string
			for _, item := range path.Get(value) {
				if items, ok := item.([]any); ok {
					for _, item := range items {
						parts = append(parts, jsonScalarText(item))
					}
				} else {
					parts = append(parts, jsonScalarText(item))
				}
			}
			data = strings.Join(parts, arg(1))
		case "validate":
			allowed := wordTokens(arg(0))
			actual := wordTokens(data)
			var matches []string
			seen := map[string]bool{}
			for _, word := range allowed {
				for _, candidate := range actual {
					if word == candidate && !seen[word] {
						matches = append(matches, word)
						seen[word] = true
						break
					}
				}
			}
			data = strings.Join(matches, ",")
		case "hexdump", "strdump":
			// Debug-only filters preserve the value. Response bodies are never
			// emitted to logs by the interpreter, even in debug mode.
		}
	}
	if len(data) > MaxTemplateBytes {
		return "", fmt.Errorf("filter output exceeds byte budget")
	}
	return data, nil
}

func wordTokens(s string) []string {
	return strings.FieldsFunc(filterLower(s), func(r rune) bool { return strings.ContainsRune(", /)(.;[]\"|:", r) })
}

func jsonScalarText(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		if date, ok := jsonDateText(s); ok {
			return date
		}
		return s
	}
	if b, ok := value.(bool); ok {
		if b {
			return "True"
		}
		return "False"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
