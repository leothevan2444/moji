package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var ErrDirectoryInUse = errors.New("storage: data directory is already in use")

type DirectoryLock struct {
	file      *os.File
	mu        sync.Mutex
	Directory string
	Created   bool
}

// AcquireDirectory uses an OS advisory lock, not the existence of a PID file.
// A crashed process releases its lock automatically; a stale filename is safe.
func AcquireDirectory(directory string) (*DirectoryLock, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(absolute, 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(absolute, ".moji-storage.lock"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	created := err == nil
	if errors.Is(err, os.ErrExist) {
		info, statErr := os.Lstat(filepath.Join(absolute, ".moji-storage.lock"))
		if statErr != nil {
			return nil, statErr
		}
		if !info.Mode().IsRegular() {
			return nil, layoutError("invalid lock file", filepath.Join(absolute, ".moji-storage.lock"), "remove the unsafe lock path manually after confirming no instance is running")
		}
		file, err = os.OpenFile(filepath.Join(absolute, ".moji-storage.lock"), os.O_RDWR, 0600)
	}
	if err != nil {
		return nil, err
	}
	if err = lockFile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("%w: %s: %v", ErrDirectoryInUse, absolute, err)
	}
	if err = file.Truncate(0); err == nil {
		_, err = file.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	}
	if err == nil {
		err = file.Sync()
	}
	if err != nil {
		unlockFile(file)
		file.Close()
		return nil, err
	}
	return &DirectoryLock{file: file, Directory: absolute, Created: created}, nil
}
func (l *DirectoryLock) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := errors.Join(unlockFile(l.file), l.file.Close())
	l.file = nil
	return err
}
