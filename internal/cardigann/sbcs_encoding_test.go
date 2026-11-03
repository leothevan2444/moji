package cardigann

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

func TestLegacyEncodingMatchesDotNetRuntimeReference(t *testing.T) {
	var reference struct {
		Runtime   string `json:"runtime"`
		CodePages map[string]struct {
			Name              string                       `json:"name"`
			EncoderBytes      int                          `json:"encoderBytes"`
			EncoderSHA256     string                       `json:"encoderSha256"`
			DecoderCharacters int                          `json:"decoderCharacters"`
			DecoderSHA256     string                       `json:"decoderSha256"`
			Samples           []struct{ Text, Hex string } `json:"samples"`
		} `json:"codePages"`
	}
	data, err := os.ReadFile("testdata/dotnet-encoding-v9.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Runtime != "9.0.0" || len(reference.CodePages) != 8 {
		t.Fatal("unexpected reference runtime or coverage")
	}
	// Invalid standalone UTF-16 surrogates become RuneError in Go input.
	// Both use '?' in these code pages. The adjacent surrogate pair in the
	// .NET BMP input produces two '?' bytes, preserving this comparison.
	bmp := make([]rune, 65536)
	for i := range bmp {
		bmp[i] = rune(i)
	}
	input := string(bmp)
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
	}
	for _, expected := range reference.CodePages {
		t.Run(expected.Name, func(t *testing.T) {
			codec, err := declaredEncoding(expected.Name)
			if err != nil || codec == nil {
				t.Fatalf("declared codec unavailable: %v", err)
			}
			encoded, err := codec.NewEncoder().String(input)
			if err != nil {
				t.Fatal(err)
			}
			if len(encoded) != expected.EncoderBytes || encodedDigest(encoded) != expected.EncoderSHA256 {
				t.Fatal("BMP encoder differs from official runtime", len(encoded), encodedDigest(encoded))
			}
			decoded, err := codec.NewDecoder().Bytes(allBytes)
			if err != nil {
				t.Fatal(err)
			}
			if utf8.RuneCount(decoded) != expected.DecoderCharacters || encodedDigest(string(decoded)) != expected.DecoderSHA256 {
				t.Fatal("single-byte decoder differs from official runtime", encodedDigest(string(decoded)))
			}
			for _, sample := range expected.Samples {
				actual, err := codec.NewEncoder().String(sample.Text)
				if err != nil || hex.EncodeToString([]byte(actual)) != sample.Hex {
					t.Errorf("sample %q: %x, %v; expected %s", sample.Text, actual, err, sample.Hex)
				}
			}
		})
	}
}

func encodedDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TestSingleByteEncodingRespectsStreamingBoundaries(t *testing.T) {
	codec, err := declaredEncoding("windows-1251")
	if err != nil {
		t.Fatal(err)
	}
	encoder := codec.NewEncoder()
	runeBytes := []byte("ē")
	buffer := make([]byte, 2)
	if nDst, nSrc, err := encoder.Transform(buffer, runeBytes[:1], false); err != transform.ErrShortSrc || nDst != 0 || nSrc != 0 {
		t.Fatalf("partial UTF-8 rune consumed: %d %d %v", nDst, nSrc, err)
	}
	if nDst, nSrc, err := encoder.Transform(buffer, runeBytes, true); err != nil || nDst != 1 || nSrc != len(runeBytes) || buffer[0] != 'e' {
		t.Fatalf("best-fit rune retry failed: %d %d %v", nDst, nSrc, err)
	}
	if nDst, nSrc, err := encoder.Transform(buffer[:1], []byte("😀"), true); err != transform.ErrShortDst || nDst != 0 || nSrc != 0 {
		t.Fatalf("surrogate replacement was split: %d %d %v", nDst, nSrc, err)
	}
	if nDst, nSrc, err := encoder.Transform(buffer, []byte("😀"), true); err != nil || nDst != 2 || nSrc != 4 || string(buffer) != "??" {
		t.Fatalf("surrogate retry failed: %d %d %v", nDst, nSrc, err)
	}
	decoder := codec.NewDecoder()
	if nDst, nSrc, err := decoder.Transform(buffer[:1], []byte{0xc6}, true); err != transform.ErrShortDst || nDst != 0 || nSrc != 0 {
		t.Fatalf("partial decoded rune emitted: %d %d %v", nDst, nSrc, err)
	}
	if nDst, nSrc, err := decoder.Transform(buffer, []byte{0xc6}, true); err != nil || nDst != 2 || nSrc != 1 || string(buffer) != "Ж" {
		t.Fatalf("decoded rune retry failed: %d %d %v", nDst, nSrc, err)
	}
}
