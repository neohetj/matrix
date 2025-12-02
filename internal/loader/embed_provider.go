package loader

import (
	"embed"
	"io/fs"

	"gitlab.com/neohet/matrix/pkg/types"
)

// EmbedProvider implements the ResourceProvider interface for an embedded filesystem.
type EmbedProvider struct {
	fs embed.FS
}

// NewEmbedProvider creates a new provider for the given embedded filesystem.
func NewEmbedProvider(fs embed.FS) *EmbedProvider {
	return &EmbedProvider{fs: fs}
}

// Open opens the named file from the embedded filesystem.
func (p *EmbedProvider) Open(name string) (fs.File, error) {
	return p.fs.Open(name)
}

// ReadFile reads a file from the embedded filesystem.
func (p *EmbedProvider) ReadFile(name string) (*types.Resource, error) {
	content, err := p.fs.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return &types.Resource{
		Content: content,
		Source:  types.FromEmbed,
	}, nil
}

// ReadDir reads the contents of a directory from the embedded filesystem.
func (p *EmbedProvider) ReadDir(name string) ([]fs.DirEntry, error) {
	return p.fs.ReadDir(name)
}

// Stat returns a FileInfo describing the named file from the embedded filesystem.
func (p *EmbedProvider) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(p.fs, name)
}

// WalkDir walks the file tree rooted at root within the embedded filesystem.
func (p *EmbedProvider) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(p.fs, root, fn)
}

// Name returns the name of the provider.
func (p *EmbedProvider) Name() string {
	return "EmbedProvider"
}

// Priority returns the priority of the provider.
func (p *EmbedProvider) Priority() int {
	return 0
}
