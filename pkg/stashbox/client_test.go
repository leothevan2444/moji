package stashbox

import (
	"context"
	"fmt"
	"testing"
)

var apikey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiI5ZGEyYWRmNy04ODIxLTRiYTEtOTcyZC01NTNkNTQ5Nzc1YWUiLCJzdWIiOiJBUElLZXkiLCJpYXQiOjE3MzQwMjI5MTN9.skQWjDQq9147WzO9qM-Ignu0Vvuc0ZQOx5JfgvdcIAE"

func TestClient(t *testing.T) {
	client := NewClient(apikey)
	me, err := client.Me(context.Background())
	if err != nil {
		t.Errorf("failed to get me: %v", err)
	}
	fmt.Printf("Me: %+v\n", me)
	version, err := client.GetVersion(context.Background())
	if err != nil {
		t.Errorf("failed to get version: %v", err)
	}
	fmt.Printf("Version: %+v\n", version)
}
