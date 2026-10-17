package mediafiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var srtTime = regexp.MustCompile(`(?m)(\d{2,}):(\d{2}):(\d{2})[,.](\d{3})[ \t]+-->[ \t]+(\d{2,}):(\d{2}):(\d{2})[,.](\d{3})`)
var suffixSafe = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

func clock(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}
func shiftSRT(text string, offset int64) (string, error) {
	count := 0
	text = strings.ReplaceAll(strings.TrimPrefix(text, "\ufeff"), "\r\n", "\n")
	text = srtTime.ReplaceAllStringFunc(text, func(line string) string {
		m := srtTime.FindStringSubmatch(line)
		values := make([]int64, 8)
		for i := range values {
			values[i], _ = strconv.ParseInt(m[i+1], 10, 64)
		}
		start := values[0]*3600000 + values[1]*60000 + values[2]*1000 + values[3]
		end := values[4]*3600000 + values[5]*60000 + values[6]*1000 + values[7]
		count++
		return clock(start+offset) + " --> " + clock(end+offset)
	})
	if count == 0 && strings.TrimSpace(text) != "" {
		return "", errors.New("subtitle has no supported timestamps")
	}
	return text, nil
}
func renumberSRT(text string) string {
	blocks := strings.Split(strings.TrimSpace(text), "\n\n")
	result := []string{}
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		timeIndex := -1
		for i, line := range lines {
			if srtTime.MatchString(line) {
				timeIndex = i
				break
			}
		}
		if timeIndex < 0 {
			continue
		}
		result = append(result, fmt.Sprintf("%d\n%s", len(result)+1, strings.Join(lines[timeIndex:], "\n")))
	}
	return strings.Join(result, "\n\n") + "\n"
}

type subtitlePiece struct {
	file   File
	part   int
	offset int64
}

func (f FFmpeg) prepareSubtitles(ctx context.Context, r Request, parts []File, durations []float64, staging string, outputs map[string]string, partAliases map[int]int) error {
	groups := map[string][]subtitlePiece{}
	offsets := map[int]int64{}
	var offset int64
	for i, v := range parts {
		offsets[v.Part] = offset
		offset += int64(durations[i]*1000 + .5)
	}
	for _, entry := range r.Plan.Files {
		if !entry.Selected || entry.Kind != Subtitle {
			continue
		}
		stem := strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
		clean := strings.ToLower(CleanName(stem))
		part := -1
		label := ""
		longest := 0
		for _, video := range r.Plan.Videos() {
			v := strings.ToLower(CleanName(strings.TrimSuffix(filepath.Base(video.Path), filepath.Ext(video.Path))))
			if (clean == v || strings.HasPrefix(clean, v+".") || strings.HasPrefix(clean, v+"_") || strings.HasPrefix(clean, v+"-")) && len(v) > longest {
				part, label, longest = video.Part, stem[len(stem)-(len(clean)-len(v)):], len(v)
			}
		}
		if part < 0 {
			// A subtitle for the whole movie is retained without a part-time offset.
			if SameCode(stem, r.Code) {
				token := codeToken.FindStringIndex(CleanName(stem))
				label = CleanName(stem)[token[1]:]
				if partToken.MatchString(label) {
					return errors.New("subtitle refers to an unselected video part")
				}
				part = 0
			} else if len(parts) == 1 {
				part = parts[0].Part
				label = stem
			} else {
				return errors.New("subtitle cannot be associated with the movie or a selected part")
			}
		}
		if canonical, ok := partAliases[part]; ok {
			part = canonical
		}
		label = strings.Trim(suffixSafe.ReplaceAllString(label, "_"), "._-")
		ext := strings.ToLower(filepath.Ext(entry.Path))
		key := label + ext
		duplicate := false
		for _, existing := range groups[key] {
			if existing.part != part {
				continue
			}
			a, err := Resolve(r.SourceRoot, existing.file.Path)
			if err != nil {
				return err
			}
			b, err := Resolve(r.SourceRoot, entry.Path)
			if err != nil {
				return err
			}
			hashA, err := FileHash(ctx, a)
			if err != nil {
				return err
			}
			hashB, err := FileHash(ctx, b)
			if err != nil {
				return err
			}
			if hashA == hashB {
				duplicate = true
				break
			}
			return errors.New("multiple different subtitle files have the same track and part")
		}
		if duplicate {
			continue
		}
		groups[key] = append(groups[key], subtitlePiece{file: entry, part: part, offset: offsets[part]})
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hashes := map[string]bool{}
	for track, key := range keys {
		pieces := groups[key]
		sort.SliceStable(pieces, func(i, j int) bool { return pieces[i].part < pieces[j].part })
		ext := strings.ToLower(filepath.Ext(pieces[0].file.Path))
		name := r.Code
		if strings.TrimSuffix(key, ext) != "" {
			name += "." + strings.TrimSuffix(key, ext)
		}
		if len(pieces) == 1 && (len(parts) == 1 || pieces[0].part == 0) {
			source, err := Resolve(r.SourceRoot, pieces[0].file.Path)
			if err != nil {
				return err
			}
			if ext == ".srt" || ext == ".ass" || ext == ".ssa" || ext == ".vtt" || ext == ".smi" {
				info, err := f.inspect(ctx, source)
				if err != nil {
					return err
				}
				valid := false
				for _, stream := range info.Streams {
					valid = valid || stream.Type == "subtitle"
				}
				if !valid {
					return errors.New("selected subtitle file has no subtitle stream")
				}
			}
			destination := filepath.Join(staging, fmt.Sprintf("subtitle-%d.partial", track))
			if err := copyMedia(ctx, source, destination); err != nil {
				return err
			}
			h, err := FileHash(ctx, destination)
			if err != nil {
				return err
			}
			if hashes[ext+":"+h] {
				continue
			}
			hashes[ext+":"+h] = true
			outputs[name+ext] = destination
			continue
		}
		if ext != ".srt" && ext != ".ass" && ext != ".ssa" && ext != ".vtt" && ext != ".smi" {
			return errors.New("SUBTITLE_MERGE_UNSUPPORTED: bitmap/unknown part subtitles cannot be time-shifted; disable subtitle retention or supply whole-movie subtitles")
		}
		var merged strings.Builder
		for i, piece := range pieces {
			if piece.part == 0 {
				return errors.New("whole-movie and per-part subtitles use the same track")
			}
			source, err := Resolve(r.SourceRoot, piece.file.Path)
			if err != nil {
				return err
			}
			textPath := filepath.Join(staging, fmt.Sprintf("sub-%d-%d.srt", track, i))
			if _, err := run(ctx, binary(f.Binary, "ffmpeg"), "-hide_banner", "-loglevel", "error", "-nostdin", "-n", "-i", source, "-map", "0:s:0", "-c:s", "srt", textPath); err != nil {
				return err
			}
			data, err := os.ReadFile(textPath)
			if err != nil {
				return err
			}
			shifted, err := shiftSRT(string(data), piece.offset)
			if err != nil {
				return err
			}
			merged.WriteString(strings.TrimSpace(shifted))
			merged.WriteString("\n\n")
		}
		destination := filepath.Join(staging, fmt.Sprintf("subtitle-%d.partial", track))
		if err := os.WriteFile(destination, []byte(renumberSRT(merged.String())), 0644); err != nil {
			return err
		}
		h, err := FileHash(ctx, destination)
		if err != nil {
			return err
		}
		if hashes[".srt:"+h] {
			continue
		}
		hashes[".srt:"+h] = true
		// Different input formats must not silently overwrite a previous track.
		outName := name + ".srt"
		if _, exists := outputs[outName]; exists {
			outName = fmt.Sprintf("%s.track%d.srt", name, track+1)
		}
		outputs[outName] = destination
	}
	return nil
}
