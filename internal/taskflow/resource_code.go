package taskflow

import (
	"regexp"
	"strings"
)

var domain = regexp.MustCompile(`(?i)[a-z0-9_-]+(?:\.[a-z0-9_-]+)*\.(?:com|net|org|xyz|cc|tv|me|cn|top|vip)(?:@)?`)
var codeToken = regexp.MustCompile(`(?i)([a-z]{2,10})[-_ ]?([0-9]{2,6})`)

func cleanResourceName(value string) string { return domain.ReplaceAllString(value, "") }
func codeMatches(value string) [][]string {
	value = cleanResourceName(value)
	matches := codeToken.FindAllStringSubmatchIndex(value, -1)
	result := [][]string{}
	for _, m := range matches {
		if m[0] > 0 && isAlnum(value[m[0]-1]) {
			continue
		}
		if m[1] < len(value) && value[m[1]] >= '0' && value[m[1]] <= '9' {
			continue
		}
		result = append(result, []string{strings.ToUpper(value[m[2]:m[3]]), value[m[4]:m[5]], value[m[1]:]})
	}
	return result
}
func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// suggestResourceCode rejects conflicting tokens instead of choosing the first.
func suggestResourceCode(value string) string {
	found := ""
	for _, m := range codeMatches(value) {
		code := m[0] + "-" + m[1]
		if found != "" && found != code {
			return ""
		}
		found = code
	}
	return found
}
