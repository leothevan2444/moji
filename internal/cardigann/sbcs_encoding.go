package cardigann

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// singleByteEncoding contains immutable .NET round-trip and best-fit mappings.
// decode contains 256 little-endian UTF-16 values; encode contains sorted
// three-byte records (little-endian UTF-16 value, encoded byte).
type singleByteEncoding struct {
	decode string
	encode string
}

func dotNetSingleByteEncoding(name string) (*singleByteEncoding, bool) {
	page, ok := dotNetSingleByteAliases[strings.ToLower(name)]
	return dotNetSingleByteCodePages[page], ok
}

func (c *singleByteEncoding) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: singleByteDecoder{c}}
}

func (c *singleByteEncoding) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: singleByteEncoder{c}}
}

func (c *singleByteEncoding) encodedRune(r rune) byte {
	low, high := 0, len(c.encode)/3
	for low < high {
		middle := low + (high-low)/2
		position := middle * 3
		candidate := rune(c.encode[position]) | rune(c.encode[position+1])<<8
		if candidate < r {
			low = middle + 1
		} else if candidate > r {
			high = middle
		} else {
			return c.encode[position+2]
		}
	}
	return '?'
}

type singleByteEncoder struct{ table *singleByteEncoding }

func (singleByteEncoder) Reset() {}

func (e singleByteEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		if !atEOF && !utf8.FullRune(src[nSrc:]) {
			return nDst, nSrc, transform.ErrShortSrc
		}
		r, size := utf8.DecodeRune(src[nSrc:])
		width := 1
		if r > 0xffff {
			// .NET best-fit fallback replaces both UTF-16 surrogate units.
			width = 2
		}
		if len(dst)-nDst < width {
			return nDst, nSrc, transform.ErrShortDst
		}
		if width == 2 {
			dst[nDst], dst[nDst+1] = '?', '?'
		} else {
			dst[nDst] = e.table.encodedRune(r)
		}
		nDst += width
		nSrc += size
	}
	return nDst, nSrc, nil
}

type singleByteDecoder struct{ table *singleByteEncoding }

func (singleByteDecoder) Reset() {}

func (d singleByteDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		position := int(src[nSrc]) * 2
		r := rune(d.table.decode[position]) | rune(d.table.decode[position+1])<<8
		if len(dst)-nDst < utf8.RuneLen(r) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc++
	}
	return nDst, nSrc, nil
}
