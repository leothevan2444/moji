package cardigann

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestResponseTextPreservesUTF8AndBoundsDecodedExpansion(t *testing.T) {
	for _, sample := range []struct {
		name, encoding, expected string
		body                     []byte
		fail                     bool
	}{
		{"utf8", "UTF-8", "\ufeffA\ufffd", []byte{0xef, 0xbb, 0xbf, 'A', 0xff}, false},
		{"expanded-limit", "ISO-8859-1", "", bytes.Repeat([]byte{0xe9}, MaxPageBytes/2+1), true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			plan, err := Load([]byte(strings.Replace(minimalDefinition, "encoding: UTF-8", "encoding: "+sample.encoding, 1)))
			if err != nil {
				t.Fatal(err)
			}
			execution, err := plan.NewExecution(&fakeSession{}, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			text, err := execution.responseText(HTTPResponse{Body: sample.body})
			if (err != nil) != sample.fail || text != sample.expected {
				t.Fatalf("response decode: length %d, err %v", len(text), err)
			}
		})
	}
}

func TestSearchResponseUsesDeclaredLatin1InsteadOfHTMLAlias(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "encoding: UTF-8", "encoding: ISO-8859-1", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, URL: r.URL, Body: []byte("<article><h2>A\x80B</h2><a href='/dl'/></article>")}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{})
	if err != nil || len(result.Results) != 1 || result.Results[0].Title != "A\u0080B" {
		t.Fatalf("declared Latin-1 was replaced by an HTML alias: %#v %v", result, err)
	}
}

func TestSearchXMLDeclarationDoesNotDecodeLegacyTextTwice(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "encoding: UTF-8", "encoding: windows-1251", 1)
	definition = strings.Replace(definition, "paths: [{path: /}]", "paths: [{path: /api, response: {type: xml}}]", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("<?xml version=\"1.0\" encoding=\"windows-1251\"?><feed><article><h2>\xd2\xe5\xf1\xf2</h2><a href='/dl'/></article></feed>")
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, URL: r.URL, Body: body}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{})
	if err != nil || len(result.Results) != 1 || result.Results[0].Title != "Тест" {
		t.Fatalf("XML declaration decoded already-decoded text: %#v %v", result, err)
	}
}

func TestLoadRejectsUnknownDeclaredEncodingAtItsSource(t *testing.T) {
	_, err := Load([]byte(strings.Replace(minimalDefinition, "encoding: UTF-8", "encoding: moji-unknown-codepage", 1)))
	diagnostic, ok := err.(Diagnostic)
	if !ok || diagnostic.Path != "/encoding" || diagnostic.Line != 6 || diagnostic.Column <= 0 {
		t.Fatalf("unknown encoding was not rejected at its declaration: %#v", err)
	}
}
