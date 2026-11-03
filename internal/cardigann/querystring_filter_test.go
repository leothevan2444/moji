package cardigann

import "testing"

func TestQueryStringFilterKeepsFirstValueAndURIDecoding(t *testing.T) {
	for _, test := range []struct{ input, argument, want string }{
		{"https://site.invalid/get?ID=one&id=two#id=third", "id", "one"},
		{"/get?ID=&id=two", "id", ""},
		{"/get?i%64=encoded", "id", "encoded"},
		{"/get?id=one;two", "id", "one;two"},
		{"/get?id=%2f%zz%2b+", "id", "/%zz+ "},
		{"/get?id=%E4%B8%AD", "id", "中"},
		{"/get?id=%E4%B8", "id", "%E4%B8"},
		{"/get?id=%FF", "id", "%FF"},
		{"/get?x=one&amp;id=two", "id", ""},
		{"/get?İD=dotted&id=normal", "id", "normal"},
		{"/get?ıd=dotless&id=normal", "id", "normal"},
		{"id=one+two", "id", ""},
		{"/get?id", "id", ""},
	} {
		if got := queryStringFilter(test.input, test.argument); got != test.want {
			t.Fatalf("%q, key %q: got %q, want %q", test.input, test.argument, got, test.want)
		}
	}
}
