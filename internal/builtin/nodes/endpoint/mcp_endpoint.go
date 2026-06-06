package endpoint

import (
	"context"
	"encoding/json"

	"github.com/neohetj/matrix/internal/registry"
	matrixmcp "github.com/neohetj/matrix/pkg/mcp"
	"github.com/neohetj/matrix/pkg/types"
)

const McpEndpointNodeType = "endpoint/mcp"

var mcpEndpointNodePrototype = &McpEndpointNode{
	BaseNode: *types.NewBaseNode(McpEndpointNodeType, types.NodeMetadata{
		Name:        "MCP Endpoint",
		Description: "Exposes module-owned business tools through a Model Context Protocol endpoint.",
		Dimension:   "Endpoint",
		Tags:        []string{"endpoint", "mcp", "tool"},
		Version:     "0.1.0",
	}),
}

func init() {
	registry.Default.GetNodeManager().Register(mcpEndpointNodePrototype)
}

// McpEndpointNode adapts a module-owned MCP tool catalog to Matrix endpoint
// discovery. Transport hosting remains outside Matrix.
type McpEndpointNode struct {
	types.BaseNode
	types.Instance
	nodeConfig types.McpEndpointNodeConfiguration
	endpoint   *matrixmcp.Endpoint
}

func (n *McpEndpointNode) New() types.Node {
	return &McpEndpointNode{BaseNode: n.BaseNode}
}

func (n *McpEndpointNode) Init(config types.ConfigMap) error {
	var nodeConfig types.McpEndpointNodeConfiguration
	data, err := json.Marshal(config)
	if err != nil {
		return types.InvalidConfiguration.Wrap(err)
	}
	if err := json.Unmarshal(data, &nodeConfig); err != nil {
		return types.InvalidConfiguration.Wrap(err)
	}
	endpoint, err := matrixmcp.NewEndpoint(nodeConfig)
	if err != nil {
		return types.InvalidConfiguration.Wrap(err)
	}
	n.nodeConfig = nodeConfig
	n.endpoint = endpoint
	return nil
}

func (n *McpEndpointNode) SetRuntimePool(pool any) error {
	return nil
}

func (n *McpEndpointNode) GetInstance() (any, error) {
	return n, nil
}

func (n *McpEndpointNode) ServerName() string {
	if n.endpoint == nil {
		return ""
	}
	return n.endpoint.ServerName()
}

func (n *McpEndpointNode) Instructions() string {
	if n.endpoint == nil {
		return ""
	}
	return n.endpoint.Instructions()
}

func (n *McpEndpointNode) ListTools(ctx context.Context) ([]types.McpToolDefinition, error) {
	if n.endpoint == nil {
		return nil, types.InvalidConfiguration
	}
	return n.endpoint.ListTools(ctx)
}

func (n *McpEndpointNode) CallTool(ctx context.Context, name string, arguments map[string]any) (types.McpToolResult, error) {
	if n.endpoint == nil {
		return types.McpToolResult{}, types.InvalidConfiguration
	}
	return n.endpoint.CallTool(ctx, name, arguments)
}

func (n *McpEndpointNode) Configuration() types.McpEndpointNodeConfiguration {
	return n.nodeConfig
}
