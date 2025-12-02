package ops

import (
	"sync"
	"time"

	"gitlab.com/neohet/matrix/internal/log"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	MachineNodeType = "ops/machine"
)

var machineNodePrototype = &MachineNode{
	BaseNode: *types.NewBaseNode(MachineNodeType, types.NodeDefinition{
		Name:        "Machine",
		Description: "Represents a physical or virtual machine in the infrastructure.",
		Dimension:   "Operational",
		Tags:        []string{"ops", "infra", "machine"},
		Version:     "1.0.0",
		Icon:        "server",
	}),
}

func init() {
	registry.Default.NodeManager.Register(machineNodePrototype)
}

// MachineNodeConfiguration holds the static configuration for a MachineNode.
type MachineNodeConfiguration struct {
	KnowledgeBaseUUID    string `json:"knowledgeBaseUUID,omitempty"`
	Address              string `json:"address"`
	CredentialSecretName string `json:"credentialSecretName,omitempty"`
}

// MachineNodeState holds the dynamic, probed state of a machine.
type MachineNodeState struct {
	OS        string    `json:"os"`
	CPU       int       `json:"cpu"`
	MemoryGB  int       `json:"memoryGB"`
	Status    string    `json:"status"` // e.g., "discovered", "unreachable"
	LastCheck time.Time `json:"lastCheck"`
}

// MachineNode represents a machine in the infrastructure.
type MachineNode struct {
	types.Instance
	types.BaseNode
	nodeConfig MachineNodeConfiguration
	nodeState  MachineNodeState
	mu         sync.RWMutex
	logger     types.Logger
}

// New creates a new instance of the MachineNode.
func (n *MachineNode) New() types.Node {
	return &MachineNode{
		BaseNode: n.BaseNode,
		logger:   log.GetLogger(),
	}
}

// Init initializes the node with its configuration.
func (n *MachineNode) Init(config types.Config) error {
	return utils.Decode(config, &n.nodeConfig)
}

// OnMsg handles incoming messages. It can respond to PROBE or GET_STATE actions.
func (n *MachineNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// Placeholder for action-based dispatching, e.g., parsing msg.Data
	// for `{"action": "PROBE"}` or `{"action": "GET_STATE"}`.
	// For now, it acts as a non-executable node.
	ctx.TellSuccess(msg)
}

// Destroy cleans up resources used by the node.
func (n *MachineNode) Destroy() {
	// No resources to clean up in this version.
}
