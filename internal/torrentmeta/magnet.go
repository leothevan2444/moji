package torrentmeta

import (
	"encoding/base32"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
)

const MaxMagnetBytes = 65536

// Magnet retains the URI's query, including every tracker and display-name
// parameter. InfoHash is the canonical lowercase hex v1 or v2 digest, preferring
// v1 for hybrid links. This validates the URI, not the referenced metainfo.
type Magnet struct {
	URI      string
	InfoHash string
}

func ParseMagnet(value string) (Magnet, error) {
	if len(value) > MaxMagnetBytes {
		return Magnet{}, errors.New("magnet exceeds byte budget")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "magnet" || parsed.User != nil {
		return Magnet{}, errors.New("invalid magnet")
	}
	var v2Hash string
	for _, xt := range parsed.Query()["xt"] {
		// BEP 9/52 encode a v2 SHA-256 topic as hex multihash: 0x12
		// (sha2-256), 0x20 (digest length), followed by 32 digest bytes.
		if strings.HasPrefix(strings.ToLower(xt), "urn:btmh:1220") {
			hash := xt[len("urn:btmh:1220"):]
			if len(hash) == 64 {
				if decoded, err := hex.DecodeString(hash); err == nil && v2Hash == "" {
					v2Hash = hex.EncodeToString(decoded)
				}
			}
			continue
		}
		if !strings.HasPrefix(strings.ToLower(xt), "urn:btih:") {
			continue
		}
		hash := xt[len("urn:btih:"):]
		var decoded []byte
		switch len(hash) {
		case 40:
			decoded, err = hex.DecodeString(hash)
		case 32:
			decoded, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(hash))
		default:
			continue
		}
		if err == nil && len(decoded) == 20 {
			return Magnet{URI: parsed.String(), InfoHash: hex.EncodeToString(decoded)}, nil
		}
	}
	if v2Hash != "" {
		return Magnet{URI: parsed.String(), InfoHash: v2Hash}, nil
	}
	return Magnet{}, errors.New("magnet has no valid BitTorrent hash")
}
