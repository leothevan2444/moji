package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// SupplementalFiles enumerates configuration and immutable delivery receipts.
// It does not include video content, image/cache files or upstream Stash state.
func SupplementalFiles(ctx context.Context, configPath, libraryRoot string) ([]BackupFile, error) {
	var files []BackupFile
	if configPath != "" {
		files = append(files, BackupFile{Name: "configuration/config.yaml", Path: configPath})
	}
	if libraryRoot == "" {
		return files, nil
	}
	directory := filepath.Join(libraryRoot, ".moji-receipts")
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		files = append(files, BackupFile{Name: filepath.Join("receipts", entry.Name()), Path: filepath.Join(directory, entry.Name())})
	}
	return files, nil
}
