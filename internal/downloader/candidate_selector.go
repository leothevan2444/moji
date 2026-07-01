package downloader

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/pkg/jackett"
)

type defaultCandidateSelector struct{}

func (defaultCandidateSelector) Select(query string, results []jackett.SearchResult, cfg config.CandidateSelectionConfig) (jackett.SearchResult, error) {
	if !cfg.Enabled {
		cfg = config.DefaultCandidateSelectionConfig()
	}
	cfg = cfg.Effective()

	candidates := make([]rankedCandidate, 0, len(results))
	for index, result := range results {
		if preferredTorrentURL(result) == "" {
			continue
		}
		candidates = append(candidates, rankedCandidate{
			index:  index,
			result: result,
		})
	}
	if len(candidates) == 0 {
		return jackett.SearchResult{}, errors.New("downloader: no downloadable torrent candidate found")
	}

	compiled := compileSelectionRules(cfg.Rules)
	sort.SliceStable(candidates, func(i, j int) bool {
		return compareRankedCandidates(query, candidates[i], candidates[j], compiled) < 0
	})
	return candidates[0].result, nil
}

type rankedCandidate struct {
	index  int
	result jackett.SearchResult
}

type compiledRule struct {
	rule          config.CandidateSelectionRule
	regexMatchers []*regexp.Regexp
}

func compileSelectionRules(rules []config.CandidateSelectionRule) []compiledRule {
	out := make([]compiledRule, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		item := compiledRule{rule: rule}
		if rule.Type == config.CandidateSelectionRuleTypeTitleMatch {
			item.regexMatchers = make([]*regexp.Regexp, len(rule.TitleMatch.Clauses))
			for i, clause := range rule.TitleMatch.Clauses {
				if clause.PatternMode != config.TitleMatchPatternModeRegex {
					continue
				}
				re, err := regexp.Compile(clause.Pattern)
				if err != nil {
					continue
				}
				item.regexMatchers[i] = re
			}
		}
		out = append(out, item)
	}
	return out
}

func compareRankedCandidates(query string, left rankedCandidate, right rankedCandidate, rules []compiledRule) int {
	for _, rule := range rules {
		if cmp := compareByRule(query, left.result, right.result, rule); cmp != 0 {
			return cmp
		}
	}
	if left.index < right.index {
		return -1
	}
	if left.index > right.index {
		return 1
	}
	return 0
}

func compareByRule(query string, left jackett.SearchResult, right jackett.SearchResult, rule compiledRule) int {
	switch rule.rule.Type {
	case config.CandidateSelectionRuleTypeIndexerPreference:
		return compareInts(indexerPreferenceRank(left, rule.rule), indexerPreferenceRank(right, rule.rule), rule.rule.Direction)
	case config.CandidateSelectionRuleTypeTitleMatch:
		return compareInts(titleMatchRank(left.Title, rule), titleMatchRank(right.Title, rule), config.CandidateSelectionDirectionAsc)
	case config.CandidateSelectionRuleTypePublishDate:
		leftTime, leftOK := parsePublishDate(left.PublishDate)
		rightTime, rightOK := parsePublishDate(right.PublishDate)
		return compareTimes(leftTime, leftOK, rightTime, rightOK, rule.rule.Direction)
	case config.CandidateSelectionRuleTypeTitleSimilarity:
		return compareInts(titleSimilarityScore(query, left.Title), titleSimilarityScore(query, right.Title), config.CandidateSelectionDirectionDesc)
	case config.CandidateSelectionRuleTypeSeeders:
		return compareInts(left.Seeders, right.Seeders, rule.rule.Direction)
	case config.CandidateSelectionRuleTypeSize:
		return compareInt64s(left.Size, right.Size, rule.rule.Direction)
	default:
		return 0
	}
}

func indexerPreferenceRank(result jackett.SearchResult, rule config.CandidateSelectionRule) int {
	trackerID := strings.TrimSpace(result.TrackerID)
	if trackerID == "" {
		trackerID = strings.TrimSpace(result.Tracker)
	}
	for index, preferred := range rule.IndexerPreference.TrackerIDs {
		if strings.EqualFold(preferred, trackerID) || strings.EqualFold(preferred, result.Tracker) {
			return index
		}
	}
	return len(rule.IndexerPreference.TrackerIDs) + 1
}

func titleMatchRank(title string, rule compiledRule) int {
	if len(rule.rule.TitleMatch.Clauses) == 0 {
		return 1
	}
	normalized := strings.ToLower(title)
	for index, clause := range rule.rule.TitleMatch.Clauses {
		if matchesTitleClause(normalized, clause, rule.regexMatchers, index) {
			if clause.Effect == config.TitleMatchEffectAvoid {
				return len(rule.rule.TitleMatch.Clauses) + 2 + index
			}
			return index
		}
	}
	return len(rule.rule.TitleMatch.Clauses) + 1
}

func matchesTitleClause(title string, clause config.TitleMatchClause, regexMatchers []*regexp.Regexp, index int) bool {
	switch clause.PatternMode {
	case config.TitleMatchPatternModeRegex:
		if index >= len(regexMatchers) || regexMatchers[index] == nil {
			return false
		}
		return regexMatchers[index].MatchString(title)
	default:
		return strings.Contains(title, strings.ToLower(clause.Pattern))
	}
}

func parsePublishDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC(), true
	}
	if parsed, err := time.Parse(time.RFC1123Z, value); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func titleSimilarityScore(query string, title string) int {
	normQuery := normalizeSimilarityText(query)
	normTitle := normalizeSimilarityText(title)
	if normQuery == "" || normTitle == "" {
		return 0
	}
	if normTitle == normQuery {
		return 10000
	}
	if strings.HasPrefix(normTitle, normQuery) {
		return 9000
	}
	if strings.Contains(normTitle, normQuery) {
		return 8000
	}

	queryTokens := tokenizeSimilarity(normQuery)
	titleTokens := tokenizeSimilarity(normTitle)
	shared := sharedTokenCount(queryTokens, titleTokens)
	score := shared * 100
	score += commonPrefixLength(normQuery, normTitle) * 10
	score += lcsLength(normQuery, normTitle)
	return score
}

func normalizeSimilarityText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r):
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
		default:
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func tokenizeSimilarity(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}

func sharedTokenCount(left []string, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	seen := make(map[string]int, len(right))
	for _, token := range right {
		seen[token]++
	}
	count := 0
	for _, token := range left {
		if seen[token] > 0 {
			count++
			seen[token]--
		}
	}
	return count
}

func commonPrefixLength(left string, right string) int {
	max := len(left)
	if len(right) < max {
		max = len(right)
	}
	count := 0
	for i := 0; i < max; i++ {
		if left[i] != right[i] {
			break
		}
		count++
	}
	return count
}

func lcsLength(left string, right string) int {
	if left == "" || right == "" {
		return 0
	}
	prev := make([]int, len(right)+1)
	curr := make([]int, len(right)+1)
	for i := 1; i <= len(left); i++ {
		for j := 1; j <= len(right); j++ {
			if left[i-1] == right[j-1] {
				curr[j] = prev[j-1] + 1
			} else if curr[j-1] > prev[j] {
				curr[j] = curr[j-1]
			} else {
				curr[j] = prev[j]
			}
		}
		copy(prev, curr)
		clear(curr)
	}
	return prev[len(right)]
}

func compareInts(left int, right int, direction config.CandidateSelectionDirection) int {
	switch direction {
	case config.CandidateSelectionDirectionAsc:
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	default:
		switch {
		case left > right:
			return -1
		case left < right:
			return 1
		default:
			return 0
		}
	}
}

func compareInt64s(left int64, right int64, direction config.CandidateSelectionDirection) int {
	switch direction {
	case config.CandidateSelectionDirectionAsc:
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	default:
		switch {
		case left > right:
			return -1
		case left < right:
			return 1
		default:
			return 0
		}
	}
}

func compareTimes(left time.Time, leftOK bool, right time.Time, rightOK bool, direction config.CandidateSelectionDirection) int {
	switch {
	case leftOK && !rightOK:
		return -1
	case !leftOK && rightOK:
		return 1
	case !leftOK && !rightOK:
		return 0
	}
	switch direction {
	case config.CandidateSelectionDirectionAsc:
		switch {
		case left.Before(right):
			return -1
		case left.After(right):
			return 1
		default:
			return 0
		}
	default:
		switch {
		case left.After(right):
			return -1
		case left.Before(right):
			return 1
		default:
			return 0
		}
	}
}
