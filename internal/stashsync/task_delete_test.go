package stashsync

import (
	"context"
	"errors"
	"reflect"
	"testing"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type taskDeleteClient struct {
	fakeIngestClient
	byID         map[string]*gql.IngestScenes_FindScenes_Scenes
	deleted      []string
	lostResponse bool
	readErr      error
}

func (*taskDeleteClient) IngestIdentity() string { return "account" }
func (c *taskDeleteClient) FindSceneForDeletion(_ context.Context, _ string, id string) (*gql.IngestScenes_FindScenes_Scenes, error) {
	return c.byID[id], c.readErr
}
func (c *taskDeleteClient) DeleteSceneWithFiles(_ context.Context, _ string, id string) error {
	c.deleted = append(c.deleted, id)
	delete(c.byID, id)
	if c.lostResponse {
		c.lostResponse = false
		return errors.New("response lost")
	}
	return nil
}
func ownedDeleteScene(id, path string) *gql.IngestScenes_FindScenes_Scenes {
	return &gql.IngestScenes_FindScenes_Scenes{ID: id, Files: []*gql.IngestScenes_FindScenes_Scenes_Files{{Path: path}}}
}

func TestTaskSceneDeletionScopesAndChecksAllScenesBeforeDeleting(t *testing.T) {
	for _, mode := range []string{"owned", "missing", "foreign", "shared", "no-files", "offline", "connection", "invalid-root"} {
		t.Run(mode, func(t *testing.T) {
			c := &taskDeleteClient{byID: map[string]*gql.IngestScenes_FindScenes_Scenes{"1": ownedDeleteScene("1", "/library/DEMO-123/a.mp4"), "2": ownedDeleteScene("2", "/library/DEMO-123/b.mp4"), "3": ownedDeleteScene("3", "/library/DEMO-123/unlinked.mp4")}}
			identity, path := "account", "/library/DEMO-123"
			switch mode {
			case "missing":
				delete(c.byID, "1")
			case "foreign":
				c.byID["2"] = ownedDeleteScene("2", "/library/DEMO-1234/other.mp4")
			case "shared":
				c.byID["2"].Files = append(c.byID["2"].Files, &gql.IngestScenes_FindScenes_Scenes_Files{Path: "/other/shared.mp4"})
			case "no-files":
				c.byID["2"].Files = nil
			case "offline":
				c.readErr = errors.New("unreachable")
			case "connection":
				identity = "another-account"
			case "invalid-root":
				path = "/"
			}
			s, _ := NewService(c, nil)
			err := s.DeleteTaskScenes(context.Background(), identity, path, []string{"1", "2", "1"})
			if mode != "owned" && mode != "missing" {
				if err == nil || len(c.deleted) > 0 {
					t.Fatal("unsafe deletion", err, c.deleted)
				}
				return
			}
			want := []string{"1", "2"}
			if mode == "missing" {
				want = []string{"2"}
			}
			if err != nil || !reflect.DeepEqual(c.deleted, want) || c.byID["3"] == nil {
				t.Fatal("incorrect scene scope", err, c.deleted)
			}
		})
	}
}
func TestTaskSceneDeletionReconcilesLostResponse(t *testing.T) {
	c := &taskDeleteClient{byID: map[string]*gql.IngestScenes_FindScenes_Scenes{"1": ownedDeleteScene("1", "/library/DEMO-123/a.mp4"), "2": ownedDeleteScene("2", "/library/DEMO-123/b.mp4")}, lostResponse: true}
	s, _ := NewService(c, nil)
	if err := s.DeleteTaskScenes(context.Background(), "account", "/library/DEMO-123", []string{"1", "2"}); err == nil {
		t.Fatal("lost response hidden")
	}
	if err := s.DeleteTaskScenes(context.Background(), "account", "/library/DEMO-123", []string{"1", "2"}); err != nil || !reflect.DeepEqual(c.deleted, []string{"1", "2"}) {
		t.Fatal("retry repeated deletion or missed remaining scene", err, c.deleted)
	}
}
