package loader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/neohetj/matrix/pkg/types"
)

// FileProvider implements the ResourceProvider interface for the local filesystem.
// It reads files from a specified base directory.
type FileProvider struct {
	baseDir  string
	fs       fs.FS
	priority int
}

// NewFileProvider creates a new provider for the given base directory.
// The base directory is the root from which files will be read.
func NewFileProvider(baseDir string, priority int) *FileProvider {
	if priority == 0 {
		priority = 50 // Default priority
	}
	if priority < -1 || priority > 100 {
		priority = 50 // Fallback to default if out of range
	}
	return &FileProvider{
		baseDir:  baseDir,
		fs:       os.DirFS(baseDir),
		priority: priority,
	}
}

// Open opens the named file.
func (p *FileProvider) Open(name string) (fs.File, error) {
	return p.fs.Open(name)
}

// ReadFile reads a file from the local filesystem.
func (p *FileProvider) ReadFile(name string) (*types.Resource, error) {
	// Use the embedded fs.FS to read the file content to ensure
	// it's relative to the baseDir.
	content, err := fs.ReadFile(p.fs, name)
	if err != nil {
		return nil, err
	}
	return &types.Resource{
		Content: content,
		Source:  types.FromExternal,
	}, nil
}

// ReadDir reads the contents of a directory from the local filesystem.
func (p *FileProvider) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(p.fs, name)
}

// Stat returns a FileInfo describing the named file from the local filesystem.
func (p *FileProvider) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(p.fs, name)
}

// WalkDir walks the file tree rooted at root.
func (p *FileProvider) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(p.fs, root, fn)
}

// Name returns the name of the provider.
func (p *FileProvider) Name() string {
	absPath, err := filepath.Abs(p.baseDir)
	if err != nil {
		return fmt.Sprintf("FileProvider(%s, error: %v)", p.baseDir, err)
	}
	return "FileProvider(" + absPath + ")"
}

// Priority returns the priority of the provider.
func (p *FileProvider) Priority() int {
	return p.priority
}
