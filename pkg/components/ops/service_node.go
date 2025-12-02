package ops

import (
	"sync"

	"gitlab.com/neohet/matrix/internal/log"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	ServiceNodeType = "ops/service"
)

var serviceNodePrototype = &ServiceNode{
	BaseNode: *types.NewBaseNode(ServiceNodeType, types.NodeDefinition{
		Name:        "Service",
		Description: "Represents a technical service or microservice.",
		Dimension:   "Operational",
		Tags:        []string{"ops", "service"},
		Version:     "1.0.0",
		Icon:        "cog",
	}),
}

func init() {
	registry.Default.NodeManager.Register(serviceNodePrototype)
}

// ServiceEndpoint describes a single API endpoint exposed by a service.
type ServiceEndpoint struct {
	EndpointRef string `json:"endpointRef,omitempty"`
	Type        string `json:"type"`   // http, grpc
	Path        string `json:"path"`   // e.g., /v1/users/login
	Method      string `json:"method"` // e.g., POST
}

// ServiceNodeConfiguration holds the static configuration for a ServiceNode.
type ServiceNodeConfiguration struct {
	KnowledgeBaseUUID string            `json:"knowledgeBaseUUID,omitempty"`
	Ports             []int             `json:"ports,omitempty"`
	RuleChainRefs     []string          `json:"ruleChainRefs,omitempty"`
	Endpoints         []ServiceEndpoint `json:"endpoints,omitempty"`
}

// ServiceNode represents a technical service in the infrastructure.
type ServiceNode struct {
	types.Instance
	types.BaseNode
	nodeConfig ServiceNodeConfiguration
	mu         sync.RWMutex
	logger     types.Logger
}

// New creates a new instance of the ServiceNode.
func (n *ServiceNode) New() types.Node {
	return &ServiceNode{
		BaseNode: n.BaseNode,
		logger:   log.GetLogger(),
	}
}

// Init initializes the node with its configuration.
func (n *ServiceNode) Init(config types.Config) error {
	return utils.Decode(config, &n.nodeConfig)
}

// OnMsg handles incoming messages.
func (n *ServiceNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// Placeholder for future logic, e.g., routing to an associated rule chain.
	ctx.TellSuccess(msg)
}

// Destroy cleans up resources used by the node.
func (n *ServiceNode) Destroy() {
	// No resources to clean up in this version.
}
