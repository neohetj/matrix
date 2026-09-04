package catalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLegacyInt64 保证旧 Catalog 的大整数、零和 false 不因归一化丢失。
func TestLegacyInt64(t *testing.T) {
	c := loadFixture(t, `version: "1"
module: sample
domain: limits
items:
  - {key: LIMIT, owner: sample, type: int64, description: limit, resolution: placeholder, default: 9223372036854775807, secret: false}
  - {key: ENABLED, owner: sample, type: bool, description: enabled, resolution: placeholder, default: true, secret: false}
`)
	resolved, issues := c.Resolve(context.Background(), Sources{})
	require.Empty(t, issues)
	require.Equal(t, int64(9223372036854775807), resolved.Values()["LIMIT"])
	resolved, issues = c.Resolve(context.Background(), Sources{Env: Values{"LIMIT": "0", "ENABLED": false}, Business: Values{"LIMIT": "42", "ENABLED": true}})
	require.Empty(t, issues)
	require.Equal(t, int64(0), resolved.Values()["LIMIT"])
	require.Equal(t, false, resolved.Values()["ENABLED"])
}

// TestFrozenNumericPrecision 覆盖真实 JSON 持久化、摘要复原和相邻大整数校验。
func TestFrozenNumericPrecision(t *testing.T) {
	c := loadFixture(t, `version: "2"
module: sample
domain: limits
items:
  - {key: LIMIT, owner: sample, type: int64, description: limit, resolution: placeholder, default: 9007199254740993, secret: false, schema: {const: 9007199254740993}}
`)
	encoded, err := json.Marshal(c.Freeze())
	require.NoError(t, err)
	var frozen Frozen
	require.NoError(t, json.Unmarshal(encoded, &frozen))
	restored, err := Restore(frozen.Clone())
	require.NoError(t, err)
	resolved, issues := restored.Resolve(context.Background(), Sources{})
	require.Empty(t, issues)
	require.Equal(t, int64(9007199254740993), resolved.Values()["LIMIT"])
	require.Empty(t, restored.ValidateProvided(map[string]any{"LIMIT": "9007199254740993"}))
	require.NotEmpty(t, restored.ValidateProvided(map[string]any{"LIMIT": "9007199254740992"}))
	require.Empty(t, restored.Validate(map[string]any{"LIMIT": int64(9007199254740993)}))
	require.NotEmpty(t, restored.Validate(map[string]any{"LIMIT": int64(9007199254740992)}))
}

func TestIntegerConversionRejectsLossyOrOverflowValues(t *testing.T) {
	item := Item{Key: "LIMIT", Type: "int64"}
	for _, raw := range []any{"9223372036854775808", uint64(9223372036854775808), 1.5, float64(9007199254740992)} {
		_, err := Convert(item, raw)
		require.Error(t, err)
	}
	value, err := Convert(item, "-9223372036854775808")
	require.NoError(t, err)
	require.Equal(t, int64(-9223372036854775808), value)
}
