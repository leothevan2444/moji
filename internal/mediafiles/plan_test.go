package mediafiles

import (
	"errors"
	"reflect"
	"testing"
)

func TestClassifyFourKindsAndContextualCode(t *testing.T) {
	files := []File{{Index: 0, Path: "hdd800.com@ssis00618A.mp4"}, {Index: 1, Path: "gc_55.com@SSIS-618B.mp4"}, {Index: 2, Path: "SSIS-618A.en.srt"}, {Index: 3, Path: "广告.mp4"}, {Index: 4, Path: "sample.mp4"}, {Index: 5, Path: "screenshot.jpg"}, {Index: 6, Path: "mystery.mp4"}}
	plan := Classify("SSIS-618", files, true)
	kinds := []Kind{}
	for _, f := range plan.Files {
		kinds = append(kinds, f.Kind)
	}
	if !reflect.DeepEqual(kinds, []Kind{Main, Main, Subtitle, Discard, Discard, Discard, UnknownVideo}) {
		t.Fatalf("wrong classification: %+v", plan.Files)
	}
	if plan.Files[0].Part != 1 || plan.Files[1].Part != 2 {
		t.Fatalf("wrong part order: %+v", plan.Files)
	}
	if !errors.Is(plan.Validate(), ErrReview) {
		t.Fatal("unknown video must require review")
	}
	if err := plan.Select([]int{1, 0}); err != nil {
		t.Fatal(err)
	}
	videos := plan.Videos()
	if videos[0].Index != 1 || videos[1].Index != 0 {
		t.Fatal("manual ordering was lost")
	}
	if !plan.Files[2].Selected {
		t.Fatal("subtitles must be retained regardless of language")
	}
	disabled := Classify("SSIS-618", files, false)
	if disabled.Files[2].Selected {
		t.Fatal("subtitle switch was ignored")
	}
}
func TestIdentityParsingDoesNotChooseAdvertisingDomain(t *testing.T) {
	cases := map[string]string{"hdd800.com@SSIS-618.mp4": "SSIS-618", "gc_55.com@SSIS-618.mp4": "SSIS-618", "OFJE-329A.mp4": "OFJE-329", "OFJE-329-3.mp4": "OFJE-329", "ssis618": "SSIS-618", "ssis00618": "SSIS-00618", "SSIS-618 SONE-001": ""}
	for value, want := range cases {
		if got := ExtractCode(value); got != want {
			t.Errorf("%s: got %s want %s", value, got, want)
		}
	}
	if !SameCode("ssis00618", "SSIS-618") || SameCode("SSIS-618", "SSIS-619") {
		t.Fatal("contextual alias comparison failed")
	}
	if ExtractCode("SONE-001") != "SONE-001" {
		t.Fatal("canonical zero padding must be retained")
	}
}
func TestPartsMustBeUnambiguousAndOrdered(t *testing.T) {
	for _, names := range [][]string{{"OFJE-329-1.mp4", "OFJE-329-3.mp4"}, {"OFJE-329.mp4", "OFJE-329.mkv"}, {"OFJE-329A.mp4", "OFJE-329A.mkv"}} {
		files := []File{}
		for i, name := range names {
			files = append(files, File{Index: i, Path: name})
		}
		p := Classify("OFJE-329", files, false)
		if !errors.Is(p.Validate(), ErrReview) {
			t.Fatalf("ambiguous files passed: %+v", p)
		}
	}
	p := Classify("OFJE-329", []File{{Index: 10, Path: "OFJE-329-10.mp4"}, {Index: 2, Path: "OFJE-329-2.mp4"}}, false)
	if p.Videos()[0].Index != 2 {
		t.Fatal("part order must be numeric")
	}
	if err := p.Select([]int{2, 2}); err == nil {
		t.Fatal("duplicate indices accepted")
	}
	if err := p.Select([]int{999}); err == nil {
		t.Fatal("unknown index accepted")
	}
}
func TestSubtitleTimeShift(t *testing.T) {
	text := "1\r\n00:00:01,250 --> 00:00:02,500\r\nHello\r\n"
	shifted, err := shiftSRT(text, 90500)
	if err != nil {
		t.Fatal(err)
	}
	if got := renumberSRT(shifted); got != "1\n00:01:31,750 --> 00:01:33,000\nHello\n" {
		t.Fatalf("wrong subtitle shift: %q", got)
	}
}
