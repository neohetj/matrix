package catalog

import (
	"context"
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestDecoderMissingAndInvalid 验证批量业务装配只对缺失使用 fallback，错误必须统一返回。
func TestDecoderMissingAndInvalid(t *testing.T) {
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), Values{"ENABLED": false, "COUNT": 0}, nil)))
	require.NoError(t, err)
	d := NewDecoder(context.Background(), r)
	require.False(t, d.Bool("ENABLED", true))
	require.Zero(t, d.Int("COUNT", 99))
	require.Zero(t, d.Int64("COUNT", 99))
	require.Equal(t, time.Second, d.Duration("PERIOD", time.Millisecond, time.Second))
	require.Equal(t, "fallback", d.String("TEXT", "business-policy"))
	require.NoError(t, d.Err())
	require.Empty(t, d.String("REQUIRED", "must-not-bypass-required"))
	require.Error(t, d.Err())
	d = NewDecoder(context.Background(), r)
	require.False(t, d.Bool("TEXT", true))
	require.Error(t, d.Err())
	require.NotContains(t, d.Err().Error(), "fallback")
}
