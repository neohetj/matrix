package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var ServiceNodePrototype = &ServiceNode{
	BaseNode: *types.NewBaseNode(ServiceNodeType, types.NodeMetadata{
		Name:        "Service",
		Description: "Deployable service node for application topology and release planning.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "service"},
		Version:     "1.0.0",
		Icon:        "service",
	}),
}

type ServiceNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.ServiceConfig
}

func (n *ServiceNode) New() types.Node {
	return &ServiceNode{BaseNode: n.BaseNode}
}

func (n *ServiceNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, ServiceNodeType)
}
