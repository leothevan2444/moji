package mediafiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func deliveryRemovalFixture(t *testing.T) (string, string, string) {
	t.Helper()
	ctx := context.Background()
	source, library := t.TempDir(), t.TempDir()
	video, subtitle := []byte("clip"), []byte("1\n00:00:00,000 --> 00:00:01,000\nSubtitle\n")
	for name, data := range map[string][]byte{"SNOS-184.mp4": video, "SNOS-184.srt": subtitle} {
		if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	plan := Classify("SNOS-184", []File{{Index: 0, Path: "SNOS-184.mp4", Size: int64(len(video))}, {Index: 1, Path: "SNOS-184.srt", Size: int64(len(subtitle))}}, true)
	result, err := (FFmpeg{}).Prepare(ctx, Request{TaskID: "task-1", Code: "SNOS-184", SourceRoot: source, TargetRoot: library, Action: "COPY", Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	return library, result.Directory, result.VideoPath
}

func TestDeliveryRemovalResumesPartialDeletionWithoutBackup(t *testing.T) {
	library, target, video := deliveryRemovalFixture(t)
	ctx := context.Background()
	plan, err := PlanDeliveryRemoval(ctx, "task-1", "SNOS-184", target)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 2 {
		t.Fatal("fixture must include movie and subtitle")
	}
	// Simulate a crash after deleting one file, before saving another checkpoint.
	if err := os.Remove(video); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDelivery(ctx, *plan); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDelivery(ctx, *plan); err != nil {
		t.Fatal("deletion must be idempotent", err)
	}
	for _, path := range []string{target, filepath.Join(library, ".moji-receipts", "task-1.json"), filepath.Join(library, ".moji-restarts")} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unexpected retained artifact %s: %v", path, err)
		}
	}
}

func TestDeliveryRemovalRejectsUnownedModifiedAndSymlinkedOutput(t *testing.T) {
	for _, mode := range []string{"extra-file", "modified", "symlink", "foreign", "modified-after-plan"} {
		t.Run(mode, func(t *testing.T) {
			library, target, video := deliveryRemovalFixture(t)
			ctx := context.Background()
			plan, err := PlanDeliveryRemoval(ctx, "task-1", "SNOS-184", target)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "extra-file":
				err = os.WriteFile(filepath.Join(target, "user-notes.txt"), []byte("preserve"), 0600)
			case "modified", "modified-after-plan":
				err = os.WriteFile(video, []byte("user modified video"), 0600)
			case "symlink":
				outside := filepath.Join(t.TempDir(), "movie.mp4")
				if err = os.Rename(video, outside); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(outside, video)
			case "foreign":
				err = os.Remove(filepath.Join(library, ".moji-receipts", "task-1.json"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "modified-after-plan" {
				err = RemoveDelivery(ctx, *plan)
			} else {
				_, err = PlanDeliveryRemoval(ctx, "task-1", "SNOS-184", target)
			}
			if err == nil {
				t.Fatal("unsafe deletion accepted")
			}
			if _, err := os.Stat(video); err != nil {
				t.Fatal("existing video lost", err)
			}
		})
	}
}

func TestDownloadRemovalDeletesSelectedDiscardedAndPartialFilesOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "torrent"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"torrent/movie.mp4", "torrent/movie.mp4.!qB", "torrent/sample.mp4", "unrelated.mp4"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	plan := DownloadRemoval{Root: root, Paths: []string{"torrent/movie.mp4", "torrent/sample.mp4"}}
	if err := RemoveDownloads(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDownloads(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "torrent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("torrent files or empty directory retained")
	}
	if _, err := os.Stat(filepath.Join(root, "unrelated.mp4")); err != nil {
		t.Fatal("unrelated download removed")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("shared download root removed")
	}
}

func TestDownloadRemovalRejectsTraversalAndSymlinks(t *testing.T) {
	for _, mode := range []string{"traversal", "symlink", "directory"} {
		t.Run(mode, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			file := filepath.Join(outside, "movie.mp4")
			if err := os.WriteFile(file, []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			name := "../outside/movie.mp4"
			switch mode {
			case "symlink":
				name = "link/movie.mp4"
				if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
					t.Fatal(err)
				}
			case "directory":
				name = "folder"
				if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := RemoveDownloads(context.Background(), DownloadRemoval{Root: root, Paths: []string{name}}); err == nil {
				t.Fatal("unsafe download deletion accepted")
			}
			if _, err := os.Stat(file); err != nil {
				t.Fatal("outside file removed")
			}
		})
	}
}
