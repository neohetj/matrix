package config

import (
	"errors"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

// TestValueTransformSnapshot 验证业务语法适配仅处理一次有效来源，错误不泄露原值。
func TestValueTransformSnapshot(t *testing.T) {
	calls := 0
	r := NewConfigResolver(WithBusinessConfig(types.ConfigMap{"FLAG": "business"}), WithEnvLookup(func(key string) (string, bool) { return "yes", true }), WithValueTransform(func(key string, raw any) (any, error) {
		calls++
		return ParseBool(raw.(string))
	}))
	spec := ConfigSpec{Key: "FLAG", Resolution: ResolutionPlaceholder}
	frozen, err := r.Snapshot([]ConfigSpec{spec})
	require.NoError(t, err)
	for range 2 {
		value, _, err := Resolve[bool](frozen, spec)
		require.NoError(t, err)
		require.True(t, value)
	}
	require.Equal(t, 1, calls)
	r = NewConfigResolver(WithEnvLookup(func(string) (string, bool) { return "secret-value", true }), WithValueTransform(func(string, any) (any, error) { return nil, errors.New("secret-value") }))
	_, _, err = Resolve[any](r, spec)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-value")
}
