package asset

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

// RelAsset represents a parsed rel:// URI.
type RelAsset struct {
	Scheme string
	Host   string
	Path   string
}

// Parse parses a rel:// URI into path.
func ParseRel(uri string) (RelAsset, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return RelAsset{}, fmt.Errorf("invalid rel uri: %w", err)
	}
	return ParseRelFromURL(u)
}

// ParseRelFromURL converts a URL object to a RelAsset struct.
func ParseRelFromURL(u *url.URL) (RelAsset, error) {
	if u.Scheme != cnst.SchemeRel {
		return RelAsset{}, fmt.Errorf("invalid rel uri scheme: %s", u.Scheme)
	}
	path := strings.TrimPrefix(u.Host+u.Path, "/")
	return RelAsset{Scheme: u.Scheme, Host: u.Host, Path: path}, nil
}

// NormalizeAssetURI returns the URI as-is for rel:// assets.
func (a RelAsset) NormalizeAssetURI(uri string) string {
	return uri
}

// Handle resolves rel:// URIs.
func (a RelAsset) Handle(uri *url.URL, ctx *AssetContext) (any, error) {
	rawPath := strings.TrimSpace(uri.Host + uri.Path)
	if rawPath == "" {
		return "", nil
	}

	// Absolute rel:// path keeps direct file read behavior.
	if filepath.IsAbs(rawPath) {
		content, err := os.ReadFile(rawPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, types.AssetNotFound.Wrap(err)
			}
			return nil, fmt.Errorf("failed to read file from %s: %w", rawPath, err)
		}
		return string(content), nil
	}

	nodeCtx := relNodeCtx(ctx)
	if nodeCtx == nil {
		return nil, types.AssetNotFound.Wrap(fmt.Errorf("relative rel asset requires node context, relPath=%q", rawPath))
	}

	sourcePath := ""
	if def := nodeCtx.SelfDef(); def != nil {
		// SourcePath comes from the node's DSL definition file (json),
		// not from the node function's Go code file.
		sourcePath = filepath.ToSlash(strings.TrimSpace(def.SourcePath))
	}
	if sourcePath == "" {
		return nil, types.AssetNotFound.Wrap(fmt.Errorf(
			"relative rel asset requires node SourcePath, chainId=%s nodeId=%s relPath=%q",
			strings.TrimSpace(nodeCtx.ChainID()),
			strings.TrimSpace(nodeCtx.NodeID()),
			rawPath,
		))
	}

	rt := nodeCtx.GetRuntime()
	if rt == nil || rt.GetEngine() == nil || rt.GetEngine().Loader() == nil {
		return nil, types.AssetNotFound.Wrap(fmt.Errorf(
			"relative rel asset requires engine loader, chainId=%s nodeId=%s sourcePath=%s relPath=%q",
			strings.TrimSpace(nodeCtx.ChainID()),
			strings.TrimSpace(nodeCtx.NodeID()),
			sourcePath,
			rawPath,
		))
	}

	// Resolve rel:// against the node DSL file directory:
	// sourcePath = code/dsl/rulechains/example/example_flow.json
	// baseDir    = code/dsl/rulechains/example
	// relPath    = ../../prompts/example/example_prompt.txt
	// resolved   = code/dsl/prompts/example/example_prompt.txt
	baseDir := path.Dir(sourcePath)
	resolvedPath := path.Clean(path.Join(baseDir, rawPath))
	res, err := rt.GetEngine().Loader().ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist) {
			return nil, types.AssetNotFound.Wrap(fmt.Errorf(
				"chainId=%s nodeId=%s sourcePath=%s resolvedPath=%s: %w",
				strings.TrimSpace(nodeCtx.ChainID()),
				strings.TrimSpace(nodeCtx.NodeID()),
				sourcePath,
				resolvedPath,
				err,
			))
		}
		return nil, types.AssetNotFound.Wrap(fmt.Errorf(
			"failed to read rel asset from loader, chainId=%s nodeId=%s sourcePath=%s resolvedPath=%s: %w",
			strings.TrimSpace(nodeCtx.ChainID()),
			strings.TrimSpace(nodeCtx.NodeID()),
			sourcePath,
			resolvedPath,
			err,
		))
	}
	return string(res.Content), nil
}

// Set writes content to the file pointed by rel:// URI.
func (a RelAsset) Set(uri *url.URL, ctx *AssetContext, value any) error {
	path := uri.Host + uri.Path
	if path == "" {
		return fmt.Errorf("empty path in rel uri")
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("unsupported value type for rel asset set: %T", value)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file to %s: %w", path, err)
	}
	return nil
}

func relNodeCtx(ctx *AssetContext) types.NodeCtx {
	if ctx == nil {
		return nil
	}
	return ctx.NodeCtx()
}
