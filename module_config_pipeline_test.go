package matrix

import (
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/stretchr/testify/require"
)

// TestModuleConfigPrivatePipelineStarts 覆盖公共 Engine 构造路径，资源仅存在于所属实例。
func TestModuleConfigPrivatePipelineStarts(t *testing.T) {
	cfg := lifecycleConfig(t, map[string]string{
		"shared/channels.json":    `{"ruleChain":{"id":"resources"},"metadata":{"nodes":[{"id":"private-pipeline-manager","type":"resource/channel_manager"}]}}`,
		"endpoints/pipeline.json": `{"id":"private-pipeline","type":"endpoint/pipeline","configuration":{"channelManager":"ref://private-pipeline-manager","exposedChannels":{"input":"input"}}}`,
	})
	e, err := New(cfg, WithModuleConfig("sample", fixtureReader("value")))
	require.NoError(t, err)
	t.Cleanup(func() { _ = e.StopActiveEndpoints(); e.SharedNodePool().Stop() })
	_, ok := e.SharedNodePool().Get("private-pipeline-manager")
	require.True(t, ok)
	_, ok = registry.Default.SharedNodePool.Get("private-pipeline-manager")
	require.False(t, ok)
}
