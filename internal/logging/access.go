package logging

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const accessLogPrefix = "access-"

// AccessWriter stores the current day's access log as plain text. On startup
// and after a date change it compresses older logs and removes expired files.
type AccessWriter struct {
	mu            sync.Mutex
	dir           string
	retentionDays int
	now           func() time.Time
	currentDate   string
	file          *os.File
}

func NewAccessWriter(dir string, retentionDays int) (*AccessWriter, error) {
	return newAccessWriter(dir, retentionDays, time.Now)
}

func newAccessWriter(dir string, retentionDays int, now func() time.Time) (*AccessWriter, error) {
	if strings.TrimSpace(dir) == "" || retentionDays < 1 || now == nil {
		return nil, errors.New("invalid access log configuration")
	}
	writer := &AccessWriter{dir: dir, retentionDays: retentionDays, now: now}
	if err := writer.rotateLocked(now()); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *AccessWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotateLocked(w.now()); err != nil {
		return 0, err
	}
	return w.file.Write(data)
}

func (w *AccessWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *AccessWriter) rotateLocked(now time.Time) error {
	today := now.Format(time.DateOnly)
	if w.file != nil && w.currentDate == today {
		return nil
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	if err := os.MkdirAll(w.dir, 0o750); err != nil {
		return err
	}
	if err := w.maintain(now, today); err != nil {
		return err
	}
	path := filepath.Join(w.dir, accessLogPrefix+today+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	w.file = file
	w.currentDate = today
	return nil
}

func (w *AccessWriter) maintain(now time.Time, today string) error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") || entry.Name() == accessLogPrefix+today+".log" {
			continue
		}
		if _, ok := parseAccessLogDate(entry.Name(), ".log", now.Location()); !ok {
			continue
		}
		if err := compressFile(filepath.Join(w.dir, entry.Name())); err != nil {
			return err
		}
	}

	cutoff := beginningOfDay(now).AddDate(0, 0, -(w.retentionDays - 1))
	entries, err = os.ReadDir(w.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		date, ok := parseAccessLogDate(entry.Name(), ".log.gz", now.Location())
		if !ok || !date.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(w.dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func compressFile(sourcePath string) error {
	targetPath := sourcePath + ".gz"
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("compressed access log already exists: %s", targetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	sourceOpen := true
	defer func() {
		if sourceOpen {
			_ = source.Close()
		}
	}()

	temporary, err := os.CreateTemp(filepath.Dir(sourcePath), filepath.Base(targetPath)+"-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	zipWriter := gzip.NewWriter(temporary)
	if _, err := io.Copy(zipWriter, source); err != nil {
		_ = zipWriter.Close()
		_ = temporary.Close()
		return err
	}
	if err := zipWriter.Close(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := source.Close(); err != nil {
		return err
	}
	sourceOpen = false
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		return err
	}
	removeTemporary = false
	return os.Remove(sourcePath)
}

func parseAccessLogDate(name, suffix string, location *time.Location) (time.Time, bool) {
	if !strings.HasPrefix(name, accessLogPrefix) || !strings.HasSuffix(name, suffix) {
		return time.Time{}, false
	}
	dateText := strings.TrimSuffix(strings.TrimPrefix(name, accessLogPrefix), suffix)
	date, err := time.ParseInLocation(time.DateOnly, dateText, location)
	return date, err == nil
}

func beginningOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
