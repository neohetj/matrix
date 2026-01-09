package asset

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/utils"
)

// ConfigAsset represents a parsed config:// URI.
type ConfigAsset struct {
	Scheme string
	Host   string
	Key    string
	Query  url.Values
}

// ParseConfig parses a config:// URI into key and query.
func ParseConfig(uri string) (ConfigAsset, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return ConfigAsset{}, fmt.Errorf("invalid config uri: %w", err)
	}
	return ParseConfigFromURL(u)
}

// ParseConfigFromURL converts a URL object to a ConfigAsset struct.
func ParseConfigFromURL(u *url.URL) (ConfigAsset, error) {
	if u.Scheme != cnst.SchemeConfig {
		return ConfigAsset{}, fmt.Errorf("invalid config uri scheme: %s", u.Scheme)
	}

	key := strings.TrimPrefix(u.Path, "/")
	return ConfigAsset{
		Scheme: u.Scheme,
		Host:   u.Host,
		Key:    key,
		Query:  u.Query(),
	}, nil
}

// NormalizeAssetURI returns the URI as-is for config:// assets.
func (a ConfigAsset) NormalizeAssetURI(uri string) string {
	return uri
}

// NewConfigAsset creates a new config:// builder state.
func NewConfigAsset() ConfigAsset {
	return ConfigAsset{Scheme: cnst.SchemeConfig, Query: url.Values{}}
}

// SetKey sets the config key.
func (a ConfigAsset) SetKey(key string) ConfigAsset {
	a.Scheme = cnst.SchemeConfig
	a.Key = strings.TrimPrefix(key, "/")
	if a.Query == nil {
		a.Query = url.Values{}
	}
	return a
}

// Scope sets the config scope query.
func (a ConfigAsset) Scope(scope string) ConfigAsset {
	if scope != "" {
		if a.Query == nil {
			a.Query = url.Values{}
		}
		a.Query.Set("scope", scope)
	}
	return a
}

// Default sets the config default query.
func (a ConfigAsset) Default(defaultVal string) ConfigAsset {
	if defaultVal != "" {
		if a.Query == nil {
			a.Query = url.Values{}
		}
		a.Query.Set("default", defaultVal)
	}
	return a
}

// Type sets the config type query.
func (a ConfigAsset) Type(typeVal string) ConfigAsset {
	if typeVal != "" {
		if a.Query == nil {
			a.Query = url.Values{}
		}
		a.Query.Set("type", typeVal)
	}
	return a
}

// Build assembles the URI string.
func (a ConfigAsset) Build() string {
	scheme := a.Scheme
	if scheme == "" {
		scheme = cnst.SchemeConfig
	}
	if a.Query == nil {
		a.Query = url.Values{}
	}
	u := url.URL{
		Scheme:   scheme,
		Path:     "/" + strings.TrimPrefix(a.Key, "/"),
		RawQuery: a.Query.Encode(),
	}
	return u.String()
}

// Handle resolves config:// URIs.
func (a ConfigAsset) Handle(uri *url.URL, ctx *AssetContext) (any, error) {
	configPath := strings.TrimPrefix(uri.Path, "/")

	query := uri.Query()
	scopeVal := query.Get("scope")
	defaultValue := query.Get("default")

	var scope []string
	if scopeVal == "-" {
		scope = []string{}
	} else if scopeVal != "" {
		scope = strings.Split(scopeVal, ",")
	} else {
		scope = []string{"business", "node", "engine", "env"}
	}

	var rawVal any
	var found bool
	searched := make([]string, 0, len(scope))

	for _, s := range scope {
		if found {
			break
		}

		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}
		searched = append(searched, trimmed)

		switch trimmed {
		case "business":
			if nodeCtx := ctx.NodeCtx(); nodeCtx != nil {
				if nodeConfig := nodeCtx.Config(); nodeConfig != nil {
					if bizAny, ok := nodeConfig["business"]; ok {
						if val, exists, err := utils.ExtractByPath(bizAny, configPath); err == nil && exists {
							rawVal = val
							found = true
						}
					}
				}
			}
		case "node":
			if nodeConfig := ctx.Config(); nodeConfig != nil {
				if val, exists, err := utils.ExtractByPath(nodeConfig, configPath); err == nil && exists {
					rawVal = val
					found = true
				}
			}
		case "engine":
			if nodeCtx := ctx.NodeCtx(); nodeCtx != nil && nodeCtx.GetRuntime() != nil && nodeCtx.GetRuntime().GetEngine() != nil {
				bizConfig := nodeCtx.GetRuntime().GetEngine().BizConfig()
				if val, exists, err := utils.ExtractByPath(bizConfig, configPath); err == nil && exists {
					rawVal = val
					found = true
				} else {
					if v, ok := nodeCtx.GetRuntime().GetEngine().GetEngineConfig(configPath); ok {
						rawVal = v
						found = true
					}
				}
			}
		case "env":
			if val := os.Getenv(configPath); val != "" {
				rawVal = val
				found = true
			} else {
				envKey := strings.ToUpper(strings.ReplaceAll(configPath, ".", "_"))
				if val := os.Getenv(envKey); val != "" {
					rawVal = val
					found = true
				}
			}
		}
		if found {
			setConfigSearchedScopes(ctx, searched)
		}
	}

	if !found {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return nil, fmt.Errorf("config key not found: %s", configPath)
	}

	return rawVal, nil
}

// Set is not supported for config:// URIs.
func (a ConfigAsset) Set(uri *url.URL, ctx *AssetContext, value any) error {
	return fmt.Errorf("setting config values via config:// is not supported")
}

// BuildWithRemainingScopes builds a new URI with remaining scopes after removing searched ones.
func (a ConfigAsset) BuildWithRemainingScopes(ctx *AssetContext) string {
	query := a.Query
	if query == nil {
		query = url.Values{}
	}
	baseScopeVal := query.Get("scope")
	var baseScopes []string
	if baseScopeVal == "-" {
		baseScopes = []string{}
	} else if baseScopeVal != "" {
		baseScopes = strings.Split(baseScopeVal, ",")
	} else {
		baseScopes = []string{"business", "node", "engine", "env"}
	}

	searched := GetConfigSearchedScopes(ctx)
	if len(searched) == 0 {
		return a.Build()
	}

	searchedSet := make(map[string]struct{}, len(searched))
	for _, s := range searched {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		searchedSet[s] = struct{}{}
	}

	remaining := make([]string, 0, len(baseScopes))
	for _, s := range baseScopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := searchedSet[s]; ok {
			continue
		}
		remaining = append(remaining, s)
	}

	if len(remaining) == 0 {
		return a.Scope("-").Build()
	}

	return a.Scope(strings.Join(remaining, ",")).Build()
}

const configScopeSearchedKey = "config_scope_searched"

type configScopeSearched struct {
	mu     sync.Mutex
	scopes []string
}

func setConfigSearchedScopes(ctx *AssetContext, scopes []string) {
	if ctx == nil {
		return
	}
	copyScopes := append([]string(nil), scopes...)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.extras[configScopeSearchedKey] = &configScopeSearched{scopes: copyScopes}
}

func GetConfigSearchedScopes(ctx *AssetContext) []string {
	if ctx == nil {
		return nil
	}
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	if v, ok := ctx.extras[configScopeSearchedKey].(*configScopeSearched); ok {
		return append([]string(nil), v.scopes...)
	}
	return nil
}

func ClearConfigSearchedScopes(ctx *AssetContext) {
	if ctx == nil {
		return
	}
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	delete(ctx.extras, configScopeSearchedKey)
}
