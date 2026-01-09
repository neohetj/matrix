package loop

import (
	"fmt"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

func init() {
	registry.Default.NodeManager.Register(breakNodePrototype)
}

const (
	BreakNodeType    = "action/break"
	MetadataKeyBreak = "break_loop"
)

var breakNodePrototype = &BreakNode{
	BaseNode: *types.NewBaseNode(BreakNodeType, types.NodeMetadata{
		Name:        "Break Loop",
		Description: "Breaks the current forEach loop by setting a break flag in the metadata.",
		Dimension:   "Action",
		Tags:        []string{"action", "loop", "break"},
		Version:     "1.0.0",
		NodeWrites: []types.ContractDef{
			{URI: asset.MetadataURI(MetadataKeyBreak), Description: "Sets the break flag to true to stop the loop."},
		},
	}),
}

// BreakNode is a component that sets a break flag in metadata.
type BreakNode struct {
	types.BaseNode
	types.Instance
}

// New creates a new instance of BreakNode.
func (n *BreakNode) New() types.Node {
	return &BreakNode{
		BaseNode: n.BaseNode,
	}
}

// Type returns the node type.
func (n *BreakNode) Type() types.NodeType {
	return BreakNodeType
}

// Init initializes the node instance.
func (n *BreakNode) Init(configuration types.ConfigMap) error {
	return nil
}

// OnMsg executes the node.
func (n *BreakNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	msg.Metadata()[fmt.Sprintf("%s_%s", MetadataKeyBreak, ctx.ChainID())] = "true"
	ctx.TellSuccess(msg)
}
