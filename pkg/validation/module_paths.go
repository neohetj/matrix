package validation

import (
	"path/filepath"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

var DefaultModuleDSLRoots = []string{"code/dsl", "common/dsl"}

func DiscoverLoaderPaths(provider types.ResourceProvider, dslRoots []string) LoaderPaths {
	if provider == nil {
		return LoaderPaths{}
	}
	if len(dslRoots) == 0 {
		dslRoots = DefaultModuleDSLRoots
	}

	var paths LoaderPaths
	ruleChainSeen := map[string]struct{}{}
	endpointSeen := map[string]struct{}{}
	sharedSeen := map[string]struct{}{}

	for _, root := range dslRoots {
		root = normalizeLoaderPath(root)
		if root == "" {
			continue
		}
		appendExistingPath(provider, ruleChainSeen, &paths.RuleChains, filepath.Join(root, "rulechains"))
		appendExistingPath(provider, endpointSeen, &paths.Endpoints, filepath.Join(root, "endpoints"))
		appendExistingPath(provider, sharedSeen, &paths.Shared, filepath.Join(root, "shared"))
	}

	return paths
}

func appendExistingPath(provider types.ResourceProvider, seen map[string]struct{}, paths *[]string, candidate string) {
	candidate = normalizeLoaderPath(candidate)
	if candidate == "" {
		return
	}
	if _, ok := seen[candidate]; ok {
		return
	}
	stat, err := provider.Stat(candidate)
	if err != nil || !stat.IsDir() {
		return
	}
	seen[candidate] = struct{}{}
	*paths = append(*paths, candidate)
}

func normalizeLoaderPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if path == "." {
		return ""
	}
	return path
}
