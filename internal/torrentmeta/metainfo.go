// Package torrentmeta parses bounded torrent metainfo and magnet URIs without application state.
package torrentmeta

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"strings"
)

const MaxBytes = 8 << 20

// Metadata retains the original metainfo bytes and the exact info dictionary hash.
type Metadata struct {
	Data                                []byte
	Name, FilePath, InfoHash            string
	Paths, IdentityNames, IdentityPaths []string
}

func Parse(data []byte) (Metadata, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return Metadata{}, errors.New("torrentmeta: invalid metainfo size")
	}
	parser := &bencodeParser{data: data}
	root, infoBytes, err := parser.parseMetainfo()
	if err != nil {
		return Metadata{}, fmt.Errorf("taskruntime: parse .torrent metadata: %w", err)
	}
	metadata := Metadata{
		Data:          data,
		Name:          root.name,
		FilePath:      root.firstPath,
		Paths:         append([]string(nil), root.paths...),
		IdentityNames: root.names,
		IdentityPaths: parser.identityPaths,
	}
	if len(metadata.Paths) == 0 && metadata.Name != "" {
		metadata.Paths = []string{metadata.Name}
	}
	if metadata.FilePath == "" && len(metadata.Paths) > 0 {
		metadata.FilePath = metadata.Paths[0]
	}
	if len(infoBytes) > 0 {
		hash := sha1.Sum(infoBytes)
		metadata.InfoHash = strings.ToUpper(fmt.Sprintf("%x", hash[:]))
	}
	if len(infoBytes) == 0 || metadata.Name == "" || parser.pos != len(data) {
		return Metadata{}, errors.New("taskruntime: incomplete or trailing torrent metadata")
	}
	return metadata, nil
}

type torrentMetainfo struct {
	name      string
	names     []string
	firstPath string
	paths     []string
}

type bencodeParser struct {
	data          []byte
	pos           int
	depth         int
	identityPaths []string
}

func (p *bencodeParser) parseMetainfo() (torrentMetainfo, []byte, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'd' {
		return torrentMetainfo{}, nil, errors.New("root value is not a dictionary")
	}
	p.pos++
	var meta torrentMetainfo
	var infoBytes []byte
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		key, err := p.parseString()
		if err != nil {
			return torrentMetainfo{}, nil, err
		}
		valueStart := p.pos
		switch key {
		case "info":
			if infoBytes != nil {
				return torrentMetainfo{}, nil, errors.New("duplicate info dictionary")
			}
			infoMeta, err := p.parseInfoDict()
			if err != nil {
				return torrentMetainfo{}, nil, err
			}
			infoBytes = append([]byte(nil), p.data[valueStart:p.pos]...)
			meta = infoMeta
		default:
			if err := p.skipValue(); err != nil {
				return torrentMetainfo{}, nil, err
			}
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return torrentMetainfo{}, nil, errors.New("unterminated root dictionary")
	}
	p.pos++
	return meta, infoBytes, nil
}

func (p *bencodeParser) parseInfoDict() (torrentMetainfo, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'd' {
		return torrentMetainfo{}, errors.New("info is not a dictionary")
	}
	p.pos++
	var meta torrentMetainfo
	seen := make(map[string]bool)
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		key, err := p.parseString()
		if err != nil {
			return torrentMetainfo{}, err
		}
		if seen[key] {
			return torrentMetainfo{}, errors.New("duplicate info key")
		}
		seen[key] = true
		switch key {
		case "name", "name.utf-8":
			value, err := p.parseString()
			if err != nil {
				return torrentMetainfo{}, err
			}
			meta.names = append(meta.names, value)
			if key == "name.utf-8" || !seen["name.utf-8"] {
				meta.name = value
			}
		case "files":
			paths, err := p.parseFiles()
			if err != nil {
				return torrentMetainfo{}, err
			}
			meta.paths = paths
			if len(paths) > 0 {
				meta.firstPath = paths[0]
			}
		default:
			if err := p.skipValue(); err != nil {
				return torrentMetainfo{}, err
			}
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return torrentMetainfo{}, errors.New("unterminated info dictionary")
	}
	p.pos++
	return meta, nil
}

func (p *bencodeParser) parseFiles() ([]string, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'l' {
		return nil, errors.New("files is not a list")
	}
	p.pos++
	paths := make([]string, 0)
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		pathValue, err := p.parseFileEntry()
		if err != nil {
			return nil, err
		}
		if pathValue != "" {
			paths = append(paths, pathValue)
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return nil, errors.New("unterminated files list")
	}
	p.pos++
	return paths, nil
}

func (p *bencodeParser) parseFileEntry() (string, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'd' {
		return "", errors.New("file entry is not a dictionary")
	}
	p.pos++
	firstPath := ""
	seen := make(map[string]bool)
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		key, err := p.parseString()
		if err != nil {
			return "", err
		}
		if seen[key] {
			return "", errors.New("duplicate file key")
		}
		seen[key] = true
		if key == "path" || key == "path.utf-8" {
			pathSegments, err := p.parseStringList()
			if err != nil {
				return "", err
			}
			value := strings.Join(pathSegments, "/")
			p.identityPaths = append(p.identityPaths, value)
			if key == "path.utf-8" || !seen["path.utf-8"] {
				firstPath = value
			}
			continue
		}
		if err := p.skipValue(); err != nil {
			return "", err
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return "", errors.New("unterminated file entry")
	}
	p.pos++
	return firstPath, nil
}

func (p *bencodeParser) parseStringList() ([]string, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'l' {
		return nil, errors.New("path is not a list")
	}
	p.pos++
	var items []string
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		item, err := p.parseString()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return nil, errors.New("unterminated string list")
	}
	p.pos++
	return items, nil
}

func (p *bencodeParser) parseString() (string, error) {
	lengthStart := p.pos
	for p.pos < len(p.data) && p.data[p.pos] != ':' {
		if p.data[p.pos] < '0' || p.data[p.pos] > '9' {
			return "", errors.New("invalid string length")
		}
		p.pos++
	}
	if p.pos >= len(p.data) || p.data[p.pos] != ':' {
		return "", errors.New("unterminated string length")
	}
	lengthValue := 0
	if lengthStart == p.pos {
		return "", errors.New("empty string length")
	}
	for _, digit := range p.data[lengthStart:p.pos] {
		// Bound lengths before arithmetic to avoid overflow and slice panics.
		if lengthValue > len(p.data)/10 || lengthValue*10+int(digit-'0') > len(p.data) {
			return "", errors.New("string exceeds buffer")
		}
		lengthValue = lengthValue*10 + int(digit-'0')
	}
	p.pos++
	if p.pos+lengthValue > len(p.data) {
		return "", errors.New("string exceeds buffer")
	}
	value := string(p.data[p.pos : p.pos+lengthValue])
	p.pos += lengthValue
	return value, nil
}

func (p *bencodeParser) skipValue() error {
	if p.depth >= 64 {
		return errors.New("torrent metadata nesting exceeds limit")
	}
	p.depth++
	defer func() { p.depth-- }()
	if p.pos >= len(p.data) {
		return io.ErrUnexpectedEOF
	}
	switch p.data[p.pos] {
	case 'd':
		p.pos++
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			if _, err := p.parseString(); err != nil {
				return err
			}
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		if p.pos >= len(p.data) {
			return errors.New("unterminated dictionary")
		}
		p.pos++
		return nil
	case 'l':
		p.pos++
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		if p.pos >= len(p.data) {
			return errors.New("unterminated list")
		}
		p.pos++
		return nil
	case 'i':
		p.pos++
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			p.pos++
		}
		if p.pos >= len(p.data) {
			return errors.New("unterminated integer")
		}
		p.pos++
		return nil
	default:
		_, err := p.parseString()
		return err
	}
}
