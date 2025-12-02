package types

import "io/fs"

// Source indicates the origin of a loaded resource.
type Source int

const (
	// FromUnknown indicates an unknown or unspecified source.
	FromUnknown Source = iota
	// FromEmbed indicates the resource was loaded from the embedded filesystem.
	FromEmbed
	// FromExternal indicates the resource was loaded from the external filesystem.
	FromExternal
	// FromEtcd indicates the resource was loaded from etcd. (For future use)
	FromEtcd
)

// String returns the string representation of the Source.
func (s Source) String() string {
	switch s {
	case FromEmbed:
		return "embed"
	case FromExternal:
		return "external"
	case FromEtcd:
		return "etcd"
	default:
		return "unknown"
	}
}

// Resource holds the content of a file and its source.
type Resource struct {
	Content []byte
	Source  Source
}

// ResourceProvider defines a unified interface for reading files from various sources,
// such as an embedded filesystem or the local disk. This abstraction is key to the
// hybrid loading strategy. It embeds standard library interfaces for better composability.
type ResourceProvider interface {
	fs.ReadDirFS
	fs.StatFS

	// ReadFile reads the file named by name and returns its content and source.
	// This is a custom method not part of the standard fs interfaces.
	ReadFile(name string) (*Resource, error)
	// WalkDir walks the file tree rooted at root.
	WalkDir(root string, fn fs.WalkDirFunc) error
	// Name returns the name of the provider.
	Name() string
	// Priority returns the priority of the provider.
	Priority() int
}
