package cardigann

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestRequestEncodingSupportsAllLabelsInPinnedCorpus(t *testing.T) {
	for _, name := range []string{"utf-8", "utf8", "iso-8859-1", "iso-8859-2", "tis-620", "iso-8859-11", "windows-1250", "windows-1251", "windows-1252", "windows-1255", "windows-1256", "windows-874"} {
		if actual, err := encodeQueryValue("ABC 123", name); err != nil || actual != "ABC+123" {
			t.Errorf("declared encoding %s: %q %v", name, actual, err)
		}
	}
}
func TestFormEncodingMatchesWebUtilityReservedCharacterContract(t *testing.T) {
	actual, err := encodeQueryValue("~!*()' +/&=", "UTF-8")
	if err != nil || actual != "%7E!*()%27+%2B%2F%26%3D" {
		t.Fatal("form character set differs", actual, err)
	}
	// ISO-8859-1 is a request encoding here, not HTML's Windows-1252 alias.
	actual, err = encodeQueryValue("\u0080é", "ISO-8859-1")
	if err != nil || actual != "%80%E9" {
		t.Fatal("request encoding used HTML charset aliases", actual, err)
	}
	result, err := ApplyFilters(context.Background(), "%80%E9", []Filter{{Name: "urldecode"}}, FilterContext{Encoding: "ISO-8859-1"})
	if err != nil || result != "\u0080é" {
		t.Fatal("legacy URL decode lost bytes", result, err)
	}
}

func TestThaiCodePageAliasUsesWindows874(t *testing.T) {
	value, err := encodeQueryValue("ไทย", "tis-620")
	if err != nil || value != "%E4%B7%C2" {
		t.Fatal("Thai code page alias differs", value, err)
	}
}

func TestGenericRequestDoesNotExpandSearchOnlyRawSyntax(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition))
	if err != nil {
		t.Fatal(err)
	}
	execution, err := plan.NewExecution(&fakeSession{}, "https://example.invalid/", nil, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	request, err := execution.requestFrom(context.Background(), RequestBlock{Path: "/fetch", Inputs: ScalarFields{{Name: "$raw", Value: "opaque=value"}}}, execution.variables, nil, execution.base, true, false)
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse(request.URL)
	if err != nil || target.Query().Get("$raw") != "opaque=value" || target.Query().Has("opaque") {
		t.Fatal("generic request gained search raw expansion", request.URL, err)
	}
}
