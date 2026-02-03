package utils

import (
	"io/fs"
	"strings"
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

// ----------------------- MockResourceProvider -----------------------
// MockResourceProvider is a mock implementation of types.ResourceProvider.
type MockResourceProvider struct {
	Files map[string]struct {
		Content string
		IsDir   bool
	}
}

func (m *MockResourceProvider) WalkDir(root string, fn fs.WalkDirFunc) error {
	for path, file := range m.Files {
		if strings.HasPrefix(path, root) {
			parts := strings.Split(path, "/")
			filename := parts[len(parts)-1]
			d := &MockDirEntry{name: filename, isDir: file.IsDir}
			if err := fn(path, d, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MockResourceProvider) Priority() int {
	return 0
}

func (m *MockResourceProvider) Name() string {
	return "mock"
}

func (m *MockResourceProvider) Open(name string) (fs.File, error) {
	if file, ok := m.Files[name]; ok && !file.IsDir {
		return &MockFSFile{name: name, content: file.Content}, nil
	}
	return nil, fs.ErrNotExist
}

func (m *MockResourceProvider) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	for path, file := range m.Files {
		if strings.HasPrefix(path, name) {
			parts := strings.Split(path, "/")
			filename := parts[len(parts)-1]
			entries = append(entries, &MockDirEntry{name: filename, isDir: file.IsDir})
		}
	}
	return entries, nil
}

func (m *MockResourceProvider) ReadFile(name string) (*types.Resource, error) {
	if file, ok := m.Files[name]; ok {
		return &types.Resource{Content: []byte(file.Content), Source: types.FromExternal}, nil
	}
	return nil, fs.ErrNotExist
}

func (m *MockResourceProvider) Stat(name string) (fs.FileInfo, error) {
	if _, ok := m.Files[name]; ok {
		return &MockFileInfo{}, nil
	}
	return nil, fs.ErrNotExist
}

// ----------------------- MockFSFile -----------------------
// MockFSFile is a mock implementation of fs.File.
type MockFSFile struct {
	name    string
	content string
	offset  int64
}

func (f *MockFSFile) Stat() (fs.FileInfo, error) { return &MockFileInfo{name: f.name}, nil }
func (f *MockFSFile) Read(b []byte) (int, error) {
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}
func (f *MockFSFile) Close() error { return nil }

// ----------------------- MockFileInfo -----------------------
// MockFileInfo is a mock implementation of fs.FileInfo.
type MockFileInfo struct {
	name  string
	isDir bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return 0 }
func (m *MockFileInfo) Mode() fs.FileMode  { return 0 }
func (m *MockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

// ----------------------- MockDirEntry -----------------------
// MockDirEntry is a mock implementation of fs.DirEntry.
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m *MockDirEntry) Name() string      { return m.name }
func (m *MockDirEntry) IsDir() bool       { return m.isDir }
func (m *MockDirEntry) Type() fs.FileMode { return 0 }
func (m *MockDirEntry) Info() (fs.FileInfo, error) {
	return &MockFileInfo{name: m.name, isDir: m.isDir}, nil
}
