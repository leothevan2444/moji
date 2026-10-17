package mediafiles

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for media integration tests")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is required for media integration tests")
	}
}
func makeClip(t *testing.T, path, color string, withSubtitle bool) {
	t.Helper()
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-f", "lavfi", "-i", "color=c=" + color + ":s=160x120:r=25:d=1", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=1"}
	if withSubtitle {
		sub := filepath.Join(t.TempDir(), "sub.srt")
		if err := os.WriteFile(sub, []byte("1\n00:00:00,100 --> 00:00:00,500\nEmbedded\n"), 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, "-i", sub)
	}
	args = append(args, "-map", "0:v:0", "-map", "1:a:0", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac")
	if withSubtitle {
		args = append(args, "-map", "2:s:0", "-c:s", "srt")
	}
	args = append(args, "-t", "1", path)
	if _, err := run(context.Background(), "ffmpeg", args...); err != nil {
		t.Fatal(err)
	}
}
func TestRealFFmpegMergesFiltersDeduplicatesAndShiftsSubtitles(t *testing.T) {
	requireFFmpeg(t)
	root := t.TempDir()
	source, target := filepath.Join(root, "downloads"), filepath.Join(root, "library")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	makeClip(t, filepath.Join(source, "hdd800.com@OFJE-329A.mkv"), "red", true)
	makeClip(t, filepath.Join(source, "gc_55.com@OFJE-329B.mkv"), "blue", true)
	if err := copyMedia(context.Background(), filepath.Join(source, "gc_55.com@OFJE-329B.mkv"), filepath.Join(source, "OFJE-329C.mkv")); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"A", "B", "C"} {
		if err := os.WriteFile(filepath.Join(source, "OFJE-329"+part+".en.srt"), []byte("1\n00:00:00,100 --> 00:00:00,500\n"+part+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyMedia(context.Background(), filepath.Join(source, "OFJE-329B.en.srt"), filepath.Join(source, "duplicate-sub.srt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(source, "OFJE-329C.en.srt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(source, "duplicate-sub.srt"), filepath.Join(source, "OFJE-329C.en.srt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "广告.mp4"), []byte("junk"), 0644); err != nil {
		t.Fatal(err)
	}
	files := []File{}
	names := []string{"hdd800.com@OFJE-329A.mkv", "gc_55.com@OFJE-329B.mkv", "OFJE-329C.mkv", "OFJE-329A.en.srt", "OFJE-329B.en.srt", "OFJE-329C.en.srt", "广告.mp4"}
	for i, name := range names {
		files = append(files, File{Index: i, Path: name})
	}
	request := Request{TaskID: "task-1", Code: "OFJE-329", SourceRoot: source, TargetRoot: target, Action: "COPY", Plan: Classify("OFJE-329", files, true)}
	result, err := (FFmpeg{}).Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(result.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected one video and subtitle only: %v", entries)
	}
	p, err := (FFmpeg{}).inspect(context.Background(), result.VideoPath)
	if err != nil {
		t.Fatal(err)
	}
	duration, err := p.duration()
	if err != nil {
		t.Fatal(err)
	}
	if duration < 1.8 || duration > 2.4 {
		t.Fatalf("duplicate was not removed or parts missing: %f", duration)
	}
	subtitle, err := os.ReadFile(filepath.Join(result.Directory, "OFJE-329.en.srt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(subtitle), "00:00:01,") {
		t.Fatalf("second subtitle was not shifted: %s", subtitle)
	}
	// Retrying after an interrupted state save recognizes only this task's output.
	if _, err := (FFmpeg{}).Prepare(context.Background(), request); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if err := os.WriteFile(result.VideoPath, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := (FFmpeg{}).Prepare(context.Background(), request); err == nil {
		t.Fatal("modified output must not be reused")
	}
}
func TestRealFFmpegSubtitleOffAndCancelledPreparation(t *testing.T) {
	requireFFmpeg(t)
	source, target := t.TempDir(), t.TempDir()
	makeClip(t, filepath.Join(source, "SSIS-618.mkv"), "green", true)
	request := Request{TaskID: "single", Code: "SSIS-618", SourceRoot: source, TargetRoot: target, Action: "COPY", Plan: Classify("SSIS-618", []File{{Path: "SSIS-618.mkv"}}, false)}
	result, err := (FFmpeg{}).Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	p, err := (FFmpeg{}).inspect(context.Background(), result.VideoPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range p.Streams {
		if s.Type == "subtitle" {
			t.Fatal("embedded subtitle retained while disabled")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request.TaskID = "cancelled"
	request.TargetRoot = t.TempDir()
	if _, err := (FFmpeg{}).Prepare(ctx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(request.TargetRoot, request.Code)); !os.IsNotExist(err) {
		t.Fatal("cancelled job published output")
	}
}
func TestResolveRejectsTraversalAndSymlinkEscape(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "video.mkv"), []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "video.mkv"), filepath.Join(root, "escape.mkv")); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "escape.mkv"); err == nil {
		t.Fatal("symlink escaped root")
	}
	if _, err := Resolve(root, "../video.mkv"); err == nil {
		t.Fatal("traversal accepted")
	}
}

func TestRealFFmpegMovePublishesBeforeRemovingSourcesAndRecovers(t *testing.T) {
	requireFFmpeg(t)
	source, target := t.TempDir(), t.TempDir()
	makeClip(t, filepath.Join(source, "SSIS-618.mkv"), "yellow", false)
	if err := os.WriteFile(filepath.Join(source, "extra.txt"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	request := Request{TaskID: "move", Code: "SSIS-618", SourceRoot: source, TargetRoot: target, Action: "MOVE", Plan: Classify("SSIS-618", []File{{Path: "SSIS-618.mkv", Index: 0}, {Path: "extra.txt", Index: 1}}, false)}
	result, err := (FFmpeg{}).Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.VideoPath); err != nil {
		t.Fatal("movie was not published")
	}
	if _, err := os.Stat(filepath.Join(source, "SSIS-618.mkv")); !os.IsNotExist(err) {
		t.Fatal("selected move source remains")
	}
	if _, err := os.Stat(filepath.Join(source, "extra.txt")); err != nil {
		t.Fatal("unselected source was removed")
	}
	if _, err := (FFmpeg{}).Prepare(context.Background(), request); err != nil {
		t.Fatalf("MOVE restart failed after sources removed: %v", err)
	}
}
