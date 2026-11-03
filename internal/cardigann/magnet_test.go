package cardigann

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGeneratedMagnetPreservesUTF8NameAndEncodesHashAsOneValue(t *testing.T) {
	hash := strings.Repeat("a", 40)
	title := "Example !*() 测试 &tr=unintended"
	value := generatedMagnetURI(hash, title)
	if !strings.Contains(value, "&dn=Example+!*()+%E6%B5%8B%E8%AF%95+%26tr%3Dunintended") {
		t.Fatal("generated magnet uses different form encoding", value)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Query().Get("dn") != title || parsed.Query().Get("xt") != "urn:btih:"+hash || len(parsed.Query()["tr"]) != 10 {
		t.Fatal("magnet parameters were changed or injected", value, err)
	}
	for _, tracker := range parsed.Query()["tr"] {
		if strings.Contains(tracker, "unintended") {
			t.Fatal("title injected an extra tracker", value)
		}
	}
	if _, err := magnetArtifact(generatedMagnetURI(hash+"&tr=unintended", title)); err == nil {
		t.Fatal("invalid hash injected an additional magnet parameter")
	}
}

func TestPublicTrackerDefaultsDoNotRewriteSiteMagnetOrCreatePrivateFallback(t *testing.T) {
	for _, privacy := range []string{"public", "private"} {
		plan, err := Load([]byte(strings.Replace(minimalDefinition, "type: public", "type: "+privacy, 1)))
		if err != nil {
			t.Fatal(err)
		}
		session := &fakeSession{respond: func(Request) (HTTPResponse, error) {
			t.Fatal("magnet processing sent HTTP")
			return HTTPResponse{}, nil
		}}
		execution, err := plan.NewExecution(session, "", nil, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		original := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40) + "&tr=https%3A%2F%2Ftracker.fixture.invalid%2Fannounce&dn=Original"
		artifact, err := execution.Fetch(context.Background(), Result{MagnetURI: original})
		if err != nil || artifact.Magnet != original {
			t.Fatal("existing site magnet was rewritten", artifact, err)
		}
		artifact, err = execution.Fetch(context.Background(), Result{InfoHash: strings.Repeat("a", 40), Title: "Example"})
		if privacy == "private" {
			if err == nil || artifact.Magnet != "" {
				t.Fatal("private hash acquired a public fallback", artifact, err)
			}
		} else if err != nil || !strings.Contains(artifact.Magnet, "&tr=udp%3A%2F%2Ftracker.opentrackr.org") {
			t.Fatal("public fallback lost approved tracker defaults", artifact, err)
		}
	}
}
