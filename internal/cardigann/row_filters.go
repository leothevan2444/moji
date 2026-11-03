package cardigann

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func andMatchLimit(filter Filter) (int64, error) {
	if len(filter.Args) == 0 {
		return -1, nil
	}
	if len(filter.Args) != 1 {
		return 0, fmt.Errorf("andmatch accepts at most one argument")
	}
	limit, err := strconv.ParseInt(strings.TrimSpace(string(filter.Args[0])), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("andmatch character limit must be a 32-bit integer")
	}
	return limit, nil
}

// andmatch follows Jackett's declarative row filter, including its explicit
// character-limit argument. Prowlarr's host-level cleanup uses different rules.
func (e *Execution) matchesRowFilters(result Result, variables Variables) (bool, error) {
	title := strings.ToLower(result.Title)
	for _, filter := range e.plan.Definition.Search.Rows.Filters {
		if filter.Name != "andmatch" {
			continue // strdump is intentionally a no-op, not a response-body log.
		}
		limit, err := andMatchLimit(filter)
		if err != nil {
			return false, err
		}
		if e.andMatchUsesIdentifier(variables) {
			continue
		}
		query := variables[".Keywords"].Text
		if strings.TrimSpace(query) == "" {
			query = strings.TrimSpace(sanitizeAndMatchFallback(variables[".Query.Q"].Text) + " " + variables[".Query.Episode"].Text)
		}
		query = andMatchPrefix(query, limit)
		for _, word := range strings.FieldsFunc(query, func(r rune) bool {
			// .NET regex \w is defined over UTF-16 code units. Supplementary
			// characters form surrogate pairs, neither of which is a word char.
			return r > 0xFFFF || !(unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Pc, r))
		}) {
			word = strings.ToLower(word)
			if utf8.RuneCountInString(word) <= 1 || word == "and" || word == "the" || word == "an" {
				continue
			}
			if !strings.Contains(title, word) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (e *Execution) andMatchUsesIdentifier(variables Variables) bool {
	hasParameter := func(mode, parameter string) bool {
		for _, supported := range e.plan.Definition.Caps.Modes[mode] {
			if supported == parameter {
				return true
			}
		}
		return false
	}
	tvIMDB := e.plan.Definition.Caps.AllowTVSearchIMDB || hasParameter("tv-search", "imdbid")
	imdb := tvIMDB || hasParameter("movie-search", "imdbid")
	for _, field := range []struct {
		name      string
		supported bool
	}{
		{"IMDBID", imdb},
		{"TMDBID", hasParameter("movie-search", "tmdbid") || hasParameter("tv-search", "tmdbid")},
		{"TVDBID", hasParameter("tv-search", "tvdbid")},
		{"DoubanID", imdb}, {"TraktID", imdb},
		{"TVMazeID", tvIMDB}, {"TVRageID", tvIMDB},
	} {
		if _, present := variables[".Query."+field.name]; field.supported && present {
			return true
		}
	}
	return false
}

func andMatchPrefix(query string, limit int64) string {
	if limit <= 0 {
		return query
	}
	var units int64
	for offset, r := range query {
		width := int64(1)
		if r > 0xFFFF {
			width = 2
		}
		if units+width > limit {
			// A truncated high surrogate is a non-word delimiter in .NET;
			// dropping it preserves the same final word tokens.
			return query[:offset]
		}
		units += width
	}
	return query
}

func sanitizeAndMatchFallback(query string) string {
	var result strings.Builder
	previousDash := false
	for _, r := range query {
		if unicode.Is(unicode.Pd, r) {
			if !previousDash {
				result.WriteByte('-')
			}
			previousDash = true
			continue
		}
		previousDash = false
		if r == '\u0060' || r == '\u00B4' || r == '\u2018' || r == '\u2019' {
			r = '\''
		}
		if r <= 0xFFFF && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) || strings.ContainsRune("-._()@/'[]+%", r)) {
			result.WriteRune(r)
		}
	}
	return result.String()
}
