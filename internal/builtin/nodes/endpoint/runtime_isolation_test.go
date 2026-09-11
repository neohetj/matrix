package endpoint

import (
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/require"
)

// TestRedisStreamRuntimeIsolation 验证已注入的池缺失规则链时不会借用全局同名链。
func TestRedisStreamRuntimeIsolation(t *testing.T) {
	global := registry.Default.RuntimePool
	registry.Default.RuntimePool = registry.NewRuntimePool()
	t.Cleanup(func() { registry.Default.RuntimePool = global })
	foreign := &utils.MockRuntime{}
	require.NoError(t, registry.Default.RuntimePool.Register("same-chain", foreign))
	local := &utils.MockRuntime{}
	private := registry.NewRuntimePool()
	require.NoError(t, private.Register("same-chain", local))
	for _, tc := range []struct {
		name string
		pool types.RuntimePool
		want types.Runtime
	}{
		{"private_hit", private, local},
		{"private_miss", registry.NewRuntimePool(), nil},
		{"legacy_without_injection", nil, foreign},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := &RedisStreamEndpointNode{config: RedisStreamEndpointConfiguration{RuleChainID: "same-chain"}, runtimePool: tc.pool}
			got, ok := n.resolveRuntime()
			require.Equal(t, tc.want != nil, ok)
			if tc.want == nil {
				require.Nil(t, got)
			} else {
				require.Same(t, tc.want, got)
			}
		})
	}
}
