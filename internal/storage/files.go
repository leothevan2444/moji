package storage

import (
	"errors"
	"github.com/leothevan2444/moji/internal/durablefile"
	"net/url"
	"os"
	"path/filepath"
)

func readOnlyURI(path string) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	q := url.Values{}
	q.Set("mode", "ro")
	q.Add("_pragma", "query_only(1)")
	u.RawQuery = q.Encode()
	return u.String()
}
func syncFile(path string) error {
	f, e := os.OpenFile(path, os.O_RDWR, 0)
	if e != nil {
		return e
	}
	return errors.Join(f.Sync(), f.Close())
}
func syncDirectory(path string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	return errors.Join(f.Sync(), f.Close())
}
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	return durablefile.Write(path, data, mode)
}
