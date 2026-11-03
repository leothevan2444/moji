package indexerdefs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
)

const MaxArchiveBytes int64 = 32 << 20
const MaxUnpackedBytes int64 = 128 << 20
const MaxArchiveEntries = 10000
const MaxBundleFiles = 4096
const MaxManifestBytes = 1 << 20

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func manifestData(manifest Manifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func validFileName(name string) bool {
	return name == "schema.json" || strings.HasSuffix(name, ".yml") && definitionPattern.MatchString(strings.TrimSuffix(name, ".yml"))
}

// StageArchive processes an untrusted upstream archive into a private staging
// directory. It never publishes an active revision or changes the core DB.
// The caller owns removal of a successful stage after Publish or cancellation.
func StageArchive(ctx context.Context, root, revision string, archive io.Reader, now time.Time) (bundle *Bundle, err error) {
	return stageArchive(ctx, root, revision, archive, now, MaxUnpackedBytes)
}

func stageArchive(ctx context.Context, root, revision string, archive io.Reader, now time.Time, storageBudget int64) (bundle *Bundle, err error) {
	if !ValidRevision(revision) {
		return nil, ErrInvalidBundle
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	if err := requireDirectory(root); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(root, ".staging-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(directory)
		}
	}()
	compressed := &io.LimitedReader{R: archive, N: MaxArchiveBytes + 1}
	gz, err := gzip.NewReader(compressed)
	if err != nil {
		return nil, fmt.Errorf("%w: gzip header", ErrInvalidBundle)
	}
	defer gz.Close()
	unpacked := &io.LimitedReader{R: gz, N: MaxUnpackedBytes + 1}
	reader := tar.NewReader(unpacked)
	bundle = &Bundle{Directory: directory, Plans: map[string]*cardigann.Plan{}, Manifest: Manifest{Version: ManifestVersion, Schema: SchemaVersion, Repository: "Prowlarr/Indexers", Revision: revision, Engine: EngineVersion, InstalledAt: now.UTC()}}
	seen := map[string]bool{}
	archiveRoot := ""
	var written int64
	for count := 0; ; count++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		header, readErr := reader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("%w: tar data", ErrInvalidBundle)
		}
		if count >= MaxArchiveEntries || unpacked.N <= 0 || compressed.N <= 0 {
			return nil, ErrBudget
		}
		if header.Size < 0 || header.Size > MaxUnpackedBytes {
			return nil, ErrBudget
		}
		// Go consumes per-file PAX records internally, but returns global PAX
		// metadata as a header. It describes the archive, never an extracted file.
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("%w: archive entry type", ErrInvalidBundle)
		}
		name := strings.TrimSuffix(header.Name, "/")
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || path.Clean(name) != name {
			return nil, fmt.Errorf("%w: archive path", ErrInvalidBundle)
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." || part == "." || part == "" {
				return nil, fmt.Errorf("%w: archive traversal", ErrInvalidBundle)
			}
		}
		parts := strings.Split(name, "/")
		if len(parts) != 4 || parts[1] != "definitions" || parts[2] != "v11" {
			continue
		}
		if archiveRoot == "" {
			archiveRoot = parts[0]
		} else if archiveRoot != parts[0] {
			return nil, fmt.Errorf("%w: mixed archive roots", ErrInvalidBundle)
		}
		fileName := parts[3]
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if !validFileName(fileName) {
			return nil, fmt.Errorf("%w: unexpected v11 file", ErrInvalidBundle)
		}
		if seen[fileName] {
			return nil, fmt.Errorf("%w: duplicate definition", ErrInvalidBundle)
		}
		seen[fileName] = true
		if len(seen) > MaxBundleFiles || header.Size > cardigann.MaxDefinitionBytes {
			return nil, ErrBudget
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, cardigann.MaxDefinitionBytes+1))
		if readErr != nil || int64(len(data)) != header.Size {
			return nil, fmt.Errorf("%w: truncated definition", ErrInvalidBundle)
		}
		if fileName != "schema.json" {
			plan, loadErr := cardigann.Load(data)
			if loadErr != nil {
				return nil, fmt.Errorf("%w: %s: %v", ErrIncompatible, fileName, loadErr)
			}
			if !definitionPattern.MatchString(plan.Definition.ID) || bundle.Plans[plan.Definition.ID] != nil {
				return nil, fmt.Errorf("%w: invalid or duplicate definition identity", ErrInvalidBundle)
			}
			bundle.Plans[plan.Definition.ID] = plan
		} else if !json.Valid(data) {
			return nil, fmt.Errorf("%w: invalid schema JSON", ErrInvalidBundle)
		}
		if int64(len(data)) > storageBudget-written {
			return nil, ErrBudget
		}
		if err := writeNewFile(filepath.Join(directory, fileName), data); err != nil {
			return nil, err
		}
		written += int64(len(data))
		bundle.Manifest.Files = append(bundle.Manifest.Files, File{Name: fileName, SHA256: digest(data), Bytes: int64(len(data))})
	}
	// Drain gzip to verify its trailer/checksum and bounded trailing members.
	if _, err := io.Copy(io.Discard, unpacked); err != nil {
		return nil, fmt.Errorf("%w: gzip trailer", ErrInvalidBundle)
	}
	if unpacked.N <= 0 || compressed.N <= 0 {
		return nil, ErrBudget
	}
	if len(bundle.Plans) == 0 || !seen["schema.json"] {
		return nil, fmt.Errorf("%w: empty or incomplete v11 library", ErrInvalidBundle)
	}
	sort.Slice(bundle.Manifest.Files, func(i, j int) bool { return bundle.Manifest.Files[i].Name < bundle.Manifest.Files[j].Name })
	bundle.Manifest.DefinitionCount = len(bundle.Plans)
	data, err := manifestData(bundle.Manifest)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > storageBudget-written {
		return nil, ErrBudget
	}
	if err := writeNewFile(filepath.Join(directory, "manifest.json"), data); err != nil {
		return nil, err
	}
	bundle.ManifestSHA256 = digest(data)
	if err := syncDirectory(directory); err != nil {
		return nil, err
	}
	return bundle, nil
}

func writeNewFile(name string, data []byte) (err error) {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if _, err = f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

func syncDirectory(name string) error {
	directory, err := os.Open(name)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

func requireDirectory(name string) error {
	info, err := os.Lstat(name)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: unsafe bundle directory", ErrInvalidBundle)
	}
	return nil
}

// Publish makes complete immutable files durable before a repository transaction
// can reference them. A crash here leaves an unreferenced directory, not a
// half-activated bundle. Existing revisions are never overwritten.
func Publish(ctx context.Context, root string, stage *Bundle) (*Bundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !ValidRevision(stage.Manifest.Revision) || filepath.Dir(stage.Directory) != filepath.Clean(root) || !strings.HasPrefix(filepath.Base(stage.Directory), ".staging-") {
		return nil, ErrInvalidBundle
	}
	target := filepath.Join(root, stage.Manifest.Revision)
	if _, err := os.Lstat(target); err == nil {
		existing, err := LoadBundle(ctx, target, "")
		if err != nil {
			return nil, err
		}
		if !sameFiles(existing.Manifest, stage.Manifest) {
			return nil, fmt.Errorf("%w: immutable revision contents changed", ErrInvalidBundle)
		}
		if err := os.RemoveAll(stage.Directory); err != nil {
			return nil, err
		}
		return existing, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.Rename(stage.Directory, target); err != nil {
		return nil, err
	}
	if err := syncDirectory(root); err != nil {
		return nil, err
	}
	stage.Directory = target
	return stage, nil
}

func sameFiles(a, b Manifest) bool {
	if a.Revision != b.Revision || a.Schema != b.Schema || len(a.Files) != len(b.Files) {
		return false
	}
	for i, file := range a.Files {
		if file != b.Files[i] {
			return false
		}
	}
	return true
}

func LoadBundle(ctx context.Context, directory, expectedManifest string) (*Bundle, error) {
	if err := requireDirectory(directory); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	read := func(name string, limit int64) ([]byte, error) {
		info, err := root.Lstat(name)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() > limit {
			return nil, ErrInvalidBundle
		}
		f, err := root.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, limit+1))
		if err != nil || int64(len(data)) > limit {
			return nil, ErrInvalidBundle
		}
		return data, nil
	}
	data, err := read("manifest.json", MaxManifestBytes)
	if err != nil {
		return nil, err
	}
	actualDigest := digest(data)
	if expectedManifest != "" && actualDigest != expectedManifest {
		return nil, fmt.Errorf("%w: manifest digest", ErrInvalidBundle)
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, ErrInvalidBundle
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, ErrInvalidBundle
	}
	if manifest.Version != ManifestVersion || manifest.Schema != SchemaVersion || manifest.Repository != "Prowlarr/Indexers" || !ValidRevision(manifest.Revision) || filepath.Base(directory) != manifest.Revision || len(manifest.Files) > MaxBundleFiles || manifest.InstalledAt.IsZero() || manifest.Engine == "" || len(manifest.Engine) > 128 {
		return nil, ErrInvalidBundle
	}
	bundle := &Bundle{Manifest: manifest, ManifestSHA256: actualDigest, Directory: directory, Plans: map[string]*cardigann.Plan{}}
	seen := map[string]bool{}
	var total int64
	previousName := ""
	for _, file := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !validFileName(file.Name) || seen[file.Name] || file.Name <= previousName || file.Bytes < 0 || file.Bytes > cardigann.MaxDefinitionBytes {
			return nil, ErrInvalidBundle
		}
		previousName = file.Name
		seen[file.Name] = true
		total += file.Bytes
		if total > MaxUnpackedBytes {
			return nil, ErrBudget
		}
		data, err := read(file.Name, cardigann.MaxDefinitionBytes)
		if err != nil {
			return nil, err
		}
		if int64(len(data)) != file.Bytes || digest(data) != file.SHA256 {
			return nil, fmt.Errorf("%w: file digest", ErrInvalidBundle)
		}
		if file.Name == "schema.json" && !json.Valid(data) {
			return nil, ErrInvalidBundle
		}
		if file.Name != "schema.json" {
			plan, err := cardigann.Load(data)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrIncompatible, file.Name)
			}
			if !definitionPattern.MatchString(plan.Definition.ID) || bundle.Plans[plan.Definition.ID] != nil {
				return nil, ErrInvalidBundle
			}
			bundle.Plans[plan.Definition.ID] = plan
		}
	}
	if len(bundle.Plans) == 0 || len(bundle.Plans) != manifest.DefinitionCount || !seen["schema.json"] {
		return nil, ErrInvalidBundle
	}
	return bundle, nil
}
