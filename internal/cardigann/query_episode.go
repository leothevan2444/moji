package cardigann

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Ep is the raw query field; Episode is the formatted search token. These are
// distinct variables in both Jackett and Prowlarr's Cardigann context.
func applyEpisodeQuery(variables Variables) error {
	seasonValue, hasSeason := variables[".Query.Season"]
	ep, hasEpisode := variables[".Query.Ep"]
	if !hasSeason && !hasEpisode {
		return nil
	}
	variables[".Query.Episode"] = TemplateValue{}
	variables[".Query.Season"] = TemplateValue{}
	if strings.TrimSpace(seasonValue.Text) == "" {
		return nil
	}
	season, err := strconv.ParseInt(strings.TrimSpace(seasonValue.Text), 10, 32)
	if err != nil {
		return fmt.Errorf("query season must be a 32-bit integer")
	}
	if season > 0 {
		variables[".Query.Season"] = TemplateValue{Text: strconv.FormatInt(season, 10)}
	}
	if season == 0 {
		return nil
	}
	if daily, err := time.Parse("2006 01/02", strconv.FormatInt(season, 10)+" "+ep.Text); err == nil {
		variables[".Query.Episode"] = TemplateValue{Text: daily.Format("2006.01.02")}
		return nil
	}
	formatted := "S" + episodeInteger(season)
	if strings.TrimSpace(ep.Text) != "" {
		number, numeric, err := coerceEpisodeInteger(ep.Text)
		if err != nil {
			return err
		}
		if numeric {
			formatted += "E" + episodeInteger(number)
		} else {
			formatted += "E" + ep.Text
		}
	}
	variables[".Query.Episode"] = TemplateValue{Text: formatted}
	return nil
}

func episodeInteger(value int64) string {
	if value < 0 {
		return "-" + episodeInteger(-value)
	}
	text := strconv.FormatInt(value, 10)
	if len(text) < 2 {
		text = "0" + text
	}
	return text
}

// ParseUtil.CoerceInt keeps digits and separators, treating a lone dot as a
// grouping separator. With both separator kinds, an all-zero fraction is valid.
// A format error keeps the raw episode; overflow is an error, not a suffix.
func coerceEpisodeInteger(value string) (int64, bool, error) {
	var normalized strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) || r == '.' || r == ',' {
			normalized.WriteRune(r)
		}
	}
	text := normalized.String()
	if text == "" {
		text = "0"
	}
	if !strings.Contains(text, ",") || !strings.Contains(text, ".") {
		text = strings.ReplaceAll(text, ".", ",")
	}
	if text[0] < '0' || text[0] > '9' {
		return 0, false, nil
	}
	if decimal := strings.IndexByte(text, '.'); decimal >= 0 && strings.Contains(text[decimal:], ",") {
		return 0, false, nil
	}
	text = strings.ReplaceAll(text, ",", "")
	parts := strings.Split(text, ".")
	if len(parts) > 2 {
		return 0, false, nil
	}
	if len(parts) == 2 {
		if strings.Trim(parts[1], "0") != "" {
			return 0, false, fmt.Errorf("query episode is not a 32-bit integer")
		}
	}
	number, err := strconv.ParseInt(parts[0], 10, 32)
	if errors.Is(err, strconv.ErrRange) {
		return 0, false, fmt.Errorf("query episode is not a 32-bit integer")
	}
	return number, err == nil, nil
}
