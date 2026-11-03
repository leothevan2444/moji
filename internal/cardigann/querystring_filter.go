package cardigann

import (
	"strings"
	"unicode/utf8"
)

func queryStringFilter(data, argument string) string {
	_, query, found := strings.Cut(data, "?")
	if !found {
		return ""
	}
	query, _, _ = strings.Cut(query, "#")
	for query != "" {
		part, rest, more := strings.Cut(query, "&")
		if part != "" {
			name, value, _ := strings.Cut(part, "=")
			// QueryHelpers keeps the first value of a case-insensitive key.
			if filterUpper(decodeQueryPart(name)) == filterUpper(argument) {
				return decodeQueryPart(value)
			}
		}
		if !more {
			break
		}
		query = rest
	}
	return ""
}

// QueryHelpers uses URI unescaping: malformed escapes and invalid UTF-8 bytes
// retain their original percent spelling. This differs from form decoding.
func decodeQueryPart(value string) string {
	var out strings.Builder
	out.Grow(len(value))
	for i := 0; i < len(value); {
		if value[i] == '+' {
			out.WriteByte(' ')
			i++
			continue
		}
		start := i
		var escaped []byte
		for i+2 < len(value) && value[i] == '%' {
			high, low := hexNibble(value[i+1]), hexNibble(value[i+2])
			if high < 0 || low < 0 {
				break
			}
			escaped = append(escaped, byte(high<<4|low))
			i += 3
		}
		if len(escaped) == 0 {
			out.WriteByte(value[i])
			i++
			continue
		}
		for offset := 0; offset < len(escaped); {
			r, size := utf8.DecodeRune(escaped[offset:])
			if r == utf8.RuneError && size == 1 {
				out.WriteString(value[start+offset*3 : start+(offset+1)*3])
			} else {
				out.Write(escaped[offset : offset+size])
			}
			offset += size
		}
	}
	return out.String()
}

func hexNibble(value byte) int {
	switch {
	case value >= '0' && value <= '9':
		return int(value - '0')
	case value >= 'a' && value <= 'f':
		return int(value-'a') + 10
	case value >= 'A' && value <= 'F':
		return int(value-'A') + 10
	default:
		return -1
	}
}
