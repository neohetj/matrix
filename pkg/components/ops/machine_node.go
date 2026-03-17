package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var MachineNodePrototype = &MachineNode{
	BaseNode: *types.NewBaseNode(MachineNodeType, types.NodeMetadata{
		Name:        "Machine",
		Description: "Physical or virtual host that backs a deployment runner.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "machine"},
		Version:     "1.0.0",
		Icon:        "machine",
	}),
}

type MachineNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.MachineConfig
}

func (n *MachineNode) New() types.Node {
	return &MachineNode{BaseNode: n.BaseNode}
}

func (n *MachineNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, MachineNodeType)
}
