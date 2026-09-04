package asset

import (
	"errors"

	"github.com/neohetj/matrix/pkg/types"
)

// FromNodeContext 从显式执行上下文构建资源解析上下文，不回退到全局资源池。
func FromNodeContext(ctx types.NodeCtx, msg types.RuleMsg) (*AssetContext, error) {
	if ctx == nil || ctx.GetRuntime() == nil {
		return nil, errors.New("node runtime unavailable for asset context")
	}
	runtime := ctx.GetRuntime()
	var pool types.NodePool
	if engine := runtime.GetEngine(); engine != nil {
		pool = engine.SharedNodePool()
	}
	if pool == nil {
		pool = runtime.GetNodePool()
	}
	if pool == nil {
		return nil, errors.New("instance node pool unavailable for asset context")
	}
	return NewAssetContext(WithNodeCtx(ctx), WithRuleMsg(msg), WithNodePool(pool)), nil
}
