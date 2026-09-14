package catalog

import (
	"context"
	"testing"

	matrixconfig "github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestConditionalRequiredSecretRejectsEmpty 验证已有条件 Secret 契约在完整解析与冻结恢复后仍拒绝空值。
func TestConditionalRequiredSecretRejectsEmpty(t *testing.T) {
	c := loadFixture(t, fixture)
	ctx := context.Background()
	resolved, issues := c.Resolve(ctx, Sources{Env: Values{"BACKEND": "remote", "TOKEN": ""}})
	require.NotEmpty(t, issues)
	require.Contains(t, issues.Error(), "TOKEN")
	require.Equal(t, "", resolved.Values()["TOKEN"])
	resolver := matrixconfig.NewConfigResolver(matrixconfig.WithValueSources(ctx, Values{"BACKEND": "remote", "TOKEN": ""}, nil))
	reader, err := NewReader(c, resolver)
	require.NoError(t, err)
	// Reader 只校验单字段，完整条件仍由执行门禁检查。
	require.NoError(t, reader.ValidateProvided(ctx))
	_, issues = c.ResolveWithResolver(reader.resolver)
	require.NotEmpty(t, issues)
	require.Contains(t, issues.Error(), "TOKEN")
	restored, err := Restore(c.Freeze())
	require.NoError(t, err)
	require.Equal(t, c.Validate(resolved.Values()), restored.Validate(resolved.Values()))
	_, issues = restored.Resolve(ctx, Sources{Env: Values{"BACKEND": "local", "TOKEN": ""}})
	require.Empty(t, issues)
}

const conditionalTextFixture = `version: "2"
module: sample
domain: conditional_text
items:
  - {key: SELECTOR, owner: sample, type: string, description: Selector, resolution: placeholder}
  - {key: TEXT, owner: sample, type: string, description: Text, resolution: placeholder, aliases: [legacy.text]}
  - {key: FALLBACK, owner: sample, type: string, description: Fallback, resolution: placeholder}
  - {key: COUNT, owner: sample, type: int, description: Count, resolution: placeholder}
  - {key: ENABLED, owner: sample, type: bool, description: Enabled, resolution: placeholder}
schema:
`

// TestConditionalRequiredTextRejectsEmpty 验证普通 YAML alias、依赖规则和组合规则在启用时拒绝空串。
func TestConditionalRequiredTextRejectsEmpty(t *testing.T) {
	for _, rule := range []struct{ name, schema string }{
		{"then", "  if: {required: [SELECTOR]}\n  then: {required: [TEXT]}\n"},
		{"dependent_required", "  dependentRequired: {SELECTOR: [TEXT]}\n"},
		{"dependent_schema", "  dependentSchemas: {SELECTOR: {required: [TEXT]}}\n"},
		{"all_of", "  allOf: [{if: {required: [SELECTOR]}, then: {required: [TEXT]}}]\n"},
	} {
		t.Run(rule.name, func(t *testing.T) {
			c := loadFixture(t, conditionalTextFixture+rule.schema)
			_, issues := c.Resolve(context.Background(), Sources{Business: Values{"SELECTOR": "", "legacy.text": ""}})
			require.NotEmpty(t, issues)
			require.Contains(t, issues.Error(), "TEXT")
			_, issues = c.Resolve(context.Background(), Sources{Business: Values{"legacy.text": ""}})
			require.Empty(t, issues)
		})
	}
}

// TestRequiredEmptyChecksPreserveConditionalPredicates 验证 required 空值约束不会重写 if/not 或错误触发另一分支。
func TestRequiredEmptyChecksPreserveConditionalPredicates(t *testing.T) {
	c := loadFixture(t, conditionalTextFixture+`  if: {required: [SELECTOR]}
  then: {required: [TEXT]}
  else: {required: [FALLBACK]}
`)
	require.Empty(t, c.Validate(map[string]any{"SELECTOR": "", "TEXT": "set", "FALLBACK": ""}))
	issues := c.Validate(map[string]any{"SELECTOR": "", "TEXT": "", "FALLBACK": "set"})
	require.NotEmpty(t, issues)
	require.Contains(t, issues.Error(), "TEXT")
	require.Empty(t, c.Validate(map[string]any{"TEXT": "", "FALLBACK": "set"}))
	require.NotEmpty(t, c.Validate(map[string]any{"TEXT": "set", "FALLBACK": ""}))
	c = loadFixture(t, conditionalTextFixture+"  not: {required: [SELECTOR]}\n")
	require.NotEmpty(t, c.Validate(map[string]any{"SELECTOR": ""}))
	require.Empty(t, c.Validate(map[string]any{"TEXT": ""}))
	for _, keyword := range []string{"anyOf", "oneOf"} {
		c = loadFixture(t, conditionalTextFixture+"  "+keyword+": [{required: [TEXT]}, {required: [FALLBACK]}]\n")
		require.Empty(t, c.Validate(map[string]any{"TEXT": "", "FALLBACK": "set"}))
		require.NotEmpty(t, c.Validate(map[string]any{"TEXT": "", "FALLBACK": ""}))
	}
	c = loadFixture(t, conditionalTextFixture+"  dependentRequired: {SELECTOR: [TEXT, COUNT, ENABLED]}\n")
	_, issues = c.Resolve(context.Background(), Sources{Env: Values{"SELECTOR": "", "TEXT": "set", "COUNT": 0, "ENABLED": false}})
	require.Empty(t, issues)
}
