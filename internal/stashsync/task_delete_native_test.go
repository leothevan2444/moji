package stashsync

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	stashclient "github.com/leothevan2444/moji/pkg/stash"
	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

// Explicit destructive integration test, restricted to a disposable local
// instance with a synthetic-only mount and a dedicated temporary fixture root.
func TestOptInStashTaskDeletionRemovesNativeFiles(t *testing.T) {
	endpoint, root := os.Getenv("MOJI_STASH_DELETE_URL"), os.Getenv("MOJI_STASH_DELETE_ROOT")
	if endpoint == "" || root == "" {
		t.Skip("set MOJI_STASH_DELETE_URL and MOJI_STASH_DELETE_ROOT for an isolated synthetic instance")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(u.Port())
	if u.Scheme != "http" || u.Hostname() != "127.0.0.1" || port < 20000 || u.Path != "/graphql" || !strings.HasPrefix(root, "/tmp/moji-stash-delete-") {
		t.Fatal("refusing non-isolated endpoint or fixture root")
	}
	roots := parityPost(t, endpoint, `query{configuration{general{stashes{path}}}}`, nil)
	var conf struct {
		General struct{ Stashes []struct{ Path string } }
	}
	if json.Unmarshal(roots["configuration"], &conf) != nil || len(conf.General.Stashes) != 1 || conf.General.Stashes[0].Path != "/moji-parity" {
		t.Fatal("refusing instance without synthetic-only media root")
	}
	video := filepath.Join(root, "media", "DEMO-123", "DEMO-123.mp4")
	funscript := strings.TrimSuffix(video, ".mp4") + ".funscript"
	note := filepath.Join(filepath.Dir(video), "keep.txt")
	for _, p := range []string{video, funscript, note} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("synthetic fixture missing", err)
		}
	}
	c := stashclient.NewClient(func() stashclient.Config { return stashclient.Config{URL: endpoint} })
	s, _ := NewService(c, nil)
	job, err := c.MetadataScan(context.Background(), gql.ScanMetadataInput{Paths: []string{"/moji-parity"}, ScanGenerateCovers: codeBool(false), ScanGenerateSprites: codeBool(false), ScanGeneratePreviews: codeBool(false), ScanGeneratePhashes: codeBool(false)})
	if err != nil {
		t.Fatal(err)
	}
	waitParityJob(t, c, job)
	scenes, err := s.FindIngestScenes(context.Background(), "/moji-parity/DEMO-123", nil)
	if err != nil || len(scenes) != 1 {
		t.Fatal("synthetic scene missing", err)
	}
	ids := []string{scenes[0].ID}
	job, err = c.MetadataGenerate(context.Background(), gql.GenerateMetadataInput{SceneIDs: ids, Covers: codeBool(true), Sprites: codeBool(true)})
	if err != nil {
		t.Fatal(err)
	}
	waitParityJob(t, c, job)
	generated := []string{}
	if err := filepath.WalkDir(filepath.Join(root, "config", "generated"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			generated = append(generated, path)
		}
		return nil
	}); err != nil || len(generated) == 0 {
		t.Fatal("no generated files to verify", err)
	}
	if err := s.DeleteTaskScenes(context.Background(), s.DeletionIdentity(), "/moji-parity/DEMO-123", ids); err != nil {
		t.Fatal(err)
	}
	for _, p := range append(generated, video, funscript) {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatal("Stash retained deleted artifact", p, err)
		}
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal("unrelated file was removed", err)
	}
	if err := s.DeleteTaskScenes(context.Background(), s.DeletionIdentity(), "/moji-parity/DEMO-123", ids); err != nil {
		t.Fatal("deletion is not retryable after confirmed absence", err)
	}
	t.Logf("verified scene, video, funscript and %d generated files removed; unrelated file retained", len(generated))
}
