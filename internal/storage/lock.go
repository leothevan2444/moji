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
	file, err := os.OpenFile(filepath.Join(absolute, ".moji-storage.lock"), os.O_CREATE|os.O_RDWR, 0600)
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
	return &DirectoryLock{file: file, Directory: absolute}, nil
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
