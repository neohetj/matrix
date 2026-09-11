package loop

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/require"
)

type countingRuntime struct {
	utils.MockRuntime
	calls int
}

// ExecuteAndWait 记录循环节点实际选择的运行时。
func (r *countingRuntime) ExecuteAndWait(_ context.Context, _ string, msg types.RuleMsg, _ func(types.RuleMsg, error)) (types.RuleMsg, error) {
	r.calls++
	return msg, nil
}

// TestForEachRuntimeIsolation 覆盖本地同名链优先与本地链缺失时禁止跨实例执行。
func TestForEachRuntimeIsolation(t *testing.T) {
	oldSubMsg := types.NewSubMsg
	types.NewSubMsg = facotry.NewSubMsg
	t.Cleanup(func() { types.NewSubMsg = oldSubMsg })
	old := registry.Default.RuntimePool
	registry.Default.RuntimePool = registry.NewRuntimePool()
	t.Cleanup(func() { registry.Default.RuntimePool = old })
	foreign := &countingRuntime{}
	require.NoError(t, registry.Default.RuntimePool.Register("child", foreign))
	for _, present := range []bool{true, false} {
		t.Run(map[bool]string{true: "private_hit", false: "private_miss"}[present], func(t *testing.T) {
			private := registry.NewRuntimePool()
			local := &countingRuntime{}
			if present {
				require.NoError(t, private.Register("child", local))
			}
			parent := &utils.MockRuntime{}
			parent.On("GetEngine").Return(&utils.MockEngine{RuntimePoolValue: private})
			ctx := utils.NewMockNodeCtx()
			ctx.SetRuntime(parent)
			n := &ForEachNode{}
			require.NoError(t, n.Init(types.ConfigMap{"chainId": "child", "mode": "RANGE", "loopSource": "rulemsg://dataT/count?sid=Int64"}))
			data := facotry.NewDataT()
			count := facotry.NewCoreObj("count", facotry.NewCoreObjDef(int64(0), "Int64", "iteration count"))
			require.NoError(t, count.SetBody(int64(1)))
			data.Set("count", count)
			n.OnMsg(ctx, facotry.NewMsg("test", "", nil, data))
			if present {
				require.NoError(t, ctx.FailureErr)
				require.Equal(t, 1, local.calls)
			} else {
				require.ErrorContains(t, ctx.FailureErr, "child")
			}
			require.Zero(t, foreign.calls)
		})
	}
}
