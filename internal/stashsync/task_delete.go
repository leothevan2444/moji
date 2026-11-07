package stashsync

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type taskDeletionClient interface {
	IngestIdentity() string
	FindSceneForDeletion(context.Context, string, string) (*gql.IngestScenes_FindScenes_Scenes, error)
	DeleteSceneWithFiles(context.Context, string, string) error
}

func (s *Service) DeletionIdentity() string {
	if client, ok := s.client.(taskDeletionClient); ok {
		return client.IngestIdentity()
	}
	return ""
}

// This is deliberately separate from restart cleanup, which retains media.
// Only persisted scene IDs are eligible; a directory search never expands them.
func (s *Service) DeleteTaskScenes(ctx context.Context, identity, path string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	client, ok := s.client.(taskDeletionClient)
	if !ok || identity == "" || client.IngestIdentity() != identity {
		return errors.New("Stash scene deletion is unavailable or its connection changed")
	}
	root := filepath.Clean(path)
	if !filepath.IsAbs(root) || root == string(filepath.Separator) || len(ids) > 16 {
		return errors.New("invalid task delivery path or too many linked scenes")
	}
	seen := map[string]bool{}
	remove := []string{}
	// Check the complete set before the first deletion.
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		scene, err := client.FindSceneForDeletion(ctx, identity, id)
		if err != nil {
			return err
		}
		if scene == nil {
			continue
		}
		if len(scene.Files) == 0 {
			return fmt.Errorf("Stash scene %s has no verifiable task files", id)
		}
		for _, file := range scene.Files {
			if file == nil || (filepath.Clean(file.Path) != root && !strings.HasPrefix(filepath.Clean(file.Path), root+string(filepath.Separator))) {
				return fmt.Errorf("Stash scene %s contains files outside this task's delivery directory", id)
			}
		}
		remove = append(remove, id)
	}
	for _, id := range remove {
		if err := client.DeleteSceneWithFiles(ctx, identity, id); err != nil {
			return err
		}
		remaining, err := client.FindSceneForDeletion(ctx, identity, id)
		if err != nil {
			return err
		}
		if remaining != nil {
			return errors.New("Stash scene still exists after deletion")
		}
	}
	return nil
}
