package asset_test

import (
	"testing"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

type contextPoolEngine struct {
	types.MatrixEngine
	pool types.NodePool
}

func (e contextPoolEngine) SharedNodePool() types.NodePool { return e.pool }

// TestFromNodeContext 验证只选择显式运行态资源池，不读取进程全局资源池。
func TestFromNodeContext(t *testing.T) {
	enginePool, runtimePool := &MockNodePool{}, &MockNodePool{}
	msg := &MockRuleMsg{}
	for _, tc := range []struct {
		name string
		node types.NodeCtx
		want types.NodePool
	}{
		{"engine", &MockNodeCtx{runtime: &MockRuntime{engine: contextPoolEngine{pool: enginePool}, pool: runtimePool}}, enginePool},
		{"runtime", &MockNodeCtx{runtime: &MockRuntime{pool: runtimePool}}, runtimePool},
		{"empty-engine", &MockNodeCtx{runtime: &MockRuntime{engine: contextPoolEngine{}, pool: runtimePool}}, runtimePool},
		{"nil-context", nil, nil},
		{"nil-runtime", &MockNodeCtx{}, nil},
		{"nil-pool", &MockNodeCtx{runtime: &MockRuntime{}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ac, err := asset.FromNodeContext(tc.node, msg)
			if tc.want == nil {
				if err == nil || ac != nil {
					t.Fatal("missing instance pool must fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if asset.GetNodePool(ac) != tc.want || ac.NodeCtx() != tc.node || ac.RuleMsg() != msg {
				t.Fatal("context lost its explicit instance binding")
			}
		})
	}
}
