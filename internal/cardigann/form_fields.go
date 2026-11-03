package cardigann

import (
	"net/url"
	"strings"
)

type formField struct{ name, value string }
type formFields struct {
	entries   []formField
	positions map[string]int
}

// Set models the upstream form dictionary: replacing a field keeps its first
// position, and a new field is appended. DOM duplicates use the last value.
func (f *formFields) Set(name, value string) {
	if position, found := f.positions[name]; found {
		f.entries[position].value = value
		return
	}
	if f.positions == nil {
		f.positions = map[string]int{}
	}
	f.positions[name] = len(f.entries)
	f.entries = append(f.entries, formField{name, value})
}

func (f formFields) has(name string) bool {
	_, found := f.positions[name]
	return found
}

func (f formFields) Encode(encoding string) (string, error) {
	pairs := make([]string, 0, len(f.entries))
	for _, field := range f.entries {
		value, err := encodeQueryValue(field.value, encoding)
		if err != nil {
			return "", err
		}
		pairs = append(pairs, field.name+"="+value)
	}
	return strings.Join(pairs, "&"), nil
}

// Login GET selector inputs use UTF-8 independently of the form's encoding.
// Preserve the existing query's bytes and order, replacing only matching keys.
func (f formFields) withQuery(target string) (string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	var pairs []string
	if u.RawQuery != "" {
		for _, part := range strings.Split(u.RawQuery, "&") {
			key, _, _ := strings.Cut(part, "=")
			name, err := url.QueryUnescape(key)
			if err != nil {
				return "", err
			}
			if !f.has(name) {
				pairs = append(pairs, part)
			}
		}
	}
	encoded, err := f.Encode("UTF-8")
	if err != nil {
		return "", err
	}
	if encoded != "" {
		pairs = append(pairs, encoded)
	}
	u.RawQuery = strings.Join(pairs, "&")
	return u.String(), nil
}
