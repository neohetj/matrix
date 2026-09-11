package catalog

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestRenderString 验证共享节点模板通过实例 Reader 获取相同的已校验值。
func TestRenderString(t *testing.T) {
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), Values{"COUNT": "0", "ENABLED": false}, nil)))
	require.NoError(t, err)
	value, err := RenderString(context.Background(), r, "${config:///TEXT?scope=env}/${config:///COUNT}/${config:///ENABLED}")
	require.NoError(t, err)
	require.Equal(t, "fallback/0/false", value)
	value, err = RenderString(context.Background(), r, "literal")
	require.NoError(t, err)
	require.Equal(t, "literal", value)
	value, err = RenderString(context.Background(), r, "${config:///PERIOD?default=2s}")
	require.NoError(t, err)
	require.Equal(t, "2s", value)
	for _, expression := range []string{"${config:///UNKNOWN}", "${config:///SECRET?default=secret-canary}", "${config:///TEXT?scope=node}", "${config:///TEXT?bad=secret-canary}", "${config:///TEXT?scope=env&scope=node}", "${config:///TEXT", "${rulemsg://secret-canary}"} {
		value, err := RenderString(context.Background(), r, expression)
		require.Error(t, err, expression)
		require.Empty(t, value)
		require.NotContains(t, err.Error(), "secret-canary")
	}
}
