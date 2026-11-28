// Package ordering provides the shared title/name comparison used by workspace lists.
package ordering

import "strings"

// Compare ignores case and compares ASCII digit runs by numeric value, without overflow.
func Compare(a, b string) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	for i, j := 0, 0; ; {
		if i == len(a) || j == len(b) {
			if i == len(a) && j == len(b) {
				return 0
			}
			if i == len(a) {
				return -1
			}
			return 1
		}
		if a[i] >= '0' && a[i] <= '9' && b[j] >= '0' && b[j] <= '9' {
			x, y := i, j
			for i < len(a) && a[i] >= '0' && a[i] <= '9' {
				i++
			}
			for j < len(b) && b[j] >= '0' && b[j] <= '9' {
				j++
			}
			left, right := strings.TrimLeft(a[x:i], "0"), strings.TrimLeft(b[y:j], "0")
			if len(left) != len(right) {
				if len(left) < len(right) {
					return -1
				}
				return 1
			}
			if comparison := strings.Compare(left, right); comparison != 0 {
				return comparison
			}
		} else {
			if a[i] != b[j] {
				if a[i] < b[j] {
					return -1
				}
				return 1
			}
			i++
			j++
		}
	}
}
