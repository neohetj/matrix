package validation

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/neohetj/matrix/internal/loader"
)

func TestDiscoverLoaderPathsIncludesCodeAndCommonDSLRoots(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"code/dsl/rulechains",
		"code/dsl/endpoints",
		"code/dsl/shared",
		"common/dsl/rulechains",
		"common/dsl/endpoints",
		"common/dsl/shared",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	paths := DiscoverLoaderPaths(loader.NewFileProvider(root, 50), nil)

	assertStringSlice(t, paths.RuleChains, []string{"code/dsl/rulechains", "common/dsl/rulechains"})
	assertStringSlice(t, paths.Endpoints, []string{"code/dsl/endpoints", "common/dsl/endpoints"})
	assertStringSlice(t, paths.Shared, []string{"code/dsl/shared", "common/dsl/shared"})
}

func TestDiscoverLoaderPathsSkipsMissingRootsAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"code/dsl/rulechains",
		"code/dsl/shared",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	paths := DiscoverLoaderPaths(loader.NewFileProvider(root, 50), []string{"code/dsl", "missing/dsl", "code/dsl"})

	assertStringSlice(t, paths.RuleChains, []string{"code/dsl/rulechains"})
	assertStringSlice(t, paths.Endpoints, nil)
	assertStringSlice(t, paths.Shared, []string{"code/dsl/shared"})
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(want) == 0 {
		if len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
