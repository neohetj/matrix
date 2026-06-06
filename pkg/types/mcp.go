package types

import "context"

// McpEndpointNodeConfiguration defines a Matrix endpoint/mcp node.
//
// The node is business-agnostic: module repositories own the tool catalog and
// Matrix owns the MCP protocol-facing adapter contract.
type McpEndpointNodeConfiguration struct {
	ServerName   string                    `json:"serverName"`
	Instructions string                    `json:"instructions,omitempty"`
	HTTP         McpHTTPConfiguration      `json:"http,omitempty"`
	AuthContexts map[string]McpAuthContext `json:"authContexts,omitempty"`
	Tools        []McpToolDefinition       `json:"tools,omitempty"`
	ToolCatalog  string                    `json:"toolCatalog,omitempty"`
}

// McpHTTPConfiguration defines defaults for HTTP-backed MCP tool targets.
type McpHTTPConfiguration struct {
	BaseURL   string `json:"baseURL,omitempty"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
}

// McpAuthContext defines how the MCP adapter resolves trusted headers before
// calling a target. Supported modes are implemented in Matrix core, while the
// owning module still decides which auth profiles are acceptable per tool.
type McpAuthContext struct {
	Mode    string            `json:"mode"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpToolDefinition describes one MCP tool exposed by a module-owned catalog.
type McpToolDefinition struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Target      McpTargetSpec  `json:"target"`
	RiskLevel   string         `json:"riskLevel,omitempty"`
	AuthContext string         `json:"authContext,omitempty"`
	Output      McpOutputSpec  `json:"output,omitempty"`
}

// McpTargetSpec declares the existing capability path behind an MCP tool.
type McpTargetSpec struct {
	Kind   string            `json:"kind"`
	ID     string            `json:"id,omitempty"`
	Method string            `json:"method,omitempty"`
	Path   string            `json:"path,omitempty"`
	URL    string            `json:"url,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// McpOutputSpec declares lightweight output normalization hints.
type McpOutputSpec struct {
	ContentType string `json:"contentType,omitempty"`
	MaxBytes    int64  `json:"maxBytes,omitempty"`
}

// McpToolContent is a single MCP tool result content item.
type McpToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// McpToolResult is the Matrix adapter-neutral representation of a tool call
// result before JSON-RPC serialization.
type McpToolResult struct {
	Content []McpToolContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

// McpEndpoint is the runtime contract implemented by endpoint/mcp nodes.
type McpEndpoint interface {
	PassiveEndpoint
	ServerName() string
	Instructions() string
	ListTools(ctx context.Context) ([]McpToolDefinition, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (McpToolResult, error)
	Configuration() McpEndpointNodeConfiguration
}
