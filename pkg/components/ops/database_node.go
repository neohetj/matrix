package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var DatabaseNodePrototype = &DatabaseNode{
	BaseNode: *types.NewBaseNode(DatabaseNodeType, types.NodeMetadata{
		Name:        "Database",
		Description: "Stateful database dependency in deployment topology.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "database"},
		Version:     "1.0.0",
		Icon:        "database",
	}),
}

type DatabaseNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.DatabaseConfig
}

func (n *DatabaseNode) New() types.Node {
	return &DatabaseNode{BaseNode: n.BaseNode}
}

func (n *DatabaseNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, DatabaseNodeType)
}
