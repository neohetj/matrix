package pipeline

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/internal/parser"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/internal/runtime"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/require"
)

// isolatedChannelPools 为私有池和全局池装载同名资源，防止仅测试不存在的全局资源。
func isolatedChannelPools(t *testing.T) (types.NodePool, *ChannelManager, *ChannelManager) {
	t.Helper()
	old := registry.Default.SharedNodePool
	global, private := registry.NewNodePool(nil), registry.NewNodePool(nil)
	registry.Default.SharedNodePool = global
	t.Cleanup(func() { private.Stop(); global.Stop(); registry.Default.SharedNodePool = old })
	managers := []*ChannelManager{}
	for _, pool := range []types.NodePool{private, global} {
		_, err := pool.NewFromNodeDef(types.NodeDef{ID: "same-manager", Type: ChannelManagerNodeType}, registry.Default.NodeManager)
		require.NoError(t, err)
		value, err := pool.GetInstance("same-manager")
		require.NoError(t, err)
		managers = append(managers, value.(*ChannelManager))
	}
	return private, managers[0], managers[1]
}

// TestPipelineEndpointPrivatePool 验证端点通过真实节点装配获得私有资源池。
func TestPipelineEndpointPrivatePool(t *testing.T) {
	pool, local, foreign := isolatedChannelPools(t)
	shared, err := pool.NewFromNodeDef(types.NodeDef{ID: "pipeline", Type: PipelineEndpointNodeType, Configuration: types.ConfigMap{
		"channelManager": "ref://same-manager", "exposedChannels": map[string]any{"input": "input"},
	}}, registry.Default.NodeManager)
	require.NoError(t, err)
	n := shared.GetNode().(*PipelineEndpointNode)
	require.NoError(t, n.Start(context.Background()))
	t.Cleanup(func() { _ = n.Stop() })
	_, err = local.Get("pipeline", "input")
	require.NoError(t, err)
	_, err = foreign.Get("pipeline", "input")
	require.Error(t, err)
}

// TestChannelPushPrivatePool 验证消息进入所属实例的通道，不进入全局同名通道。
func TestChannelPushPrivatePool(t *testing.T) {
	pool, local, foreign := isolatedChannelPools(t)
	localCh, foreignCh := make(chan types.RuleMsg, 1), make(chan types.RuleMsg, 1)
	local.Register("pipeline", "input", localCh)
	foreign.Register("pipeline", "input", foreignCh)
	def, err := (&parser.JsonParser{}).DecodeRuleChain([]byte(`{"ruleChain":{"id":"push-flow"},"metadata":{"nodes":[{"id":"push","type":"action/channel_push","configuration":{"channelManager":"ref://same-manager","pipelineId":"pipeline","channelName":"input"}}]}}`))
	require.NoError(t, err)
	rt, err := runtime.NewDefaultRuntime(nil, def, runtime.WithNodePool(pool))
	require.NoError(t, err)
	t.Cleanup(rt.Destroy)
	node, ok := rt.GetChainInstance().GetNode("push")
	require.True(t, ok)
	ctx := utils.NewMockNodeCtx()
	node.OnMsg(ctx, types.NewMsg("test", "", nil, types.NewDataT()))
	require.NoError(t, ctx.FailureErr)
	require.Len(t, localCh, 1)
	require.Empty(t, foreignCh)
}

// TestPipelineRuntimeIsolation 验证本实例规则链查找失败时不会转到全局池。
func TestPipelineRuntimeIsolation(t *testing.T) {
	old := registry.Default.RuntimePool
	registry.Default.RuntimePool = registry.NewRuntimePool()
	t.Cleanup(func() { registry.Default.RuntimePool = old })
	foreign, local := &utils.MockRuntime{}, &utils.MockRuntime{}
	require.NoError(t, registry.Default.RuntimePool.Register("same-chain", foreign))
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
			n := &PipelineEndpointNode{runtimePool: tc.pool}
			got, ok := n.resolveStageRuntime("same-chain")
			require.Equal(t, tc.want != nil, ok)
			if tc.want == nil {
				require.Nil(t, got)
			} else {
				require.Same(t, tc.want, got)
			}
		})
	}
}
