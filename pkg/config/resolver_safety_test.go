package config

import (
	"context"
	"errors"
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/stretchr/testify/require"
)

type failingConfigSource struct{}

// Lookup 模拟含凭据的来源错误。
func (failingConfigSource) Lookup(context.Context, string) (any, bool, error) {
	return nil, false, errors.New("driver-secret")
}

// TestResolverErrorsDoNotEchoValues 验证普通读取入口也不会输出转换值或来源错误。
func TestResolverErrorsDoNotEchoValues(t *testing.T) {
	r := NewConfigResolver(WithValueSources(context.Background(), failingConfigSource{}, nil))
	_, _, err := Resolve[string](r, ConfigSpec{Key: "TOKEN", Secret: true})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "driver-secret")
	r = NewConfigResolver(WithEnvLookup(func(string) (string, bool) { return "conversion-secret", true }))
	_, _, err = Resolve[int64](r, ConfigSpec{Key: "NUMBER", Type: cnst.INT64})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "conversion-secret")
}
