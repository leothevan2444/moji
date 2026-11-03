package cardigann

import (
	"context"
	"testing"
)

func TestStringFiltersUseFirstUTF16CodeUnit(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
		args              Arguments
	}{
		{"split", "left::middle:right", "right", Arguments{"::", "-1"}},
		{"split", "left😁middle😀right", "left", Arguments{"😀", "0"}},
		{"split", "first中second文third", "second文third", Arguments{"中洲", "-1"}},
		{"trim", "xxxyVALUEyxxx", "yVALUEy", Arguments{"xy"}},
		{"trim", "中中aVALUEa中中", "aVALUEa", Arguments{"中a"}},
		{"trim", "VALUE😀", "VALUE😀", Arguments{"😀"}},
		{"trim", "🎉VALUE😀", "🎉VALUE😀", Arguments{"😀"}},
	} {
		t.Run(test.name+test.input, func(t *testing.T) {
			got, err := ApplyFilters(context.Background(), test.input, []Filter{{Name: test.name, Args: test.args}}, FilterContext{})
			if err != nil || got != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestStringFiltersRejectEmptyTrimAndUnpairedSurrogateOutput(t *testing.T) {
	for _, test := range []struct {
		input  string
		filter Filter
	}{
		{"text", Filter{Name: "trim", Args: Arguments{""}}},
		{"😀text", Filter{Name: "trim", Args: Arguments{"😀"}}},
		{"a😀b", Filter{Name: "split", Args: Arguments{"😀", "1"}}},
	} {
		if got, err := ApplyFilters(context.Background(), test.input, []Filter{test.filter}, FilterContext{}); err == nil {
			t.Fatalf("invalid filter output accepted as %q", got)
		}
	}
}

func TestHTMLEncodePreservesUpstreamEntitySpelling(t *testing.T) {
	const input = `A "double" 'single' & <tag> café 中文 😀`
	const want = `A &quot;double&quot; &#39;single&#39; &amp; &lt;tag&gt; caf&#233; 中文 &#128512;`
	got, err := ApplyFilters(context.Background(), input, []Filter{{Name: "htmlencode"}}, FilterContext{})
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
	decoded, err := ApplyFilters(context.Background(), got, []Filter{{Name: "htmldecode"}}, FilterContext{})
	if err != nil || decoded != input {
		t.Fatalf("entity round trip lost text: %q, %v", decoded, err)
	}
}
