package cardigann

import (
	"fmt"
	"strconv"
	"strings"
)

// These Torznab fields are nullable Int32 values in the upstream query models.
// The public named-parameter boundary uses strings; canonicalize before either
// templating or keyword assembly so "002026" does not become a different year.
func applyNumericQueryFields(variables Variables) error {
	for _, name := range []string{"Year", "TVDBID", "TVRageID", "TMDBID", "TVMazeID", "TraktID", "DoubanID"} {
		key := ".Query." + name
		value, present := variables[key]
		if !present {
			continue
		}
		number, err := strconv.ParseInt(strings.TrimSpace(value.Text), 10, 32)
		if err != nil {
			return fmt.Errorf("query %s must be a 32-bit integer", name)
		}
		variables[key] = TemplateValue{Text: strconv.FormatInt(number, 10)}
	}
	return nil
}
