package film

import (
	"strings"
	"testing"
	"time"
)

func TestCompleteCodesRemainDistinct(t *testing.T) {
	for _, code := range []string{"300MIUM-1437", "T38-042", "081124_001-1pon", "34ID-003", "MMR-AA244", "MBR-AA244", "SGKI-105A", "SGKI-105B"} {
		v := Snapshot{Endpoint: "https://example.test/graphql", SceneID: "scene", Code: code}
		if v.Validate() != nil || Code(" "+code+" ") != strings.ToUpper(code) || !Match(code, "[FHD] "+code+".mp4") {
			t.Fatal("complete identity rejected", code)
		}
	}
	for _, pair := range [][2]string{{"SGKI-119", "SGKI-119B"}, {"300MIUM-1437", "MIUM-1437"}, {"MMR-AA244", "MBR-AA244"}, {"SGKI-105A", "SGKI-105B"}, {"ABC-123", "ABC-1234"}, {"ABC-123", "XABC-123"}, {"ABC-123", "ABC-00123"}, {"081124_001-1pon", "081124_001-carib"}} {
		if Match(pair[0], pair[1]) {
			t.Fatal("different identities matched", pair)
		}
	}
}
func TestFilmDigestAndClone(t *testing.T) {
	v := Snapshot{Endpoint: "https://example.test/graphql", SceneID: "scene", Code: "T38-042", Performers: []Person{{ID: "one", Name: "Actor"}}}
	copy := v.Clone()
	copy.ConfirmedAt = time.Now()
	if v.Digest() != copy.Digest() {
		t.Fatal("observation time invalidates confirmation")
	}
	copy.Performers[0].Name = "Changed"
	if v.Performers[0].Name != "Actor" || copy.Digest() == v.Digest() {
		t.Fatal("mutable snapshot or incomplete digest")
	}
	for _, code := range []string{"../film", "a/b", ".", "..", "a\\b"} {
		name := SafeName(code)
		if name == code || name == "" {
			t.Fatal("unsafe output name", code)
		}
	}
}
