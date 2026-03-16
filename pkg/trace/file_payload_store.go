package trace

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

const defaultPayloadCleanupInterval = time.Minute

var errPayloadNotFound = errors.New("trace payload not found")

type PayloadStore interface {
	SaveMessage(executionID, logID, source string, raw []byte) error
	LoadMessage(executionID, logID, source string) ([]byte, error)
}

type ImageSnapshotter interface {
	SnapshotImage(executionID, logID, source, originalPath string) (string, error)
}

type FilePayloadStore struct {
	baseDir string
	ttl     time.Duration
}

func NewFilePayloadStore(baseDir string, ttl time.Duration) *FilePayloadStore {
	resolvedBaseDir := baseDir
	if resolvedBaseDir == "" {
		resolvedBaseDir = filepath.Join(os.TempDir(), "morpheus-trace-payloads")
	}

	store := &FilePayloadStore{
		baseDir: resolvedBaseDir,
		ttl:     ttl,
	}
	go store.cleanupLoop(defaultPayloadCleanupInterval)
	return store
}

func (s *FilePayloadStore) SaveMessage(executionID, logID, source string, raw []byte) error {
	if executionID == "" || logID == "" || source == "" {
		return errors.New("missing trace payload path parts")
	}

	filePath := s.payloadFilePath(executionID, logID, source)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, filePath)
}

func (s *FilePayloadStore) LoadMessage(executionID, logID, source string) ([]byte, error) {
	filePath := s.payloadFilePath(executionID, logID, source)
	raw, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errPayloadNotFound
	}
	return raw, err
}

func (s *FilePayloadStore) SnapshotImage(executionID, logID, source, originalPath string) (string, error) {
	if executionID == "" || logID == "" || source == "" || originalPath == "" {
		return "", errors.New("missing trace image snapshot path parts")
	}

	sourceFile, err := os.Open(originalPath)
	if err != nil {
		return "", err
	}
	_ = sourceFile.Close()

	targetPath := s.imageAssetPath(executionID, logID, source, originalPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}

	_ = os.Remove(targetPath)
	if err := os.Link(originalPath, targetPath); err == nil {
		return targetPath, nil
	}

	src, err := os.Open(originalPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	tmpPath := targetPath + ".tmp"
	dst, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", closeErr
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return targetPath, nil
}

func (s *FilePayloadStore) payloadFilePath(executionID, logID, source string) string {
	return filepath.Join(s.baseDir, executionID, logID, source+".json")
}

func (s *FilePayloadStore) imageAssetPath(executionID, logID, source, originalPath string) string {
	ext := filepath.Ext(originalPath)
	if ext == "" {
		ext = ".img"
	}
	return filepath.Join(s.baseDir, executionID, logID, source+ext)
}

func (s *FilePayloadStore) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		_ = filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if time.Since(info.ModTime()) <= s.ttl {
				return nil
			}
			_ = os.Remove(path)
			return nil
		})

		_ = filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || !info.IsDir() || path == s.baseDir {
				return nil
			}
			entries, readErr := os.ReadDir(path)
			if readErr == nil && len(entries) == 0 {
				_ = os.Remove(path)
			}
			return nil
		})
	}
}
