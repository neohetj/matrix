package loader

import (
	"context"
	"errors"
	"io/fs"
	"path" // Use path for virtual paths
	"path/filepath"
	"sort"

	"gitlab.com/neohet/matrix/pkg/types"
)

// HybridLoader implements the ResourceProvider interface by composing multiple
// providers in a prioritized order. It tries each provider sequentially until
// one succeeds.
type HybridLoader struct {
	providers []types.ResourceProvider
	logger    types.Logger
}

// NewHybridLoader creates a new loader with a list of providers and a logger.
// The providers are tried in the order they are given.
func NewHybridLoader(logger types.Logger, providers ...types.ResourceProvider) *HybridLoader {
	hl := &HybridLoader{
		providers: providers,
		logger:    logger,
	}
	hl.sortProviders()
	hl.logProviderOrder()
	return hl
}

func (l *HybridLoader) logProviderOrder() {
	if l.logger == nil {
		return
	}
	providerNames := make([]string, len(l.providers))
	for i, p := range l.providers {
		providerNames[i] = p.Name()
	}
	l.logger.Infof(context.Background(), "HybridLoader provider search order: %v", providerNames)
}

func (l *HybridLoader) sortProviders() {
	sort.SliceStable(l.providers, func(i, j int) bool {
		return l.providers[i].Priority() > l.providers[j].Priority()
	})
}

// Open opens the named file by trying each provider in order.
func (l *HybridLoader) Open(name string) (fs.File, error) {
	name = filepath.ToSlash(name)
	var lastErr error
	for _, p := range l.providers {
		file, err := p.Open(name)
		if err == nil {
			return file, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// ReadFile reads a file by trying each provider in order.
// It returns the resource from the first provider that succeeds.
// If all providers fail, it returns the error from the last provider.
func (l *HybridLoader) ReadFile(name string) (*types.Resource, error) {
	name = filepath.ToSlash(name)
	var lastErr error
	for _, p := range l.providers {
		res, err := p.ReadFile(name)
		if err == nil {
			if l.logger != nil {
				l.logger.Infof(context.Background(), "Resource '%s' loaded from source: %s", name, res.Source)
			}
			return res, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// ReadDir reads a directory by trying each provider in order and merging the results.
// It prioritizes entries from earlier providers in case of name conflicts.
func (l *HybridLoader) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.ToSlash(name)
	var mergedEntries []fs.DirEntry
	seen := make(map[string]bool)
	found := false

	for _, p := range l.providers {
		entries, err := p.ReadDir(name)
		if err == nil {
			found = true
			for _, entry := range entries {
				if !seen[entry.Name()] {
					mergedEntries = append(mergedEntries, entry)
					seen[entry.Name()] = true
				}
			}
		}
	}

	if !found {
		return nil, fs.ErrNotExist
	}
	return mergedEntries, nil
}

// Stat returns a FileInfo by trying each provider in order.
// It returns the FileInfo from the first provider that succeeds.
func (l *HybridLoader) Stat(name string) (fs.FileInfo, error) {
	name = filepath.ToSlash(name)
	var lastErr error
	for _, p := range l.providers {
		var err error
		info, err := p.Stat(name)
		if err == nil {
			return info, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// AddProvider adds a new provider to the end of the priority list.
func (l *HybridLoader) AddProvider(p types.ResourceProvider) {
	l.providers = append(l.providers, p)
	l.sortProviders()
	l.logProviderOrder()
}

// WalkDir walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. It merges the contents from all providers.
func (l *HybridLoader) WalkDir(root string, fn fs.WalkDirFunc) error {
	root = filepath.ToSlash(root)
	// Stat the root directory first to ensure it exists in at least one provider.
	info, err := l.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("WalkDir root is not a directory: " + root)
	}

	// Call the callback for the root directory itself.
	err = fn(root, fs.FileInfoToDirEntry(info), nil)
	if err == fs.SkipDir {
		return nil
	}
	if err != nil {
		return err
	}

	return l.walk(root, fn)
}

func (l *HybridLoader) walk(currentPath string, fn fs.WalkDirFunc) error {
	entries, err := l.ReadDir(currentPath)
	if err != nil {
		return fn(currentPath, nil, err)
	}

	for _, entry := range entries {
		// Always use the 'path' package for manipulating virtual paths.
		fullPath := path.Join(currentPath, entry.Name())
		err := fn(fullPath, entry, nil)
		if err != nil {
			if err == fs.SkipDir && entry.IsDir() {
				continue
			}
			return err
		}

		if entry.IsDir() {
			if err := l.walk(fullPath, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// Name returns the name of the provider.
func (l *HybridLoader) Name() string {
	return "HybridLoader"
}

// Priority returns the priority of the provider.
func (l *HybridLoader) Priority() int {
	// HybridLoader itself doesn't have a priority, but we need to implement the interface.
	// Return a neutral value.
	return -1
}
