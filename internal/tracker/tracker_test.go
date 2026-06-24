package tracker

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var javID = "SONE-631"

func TestJackett(t *testing.T) {
	if os.Getenv("MOJI_RUN_INTEGRATION") != "1" {
		t.Skip("set MOJI_RUN_INTEGRATION=1 to run Jackett integration tests")
	}

	jackettURL := os.Getenv("MOJI_JACKETT_URL")
	jackettAPIKey := os.Getenv("MOJI_JACKETT_API_KEY")
	if jackettURL == "" || jackettAPIKey == "" {
		t.Skip("set MOJI_JACKETT_URL and MOJI_JACKETT_API_KEY to run Jackett integration tests")
	}

	jackett := NewJackettService(func() JackettConfig {
		return JackettConfig{URL: jackettURL, APIKey: jackettAPIKey}
	})

	results, err := jackett.Search(javID,
		WithTrackers([]string{"sukebeinyaasi", "onejav", "u3c3"}),
	)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	for _, result := range results {
		t.Logf("Tracker: %s | Categories: %s | Title: %s\nLink: %s\n--------------------------------",
			result.Tracker, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(result.Category)), ","), "[]"),
			result.Title, result.Link)
	}
}
