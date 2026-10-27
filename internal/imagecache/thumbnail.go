package imagecache

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// One fixed, versioned derivative avoids arbitrary resize sizes filling the cache.
// This fits the list's portrait frame at 2x pixel density without cropping.
const avatarSuffix = "-avatar-v1"
const avatarWidth, avatarHeight = 192, 320
const maxThumbnailPixels = 32_000_000

func (s *Service) loadAvatar(ctx context.Context, key string) (imageResponse, error) {
	variantKey := key + avatarSuffix
	if path, contentType, ok := s.findObject(variantKey); ok {
		s.touchIfStale(path)
		return imageResponse{Path: path, ContentType: contentType}, nil
	}
	// Share the original fetch with concurrent original and thumbnail requests.
	v, err, _ := s.group.Do(key, func() (any, error) { return s.load(ctx, key) })
	if err != nil {
		return imageResponse{}, err
	}
	original := v.(imageResponse)
	// AVIF is accepted by the original proxy but has no Go decoder here.
	// Preserve its existing display behavior rather than returning a broken image.
	if original.ContentType == "image/avif" {
		return original, nil
	}
	select {
	case s.thumbnailSlots <- struct{}{}:
		defer func() { <-s.thumbnailSlots }()
	case <-ctx.Done():
		return imageResponse{}, ctx.Err()
	}
	var reader io.ReadSeeker
	if original.Data != nil {
		reader = bytes.NewReader(original.Data)
	} else {
		f, err := os.Open(original.Path)
		if err != nil {
			return imageResponse{}, err
		}
		defer f.Close()
		reader = f
	}
	data, contentType, err := makeAvatar(ctx, reader)
	if err != nil {
		return imageResponse{}, err
	}
	if !normalizeConfig(s.config()).Enabled {
		return imageResponse{Data: data, ContentType: contentType}, nil
	}
	path := filepath.Join(s.objectsDir, variantKey+extensionFor(contentType))
	if err := writeAtomic(path, data, 0o600); err != nil {
		return imageResponse{}, err
	}
	s.requestCleanup()
	return imageResponse{Path: path, ContentType: contentType}, nil
}

func makeAvatar(ctx context.Context, reader io.ReadSeeker) ([]byte, string, error) {
	cfg, _, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, "", err
	}
	// Check before Decode allocates the full source image.
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxThumbnailPixels {
		return nil, "", errors.New("imagecache: thumbnail source exceeds pixel limit")
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, "", err
	}
	src, _, err := image.Decode(reader)
	if err != nil {
		return nil, "", err
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	w, h := cfg.Width, cfg.Height
	if w > avatarWidth {
		h = max(1, h*avatarWidth/w)
		w = avatarWidth
	}
	if h > avatarHeight {
		w = max(1, w*avatarHeight/h)
		h = avatarHeight
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	var output bytes.Buffer
	if !dst.Opaque() {
		err = png.Encode(&output, dst)
		return output.Bytes(), "image/png", err
	}
	err = jpeg.Encode(&output, dst, &jpeg.Options{Quality: 82})
	return output.Bytes(), "image/jpeg", err
}
