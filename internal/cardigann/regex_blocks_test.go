package cardigann

import (
	"fmt"
	"testing"
)

func TestRegexUnicodeBlocksUseDotNetBMPBoundaries(t *testing.T) {
	for _, block := range []struct {
		name        string
		first, last rune
	}{
		{"IsCyrillic", '\u0400', '\u04FF'},
		{"IsCJKUnifiedIdeographs", '\u4E00', '\u9FFF'},
	} {
		for _, sample := range []struct {
			pattern string
			negated bool
		}{
			{`^\p{%s}$`, false}, {`^[\p{%s}]$`, false},
			{`^\P{%s}$`, true}, {`^[\P{%s}]$`, true},
		} {
			pattern := fmt.Sprintf(sample.pattern, block.name)
			re, err := boundedRegex(pattern)
			if err != nil {
				t.Fatal(pattern, err)
			}
			for _, r := range []rune{block.first - 1, block.first, block.last, block.last + 1} {
				want := r >= block.first && r <= block.last
				if sample.negated {
					want = !want
				}
				got, err := re.MatchString(string(r))
				if err != nil || got != want {
					t.Errorf("%s matches U+%04X=%v, want %v (%v)", pattern, r, got, want, err)
				}
			}
		}
	}
}

func TestRegexBlockNormalizationPreservesEscapedLiteralsAndUnknownErrors(t *testing.T) {
	re, err := boundedRegex(`^\\p\{IsCJKUnifiedIdeographs\}$`)
	if err != nil {
		t.Fatal(err)
	}
	if matches, err := re.MatchString(`\p{IsCJKUnifiedIdeographs}`); err != nil || !matches {
		t.Fatal("escaped property literal was rewritten", err)
	}
	if _, err := boundedRegex(`\p{IsNotARealBlock}`); err == nil {
		t.Fatal("unknown property silently accepted")
	}
}
