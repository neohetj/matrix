package catalog

import (
	"context"
	"testing"

	matrixconfig "github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

const emptyValueFixture = `version: "2"
module: sample
domain: empty_values
items:
  - {key: MODE, owner: sample, type: string, description: Mode, resolution: placeholder, required: true, default: local, aliases: [legacy.mode], schema: {enum: [local, remote]}}
  - {key: REQUIRED_TEXT, owner: sample, type: string, description: Required text, resolution: placeholder, required: true, default: default-text}
  - {key: OPTIONAL_TEXT, owner: sample, type: string, description: Optional text, resolution: placeholder, default: default-text}
  - {key: TOKEN, owner: sample, type: secret, description: Token, resolution: placeholder, secret: true, schema: {pattern: '\S'}}
`

// TestCatalogResolveRejectsExplicitEmptyRuntimeFields 验证完整运行解析不会把空输入变为默认或低优先级来源。
func TestCatalogResolveRejectsExplicitEmptyRuntimeFields(t *testing.T) {
	c := loadFixture(t, emptyValueFixture)
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		sources Sources
		badKey  string
	}{
		{"environment_before_business", Sources{Env: Values{"MODE": ""}, Business: Values{"MODE": "remote"}}, "MODE"},
		{"business_alias_before_default", Sources{Business: Values{"legacy.mode": ""}}, "MODE"},
		{"required_text_without_schema", Sources{Env: Values{"REQUIRED_TEXT": ""}}, "REQUIRED_TEXT"},
		{"secret_empty_does_not_read_business", Sources{Env: Values{"TOKEN": ""}, Business: Values{"TOKEN": "forbidden-secret"}}, "TOKEN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolved, issues := c.Resolve(ctx, tc.sources)
			require.NotEmpty(t, issues)
			require.Contains(t, issues.Error(), tc.badKey)
			require.Equal(t, "", resolved.Values()[tc.badKey])
			require.NotContains(t, issues.Error(), "forbidden-secret")
		})
	}
	resolved, issues := c.Resolve(ctx, Sources{Env: Values{"OPTIONAL_TEXT": ""}})
	require.Empty(t, issues)
	require.Equal(t, "local", resolved.Values()["MODE"])
	require.Equal(t, "default-text", resolved.Values()["REQUIRED_TEXT"])
	require.Equal(t, "", resolved.Values()["OPTIONAL_TEXT"])
	require.NotContains(t, resolved.Values(), "TOKEN")
}

// TestReaderRetainsEmptyRuntimeSources 验证宿主来源、转换和冻结 Reader 都将空串交给字段校验。
func TestReaderRetainsEmptyRuntimeSources(t *testing.T) {
	c := loadFixture(t, emptyValueFixture)
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		options []matrixconfig.ResolverOption
	}{
		{"named_source", []matrixconfig.ResolverOption{matrixconfig.WithValueSources(ctx, Values{"MODE": ""}, nil)}},
		{"environment", []matrixconfig.ResolverOption{matrixconfig.WithEnvLookup(func(key string) (string, bool) { return "", key == "MODE" })}},
		{"yaml_alias", []matrixconfig.ResolverOption{matrixconfig.WithBusinessConfig(types.ConfigMap{"legacy": map[string]any{"mode": ""}}), matrixconfig.WithEnvLookup(func(string) (string, bool) { return "", false })}},
		{"transform", []matrixconfig.ResolverOption{matrixconfig.WithValueSources(ctx, Values{"MODE": "opaque"}, nil), matrixconfig.WithValueTransform(func(string, any) (any, error) { return "", nil })}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewReader(c, matrixconfig.NewConfigResolver(tc.options...))
			require.NoError(t, err)
			require.ErrorContains(t, reader.ValidateProvided(ctx), "MODE")
			_, _, err = Read[string](ctx, reader, "MODE")
			require.ErrorContains(t, err, "MODE")
		})
	}
	env := Values{"MODE": "", "REQUIRED_TEXT": ""}
	reader, err := NewReader(c, matrixconfig.NewConfigResolver(matrixconfig.WithValueSources(ctx, env, nil)))
	require.NoError(t, err)
	env["MODE"] = "local"
	env["REQUIRED_TEXT"] = "later-value"
	for _, key := range []string{"MODE", "REQUIRED_TEXT"} {
		_, _, err := Read[string](ctx, reader, key)
		require.ErrorContains(t, err, key)
	}
	// 草稿可保留尚未完成的输入；运行快照和完整执行校验始终严格。
	require.Empty(t, c.ValidateProvided(map[string]any{"MODE": ""}))
	require.NotEmpty(t, c.Validate(map[string]any{"MODE": "", "REQUIRED_TEXT": ""}))
}
