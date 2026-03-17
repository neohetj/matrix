package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var VolumeNodePrototype = &VolumeNode{
	BaseNode: *types.NewBaseNode(VolumeNodeType, types.NodeMetadata{
		Name:        "Volume",
		Description: "Persistent storage resource in deployment topology.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "storage"},
		Version:     "1.0.0",
		Icon:        "volume",
	}),
}

type VolumeNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.VolumeConfig
}

func (n *VolumeNode) New() types.Node {
	return &VolumeNode{BaseNode: n.BaseNode}
}

func (n *VolumeNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, VolumeNodeType)
}
