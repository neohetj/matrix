package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var RunnerNodePrototype = &RunnerNode{
	BaseNode: *types.NewBaseNode(RunnerNodeType, types.NodeMetadata{
		Name:        "Runner",
		Description: "Deployment executor target that receives release bundles.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "runner"},
		Version:     "1.0.0",
		Icon:        "runner",
	}),
}

type RunnerNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.RunnerConfig
}

func (n *RunnerNode) New() types.Node {
	return &RunnerNode{BaseNode: n.BaseNode}
}

func (n *RunnerNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, RunnerNodeType)
}
