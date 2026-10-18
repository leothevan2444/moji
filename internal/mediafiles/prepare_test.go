package mediafiles

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
func TestRealFFmpegSingleFilePreservesEmbeddedSubtitlesAndCancelledPreparation(t *testing.T) {
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
	hasSubtitle := false
	for _, s := range p.Streams {
		hasSubtitle = hasSubtitle || s.Type == "subtitle"
	}
	if !hasSubtitle {
		t.Fatal("direct delivery must preserve embedded subtitles")
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

func TestRealFFmpegMultiplePartsStillMergeAndApplySubtitlePolicy(t *testing.T) {
	requireFFmpeg(t)
	source, target := t.TempDir(), t.TempDir()
	makeClip(t, filepath.Join(source, "SSIS-618A.mkv"), "red", true)
	makeClip(t, filepath.Join(source, "SSIS-618B.mkv"), "blue", true)
	request := Request{TaskID: "parts", Code: "SSIS-618", SourceRoot: source, TargetRoot: target, Action: "COPY", Plan: Classify("SSIS-618", []File{{Path: "SSIS-618A.mkv"}, {Path: "SSIS-618B.mkv"}}, false)}
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
			t.Fatal("merged output retained disabled subtitles")
		}
	}
	duration, err := p.duration()
	if err != nil || duration < 1.8 || duration > 2.4 {
		t.Fatalf("merged duration = %f, err = %v", duration, err)
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

// Deliberately invalid tool paths make any accidental media-tool call fail.
func withoutMediaTools(t *testing.T) FFmpeg {
	t.Helper()
	return FFmpeg{Binary: filepath.Join(t.TempDir(), "missing-ffmpeg"), ProbeBinary: filepath.Join(t.TempDir(), "missing-ffprobe")}
}

func TestSingleFileDeliveryWithoutMediaTools(t *testing.T) {
	for _, action := range []string{"COPY", "MOVE"} {
		for _, keep := range []bool{false, true} {
			t.Run(action+"/subtitles="+fmt.Sprint(keep), func(t *testing.T) {
				source, target := t.TempDir(), t.TempDir()
				video := []byte("original container, tracks, chapters and metadata")
				subtitle := []byte("1\n00:00:00,100 --> 00:00:00,500\nCaption\n")
				for name, data := range map[string][]byte{"hhd800.com@IDBD-998.mp4": video, "IDBD-998.en.srt": subtitle, "extra.txt": []byte("extra")} {
					if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
				request := Request{TaskID: "single", Code: "IDBD-998", SourceRoot: source, TargetRoot: target, Action: action,
					Plan: Classify("IDBD-998", []File{{Index: 0, Path: "hhd800.com@IDBD-998.mp4", Size: int64(len(video))}, {Index: 1, Path: "IDBD-998.en.srt"}, {Index: 2, Path: "extra.txt"}}, keep)}
				preparer := withoutMediaTools(t)
				result, err := preparer.Prepare(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				want := filepath.Join(target, "IDBD-998", "IDBD-998.mp4")
				if result.VideoPath != want {
					t.Fatalf("video path = %s, want %s", result.VideoPath, want)
				}
				got, err := os.ReadFile(want)
				if err != nil || !bytes.Equal(got, video) {
					t.Fatalf("video bytes changed: %v", err)
				}
				entries, err := os.ReadDir(result.Directory)
				if err != nil {
					t.Fatal(err)
				}
				count := 1
				if keep {
					count++
					got, err := os.ReadFile(filepath.Join(result.Directory, "IDBD-998.en.srt"))
					if err != nil || !bytes.Equal(got, subtitle) {
						t.Fatalf("subtitle bytes changed: %v", err)
					}
				}
				if len(entries) != count {
					t.Fatalf("unexpected delivered files: %v", entries)
				}
				for name, removed := range map[string]bool{"hhd800.com@IDBD-998.mp4": action == "MOVE", "IDBD-998.en.srt": action == "MOVE" && keep, "extra.txt": false} {
					_, err := os.Stat(filepath.Join(source, name))
					if removed && !os.IsNotExist(err) || !removed && err != nil {
						t.Fatalf("source %s removed=%v: %v", name, removed, err)
					}
				}
				retried, err := preparer.Prepare(context.Background(), request)
				if err != nil || retried != result {
					t.Fatalf("retry result=%+v err=%v", retried, err)
				}
				if err := os.WriteFile(want, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := preparer.Prepare(context.Background(), request); err == nil {
					t.Fatal("modified output reused")
				}
			})
		}
	}
}

func TestDuplicatePartsDeliverOriginalWithoutMediaTools(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	for _, name := range []string{"IDBD-998A.mp4", "IDBD-998B.mp4"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("identical movie"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := Request{TaskID: "duplicates", Code: "IDBD-998", SourceRoot: source, TargetRoot: target, Action: "COPY", Plan: Classify("IDBD-998", []File{{Path: "IDBD-998A.mp4"}, {Path: "IDBD-998B.mp4"}}, false)}
	result, err := withoutMediaTools(t).Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(result.VideoPath) != ".mp4" {
		t.Fatalf("deduplicated movie was remuxed: %+v", result)
	}
}

func TestSingleFileDeliveryRejectsSizeMismatchAndCancellation(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "IDBD-998.mp4"), []byte("video"), 0600); err != nil {
		t.Fatal(err)
	}
	request := Request{TaskID: "blocked", Code: "IDBD-998", SourceRoot: source, TargetRoot: target, Action: "MOVE", Plan: Classify("IDBD-998", []File{{Path: "IDBD-998.mp4", Size: 100}}, false)}
	preparer := withoutMediaTools(t)
	if _, err := preparer.Prepare(context.Background(), request); err == nil || !strings.Contains(err.Error(), "size") {
		t.Fatalf("expected size mismatch, got %v", err)
	}
	request.Plan.Files[0].Size = 5
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := preparer.Prepare(ctx, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, request.Code)); !os.IsNotExist(err) {
		t.Fatal("failed preparation published output")
	}
	if _, err := os.Stat(filepath.Join(source, "IDBD-998.mp4")); err != nil {
		t.Fatal("failed MOVE removed source")
	}
}

func TestLegacyReceiptRecoversRemuxedSingleFile(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "IDBD-998.mp4"), []byte("movie"), 0600); err != nil {
		t.Fatal(err)
	}
	request := Request{TaskID: "legacy", Code: "IDBD-998", SourceRoot: source, TargetRoot: target, Action: "MOVE", Plan: Classify("IDBD-998", []File{{Path: "IDBD-998.mp4"}}, false)}
	preparer := withoutMediaTools(t)
	result, err := preparer.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	// Model an old .mkv output/receipt without the new VideoName field.
	legacyPath := filepath.Join(result.Directory, "IDBD-998.mkv")
	if err := os.Rename(result.VideoPath, legacyPath); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(target, ".moji-receipts", "legacy.json")
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved receipt
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Files["IDBD-998.mkv"] = saved.Files[saved.VideoName]
	delete(saved.Files, saved.VideoName)
	saved.VideoName = ""
	data, err = json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	result, err = preparer.Prepare(context.Background(), request)
	if err != nil || result.VideoPath != legacyPath {
		t.Fatalf("legacy recovery: %+v, %v", result, err)
	}
}
