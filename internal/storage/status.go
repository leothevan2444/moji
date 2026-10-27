package storage

import (
	"context"
	"os"
)

type DatabaseStatus struct {
	Path                        string `json:"path"`
	Engine                      string `json:"sqlite_version"`
	Schema                      int    `json:"schema_version"`
	Pages, FreePages, PageBytes int64
	FileBytes, WALBytes         int64
	Metrics                     Metrics
}

func (m *Manager) Status(ctx context.Context) (map[string]DatabaseStatus, error) {
	out := map[string]DatabaseStatus{}
	for role, d := range map[string]*Database{"core": m.Core, "runtime": m.Runtime, "cache": m.Cache} {
		result := DatabaseStatus{Path: d.Path, Metrics: d.Metrics()}
		if err := d.Reader.GetContext(ctx, &result.Engine, "SELECT sqlite_version()"); err != nil {
			return nil, err
		}
		version, err := ReadVersion(ctx, d.Reader)
		if err != nil {
			return nil, err
		}
		result.Schema = version
		for _, query := range []struct {
			SQL  string
			Dest *int64
		}{{"PRAGMA page_count", &result.Pages}, {"PRAGMA freelist_count", &result.FreePages}, {"PRAGMA page_size", &result.PageBytes}} {
			if err := d.Reader.GetContext(ctx, query.Dest, query.SQL); err != nil {
				return nil, err
			}
		}
		info, err := os.Stat(d.Path)
		if err != nil {
			return nil, err
		}
		result.FileBytes = info.Size()
		if info, err := os.Stat(d.Path + "-wal"); err == nil {
			result.WALBytes = info.Size()
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		out[role] = result
	}
	return out, nil
}
