package mediafiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeliveryRemoval is persisted before deletion, so interrupted cleanup can
// validate and remove only the remaining task-owned files.
type DeliveryRemoval struct {
	TaskID string
	Code   string
	Target string
	Files  map[string]string
}

func cleanupPath(root, path string) error {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return errors.New("invalid cleanup root")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("cleanup path escapes root")
	}
	for p := path; p != root; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return errors.New("cleanup path contains a symlink")
		}
	}
	return nil
}

func (p DeliveryRemoval) receiptPath() string {
	return filepath.Join(filepath.Dir(p.Target), ".moji-receipts", p.TaskID+".json")
}

func (p DeliveryRemoval) validate(ctx context.Context) error {
	if ExtractCode(p.Code) != p.Code || p.Code == "" || p.TaskID == "" || p.TaskID == "." || p.TaskID == ".." || filepath.Base(p.TaskID) != p.TaskID ||
		!filepath.IsAbs(p.Target) || filepath.Clean(p.Target) != p.Target || filepath.Base(p.Target) != p.Code {
		return errors.New("invalid delivery ownership")
	}
	root := filepath.Dir(p.Target)
	for _, path := range []string{p.Target, p.receiptPath()} {
		if err := cleanupPath(root, path); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(p.Target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, entry := range entries {
		if _, owned := p.Files[entry.Name()]; !owned {
			return errors.New("delivery contains unowned files")
		}
	}
	for name, want := range p.Files {
		if name == "." || name == ".." || filepath.Base(name) != name || want == "" {
			return errors.New("invalid delivery receipt filename")
		}
		path := filepath.Join(p.Target, name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("delivery output is not a regular file")
		}
		hash, err := FileHash(ctx, path)
		if err != nil {
			return err
		}
		if hash != want {
			return errors.New("delivery output has changed; review it before restarting")
		}
	}
	return nil
}

func PlanDeliveryRemoval(ctx context.Context, taskID, code, target string) (*DeliveryRemoval, error) {
	p := &DeliveryRemoval{TaskID: taskID, Code: code, Target: target}
	root := filepath.Dir(target)
	if err := cleanupPath(root, p.receiptPath()); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p.receiptPath())
	if errors.Is(err, os.ErrNotExist) {
		// Already absent or empty output needs no receipt. Any remaining file
		// will fail the ownership check against this empty removal manifest.
		p.Files = map[string]string{}
		if err := p.validate(ctx); err != nil {
			return nil, err
		}
		return p, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read task delivery receipt: %w", err)
	}
	var saved receipt
	if json.Unmarshal(data, &saved) != nil || saved.Signature == "" || len(saved.Files) == 0 {
		return nil, errors.New("invalid task delivery receipt")
	}
	p.Files = saved.Files
	if err := p.validate(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func RemoveDelivery(ctx context.Context, p DeliveryRemoval) error {
	if err := p.validate(ctx); err != nil {
		return err
	}
	for name := range p.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.Remove(filepath.Join(p.Target, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// Remove, rather than RemoveAll, refuses unexpected new files or subfolders.
	if err := os.Remove(p.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(p.receiptPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// DownloadRemoval complements qBittorrent's deleteFiles request after a crash,
// or when the old torrent has already been removed. It never removes the shared
// download root or files outside the torrent's saved file list.
type DownloadRemoval struct {
	Root  string
	Paths []string
}

func (p DownloadRemoval) Validate() error {
	for _, name := range p.Paths {
		if name == "" || name == "." || name == ".." || filepath.IsAbs(name) || strings.ContainsAny(name, "\\\r\n\x00") {
			return errors.New("invalid torrent cleanup filename")
		}
		path := filepath.Join(p.Root, name)
		if err := cleanupPath(p.Root, path); err != nil {
			return err
		}
		for _, candidate := range []string{path, path + ".!qB"} {
			if err := cleanupPath(p.Root, candidate); err != nil {
				return err
			}
			info, err := os.Lstat(candidate)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err == nil && !info.Mode().IsRegular() {
				return errors.New("download cleanup target is not a regular file")
			}
		}
	}
	return nil
}

func RemoveDownloads(ctx context.Context, p DownloadRemoval) error {
	if err := p.Validate(); err != nil {
		return err
	}
	for _, name := range p.Paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(p.Root, name)
		for _, candidate := range []string{path, path + ".!qB"} {
			if err := os.Remove(candidate); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		for dir := filepath.Dir(path); dir != p.Root; dir = filepath.Dir(dir) {
			entries, err := os.ReadDir(dir)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if len(entries) != 0 {
				break
			}
			if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}
