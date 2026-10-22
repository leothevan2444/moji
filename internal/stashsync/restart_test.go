package stashsync

import (
	"context"
	"errors"
	"reflect"
	"testing"

	graphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type restartClient struct {
	fakeIngestClient
	removed   []string
	removeErr error
}

func (c *restartClient) RemoveIngestScene(_ context.Context, id string) error {
	if c.removeErr != nil {
		return c.removeErr
	}
	c.removed = append(c.removed, id)
	return nil
}

func TestRestartSceneCleanupRejectsSharedAndUnexpectedScenesBeforeDeleting(t *testing.T) {
	for _, mode := range []string{"owned", "shared", "unexpected", "failure"} {
		t.Run(mode, func(t *testing.T) {
			c := &restartClient{}
			c.scenes = []*graphql.IngestScenes_FindScenes_Scenes{
				{ID: "1", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/SNOS-184/SNOS-184.mp4"}}},
				{ID: "neighbor", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/SNOS-1840/movie.mp4"}}},
			}
			switch mode {
			case "shared":
				c.scenes = append(c.scenes, &graphql.IngestScenes_FindScenes_Scenes{ID: "2", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/SNOS-184/movie.mp4"}, {Path: "/other/movie.mp4"}}})
			case "unexpected":
				c.scenes = append(c.scenes, &graphql.IngestScenes_FindScenes_Scenes{ID: "3", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/SNOS-184/another.mp4"}}})
			case "failure":
				c.removeErr = errors.New("Stash offline")
			}
			s, _ := NewService(c, nil)
			err := s.RemoveIngestScenes(context.Background(), "/library/SNOS-184", []string{"1", "2"})
			if mode != "owned" {
				if err == nil || len(c.removed) != 0 {
					t.Fatalf("unsafe or failed cleanup changed scenes: %v %v", c.removed, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(c.removed, []string{"1"}) {
				t.Fatalf("incorrect removal scope: %v %v", c.removed, err)
			}
			c.scenes = nil
			if err := s.RemoveIngestScenes(context.Background(), "/library/SNOS-184", []string{"1"}); err != nil {
				t.Fatal("missing records must be safe on retry", err)
			}
		})
	}
}
