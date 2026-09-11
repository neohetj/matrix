package catalog

import (
	"strings"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

// FromAssetContext 只从当前 Engine 获取显式模块 Reader，不重建来源或读取进程环境。
func FromAssetContext(ctx *asset.AssetContext, moduleID string) (types.ConfigReader, error) {
	if ctx == nil || ctx.NodeCtx() == nil || strings.TrimSpace(moduleID) == "" {
		return nil, types.ErrConfigReaderUnavailable
	}
	runtime := ctx.NodeCtx().GetRuntime()
	if runtime == nil || runtime.GetEngine() == nil {
		return nil, types.ErrConfigReaderUnavailable
	}
	provider, ok := runtime.GetEngine().(types.ConfigReaderProvider)
	if !ok {
		return nil, types.ErrConfigReaderUnavailable
	}
	reader, found := provider.ConfigReader(moduleID)
	if !found || reader == nil {
		return nil, types.ErrConfigReaderUnavailable
	}
	return reader, nil
}
