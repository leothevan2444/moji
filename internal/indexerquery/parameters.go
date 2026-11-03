// Package indexerquery owns named query-field qualification shared by the
// provider-neutral entrypoint and the standalone Cardigann interpreter.
package indexerquery

import (
	"errors"
	"strings"
)

var ErrInvalid = errors.New("invalid named query parameters")

// These are the writable structured fields used by the Cardigann host query.
// Mode, keywords, categories, pagination and derived variables have typed owners
// and cannot also be supplied through this map. Ep is raw; Episode is derived.
var canonicalNames = map[string]string{
	"year": "Year", "season": "Season", "ep": "Ep",
	"imdbid": "IMDBID", "tmdbid": "TMDBID", "tvdbid": "TVDBID",
	"tvmazeid": "TVMazeID", "tvrageid": "TVRageID", "traktid": "TraktID", "doubanid": "DoubanID",
	"artist": "Artist", "album": "Album", "label": "Label", "track": "Track", "genre": "Genre",
	"author": "Author", "title": "Title", "publisher": "Publisher",
}

// NormalizeParameters returns an owned map without changing value spelling or
// presence. Case aliases are accepted individually, but duplicates are rejected
// even when their values agree, avoiding map-order-dependent interpretation.
func NormalizeParameters(input map[string]string) (map[string]string, error) {
	if len(input) > 64 {
		return nil, ErrInvalid
	}
	result := make(map[string]string, len(input))
	for name, value := range input {
		if len(name) > 64 || len(value) > 4096 {
			return nil, ErrInvalid
		}
		canonical, known := canonicalNames[strings.ToLower(name)]
		if !known {
			return nil, ErrInvalid
		}
		if _, duplicate := result[canonical]; duplicate {
			return nil, ErrInvalid
		}
		result[canonical] = value
	}
	return result, nil
}
