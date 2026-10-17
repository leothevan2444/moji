// Package mediafiles owns file classification and clean, single-movie delivery.
package mediafiles

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Kind string

const (
	Main         Kind = "MAIN"
	Subtitle     Kind = "SUBTITLE"
	Discard      Kind = "DISCARD"
	UnknownVideo Kind = "UNKNOWN_VIDEO"
)

type File struct {
	Index    int    `json:"index"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Kind     Kind   `json:"kind"`
	Part     int    `json:"part"`
	Selected bool   `json:"selected"`
	Reason   string `json:"reason"`
}
type Plan struct {
	PauseRequested bool   `json:"pauseRequested"`
	Files          []File `json:"files"`
	Confirmed      bool   `json:"confirmed"`
	KeepSubtitles  bool   `json:"keepSubtitles"`
	Applied        bool   `json:"applied"`
}

var ErrReview = errors.New("media file selection requires review")
var domain = regexp.MustCompile(`(?i)[a-z0-9_-]+(?:\.[a-z0-9_-]+)*\.(?:com|net|org|xyz|cc|tv|me|cn|top|vip)(?:@)?`)
var codeToken = regexp.MustCompile(`(?i)([a-z]{2,10})[-_ ]?([0-9]{2,6})`)
var partToken = regexp.MustCompile(`(?i)^(?:[-_. ]*(?:part|cd|pt)[-_. ]*([0-9]{1,2})|[-_. ]+([0-9]{1,2})|([a-z]))(?:$|[-_. ])`)
var junk = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:sample|trailer|preview|advert|advertisement|promo)(?:$|[^a-z0-9])|广告|廣告|宣传|宣傳|样片|樣片|试看|試看|预告|預告`)

func CleanName(value string) string { return domain.ReplaceAllString(value, "") }
func codeMatches(value string) [][]string {
	value = CleanName(value)
	matches := codeToken.FindAllStringSubmatchIndex(value, -1)
	result := [][]string{}
	for _, m := range matches {
		if m[0] > 0 && isAlnum(value[m[0]-1]) {
			continue
		}
		if m[1] < len(value) && value[m[1]] >= '0' && value[m[1]] <= '9' {
			continue
		}
		result = append(result, []string{strings.ToUpper(value[m[2]:m[3]]), value[m[4]:m[5]], value[m[1]:]})
	}
	return result
}
func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// ExtractCode rejects conflicting identities rather than choosing the first match.
func ExtractCode(value string) string {
	found := ""
	for _, m := range codeMatches(value) {
		code := m[0] + "-" + m[1]
		if found != "" && found != code {
			return ""
		}
		found = code
	}
	return found
}
func SameCode(a, b string) bool {
	x, y := codeMatches(a), codeMatches(b)
	if len(x) != 1 || len(y) != 1 {
		return false
	}
	number := func(s string) string {
		s = strings.TrimLeft(s, "0")
		if s == "" {
			return "0"
		}
		return s
	}
	return x[0][0] == y[0][0] && number(x[0][1]) == number(y[0][1])
}
func Video(value string) bool {
	switch strings.ToLower(path.Ext(value)) {
	case ".mp4", ".mkv", ".avi", ".wmv", ".mov", ".ts", ".m2ts", ".webm", ".mpeg", ".mpg", ".m4v", ".iso":
		return true
	}
	return false
}
func IsSubtitle(value string) bool {
	switch strings.ToLower(path.Ext(value)) {
	case ".srt", ".ass", ".ssa", ".vtt", ".sub", ".idx", ".sup", ".smi":
		return true
	}
	return false
}
func Classify(code string, files []File, keep bool) Plan {
	p := Plan{Files: append([]File(nil), files...), KeepSubtitles: keep}
	for i := range p.Files {
		f := &p.Files[i]
		f.Kind, f.Selected, f.Part, f.Reason = Discard, false, 0, "NON_MEDIA"
		switch {
		case junk.MatchString(CleanName(f.Path)):
			f.Reason = "ADVERTISEMENT_OR_SAMPLE"
		case IsSubtitle(f.Path):
			f.Kind, f.Selected, f.Reason = Subtitle, keep, "SUBTITLE_POLICY"
			if candidate := ExtractCode(path.Base(f.Path)); candidate != "" && !SameCode(candidate, code) {
				f.Selected = false
				f.Reason = "SUBTITLE_OTHER_RELEASE"
			}
		case Video(f.Path):
			f.Kind, f.Reason = UnknownVideo, "IDENTITY_UNCERTAIN"
			name := path.Base(strings.ReplaceAll(f.Path, "\\", "/"))
			for _, token := range codeMatches(strings.TrimSuffix(name, path.Ext(name))) {
				if !SameCode(token[0]+"-"+token[1], code) {
					continue
				}
				f.Kind, f.Selected, f.Reason = Main, true, "TARGET_CODE"
				if m := partToken.FindStringSubmatch(token[2]); m != nil {
					if m[3] != "" {
						f.Part = int(strings.ToUpper(m[3])[0]-'A') + 1
					} else {
						n := m[1]
						if n == "" {
							n = m[2]
						}
						f.Part, _ = strconv.Atoi(n)
					}
				}
				break
			}
		}
	}
	return p
}
func (p Plan) Videos() []File {
	files := []File{}
	for _, f := range p.Files {
		if f.Selected && f.Kind == Main {
			files = append(files, f)
		}
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Part < files[j].Part })
	return files
}
func (p Plan) Validate() error {
	videos := p.Videos()
	if len(videos) == 0 {
		return ErrReview
	}
	if !p.Confirmed {
		for _, f := range p.Files {
			if f.Kind == UnknownVideo {
				return ErrReview
			}
		}
	}
	if len(videos) > 1 {
		for i, f := range videos {
			if f.Part != i+1 {
				return fmt.Errorf("%w: duplicate, missing, or ambiguous part order", ErrReview)
			}
		}
	}
	return nil
}
func (p *Plan) Select(indices []int) error {
	if len(indices) == 0 {
		return ErrReview
	}
	selected := map[int]int{}
	for i, id := range indices {
		if _, ok := selected[id]; ok {
			return errors.New("duplicate selected file index")
		}
		selected[id] = i + 1
		valid := false
		for _, f := range p.Files {
			if f.Index == id && (f.Kind == Main || f.Kind == UnknownVideo) {
				valid = true
			}
		}
		if !valid {
			return errors.New("selected file is not a video candidate")
		}
	}
	for i := range p.Files {
		f := &p.Files[i]
		if f.Kind == Main || f.Kind == UnknownVideo {
			part, ok := selected[f.Index]
			f.Selected = ok
			if ok {
				f.Kind, f.Part, f.Reason = Main, part, "USER_SELECTED"
			} else {
				f.Reason = "USER_EXCLUDED"
			}
		}
	}
	p.Confirmed, p.Applied = true, false
	return p.Validate()
}
