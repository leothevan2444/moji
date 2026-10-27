package imagecache

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func portraitJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1200, 1800))
	for y := 0; y < 1800; y++ {
		for x := 0; x < 1200; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x * 7), uint8(y * 11), uint8(x + y), 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestAvatarPreservesOriginalAndCachesSmallDerivative(t *testing.T) {
	s := newTestService(t, DefaultConfig())
	original := portraitJPEG(t)
	var calls atomic.Int32
	s.client.Transport = imageTransport("image/jpeg", original, &calls, nil)
	proxy, err := s.Register(context.Background(), Descriptor{Kind: SourceStash, InstanceURL: "http://stash.test", ImageURL: "/portrait.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	get := func(url string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", url, rec.Code, rec.Body.String())
		}
		return rec
	}
	avatar := get(proxy + "?variant=avatar")
	cfg, _, err := image.DecodeConfig(bytes.NewReader(avatar.Body.Bytes()))
	if err != nil || cfg.Width != 192 || cfg.Height != 288 {
		t.Fatalf("wrong thumbnail dimensions: %+v %v", cfg, err)
	}
	if avatar.Body.Len() >= len(original)/10 {
		t.Fatalf("thumbnail insufficiently reduced: %d / %d bytes", avatar.Body.Len(), len(original))
	}
	full := get(proxy)
	if !bytes.Equal(full.Body.Bytes(), original) || full.Header().Get("ETag") == avatar.Header().Get("ETag") {
		t.Fatal("thumbnail changed the original or shares its ETag")
	}
	conditional := httptest.NewRequest(http.MethodGet, proxy+"?variant=avatar", nil)
	conditional.Header.Set("If-None-Match", avatar.Header().Get("ETag"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, conditional)
	if rec.Code != 304 || rec.Body.Len() != 0 {
		t.Fatalf("conditional thumbnail: %d", rec.Code)
	}
	// Restart and evict the original: the derivative must still be served from disk.
	key := strings.TrimPrefix(proxy, "/api/images/")
	if err := os.Remove(filepath.Join(s.objectsDir, key+".jpg")); err != nil {
		t.Fatal(err)
	}
	restarted, err := New(s.dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	restarted.client.Transport = imageTransport("image/jpeg", original, &calls, nil)
	mux = http.NewServeMux()
	restarted.RegisterHandler(mux)
	if !bytes.Equal(get(proxy+"?variant=avatar").Body.Bytes(), avatar.Body.Bytes()) || calls.Load() != 1 {
		t.Fatal("cached thumbnail was regenerated or original was refetched")
	}
	if !restarted.hasObject(key) {
		t.Fatal("registration must remain while a derivative is cached")
	}
	if _, err := restarted.Clear(context.Background()); err != nil {
		t.Fatal(err)
	}
	st, _ := restarted.Status(context.Background())
	if st.EntryCount != 0 {
		t.Fatal("clear left thumbnail files behind")
	}
}

func TestAvatarWithoutDiskCache(t *testing.T) {
	s := newTestService(t, Config{Enabled: false})
	var calls atomic.Int32
	s.client.Transport = imageTransport("image/jpeg", portraitJPEG(t), &calls, nil)
	proxy, _ := s.Register(context.Background(), Descriptor{Kind: SourceStash, InstanceURL: "http://stash.test", ImageURL: "/portrait.jpg"})
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, proxy+"?variant=avatar", nil))
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("uncached thumbnail: %d %v", rec.Code, rec.Header())
	}
	st, _ := s.Status(context.Background())
	if st.EntryCount != 0 {
		t.Fatal("disabled cache persisted original or derivative")
	}
}

func TestAvatarConcurrentRequestsShareGeneration(t *testing.T) {
	s := newTestService(t, DefaultConfig())
	var calls atomic.Int32
	s.client.Transport = imageTransport("image/jpeg", portraitJPEG(t), &calls, nil)
	proxy, _ := s.Register(context.Background(), Descriptor{Kind: SourceStash, InstanceURL: "http://stash.test", ImageURL: "/portrait.jpg"})
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, proxy+"?variant=avatar", nil))
			if rec.Code != 200 {
				t.Errorf("concurrent thumbnail: %d", rec.Code)
			}
		})
	}
	wg.Wait()
	st, _ := s.Status(context.Background())
	if calls.Load() != 1 || st.EntryCount != 2 {
		t.Fatalf("duplicate original fetch or thumbnail: calls=%d entries=%d", calls.Load(), st.EntryCount)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, proxy+"?variant=999999", nil))
	if rec.Code != 400 || calls.Load() != 1 {
		t.Fatal("unbounded variants should be rejected before fetching")
	}
}

func TestAvatarGeometryTransparencyAndPixelLimit(t *testing.T) {
	for _, size := range []image.Point{{20, 30}, {800, 100}, {100, 1000}} {
		t.Run(size.String(), func(t *testing.T) {
			var buf bytes.Buffer
			if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))); err != nil {
				t.Fatal(err)
			}
			data, contentType, err := makeAvatar(context.Background(), bytes.NewReader(buf.Bytes()))
			if err != nil || contentType != "image/png" {
				t.Fatalf("transparency lost: %s %v", contentType, err)
			}
			cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil || cfg.Width > avatarWidth || cfg.Height > avatarHeight || cfg.Width > size.X || cfg.Height > size.Y {
				t.Fatalf("invalid dimensions: %+v %v", cfg, err)
			}
		})
	}
	var pngHeader bytes.Buffer
	_ = png.Encode(&pngHeader, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	data := pngHeader.Bytes()
	binary.BigEndian.PutUint32(data[16:20], 8000)
	binary.BigEndian.PutUint32(data[20:24], 8000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	if _, _, err := makeAvatar(context.Background(), bytes.NewReader(data)); err == nil || !strings.Contains(err.Error(), "pixel limit") {
		t.Fatalf("oversized source must be rejected before decode: %v", err)
	}
	if _, _, err := makeAvatar(context.Background(), bytes.NewReader(tinyPNG)); err == nil {
		t.Fatal("corrupt source must not be cached as an image")
	}
}
