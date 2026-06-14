package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type rollingFileWriter struct {
	path       string
	maxSize    int64
	maxBackups int

	mu   sync.Mutex
	file *os.File
	size int64
}

func newRollingFileWriter(path string, maxSize int64, maxBackups int) (*rollingFileWriter, error) {
	writer := &rollingFileWriter{
		path:       path,
		maxSize:    maxSize,
		maxBackups: maxBackups,
	}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rollingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}

	if w.maxSize > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rollingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

func (w *rollingFileWriter) open() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rollingFileWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log file: %w", err)
		}
		w.file = nil
	}

	if w.maxBackups > 0 {
		for i := w.maxBackups - 1; i >= 1; i-- {
			src := fmt.Sprintf("%s.%d", w.path, i)
			dst := fmt.Sprintf("%s.%d", w.path, i+1)
			if _, err := os.Stat(src); err == nil {
				if err := os.Rename(src, dst); err != nil {
					return fmt.Errorf("rotate log backup %s to %s: %w", src, dst, err)
				}
			}
		}
		if _, err := os.Stat(w.path); err == nil {
			if err := os.Rename(w.path, fmt.Sprintf("%s.1", w.path)); err != nil {
				return fmt.Errorf("rotate active log file: %w", err)
			}
		}
	} else {
		if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove oversized log file: %w", err)
		}
	}

	w.size = 0
	return w.open()
}
