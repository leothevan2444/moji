// moji-storage performs offline preparation and recovery without initializing
// any qB, Stash, tracker or HTTP client.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/datamigration"
	"github.com/leothevan2444/moji/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: moji-storage prepare|backup|verify-backup|restore|status|activate|compact [flags]")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	flags := flag.NewFlagSet(os.Args[1], flag.ContinueOnError)
	source := flags.String("source", "", "standalone consistent legacy core backup")
	cache := flags.String("cache", "", "standalone consistent legacy cache backup")
	destination := flags.String("destination", "", "new staging directory (must not exist)")
	manifest := flags.String("manifest", "", "backup manifest path")
	protectBackup := flags.Bool("protect", false, "retain this backup until explicitly removed")
	backupReason := flags.String("reason", "manual", "reason recorded in the backup manifest")
	backupDirectory := flags.String("backup-dir", "", "core backup target directory (default: data-dir/backups)")
	generation := flags.String("generation", "", "prepared generation name beneath data-dir/.storage/generations")
	configPath := flags.String("config", "", "config YAML to include with core backups (defaults to data-dir/config.yaml when present)")
	directory := flags.String("data-dir", ".", "Moji data directory")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	supplemental := func(work context.Context) ([]storage.BackupFile, error) {
		path := *configPath
		if path == "" {
			path = filepath.Join(*directory, "config.yaml")
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				return nil, nil
			} else if err != nil {
				return nil, err
			}
		}
		store, err := config.OpenStore(path)
		if err != nil {
			return nil, err
		}
		return storage.SupplementalFiles(work, path, store.Config().Ingest.Library.MojiRoot)
	}
	var result any
	switch os.Args[1] {
	case "prepare":
		if *source == "" || *destination == "" {
			return fmt.Errorf("prepare requires -source and -destination")
		}
		prepared, err := datamigration.Prepare(ctx, *source, *cache, *destination)
		if err != nil {
			return err
		}
		result = prepared
	case "verify-backup":
		if *manifest == "" {
			return fmt.Errorf("verify-backup requires -manifest")
		}
		verified, err := storage.VerifyBackup(ctx, *manifest)
		if err != nil {
			return err
		}
		result = verified
	case "restore":
		if *manifest == "" {
			return fmt.Errorf("restore requires -manifest")
		}
		files, err := supplemental(ctx)
		if err != nil {
			return err
		}
		restored, err := storage.RestoreCore(ctx, *directory, *manifest, files...)
		if err != nil {
			return err
		}
		result = restored
	case "activate":
		if *generation == "" {
			return fmt.Errorf("activate requires -generation")
		}
		files, err := supplemental(ctx)
		if err != nil {
			return err
		}
		preserved, err := storage.ActivateGeneration(ctx, *directory, *generation, files...)
		if err != nil {
			return err
		}
		result = struct {
			Layout            storage.ActiveLayout
			PreviousRawBackup string
		}{storage.ActiveLayout{Format: 1, Generation: *generation}, preserved}
	case "backup", "status", "compact":
		active, err := storage.ResolveActiveDirectory(*directory)
		if err != nil {
			return err
		}
		info, err := os.Stat(filepath.Join(active, "moji.db"))
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("backup requires an existing nonempty core")
		}
		options := storage.DefaultManagerOptions(*directory)
		options.Supplemental = supplemental
		options.BackupDirectory = *backupDirectory
		manager, err := storage.OpenManager(ctx, options)
		if err != nil {
			return err
		}
		defer manager.Close()
		if os.Args[1] == "status" {
			status, err := manager.Status(ctx)
			if err != nil {
				return err
			}
			result = status
			break
		}
		reason, protected := *backupReason, *protectBackup
		if os.Args[1] == "compact" {
			reason, protected = "before-compaction", true
		}
		snapshot, err := manager.Backup(ctx, reason, protected)
		if err != nil {
			return err
		}
		if os.Args[1] == "compact" {
			compacted, err := manager.Core.Compact(ctx)
			if err != nil {
				return err
			}
			result = struct {
				Backup     storage.BackupManifest
				Compaction storage.CompactionResult
			}{snapshot, compacted}
		} else {
			result = snapshot
		}
	default:
		return fmt.Errorf("unknown storage command %q", os.Args[1])
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
