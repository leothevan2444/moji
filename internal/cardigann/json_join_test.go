package cardigann

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestJSONJoinEnumeratesOneTokenWithoutLosingOrderOrPrecision(t *testing.T) {
	for _, test := range []struct{ input, path, want string }{
		{`{"values":["one","two"]}`, "values", "one|two"},
		{`{"values":null}`, "values", ""},
		{`{"values":"text"}`, "values", ""},
		{`{"values":{"z":1,"a":2,"z":3}}`, "values", `"z": 3|"a": 2`},
		{`{"values":[9223372036854775807,123456789012345678901234567890]}`, "values", "9223372036854775807|123456789012345678901234567890"},
		{`{"values":[1.0,1000000.0,1e17,1e-5]}`, "values", "1|1000000|1E+17|1E-05"},
		{`{"nested":{"values":["quote\"","\u4e2d\/"]}}`, "nested.values", "quote\"|中/"},
		{`{"values":[{"z":1,"a":2}]}`, "values", "{\n  \"z\": 1,\n  \"a\": 2\n}"},
		{`{"values":[{"stamp":"2024-10-30T10:11:12.1234567+05:30"}]}`, "values", "{\n  \"stamp\": \"2024-10-30T04:41:12.1234567+00:00\"\n}"},
	} {
		got, err := jsonJoinArray(context.Background(), test.input, test.path, "|")
		if err != nil || got != test.want {
			t.Fatalf("%s: got %q, %v; want %q", test.input, got, err, test.want)
		}
	}
}

func TestJSONJoinRejectsInvalidSelectionAndBoundsExpansion(t *testing.T) {
	for _, test := range []struct{ input, path, separator string }{
		{`{"values":[1,2]}`, "values[*]", "|"},
		{`{}`, "values", "|"},
		{`[1,2]`, "$", "|"},
		{`{"values":[]} {}`, "values", "|"},
		{`{"values":[1,2]}`, "values", strings.Repeat("x", MaxTemplateBytes)},
		{`{"values":` + strings.Repeat("[", 65) + `0` + strings.Repeat("]", 65) + `}`, "values", "|"},
		{`{"values":[` + strings.Repeat("0,", 50000) + `0]}`, "values", "|"},
	} {
		if _, err := jsonJoinArray(context.Background(), test.input, test.path, test.separator); err == nil {
			t.Fatal("invalid or over-budget join accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := jsonJoinArray(ctx, `{"values":[1]}`, "values", "|"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
