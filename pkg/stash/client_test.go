package stash

import (
	"context"
	"fmt"
	"testing"
)

var host = "http://localhost:9999/graphql"
var apiKey = ""

func TestClient(t *testing.T) {
	client := NewClient(host, apiKey)
	version, err := client.GetVersion(context.Background())
	if err != nil {
		t.Errorf("failed to get version: %v", err)
	}
	fmt.Printf("Version: %+v\n", *version)
}
