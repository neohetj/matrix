package fs

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// httpFS implements the http.FileSystem interface by wrapping a ResourceProvider.
// This allows serving files from our hybrid loader (embed or external) over HTTP.
type httpFS struct {
	provider types.ResourceProvider
	basePath string
}

// NewHttpFS creates a new http.FileSystem that serves files from the given
// provider, rooted at the specified basePath.
func NewHttpFS(provider types.ResourceProvider, basePath string) http.FileSystem {
	return &httpFS{
		provider: provider,
		basePath: basePath,
	}
}

// Open opens the named file for reading.
func (h *httpFS) Open(name string) (http.File, error) {
	// Trim leading slash from the path provided by the http.FileServer
	// to ensure correct joining with the basePath.
	cleanName := strings.TrimPrefix(name, "/")
	fullName := path.Join(h.basePath, cleanName)

	// First, try to open as a file.
	res, fileErr := h.provider.ReadFile(fullName)
	if fileErr == nil {
		// It's a file, return its content.
		stat, err := h.provider.Stat(fullName)
		if err != nil {
			return nil, err
		}
		return newHttpFile(name, res.Content, stat), nil
	}

	// If it's not a file, try to open as a directory.
	stat, dirErr := h.provider.Stat(fullName)
	if dirErr == nil && stat.IsDir() {
		// It's a directory. Return a file handle that represents a directory.
		return newHttpFile(name, nil, stat), nil
	}

	// If it's neither a file nor a directory, return the original file error.
	return nil, fileErr
}

// httpFile implements the http.File interface for an in-memory file or directory.
type httpFile struct {
	*bytes.Reader
	name string
	info fs.FileInfo
}

// newHttpFile creates a new http.File.
// If content is nil, it represents a directory.
func newHttpFile(name string, content []byte, info fs.FileInfo) *httpFile {
	var reader *bytes.Reader
	if content != nil {
		reader = bytes.NewReader(content)
	} else {
		// For directories, create an empty reader.
		reader = bytes.NewReader([]byte{})
	}
	return &httpFile{
		Reader: reader,
		name:   name,
		info:   info,
	}
}

// Close is a no-op for an in-memory file.
func (f *httpFile) Close() error {
	return nil
}

// Readdir is not fully implemented for simplicity, as go-zero's static file
// server doesn't rely on it when serving individual files.
// A full implementation would require ReadDir on the provider.
func (f *httpFile) Readdir(count int) ([]fs.FileInfo, error) {
	// http.FileServer requires a non-nil slice for directories to serve index.html.
	if f.info.IsDir() {
		return []fs.FileInfo{}, nil
	}
	// For files, this should indicate it's not a directory.
	return nil, fs.ErrNotExist
}

// Stat returns the file's info.
func (f *httpFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}
