package cardigann

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func splitFilter(data, separator string, index int) (string, error) {
	first, _ := utf8.DecodeRuneInString(separator)
	if first <= 0xffff {
		parts := strings.Split(data, string(first))
		if index < 0 {
			index += len(parts)
		}
		if index < 0 || index >= len(parts) {
			return "", fmt.Errorf("split position out of range")
		}
		return parts[index], nil
	}
	// C# sep[0] is one UTF-16 code unit, including when the visible separator
	// occupies a surrogate pair. Split cannot use the entire first rune.
	separatorUnit := utf16.Encode([]rune{first})[0]
	units := utf16.Encode([]rune(data))
	parts := [][]uint16{}
	start := 0
	for i, unit := range units {
		if unit == separatorUnit {
			parts = append(parts, units[start:i])
			start = i + 1
		}
	}
	parts = append(parts, units[start:])
	if index < 0 {
		index += len(parts)
	}
	if index < 0 || index >= len(parts) {
		return "", fmt.Errorf("split position out of range")
	}
	return filterUTF16Text(parts[index])
}

func trimFilter(data, cutset string) (string, error) {
	if cutset == "" {
		return "", fmt.Errorf("trim character is empty")
	}
	first, _ := utf8.DecodeRuneInString(cutset)
	if first <= 0xffff {
		return strings.Trim(data, string(first)), nil
	}
	unit := utf16.Encode([]rune{first})[0]
	units := utf16.Encode([]rune(data))
	for len(units) > 0 && units[0] == unit {
		units = units[1:]
	}
	for len(units) > 0 && units[len(units)-1] == unit {
		units = units[:len(units)-1]
	}
	return filterUTF16Text(units)
}

func filterUTF16Text(units []uint16) (string, error) {
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u >= 0xd800 && u <= 0xdbff {
			if i+1 == len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
				return "", fmt.Errorf("filter produced an unpaired UTF-16 surrogate")
			}
			i++
		} else if u >= 0xdc00 && u <= 0xdfff {
			return "", fmt.Errorf("filter produced an unpaired UTF-16 surrogate")
		}
	}
	return string(utf16.Decode(units)), nil
}

// WebUtility.HtmlEncode uses named quotation entities and numeric Latin-1 and
// supplementary-plane entities; Go's html.EscapeString follows other choices.
func htmlEncodeFilter(data string) string {
	var out strings.Builder
	out.Grow(len(data))
	for _, r := range data {
		switch r {
		case '&':
			out.WriteString("&amp;")
		case '<':
			out.WriteString("&lt;")
		case '>':
			out.WriteString("&gt;")
		case '"':
			out.WriteString("&quot;")
		case '\'':
			out.WriteString("&#39;")
		default:
			if r >= 0xa0 && r <= 0xff || r > 0xffff {
				out.WriteString("&#")
				out.WriteString(strconv.FormatInt(int64(r), 10))
				out.WriteByte(';')
			} else {
				out.WriteRune(r)
			}
		}
	}
	return out.String()
}
