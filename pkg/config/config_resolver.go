package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

type Resolution string

const (
	ResolutionPlaceholder  Resolution = "placeholder"
	ResolutionNodeExplicit Resolution = "node_explicit"
)

type ValueSource string

const (
	SourceNone         ValueSource = ""
	SourceEnv          ValueSource = "env"
	SourceYAML         ValueSource = "yaml"
	SourceDefault      ValueSource = "default"
	SourceNodeExplicit ValueSource = "node_explicit"
)

const RedactedValue = "[redacted]"

var (
	ErrRequiredConfig = errors.New("required config missing")
	ErrRequiredSecret = errors.New("required secret missing")
)

type ConfigSpec struct {
	Key        string
	Type       cnst.MType
	Resolution Resolution
	Required   bool
	Secret     bool
	Default    any
	Explicit   any
	Aliases    []string
}

type ResolveMeta struct {
	Key    string
	Alias  string
	Source ValueSource
	Secret bool
}

func (m ResolveMeta) SafeValue(raw any) any {
	if m.Secret {
		return RedactedValue
	}
	return raw
}

type ConfigResolver struct {
	businessConfig types.ConfigMap
	envLookup      asset.EnvLookup
}

type ResolverOption func(*ConfigResolver)

func NewConfigResolver(opts ...ResolverOption) *ConfigResolver {
	r := &ConfigResolver{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func NewConfigResolverFromMatrixConfig(cfg MatrixConfig, opts ...ResolverOption) *ConfigResolver {
	options := append([]ResolverOption{WithBusinessConfig(cfg.Business)}, opts...)
	return NewConfigResolver(options...)
}

func WithBusinessConfig(cfg types.ConfigMap) ResolverOption {
	return func(r *ConfigResolver) {
		r.businessConfig = cfg
	}
}

func WithEnvLookup(lookup asset.EnvLookup) ResolverOption {
	return func(r *ConfigResolver) {
		r.envLookup = lookup
	}
}

func Resolve[T any](r *ConfigResolver, spec ConfigSpec) (T, ResolveMeta, error) {
	var zero T
	if r == nil {
		r = NewConfigResolver()
	}
	if spec.Key == "" {
		return zero, ResolveMeta{}, fmt.Errorf("%w: empty key", ErrRequiredConfig)
	}

	resolution := spec.Resolution
	if resolution == "" {
		resolution = ResolutionPlaceholder
	}

	switch resolution {
	case ResolutionNodeExplicit:
		return resolveTyped[T](spec.Explicit, spec, ResolveMeta{Key: spec.Key, Source: SourceNodeExplicit, Secret: spec.Secret})
	case ResolutionPlaceholder:
		return resolvePlaceholder[T](r, spec)
	default:
		return zero, ResolveMeta{Key: spec.Key, Secret: spec.Secret}, fmt.Errorf("%w: unsupported resolution %q", ErrRequiredConfig, resolution)
	}
}

func resolvePlaceholder[T any](r *ConfigResolver, spec ConfigSpec) (T, ResolveMeta, error) {
	if val, meta, ok, err := r.resolveFromCandidates(spec, "env", SourceEnv); err != nil {
		var zero T
		return zero, meta, err
	} else if ok {
		return resolveTyped[T](val, spec, meta)
	}

	if !spec.Secret {
		if val, meta, ok, err := r.resolveFromCandidates(spec, "engine", SourceYAML); err != nil {
			var zero T
			return zero, meta, err
		} else if ok {
			return resolveTyped[T](val, spec, meta)
		}
	}

	if !spec.Secret && spec.Default != nil {
		return resolveTyped[T](spec.Default, spec, ResolveMeta{Key: spec.Key, Source: SourceDefault, Secret: spec.Secret})
	}

	var zero T
	if spec.Secret {
		if !spec.Required {
			return zero, ResolveMeta{Key: spec.Key, Secret: true}, nil
		}
		return zero, ResolveMeta{Key: spec.Key, Secret: true}, fmt.Errorf("%w: %s", ErrRequiredSecret, spec.Key)
	}
	if spec.Required {
		return zero, ResolveMeta{Key: spec.Key, Secret: spec.Secret}, fmt.Errorf("%w: %s", ErrRequiredConfig, spec.Key)
	}
	return zero, ResolveMeta{Key: spec.Key, Secret: spec.Secret}, nil
}

func (r *ConfigResolver) resolveFromCandidates(spec ConfigSpec, scope string, source ValueSource) (any, ResolveMeta, bool, error) {
	for _, key := range candidateKeys(spec) {
		raw, ok, err := r.resolveAssetRaw(key, scope)
		meta := ResolveMeta{Key: spec.Key, Alias: aliasFor(spec.Key, key), Source: source, Secret: spec.Secret}
		if err != nil {
			return nil, meta, false, err
		}
		if ok {
			return raw, meta, true, nil
		}
	}
	return nil, ResolveMeta{Key: spec.Key, Secret: spec.Secret}, false, nil
}

func (r *ConfigResolver) resolveAssetRaw(key string, scope string) (any, bool, error) {
	uri := asset.NewConfigAsset().SetKey(key).Scope(scope).Build()
	ctx := asset.NewAssetContext(
		asset.WithEngineConfig(r.businessConfig),
		asset.WithEnvLookup(r.lookupEnv),
	)

	raw, err := (asset.Asset[any]{URI: uri}).Resolve(ctx)
	if err != nil {
		if types.IsFault(err, cnst.CodeAssetNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if isEmptyConfigValue(raw) {
		return nil, false, nil
	}
	return raw, true, nil
}

func (r *ConfigResolver) lookupEnv(key string) (string, bool) {
	if r != nil && r.envLookup != nil {
		return r.envLookup(key)
	}
	return os.LookupEnv(key)
}

func resolveTyped[T any](raw any, spec ConfigSpec, meta ResolveMeta) (T, ResolveMeta, error) {
	var zero T
	if raw == nil {
		return zero, meta, nil
	}
	if val, ok := raw.(T); ok {
		return val, meta, nil
	}

	targetType := spec.Type
	if targetType == "" {
		if inferred, ok := utils.InferMType(zero); ok {
			targetType = inferred
			if targetType == cnst.INT64 {
				targetType = cnst.INT
			}
		}
	}
	if targetType == "" {
		return zero, meta, fmt.Errorf("unsupported config target type %T", zero)
	}

	converted, err := utils.Convert(raw, targetType)
	if err != nil {
		return zero, meta, err
	}
	if val, ok := converted.(T); ok {
		return val, meta, nil
	}
	return zero, meta, fmt.Errorf("converted config value %v (type %T) cannot be asserted to %T", converted, converted, zero)
}

func candidateKeys(spec ConfigSpec) []string {
	keys := make([]string, 0, 1+len(spec.Aliases))
	keys = append(keys, spec.Key)
	keys = append(keys, spec.Aliases...)
	return keys
}

func aliasFor(primary string, candidate string) string {
	if candidate == primary {
		return ""
	}
	return candidate
}

func isEmptyConfigValue(raw any) bool {
	if raw == nil {
		return true
	}
	if s, ok := raw.(string); ok {
		return s == ""
	}
	return false
}
