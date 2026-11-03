package cardigann

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

// declaredEncoding uses definition labels, not HTML aliases (which interpret
// ISO-8859-1 as Windows-1252). A nil result denotes UTF-8.
func declaredEncoding(name string) (encoding.Encoding, error) {
	if name == "" || strings.EqualFold(name, "utf-8") || strings.EqualFold(name, "utf8") {
		return nil, nil
	}
	if enc, ok := dotNetSingleByteEncoding(name); ok {
		return enc, nil
	}
	enc, err := ianaindex.MIME.Encoding(name)
	if err != nil || enc == nil {
		return nil, fmt.Errorf("unknown character encoding")
	}
	// Resolve IANA aliases of built-in .NET encodings such as Latin-1.
	if canonical, err := ianaindex.MIME.Name(enc); err == nil {
		if exact, ok := dotNetSingleByteEncoding(canonical); ok {
			return exact, nil
		}
	}
	return enc, nil
}

func encodeQueryValue(value, name string) (string, error) {
	enc, err := declaredEncoding(name)
	if err != nil {
		return "", err
	}
	if enc != nil {
		value, err = enc.NewEncoder().String(value)
		if err != nil {
			return "", fmt.Errorf("URL character encoding failed")
		}
	}
	return encodeFormBytes(value), nil
}

// WebUtility form encoding uses the RFC 1738 safe set. Go's query encoder
// differs for tilde, exclamation, parentheses and asterisk.
var formEscapes = strings.NewReplacer("%21", "!", "%28", "(", "%29", ")", "%2A", "*", "~", "%7E")

func encodeFormBytes(value string) string { return formEscapes.Replace(url.QueryEscape(value)) }
