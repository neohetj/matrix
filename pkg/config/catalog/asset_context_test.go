package catalog

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

type readerEngine struct {
	types.MatrixEngine
	readers map[string]types.ConfigReader
}

// ConfigReader 模拟由 Engine 显式注册的模块读取器。
func (e readerEngine) ConfigReader(moduleID string) (types.ConfigReader, bool) {
	r, ok := e.readers[moduleID]
	return r, ok
}

type readerRuntime struct {
	types.Runtime
	engine types.MatrixEngine
}

// GetEngine 返回当前运行时的 Engine，不提供进程全局回退。
func (r readerRuntime) GetEngine() types.MatrixEngine { return r.engine }

type readerNodeContext struct {
	types.NodeCtx
	runtime types.Runtime
}

// GetRuntime 为测试提供实例运行时。
func (n readerNodeContext) GetRuntime() types.Runtime { return n.runtime }

// TestFromAssetContextRequiresExplicitModule 验证请求期与实例 Reader 一致并拒绝缺失绑定。
func TestFromAssetContextRequiresExplicitModule(t *testing.T) {
	r, err := NewReader(readerFixture(t), config.NewConfigResolver(config.WithValueSources(context.Background(), nil, nil)))
	require.NoError(t, err)
	ctx := asset.NewAssetContext(asset.WithNodeCtx(readerNodeContext{runtime: readerRuntime{engine: readerEngine{readers: map[string]types.ConfigReader{"sample": r}}}}))
	got, err := FromAssetContext(ctx, "sample")
	require.NoError(t, err)
	require.Same(t, r, got)
	_, err = FromAssetContext(ctx, "missing")
	require.ErrorIs(t, err, types.ErrConfigReaderUnavailable)
	_, err = FromAssetContext(nil, "sample")
	require.ErrorIs(t, err, types.ErrConfigReaderUnavailable)
	_, err = FromAssetContext(asset.NewAssetContext(), "sample")
	require.ErrorIs(t, err, types.ErrConfigReaderUnavailable)
}
