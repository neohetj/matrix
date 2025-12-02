package link

import (
	"fmt"

	"gitlab.com/neohet/matrix/internal/log"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	EndpointLinkNodeType = "flow/endpointLink"
)

var endpointLinkNodePrototype = &EndpointLinkNode{
	BaseNode: *types.NewBaseNode(EndpointLinkNodeType, types.NodeDefinition{
		Name:        "Endpoint Link",
		Description: "A virtual node used to link to another endpoint in the UI.",
		Dimension:   "Flow Control",
		Tags:        []string{"flow", "link", "endpoint"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(endpointLinkNodePrototype)
}

// EndpointLinkNodeConfiguration defines the configuration for the EndpointLinkNode.
type EndpointLinkNodeConfiguration struct {
	EndpointRef string `json:"endpointRef"`
}

// EndpointLinkNode is a virtual node that does nothing at runtime but provides a link
// to another endpoint for visualization purposes.
type EndpointLinkNode struct {
	types.Instance
	types.BaseNode
	nodeConfig EndpointLinkNodeConfiguration
	logger     types.Logger
}

// New creates a new instance of the node.
func (n *EndpointLinkNode) New() types.Node {
	return &EndpointLinkNode{
		BaseNode: n.BaseNode,
		logger:   log.GetLogger(),
	}
}

// Init initializes the node with its configuration.
func (n *EndpointLinkNode) Init(config types.Config) error {
	err := utils.Decode(config, &n.nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to decode node configuration: %w", err)
	}
	if n.nodeConfig.EndpointRef == "" {
		return fmt.Errorf("endpointRef is required")
	}
	return nil
}

// OnMsg implements the core logic of the node. For this node, it's a no-op.
func (n *EndpointLinkNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// This node is for visualization only, so we just pass the message through.
	n.logger.Debugf(ctx.GetContext(), "EndpointLinkNode is a virtual node and performs no action.")
	ctx.TellSuccess(msg)
}

// Destroy cleans up resources used by the node.
func (n *EndpointLinkNode) Destroy() {
	// No resources to clean up.
}
