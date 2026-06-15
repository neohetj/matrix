package config

import (
	"errors"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestConfigResolverPlaceholderUsesEnvBeforeYAMLAndDefault(t *testing.T) {
	resolver := NewConfigResolver(
		WithBusinessConfig(types.ConfigMap{
			"IDENTITYX_QUERY_TIMEOUT_MS": "3000",
		}),
		WithEnvLookup(func(key string) (string, bool) {
			if key == "IDENTITYX_QUERY_TIMEOUT_MS" {
				return "5000", true
			}
			return "", false
		}),
	)

	value, meta, err := Resolve[int](resolver, ConfigSpec{
		Key:        "IDENTITYX_QUERY_TIMEOUT_MS",
		Resolution: ResolutionPlaceholder,
		Default:    "1000",
	})

	assert.NoError(t, err)
	assert.Equal(t, 5000, value)
	assert.Equal(t, SourceEnv, meta.Source)
}

func TestConfigResolverPlaceholderFallsBackToYAMLThenDefault(t *testing.T) {
	resolver := NewConfigResolver(
		WithBusinessConfig(types.ConfigMap{
			"IDENTITYX_QUERY_TIMEOUT_MS": "3000",
		}),
		WithEnvLookup(func(string) (string, bool) {
			return "", false
		}),
	)

	value, meta, err := Resolve[int](resolver, ConfigSpec{
		Key:        "IDENTITYX_QUERY_TIMEOUT_MS",
		Resolution: ResolutionPlaceholder,
		Default:    "1000",
	})

	assert.NoError(t, err)
	assert.Equal(t, 3000, value)
	assert.Equal(t, SourceYAML, meta.Source)

	value, meta, err = Resolve[int](resolver, ConfigSpec{
		Key:        "MISSING_TIMEOUT_MS",
		Resolution: ResolutionPlaceholder,
		Default:    "1000",
	})

	assert.NoError(t, err)
	assert.Equal(t, 1000, value)
	assert.Equal(t, SourceDefault, meta.Source)
}

func TestConfigResolverPlaceholderUsesAlias(t *testing.T) {
	resolver := NewConfigResolver(
		WithEnvLookup(func(key string) (string, bool) {
			if key == "LEGACY_TIMEOUT_MS" {
				return "2500", true
			}
			return "", false
		}),
	)

	value, meta, err := Resolve[int](resolver, ConfigSpec{
		Key:        "IDENTITYX_QUERY_TIMEOUT_MS",
		Aliases:    []string{"LEGACY_TIMEOUT_MS"},
		Resolution: ResolutionPlaceholder,
	})

	assert.NoError(t, err)
	assert.Equal(t, 2500, value)
	assert.Equal(t, SourceEnv, meta.Source)
	assert.Equal(t, "LEGACY_TIMEOUT_MS", meta.Alias)
}

func TestConfigResolverRequiredPlaceholderMissing(t *testing.T) {
	resolver := NewConfigResolver(
		WithEnvLookup(func(string) (string, bool) {
			return "", false
		}),
	)

	_, meta, err := Resolve[string](resolver, ConfigSpec{
		Key:        "IDENTITYX_REQUIRED_VALUE",
		Resolution: ResolutionPlaceholder,
		Required:   true,
	})

	assert.True(t, errors.Is(err, ErrRequiredConfig))
	assert.Equal(t, SourceNone, meta.Source)
}

func TestConfigResolverSecretDoesNotFallbackToYAML(t *testing.T) {
	resolver := NewConfigResolver(
		WithBusinessConfig(types.ConfigMap{
			"IDENTITYX_JWT_SECRET": "from-yaml",
		}),
		WithEnvLookup(func(string) (string, bool) {
			return "", false
		}),
	)

	_, meta, err := Resolve[string](resolver, ConfigSpec{
		Key:        "IDENTITYX_JWT_SECRET",
		Resolution: ResolutionPlaceholder,
		Secret:     true,
		Required:   true,
	})

	assert.True(t, errors.Is(err, ErrRequiredSecret))
	assert.Equal(t, SourceNone, meta.Source)
}

func TestConfigResolverSecretMetaRedactsValue(t *testing.T) {
	resolver := NewConfigResolver(
		WithEnvLookup(func(key string) (string, bool) {
			if key == "IDENTITYX_JWT_SECRET" {
				return "super-secret", true
			}
			return "", false
		}),
	)

	value, meta, err := Resolve[string](resolver, ConfigSpec{
		Key:        "IDENTITYX_JWT_SECRET",
		Resolution: ResolutionPlaceholder,
		Secret:     true,
		Required:   true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "super-secret", value)
	assert.Equal(t, SourceEnv, meta.Source)
	assert.Equal(t, RedactedValue, meta.SafeValue(value))
}

func TestConfigResolverOptionalSecretMissingDoesNotFallback(t *testing.T) {
	resolver := NewConfigResolver(
		WithBusinessConfig(types.ConfigMap{
			"IDENTITYX_OPTIONAL_SECRET": "from-yaml",
		}),
		WithEnvLookup(func(string) (string, bool) {
			return "", false
		}),
	)

	value, meta, err := Resolve[string](resolver, ConfigSpec{
		Key:        "IDENTITYX_OPTIONAL_SECRET",
		Resolution: ResolutionPlaceholder,
		Secret:     true,
		Required:   false,
	})

	assert.NoError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, SourceNone, meta.Source)
	assert.Equal(t, RedactedValue, meta.SafeValue(value))
}

func TestConfigResolverNodeExplicitWinsOverEnvAndYAML(t *testing.T) {
	resolver := NewConfigResolver(
		WithBusinessConfig(types.ConfigMap{
			"COLLECTION": "from-yaml",
		}),
		WithEnvLookup(func(key string) (string, bool) {
			if key == "COLLECTION" {
				return "from-env", true
			}
			return "", false
		}),
	)

	value, meta, err := Resolve[string](resolver, ConfigSpec{
		Key:        "COLLECTION",
		Resolution: ResolutionNodeExplicit,
		Explicit:   "identity_profiles",
	})

	assert.NoError(t, err)
	assert.Equal(t, "identity_profiles", value)
	assert.Equal(t, SourceNodeExplicit, meta.Source)
}
