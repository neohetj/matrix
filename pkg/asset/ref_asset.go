package asset

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

// RefAsset represents a parsed ref:// URI.
type RefAsset struct {
	Scheme string
	Host   string
	RefID  string
}

// ParseRef parses a ref:// URI into a RefAsset struct.
// It handles the validation of the URI scheme.
func ParseRef(uri string) (RefAsset, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return RefAsset{}, fmt.Errorf("invalid ref uri: %w", err)
	}

	return ParseRefFromURL(u)
}

// ParseRefFromURL converts a URL object to a RefAsset struct.
// This function assumes the scheme has already been validated.
func ParseRefFromURL(u *url.URL) (RefAsset, error) {
	if u.Scheme != cnst.SchemeRef {
		return RefAsset{}, fmt.Errorf("invalid ref uri scheme: %s", u.Scheme)
	}

	trimmedPath := strings.TrimPrefix(u.Path, "/")
	if u.Host == "" {
		return RefAsset{Scheme: u.Scheme, Host: u.Host, RefID: trimmedPath}, nil
	}
	if trimmedPath == "" {
		return RefAsset{Scheme: u.Scheme, Host: u.Host, RefID: u.Host}, nil
	}
	// Both host and path are present, join them with a slash.
	refID := u.Host + "/" + trimmedPath
	return RefAsset{Scheme: u.Scheme, Host: u.Host, RefID: refID}, nil
}

// NormalizeAssetURI returns the URI as-is for ref:// assets.
func (a RefAsset) NormalizeAssetURI(uri string) string {
	return uri
}

// Handle resolves ref:// URIs.
func (a RefAsset) Handle(uri *url.URL, ctx *AssetContext) (any, error) {
	refAsset, err := ParseRefFromURL(uri)
	if err != nil {
		return nil, err
	}
	nodeId := refAsset.RefID

	if nodeId == "" {
		return nil, fmt.Errorf("empty ref node id")
	}

	pool := GetNodePool(ctx)
	if pool == nil {
		return nil, fmt.Errorf("node pool not found in asset context")
	}

	instance, err := pool.GetInstance(nodeId)
	if err != nil {
		return nil, types.AssetNotFound.Wrap(err)
	}

	return instance, nil
}

// Set is not supported for ref:// URIs.
func (a RefAsset) Set(uri *url.URL, ctx *AssetContext, value any) error {
	return fmt.Errorf("setting values via ref:// is not supported")
}
