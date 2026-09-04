package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLegacyFloatCatalog 保留旧 float 的双精度数值语义，拒绝非有限数。
func TestLegacyFloatCatalog(t *testing.T) {
	c, err := Load(context.Background(), Documents{{Name: "sample_catalog.yaml", Content: []byte(`version: "1"
module: sample
domain: config
items:
  - {key: RATIO, owner: sample, type: float, description: ratio, resolution: placeholder, default: "0.3"}
`)}})
	require.NoError(t, err)
	resolved, issues := c.Resolve(context.Background(), Sources{})
	require.Empty(t, issues)
	value, _ := resolved.Lookup("RATIO")
	require.Equal(t, 0.3, value)
	for _, raw := range []string{"NaN", "+Inf", "-Inf"} {
		_, issues := c.Resolve(context.Background(), Sources{Env: Values{"RATIO": raw}})
		require.NotEmpty(t, issues)
	}
}
