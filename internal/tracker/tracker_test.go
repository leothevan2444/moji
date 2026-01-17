package tracker

import (
	"fmt"
	"strings"
	"testing"
)

var jackettURL = "http://homeserver0.local:9118"
var jackettAPIKey = "yivm2eqtrspwajjwhi33cawxzclcagxe"
var javID = "SONE-631"

func TestJackett(t *testing.T) {
	jackett := NewJackettService(jackettURL, jackettAPIKey)

	results, err := jackett.Search(javID,
		WithTrackers([]string{"sukebeinyaasi", "onejav", "u3c3"}),
	)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		return
	}
	for _, result := range results {
		fmt.Printf("Tracker: %s | Categories: %s | Title: %s\nLink: %s\n--------------------------------\n",
			result.Tracker, strings.Trim(strings.Join(strings.Fields(fmt.Sprint(result.Category)), ","), "[]"),
			result.Title, result.Link)
	}
}
