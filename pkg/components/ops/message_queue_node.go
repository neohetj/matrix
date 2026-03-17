package ops

import (
	"github.com/neohetj/matrix/pkg/ops/model"
	"github.com/neohetj/matrix/pkg/types"
)

var MessageQueueNodePrototype = &MessageQueueNode{
	BaseNode: *types.NewBaseNode(MessageQueueNodeType, types.NodeMetadata{
		Name:        "Message Queue",
		Description: "Messaging infrastructure node in deployment topology.",
		Dimension:   "Operations",
		Tags:        []string{"ops", "topology", "mq"},
		Version:     "1.0.0",
		Icon:        "message-queue",
	}),
}

type MessageQueueNode struct {
	types.BaseNode
	types.Instance
	nodeConfig model.MessageQueueConfig
}

func (n *MessageQueueNode) New() types.Node {
	return &MessageQueueNode{BaseNode: n.BaseNode}
}

func (n *MessageQueueNode) Init(cfg types.ConfigMap) error {
	return decodeNodeConfig(cfg, &n.nodeConfig, MessageQueueNodeType)
}
