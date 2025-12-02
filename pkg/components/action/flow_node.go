package action

import (
	"fmt"

	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	FlowNodeType = "action/flow"
)

var flowNodePrototype = &FlowNode{
	BaseNode: *types.NewBaseNode(FlowNodeType, types.NodeDefinition{
		Name:        "Flow",
		Description: "Executes a sub-chain synchronously and continues the flow upon its completion.",
		Dimension:   "Action",
		Tags:        []string{"action", "flow", "subchain"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(flowNodePrototype)
}

// FlowNodeConfiguration holds the instance-specific configuration.
type FlowNodeConfiguration struct {
	ChainId    string `json:"chainId"`
	FromNodeId string `json:"fromNodeId,omitempty"`
}

// FlowNode is a component that executes another rule chain.
type FlowNode struct {
	types.BaseNode
	types.Instance
	nodeConfig FlowNodeConfiguration
}

// New creates a new instance of FlowNode.
func (n *FlowNode) New() types.Node {
	return &FlowNode{
		BaseNode: n.BaseNode,
	}
}

// Type returns the node type.
func (n *FlowNode) Type() types.NodeType {
	return FlowNodeType
}

// Init initializes the node instance with its specific configuration.
func (n *FlowNode) Init(configuration types.Config) error {
	if err := utils.Decode(configuration, &n.nodeConfig); err != nil {
		return types.ErrInvalidConfiguration.Wrap(fmt.Errorf("failed to decode flow node config: %w", err))
	}
	if n.nodeConfig.ChainId == "" {
		return types.ErrInvalidConfiguration.Wrap(fmt.Errorf("'chainId' is not specified for node %s", n.ID()))
	}
	return nil
}

// OnMsg executes the sub-chain synchronously.
func (n *FlowNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// 1. Look up the target runtime from the global default runtime pool.
	targetRuntime, ok := registry.Default.RuntimePool.Get(n.nodeConfig.ChainId)
	if !ok {
		ctx.TellFailure(msg, types.ErrNodeNotFound.Wrap(fmt.Errorf("target chain with id '%s' not found in default runtime pool", n.nodeConfig.ChainId)))
		return
	}

	ctx.Info("Entering sub-chain synchronously", "chainId", n.nodeConfig.ChainId, "fromNodeId", n.nodeConfig.FromNodeId)

	// 2. Execute the sub-chain synchronously and wait for the result.
	// The FromNodeId from config is passed to the execution context.
	finalMsg, err := targetRuntime.ExecuteAndWait(ctx.GetContext(), n.nodeConfig.FromNodeId, msg, nil)

	// 3. Propagate the result to the parent chain.
	if err != nil {
		ctx.Error("Sub-chain execution failed", "chainId", n.nodeConfig.ChainId, "error", err)
		ctx.TellFailure(finalMsg, err)
	} else {
		ctx.Info("Sub-chain execution completed successfully", "chainId", n.nodeConfig.ChainId)
		ctx.TellSuccess(finalMsg)
	}
}
