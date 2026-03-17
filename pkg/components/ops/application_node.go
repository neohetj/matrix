package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var ApplicationNodePrototype = &ApplicationNode{
	BaseNode: *types.NewBaseNode(ApplicationNodeType, types.NodeMetadata{
		Name:        "Application",
		Description: "Logical application boundary in a deployment topology.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "application"},
		Version:     "1.0.0",
		Icon:        "grid",
	}),
}

type ApplicationNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.ApplicationConfig
}

func (n *ApplicationNode) New() types.Node {
	return &ApplicationNode{BaseNode: n.BaseNode}
}

func (n *ApplicationNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, ApplicationNodeType)
}
