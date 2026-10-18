package mediafiles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type Request struct {
	TaskID     string
	Code       string
	SourceRoot string
	TargetRoot string
	Action     string
	Plan       Plan
}
type Result struct {
	Directory string
	VideoPath string
}
type Preparer interface {
	Prepare(context.Context, Request) (Result, error)
}
type FFmpeg struct {
	Binary      string
	ProbeBinary string
}
type probe struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []stream `json:"streams"`
}
type stream struct {
	Type     string `json:"codec_type"`
	Codec    string `json:"codec_name"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Rate     string `json:"sample_rate"`
	Channels int    `json:"channels"`
	TimeBase string `json:"time_base"`
	Extra    string `json:"extradata_hash"`
}

func (p probe) duration() (float64, error) {
	n, err := strconv.ParseFloat(p.Format.Duration, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 {
		return 0, errors.New("media duration is invalid")
	}
	return n, nil
}
func binary(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// Limit diagnostics: external tools may report file metadata for a large input.
type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 32768 {
		left := 32768 - b.Len()
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}
func run(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var out, diagnostic limitedBuffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostic
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%s: %w: %s", filepath.Base(bin), err, diagnostic.String())
	}
	return out.Bytes(), nil
}
func (f FFmpeg) inspect(ctx context.Context, path string) (probe, error) {
	data, err := run(ctx, binary(f.ProbeBinary, "ffprobe"), "-v", "error", "-protocol_whitelist", "file,crypto,data", "-show_streams", "-show_format", "-show_data_hash", "sha256", "-of", "json", path)
	if err != nil {
		return probe{}, err
	}
	var p probe
	err = json.Unmarshal(data, &p)
	return p, err
}
func FileHash(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	buf := make([]byte, 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, e := file.Read(buf)
		if n > 0 {
			_, _ = h.Write(buf[:n])
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", e
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Resolve disallows traversal and symlink escapes from a qBittorrent file entry.
func Resolve(root, name string) (string, error) {
	if strings.ContainsAny(name, "\r\n\x00") || filepath.IsAbs(name) {
		return "", errors.New("unsafe torrent file path")
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(strings.ReplaceAll(name, "\\", "/"))))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("torrent file escapes source root")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("torrent entry is not a regular file")
	}
	return path, nil
}

type receipt struct {
	Signature string            `json:"signature"`
	Files     map[string]string `json:"files"`
	VideoName string            `json:"videoName,omitempty"`
}

func (f FFmpeg) Prepare(ctx context.Context, r Request) (Result, error) {
	if err := r.Plan.Validate(); err != nil {
		return Result{}, err
	}
	if ExtractCode(r.Code) != r.Code || strings.ContainsAny(r.TaskID, "/\\\x00") || r.TaskID == "" {
		return Result{}, errors.New("invalid delivery identity")
	}
	if r.Action != "COPY" && r.Action != "MOVE" {
		return Result{}, errors.New("clean video delivery requires COPY or MOVE")
	}
	// Downloaded sources stay untouched until publication succeeds, including MOVE.
	if strings.ContainsAny(r.SourceRoot+r.TargetRoot, "\r\n\x00") {
		return Result{}, errors.New("invalid media root")
	}
	r.SourceRoot = filepath.Clean(r.SourceRoot)
	r.TargetRoot = filepath.Clean(r.TargetRoot)
	if !filepath.IsAbs(r.SourceRoot) || !filepath.IsAbs(r.TargetRoot) {
		return Result{}, errors.New("media roots must be absolute")
	}
	final := filepath.Join(r.TargetRoot, r.Code)
	result := Result{Directory: final, VideoPath: filepath.Join(final, r.Code+".mkv")}
	encoded, _ := json.Marshal(struct {
		Code, Source string
		Plan         Plan
	}{r.Code, r.SourceRoot, r.Plan})
	sig := sha256.Sum256(encoded)
	signature := hex.EncodeToString(sig[:])
	receiptDir := filepath.Join(r.TargetRoot, ".moji-receipts")
	receiptPath := filepath.Join(receiptDir, r.TaskID+".json")
	if _, err := os.Lstat(final); err == nil {
		data, err := os.ReadFile(receiptPath)
		if err != nil {
			return Result{}, errors.New("delivery target already exists and is not owned by this task")
		}
		var saved receipt
		if json.Unmarshal(data, &saved) != nil || saved.Signature != signature {
			return Result{}, errors.New("delivery target has a different file plan")
		}
		entries, err := os.ReadDir(final)
		if err != nil {
			return Result{}, err
		}
		if len(entries) != len(saved.Files) || len(saved.Files) == 0 {
			return Result{}, errors.New("delivery output has changed")
		}
		for name, want := range saved.Files {
			actual, err := FileHash(ctx, filepath.Join(final, name))
			if err != nil || actual != want {
				return Result{}, errors.New("delivery output has changed")
			}
		}
		videoName := saved.VideoName
		if videoName == "" {
			videoName = r.Code + ".mkv" // Receipts from before direct single-file delivery.
		}
		if filepath.Base(videoName) != videoName || saved.Files[videoName] == "" {
			return Result{}, errors.New("delivery receipt has no movie output")
		}
		result.VideoPath = filepath.Join(final, videoName)
		if err := cleanupSources(ctx, r); err != nil {
			return Result{}, err
		}
		return result, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, err
	}
	if err := os.MkdirAll(r.TargetRoot, 0755); err != nil {
		return Result{}, err
	}
	staging, err := os.MkdirTemp(r.TargetRoot, ".moji-preparing-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(staging)
	// Other Stash scans should ignore the private work directory as well.
	if err := os.WriteFile(filepath.Join(staging, ".stashignore"), []byte("*\n"), 0600); err != nil {
		return Result{}, err
	}
	videos := r.Plan.Videos()
	sources := []string{}
	parts := []File{}
	seen := map[string]File{}
	partAliases := map[int]int{}
	sourceHashes := []string{}
	for _, v := range videos {
		source, err := Resolve(r.SourceRoot, v.Path)
		if err != nil {
			return Result{}, err
		}
		info, err := os.Stat(source)
		if err != nil {
			return Result{}, err
		}
		if v.Size > 0 && info.Size() != v.Size {
			return Result{}, errors.New("video file size does not match the torrent manifest")
		}
		hash, err := FileHash(ctx, source)
		if err != nil {
			return Result{}, err
		}
		if previous, ok := seen[hash]; ok {
			partAliases[v.Part] = previous.Part
			continue
		}
		seen[hash] = v
		partAliases[v.Part] = v.Part
		sources = append(sources, source)
		parts = append(parts, v)
		sourceHashes = append(sourceHashes, hash)
	}
	if len(sources) == 0 {
		return Result{}, errors.New("no main video remains after deduplication")
	}
	output := filepath.Join(staging, "video.partial")
	videoName := r.Code + ".mkv"
	durations := []float64{0} // Single-file subtitles need no time offsets.
	if len(sources) == 1 {
		videoName = r.Code + filepath.Ext(parts[0].Path)
		if err := copyMedia(ctx, sources[0], output); err != nil {
			return Result{}, err
		}
	} else {
		durations, err = f.merge(ctx, r, sources, staging, output)
		if err != nil {
			return Result{}, err
		}
	}
	result.VideoPath = filepath.Join(final, videoName)
	outputs := map[string]string{videoName: output}
	if r.Plan.KeepSubtitles {
		if err := f.prepareSubtitles(ctx, r, parts, durations, staging, outputs, partAliases); err != nil {
			return Result{}, err
		}
	}
	saved := receipt{Signature: signature, Files: map[string]string{}, VideoName: videoName}
	publish := filepath.Join(staging, "publish")
	if err := os.Mkdir(publish, 0755); err != nil {
		return Result{}, err
	}
	for name, source := range outputs {
		hash, err := FileHash(ctx, source)
		if err != nil {
			return Result{}, err
		}
		if name == videoName && len(sources) == 1 && hash != sourceHashes[0] {
			return Result{}, errors.New("copied video does not match the source")
		}
		saved.Files[name] = hash
		if err := os.Rename(source, filepath.Join(publish, name)); err != nil {
			return Result{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(receiptDir, 0755); err != nil {
		return Result{}, err
	}
	data, _ := json.Marshal(saved)
	tmp, err := os.CreateTemp(receiptDir, "receipt-")
	if err != nil {
		return Result{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	_, err = tmp.Write(data)
	closeErr := tmp.Close()
	if err != nil {
		return Result{}, err
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	if err := os.Rename(tmpPath, receiptPath); err != nil {
		return Result{}, err
	}
	// A pre-existing target is never replaced. Publication is on the same FS.
	if _, err := os.Lstat(final); !errors.Is(err, os.ErrNotExist) {
		return Result{}, errors.New("delivery target appeared during preparation")
	}
	if err := os.Rename(publish, final); err != nil {
		return Result{}, err
	}
	if err := cleanupSources(ctx, r); err != nil {
		return Result{}, err
	}
	return result, nil
}

// merge probes only distinct movie parts, then concatenates compatible streams.
func (f FFmpeg) merge(ctx context.Context, r Request, sources []string, staging, output string) ([]float64, error) {
	durations := make([]float64, 0, len(sources))
	probes := make([]probe, 0, len(sources))
	for _, source := range sources {
		p, err := f.inspect(ctx, source)
		if err != nil {
			return nil, err
		}
		d, err := p.duration()
		if err != nil {
			return nil, err
		}
		probes = append(probes, p)
		durations = append(durations, d)
	}
	selectedStreams := func(p probe) []stream {
		out := []stream{}
		for _, s := range p.Streams {
			if s.Type == "video" || s.Type == "audio" || r.Plan.KeepSubtitles && s.Type == "subtitle" {
				out = append(out, s)
			}
		}
		return out
	}
	first := selectedStreams(probes[0])
	hasVideo := false
	for _, stream := range first {
		hasVideo = hasVideo || stream.Type == "video"
	}
	if !hasVideo {
		return nil, errors.New("main file has no video stream")
	}
	for _, p := range probes[1:] {
		if !reflect.DeepEqual(first, selectedStreams(p)) {
			return nil, errors.New("MERGE_INCOMPATIBLE: parts have different stream parameters; a lossless merge is not safe")
		}
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-n"}
	var list strings.Builder
	for _, source := range sources {
		list.WriteString("file '" + strings.ReplaceAll(source, "'", "'\\''") + "'\n")
	}
	listPath := filepath.Join(staging, "concat.txt")
	if err := os.WriteFile(listPath, []byte(list.String()), 0600); err != nil {
		return nil, err
	}
	args = append(args, "-f", "concat", "-safe", "0", "-protocol_whitelist", "file,crypto,data", "-i", listPath)
	args = append(args, "-map", "0:v:0", "-map", "0:a?")
	if r.Plan.KeepSubtitles {
		args = append(args, "-map", "0:s?")
	}
	args = append(args, "-map_metadata", "-1", "-map_chapters", "-1", "-c", "copy")
	if r.Plan.KeepSubtitles {
		index := 0
		for _, stream := range probes[0].Streams {
			if stream.Type == "subtitle" {
				if stream.Codec == "mov_text" || stream.Codec == "webvtt" {
					args = append(args, fmt.Sprintf("-c:s:%d", index), "srt")
				}
				index++
			}
		}
	}
	args = append(args, "-f", "matroska", output)
	if _, err := run(ctx, binary(f.Binary, "ffmpeg"), args...); err != nil {
		return nil, err
	}
	p, err := f.inspect(ctx, output)
	if err != nil {
		return nil, err
	}
	actual, err := p.duration()
	if err != nil {
		return nil, err
	}
	total := 0.0
	for _, d := range durations {
		total += d
	}
	if math.Abs(actual-total) > math.Max(.25*float64(len(sources)), total*.001) {
		return nil, errors.New("merged video duration does not match its parts")
	}
	return durations, nil
}

func copyMedia(ctx context.Context, source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	buf := make([]byte, 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			out.Close()
			return err
		}
		n, e := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				out.Close()
				return err
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			out.Close()
			return e
		}
	}
	return out.Close()
}

func cleanupSources(ctx context.Context, r Request) error {
	if r.Action != "MOVE" {
		return nil
	}
	for _, entry := range r.Plan.Files {
		if !entry.Selected {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		source, err := Resolve(r.SourceRoot, entry.Path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if err := os.Remove(source); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
