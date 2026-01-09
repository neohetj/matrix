package asset

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/neohetj/matrix/pkg/cnst"
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
	path := uri.Host + uri.Path
	if path == "" {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from %s: %w", path, err)
	}
	return string(content), nil
}

// Set is not supported for rel:// URIs.
func (a RelAsset) Set(uri *url.URL, ctx *AssetContext, value any) error {
	return fmt.Errorf("setting values via rel:// is not supported")
}
