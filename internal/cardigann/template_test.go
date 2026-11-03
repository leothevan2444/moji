package cardigann

import (
	"context"
	"strings"
	"testing"
)

func TestTemplateCommunitySyntax(t *testing.T) {
	v := Variables{".Config.filter-id": {Text: "2"}, ".Result.title": {Text: "ABCD-123"}, ".Categories": {Items: []string{"a", "b"}}, ".True": {Text: "True"}, ".Query.Keywords": {Text: "a b/c"}}
	cases := map[string]string{
		`{{ .Config.filter-id }}`:                                                                          "2",
		`{{ if .False }}no{{ else }}{{ .Result.title }}{{ end }}`:                                          "ABCD-123",
		`{{ if and (.True) (ne .Result.title "NULL") }}yes{{ else }}no{{ end }}`:                           "yes",
		`{{ if .Categories }}{{ range $i, $e := .Categories }}{{ $i }}={{ . }};{{end}}{{else}}none{{end}}`: "0=a;1=b;",
		`{{ join .Categories "," }}`:                                                                       "a,b",
		`{{ re_replace .Query.Keywords "[^a-z]+" "%" }}`:                                                   "a%b%c",
		`{{ or .Missing .Result.title }}`:                                                                  "ABCD-123",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			tmpl, err := CompileTemplate(input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tmpl.Render(context.Background(), v)
			if err != nil || got != want {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		})
	}
}

func TestTemplateRejectsExecutableOrMalformedActions(t *testing.T) {
	for _, input := range []string{`{{ exec "sh" }}`, `{{ .X`, `{{ if .X }}missing end`, `{{end}}`, `{{ eq .X }}`, `{{ .X | print }}`} {
		if _, err := CompileTemplate(input); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	tmpl, err := CompileTemplate(`{{range .Categories}}{{.}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tmpl.Render(context.Background(), Variables{".Categories": {Items: []string{strings.Repeat("x", MaxTemplateBytes), "extra"}}})
	if err == nil {
		t.Fatal("unbounded output")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = tmpl.Render(ctx, nil); err != context.Canceled {
		t.Fatalf("cancellation lost: %v", err)
	}
}
