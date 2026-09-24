package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

// readerFixture 构造自包含定义，不读取业务仓库或真实环境。
func readerFixture(t *testing.T) *Catalog {
	t.Helper()
	c, err := Load(context.Background(), Documents{{Name: "sample_catalog.yaml", Content: []byte(`version: "2"
module: sample
domain: settings
items:
  - {key: TEXT, owner: sample, type: string, description: text, resolution: placeholder, aliases: [OLD_TEXT], default: fallback}
  - {key: ENABLED, owner: sample, type: bool, description: switch, resolution: placeholder, default: true}
  - {key: COUNT, owner: sample, type: int64, description: count, resolution: placeholder, default: 9}
  - {key: SECRET, owner: sample, type: secret, secret: true, description: credential, resolution: placeholder}
  - {key: REQUIRED, owner: sample, type: string, description: required, required: true, resolution: placeholder}
  - {key: PERIOD, owner: sample, type: duration, description: period, resolution: placeholder}
  - {key: CHOICE, owner: sample, type: string, description: choice, resolution: placeholder, schema: {enum: [a, b]}}
schema:
  if: {properties: {ENABLED: {const: true}}, required: [ENABLED]}
  then: {required: [REQUIRED]}
`)}})
	require.NoError(t, err)
	return c
}

// TestReaderFrozenPriority 验证来源顺序、别名及构造后输入不可变。
func TestReaderFrozenPriority(t *testing.T) {
	env := Values{"OLD_TEXT": "env", "ENABLED": false, "COUNT": "9223372036854775807"}
	business := Values{"TEXT": "business", "SECRET": "must-not-be-read"}
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), env, business)))
	require.NoError(t, err)
	env["OLD_TEXT"] = "mutated"
	text, found, err := Read[string](context.Background(), r, "TEXT")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "env", text)
	enabled, found, err := Read[bool](context.Background(), r, "ENABLED")
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, enabled)
	count, _, err := Read[int64](context.Background(), r, "COUNT")
	require.NoError(t, err)
	require.Equal(t, int64(math.MaxInt64), count)
	_, found, err = Read[string](context.Background(), r, "SECRET")
	require.NoError(t, err)
	require.False(t, found)
}

// TestReaderExplicitPresence 验证节点 false/0 优先且不得注入节点明文 Secret。
func TestReaderExplicitPresence(t *testing.T) {
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), nil, nil)))
	require.NoError(t, err)
	value, found, err := ReadNode[bool](context.Background(), r, "ENABLED", types.ConfigOverride{Present: true, Value: false})
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, value)
	n, _, err := ReadNode[int64](context.Background(), r, "COUNT", types.ConfigOverride{Present: true, Value: 0})
	require.NoError(t, err)
	require.Zero(t, n)
	_, _, err = ReadNode[string](context.Background(), r, "SECRET", types.ConfigOverride{Present: true, Value: "node-secret"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "node-secret")
}

// TestReaderFieldValidationOnly 验证单字段校验不误触发其他字段的全局必填。
func TestReaderFieldValidationOnly(t *testing.T) {
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), Values{"CHOICE": "invalid-secret"}, nil)))
	require.NoError(t, err)
	_, _, err = Read[bool](context.Background(), r, "ENABLED")
	require.NoError(t, err)
	_, _, err = Read[string](context.Background(), r, "REQUIRED")
	require.Error(t, err)
	_, _, err = Read[string](context.Background(), r, "UNKNOWN")
	require.ErrorContains(t, err, "unknown_key")
	_, _, err = Read[string](context.Background(), r, "CHOICE")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "invalid-secret")
}

// TestReaderDurationUnits 验证显式裸数单位、零值与溢出拒绝。
func TestReaderDurationUnits(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want time.Duration
		bad  bool
	}{
		{"3", 3 * time.Millisecond, false}, {"0", 0, false}, {"0s", 0, false},
		{"2s", 2 * time.Second, false}, {"9223372036854775807", 0, true}, {"secret-invalid", 0, true},
	} {
		r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), Values{"PERIOD": tc.raw}, nil)))
		require.NoError(t, err)
		value, found, err := ReadDuration(context.Background(), r, "PERIOD", time.Millisecond)
		if tc.bad {
			require.Error(t, err)
			require.NotContains(t, err.Error(), tc.raw)
			continue
		}
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, tc.want, value)
	}
}

// TestReaderIntegerDuration 验证整数类型配置能够按显式单位读取。
func TestReaderIntegerDuration(t *testing.T) {
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), nil, nil)))
	require.NoError(t, err)
	value, found, err := ReadDuration(context.Background(), r, "COUNT", time.Millisecond)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 9*time.Millisecond, value)
}

type failingReaderSource struct{}

// Lookup 模拟来源错误包含敏感值，调用方只能收到安全错误。
func (failingReaderSource) Lookup(context.Context, string) (any, bool, error) {
	return nil, false, errors.New("source-secret")
}

// TestReaderSafeProjection 验证来源错误和 JSON/String 投影均不暴露 Secret。
func TestReaderSafeProjection(t *testing.T) {
	_, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), failingReaderSource{}, nil)))
	require.Error(t, err)
	require.NotContains(t, err.Error(), "source-secret")
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), Values{"SECRET": "value-secret"}, nil)))
	require.NoError(t, err)
	raw, err := json.Marshal(r)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(raw), "value-secret"))
	require.NotContains(t, r.String(), "value-secret")
}
