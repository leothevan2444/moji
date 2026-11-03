package cardigann

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/indexercategory"
	"github.com/leothevan2444/moji/internal/indexerquery"
)

type Result struct {
	Title       string
	DownloadURL string
	MagnetURI   string
	InfoHash    string
	DetailsURL  string
	Description string
	PosterURL   string
	Categories  []string
	Size        int64
	Seeders     *int
	Leechers    *int
	PublishedAt *time.Time
	Fields      map[string]string
}

type RowIssue struct {
	Field string
	Code  string
}
type SearchResult struct {
	Results       []Result
	Issues        []RowIssue
	Truncated     bool
	ReportedCount *int64
}

func (e *Execution) Search(ctx context.Context, query Query) (SearchResult, error) {
	parameters, err := indexerquery.NormalizeParameters(query.Parameters)
	if err != nil {
		return SearchResult{}, queryError(err.Error())
	}
	mode := query.Mode
	if mode == "" {
		mode = "search"
	}
	capability := map[string]string{"search": "search", "movie": "movie-search", "tvsearch": "tv-search", "music": "music-search", "book": "book-search"}[mode]
	if _, ok := e.plan.Definition.Caps.Modes[capability]; !ok || capability == "" {
		return SearchResult{}, fmt.Errorf("query mode is not supported by this definition")
	}
	variables := e.variables.Clone()
	variables[".Query.Type"] = TemplateValue{Text: mode}
	variables[".Query.Q"] = TemplateValue{Text: query.Keywords}
	if query.Limit != nil {
		variables[".Query.Limit"] = TemplateValue{Text: strconv.Itoa(*query.Limit)}
	}
	if query.Offset != nil {
		variables[".Query.Offset"] = TemplateValue{Text: strconv.Itoa(*query.Offset)}
	}
	originalCategories := make([]string, len(query.Categories))
	for i, category := range query.Categories {
		originalCategories[i] = strconv.Itoa(category)
	}
	variables[".Query.Categories"] = TemplateValue{Items: originalCategories}
	for key, value := range parameters {
		variables[".Query."+key] = TemplateValue{Text: value}
	}
	if err := applyNumericQueryFields(variables); err != nil {
		return SearchResult{}, queryError(err.Error())
	}
	if err := applyEpisodeQuery(variables); err != nil {
		return SearchResult{}, queryError(err.Error())
	}
	applyDerivedQueryFields(variables, mode)
	var tokens []string
	for _, key := range []string{"Q", "Series", "Movie", "Year", "Episode"} {
		if value := variables[".Query."+key].Text; strings.TrimSpace(value) != "" {
			tokens = append(tokens, value)
		}
	}
	combined := strings.Join(tokens, " ")
	variables[".Query.Keywords"] = TemplateValue{Text: combined}
	keywords, err := ApplyFilters(ctx, combined, e.plan.Definition.Search.KeywordsFilters, FilterContext{Variables: variables, Now: e.now, Encoding: e.plan.Definition.Encoding})
	if err != nil {
		return SearchResult{}, err
	}
	variables[".Keywords"] = TemplateValue{Text: keywords}
	paths := e.plan.Definition.Search.Paths
	if len(paths) == 0 && e.plan.Definition.Search.Path != "" {
		paths = []SearchPath{{RequestBlock: RequestBlock{Path: e.plan.Definition.Search.Path}}}
	}
	categories := e.mapCategories(query.Categories)
	seenRequests, seenResults := map[string]bool{}, map[string]bool{}
	var result SearchResult
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		selected, eligible := pathCategories(categories, path.Categories)
		if !eligible {
			continue
		}
		local := variables.Clone()
		local[".Categories"] = TemplateValue{Items: selected}
		// Inheritance appends parameter groups; a local key does not replace
		// the shared key. Servers may intentionally consume repeated values.
		var inputs []requestInputGroup
		if path.InheritInputs == nil || *path.InheritInputs {
			inputs = append(inputs, requestInputGroup{values: e.plan.Definition.Search.Inputs, expandRaw: true})
		}
		inputs = append(inputs, requestInputGroup{values: path.Inputs, expandRaw: true})
		request, err := e.requestFromInputGroups(ctx, path.RequestBlock, local, e.plan.Definition.Search.Headers, e.base, e.plan.Definition.Search.AllowEmptyInputs, true, inputs)
		if err != nil {
			return result, err
		}
		request.FollowRedirect = path.FollowRedirect || e.plan.Definition.FollowRedirect
		identity := request.Method + "\n" + request.URL + "\n" + string(request.Body)
		if request.Method == http.MethodGet && seenRequests[identity] {
			continue
		}
		seenRequests[identity] = true
		response, err := e.request(ctx, request)
		if err != nil {
			if e.plan.Definition.Login != nil && (response.Status == http.StatusUnauthorized || response.Status == http.StatusForbidden) {
				return result, ErrAuthentication
			}
			return result, err
		}
		if err := e.checkSearchAuthentication(ctx, response); err != nil {
			return result, err
		}
		page, err := e.parseSearchResponse(ctx, response, path.Response, local)
		if err != nil {
			return result, err
		}
		result.Issues = append(result.Issues, page.Issues...)
		result.Truncated = result.Truncated || page.Truncated
		if page.ReportedCount != nil {
			result.ReportedCount = page.ReportedCount
		}
		for _, release := range page.Results {
			identity := release.DownloadURL + "\n" + release.MagnetURI + "\n" + release.Title
			if seenResults[identity] {
				continue
			}
			seenResults[identity] = true
			if len(result.Results) >= MaxSearchResults {
				result.Truncated = true
				continue
			}
			result.Results = append(result.Results, release)
		}
	}
	if query.Limit != nil && *query.Limit > 0 && len(result.Results) > *query.Limit {
		result.Results = result.Results[:*query.Limit]
		result.Truncated = true
	}
	return result, nil
}

func (e *Execution) mapCategories(requested []int) []string {
	var mapped, defaults []string
	seen := map[string]bool{}
	add := func(value string) {
		if !seen[value] {
			mapped = append(mapped, value)
			seen[value] = true
		}
	}
	match := func(category string) bool {
		for _, wanted := range requested {
			name, known := indexercategory.Name(wanted)
			if known && (category == name || strings.HasPrefix(category, name+"/")) {
				return true
			}
		}
		return false
	}
	for _, category := range e.plan.Definition.Caps.Categories {
		if match(string(category.Value)) {
			add(category.Name)
		}
	}
	for _, category := range e.plan.Definition.Caps.CategoryMappings {
		if category.Default {
			defaults = append(defaults, string(category.ID))
		}
		if match(category.Cat) {
			add(string(category.ID))
		}
	}
	if len(mapped) > 0 {
		return mapped
	}
	return defaults
}

func pathCategories(categories []string, restricted []Scalar) ([]string, bool) {
	if len(restricted) == 0 || len(categories) == 0 {
		return categories, true
	}
	negative := restricted[0] == "!"
	var intersection []string
	for _, category := range categories {
		found := false
		for _, r := range restricted {
			if category == string(r) {
				found = true
			}
		}
		if found {
			intersection = append(intersection, category)
		}
	}
	if negative {
		return intersection, len(intersection) == 0
	}
	return intersection, len(intersection) > 0
}

func (e *Execution) applyField(result *Result, name, value string, modifiers []string, base *url.URL) (string, error) {
	has := func(modifier string) bool {
		for _, m := range modifiers {
			if m == modifier {
				return true
			}
		}
		return false
	}
	if has("append") && name != "category" && name != "categorydesc" {
		value = result.Fields[name] + value
	}
	switch name {
	case "title":
		result.Title = value
	case "description":
		result.Description = value
	case "download", "details", "poster", "comments":
		if value == "" {
			break
		}
		resolved, err := resolveURL(base, value, name == "download")
		if err != nil {
			return "", err
		}
		value = resolved
		switch name {
		case "download":
			if strings.HasPrefix(value, "magnet:") {
				result.MagnetURI = value
			} else {
				result.DownloadURL = value
			}
		case "details":
			result.DetailsURL = value
		case "poster":
			result.PosterURL = value
		}
	case "magnet":
		if value != "" {
			parsed, err := url.Parse(value)
			if err != nil || parsed.Scheme != "magnet" {
				return "", fmt.Errorf("invalid magnet value")
			}
			result.MagnetURI = value
		}
	case "infohash":
		result.InfoHash = value
	case "size":
		size, err := parseSize(value)
		if err != nil {
			return "", err
		}
		result.Size = size
		value = strconv.FormatInt(size, 10)
	case "seeders", "leechers":
		count, err := parseCount(value)
		if err != nil {
			return "", err
		}
		if count >= 5000000 {
			count = 0
		}
		if name == "seeders" {
			result.Seeders = &count
		} else {
			result.Leechers = &count
		}
		value = strconv.Itoa(count)
	case "date":
		parsed, err := parseRuleDate(value, "", e.now)
		if err != nil {
			return "", err
		}
		result.PublishedAt = &parsed
		value = formatFilterDate(parsed)
	case "category", "categorydesc":
		var mapped []string
		if name == "category" {
			for _, category := range e.plan.Definition.Caps.Categories {
				if strings.EqualFold(category.Name, value) {
					mapped = append(mapped, string(category.Value))
				}
			}
		}
		for _, category := range e.plan.Definition.Caps.CategoryMappings {
			if name == "category" && strings.EqualFold(string(category.ID), value) || name == "categorydesc" && strings.EqualFold(category.Desc, value) {
				mapped = append(mapped, category.Cat)
			}
		}
		if len(mapped) > 0 && has("noappend") {
			result.Categories = nil
		}
		for _, category := range mapped {
			found := false
			for _, old := range result.Categories {
				if old == category {
					found = true
				}
			}
			if !found {
				result.Categories = append(result.Categories, category)
			}
		}
	}
	result.Fields[name] = value
	return value, nil
}

func parseCount(value string) (int, error) {
	var text strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			text.WriteRune(r)
		} else if r != ' ' && r != ',' && r != '.' && r != '\u00a0' && r != '+' {
			return 0, fmt.Errorf("invalid count")
		}
	}
	if text.Len() == 0 {
		return 0, fmt.Errorf("missing count")
	}
	n, err := strconv.ParseInt(text.String(), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("count exceeds range")
	}
	return int(n), nil
}

func parseSize(value string) (int64, error) {
	text := strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(value, "\u00a0", " ")))
	units := []struct {
		name   string
		factor float64
	}{{"TIB", 1 << 40}, {"TB", 1 << 40}, {"GIB", 1 << 30}, {"GB", 1 << 30}, {"MIB", 1 << 20}, {"MB", 1 << 20}, {"KIB", 1 << 10}, {"KB", 1 << 10}, {"BYTES", 1}, {"BYTE", 1}, {"B", 1}}
	factor := float64(1)
	for _, unit := range units {
		if strings.HasSuffix(text, unit.name) {
			factor = unit.factor
			text = strings.TrimSpace(strings.TrimSuffix(text, unit.name))
			break
		}
	}
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, " ", "")
	number, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number*factor > float64(1<<62) {
		return 0, fmt.Errorf("invalid size")
	}
	return int64(number * factor), nil
}
