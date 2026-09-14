package config

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

// TestResolverPreservesExplicitEmptyValues 验证已提供空串保留来源，不落到 alias、YAML 或默认值。
func TestResolverPreservesExplicitEmptyValues(t *testing.T) {
	for _, tc := range []struct {
		name     string
		options  []ResolverOption
		wantFrom ValueSource
	}{
		{
			name: "environment_before_alias_and_yaml",
			options: []ResolverOption{WithBusinessConfig(types.ConfigMap{"MODE": "yaml"}), WithEnvLookup(func(key string) (string, bool) {
				return map[string]string{"MODE": "", "LEGACY_MODE": "alias"}[key], key == "MODE" || key == "LEGACY_MODE"
			})},
			wantFrom: SourceEnv,
		},
		{
			name:     "named_environment",
			options:  []ResolverOption{WithValueSources(context.Background(), snapshotValues{"MODE": ""}, snapshotValues{"MODE": "yaml"})},
			wantFrom: SourceEnv,
		},
		{
			name:     "yaml_alias",
			options:  []ResolverOption{WithBusinessConfig(types.ConfigMap{"LEGACY_MODE": ""}), WithEnvLookup(func(string) (string, bool) { return "", false })},
			wantFrom: SourceYAML,
		},
		{
			name:     "transformed_environment",
			options:  []ResolverOption{WithValueSources(context.Background(), snapshotValues{"MODE": "source-value"}, nil), WithValueTransform(func(string, any) (any, error) { return "", nil })},
			wantFrom: SourceEnv,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewConfigResolver(tc.options...)
			spec := ConfigSpec{Key: "MODE", Aliases: []string{"LEGACY_MODE"}, Default: "default"}
			for _, freeze := range []bool{false, true} {
				reader := r
				if freeze {
					var err error
					reader, err = r.Snapshot([]ConfigSpec{spec})
					require.NoError(t, err)
				}
				value, meta, err := Resolve[string](reader, spec)
				require.NoError(t, err)
				require.Empty(t, value)
				require.Equal(t, tc.wantFrom, meta.Source)
			}
		})
	}
}

// TestResolverNormalizedEnvironmentPreservesEmpty 验证点号键的环境名称兼容仍保留显式空值。
func TestResolverNormalizedEnvironmentPreservesEmpty(t *testing.T) {
	r := NewConfigResolver(WithEnvLookup(func(key string) (string, bool) {
		value, found := map[string]string{"APP_MODE": "", "LEGACY_MODE": "legacy"}[key]
		return value, found
	}))
	value, meta, err := Resolve[string](r, ConfigSpec{Key: "app.mode", Aliases: []string{"legacy.mode"}, Default: "default"})
	require.NoError(t, err)
	require.Empty(t, value)
	require.Equal(t, SourceEnv, meta.Source)
	require.Empty(t, meta.Alias)
}

// TestResolverRequiredEmptyDoesNotFallback 验证 required 空串报错，Secret 不读取业务或默认来源。
func TestResolverRequiredEmptyDoesNotFallback(t *testing.T) {
	for _, secret := range []bool{false, true} {
		r := NewConfigResolver(WithValueSources(context.Background(), snapshotValues{"VALUE": ""}, failingConfigSource{}))
		_, meta, err := Resolve[string](r, ConfigSpec{Key: "VALUE", Required: true, Secret: secret, Default: "fallback"})
		if secret {
			require.ErrorIs(t, err, ErrRequiredSecret)
		} else {
			require.ErrorIs(t, err, ErrRequiredConfig)
		}
		require.Equal(t, SourceEnv, meta.Source)
	}
}
