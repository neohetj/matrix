package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var NetworkNodePrototype = &NetworkNode{
	BaseNode: *types.NewBaseNode(NetworkNodeType, types.NodeMetadata{
		Name:        "Network",
		Description: "Logical network segment used by deployable topology nodes.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "network"},
		Version:     "1.0.0",
		Icon:        "network",
	}),
}

type NetworkNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.NetworkConfig
}

func (n *NetworkNode) New() types.Node {
	return &NetworkNode{BaseNode: n.BaseNode}
}

func (n *NetworkNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, NetworkNodeType)
}
