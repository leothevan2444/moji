package torrentmeta

import (
	"strings"
	"testing"
)

func TestParseMagnetRetainsMetadataAndNormalizesV1Identity(t *testing.T) {
	const hash = "0123456789abcdef0123456789abcdef01234567"
	for _, topic := range []string{
		"urn:btih:" + hash,
		"URN:BTIH:" + strings.ToUpper(hash),
		"urn:btih:AERUKZ4JVPG66AJDIVTYTK6N54ASGRLH",
		"urn:btih:aerukz4jvpg66ajdivtytk6n54asgrlh",
	} {
		value := "magnet:?xt=urn:sha1:ignored&xt=urn:btih:bad&xt=" + topic + "&dn=ABCD%20123&tr=udp%3A%2F%2Fone.fixture.invalid%3A80&tr=https%3A%2F%2Ftwo.fixture.invalid%2Fannounce"
		parsed, err := ParseMagnet(value)
		if err != nil || parsed.URI != value || parsed.InfoHash != hash {
			t.Fatalf("magnet metadata or identity changed: %+v, %v", parsed, err)
		}
	}
}

func TestParseMagnetRejectsInvalidIdentityAndBoundsInput(t *testing.T) {
	for _, value := range []string{
		"", "https://fixture.invalid/?xt=urn:btih:" + strings.Repeat("a", 40),
		"magnet:?xt=urn:btih:a", "magnet:?xt=urn:sha1:" + strings.Repeat("a", 40),
		"magnet:?xt=urn:btih:" + strings.Repeat("0", 32),
		"magnet:?xt=urn:btih:" + strings.Repeat("g", 40),
		"magnet:?xt=urn:btmh:1220" + strings.Repeat("a", 63),
		"magnet:?xt=urn:btmh:1220" + strings.Repeat("g", 64),
		"magnet:?xt=urn:btmh:1320" + strings.Repeat("a", 64),
		"magnet://user:password@fixture.invalid/?xt=urn:btih:" + strings.Repeat("a", 40),
	} {
		if _, err := ParseMagnet(value); err == nil {
			t.Fatal("invalid magnet accepted")
		}
	}
	prefix := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40) + "&dn="
	value := prefix + strings.Repeat("x", MaxMagnetBytes-len(prefix))
	if _, err := ParseMagnet(value); err != nil {
		t.Fatal("valid boundary magnet rejected", err)
	}
	if _, err := ParseMagnet(value + "x"); err == nil {
		t.Fatal("oversized magnet accepted")
	}
}

func TestParseMagnetRetainsV2AndHybridTopics(t *testing.T) {
	const v1 = "0123456789abcdef0123456789abcdef01234567"
	v2 := strings.Repeat("A", 64)
	for _, sample := range []struct{ uri, hash string }{
		{"magnet:?xt=urn:btmh:1220" + v2 + "&tr=https%3A%2F%2Ffixture.invalid%2Fannounce", strings.ToLower(v2)},
		{"magnet:?xt=urn:btmh:1220" + v2 + "&xt=urn:btih:" + v1, v1},
		{"magnet:?xt=urn:btih:" + v1 + "&xt=urn:btmh:1220" + v2, v1},
	} {
		parsed, err := ParseMagnet(sample.uri)
		if err != nil || parsed.URI != sample.uri || parsed.InfoHash != sample.hash {
			t.Fatalf("valid v2/hybrid URI or identity changed: %+v %v", parsed, err)
		}
	}
}
