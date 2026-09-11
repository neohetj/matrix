package matrix

import (
	"fmt"
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/internal/runtime"
	"github.com/neohetj/matrix/pkg/types"
)

type poolFixtureNode struct {
	*types.BaseNode
	types.Instance
	pool types.NodePool
}

func (n *poolFixtureNode) New() types.Node {
	return &poolFixtureNode{BaseNode: types.NewBaseNode(n.Type(), types.NodeMetadata{})}
}
func (n *poolFixtureNode) SetNodePool(pool types.NodePool) { n.pool = pool }
func (n *poolFixtureNode) Init(types.ConfigMap) error {
	if n.pool == nil {
		return fmt.Errorf("node pool missing before Init")
	}
	return nil
}
func (n *poolFixtureNode) GetInstance() (any, error) { return n.pool, nil }

// TestNodePoolInjectedBeforeInit 验证 shared 和 runtime 节点在 Init 前接收所属实例资源池。
func TestNodePoolInjectedBeforeInit(t *testing.T) {
	reg := registry.NewRegistry()
	_ = reg.NodeManager.Register(&poolFixtureNode{BaseNode: types.NewBaseNode("test/pool-aware", types.NodeMetadata{})})
	shared, err := reg.SharedNodePool.NewFromNodeDef(types.NodeDef{ID: "shared", Type: "test/pool-aware"}, reg.NodeManager)
	if err != nil {
		t.Fatal(err)
	}
	if shared.GetNode().(*poolFixtureNode).pool != reg.SharedNodePool {
		t.Fatal("shared pool not bound")
	}
	e := &MatrixEngine{registry: reg}
	def := &types.RuleChainDef{}
	def.Metadata.Nodes = []types.NodeDef{{ID: "local", Type: "test/pool-aware"}}
	rt, err := runtime.NewDefaultRuntime(nil, def, runtime.WithEngine(e))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Destroy)
	node, _ := rt.GetChainInstance().GetNode("local")
	if node.(*poolFixtureNode).pool != reg.SharedNodePool {
		t.Fatal("runtime pool not bound")
	}
}
