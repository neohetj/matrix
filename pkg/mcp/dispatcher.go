package mcp

import (
	"context"

	"github.com/neohetj/matrix/pkg/types"
)

// ResolvedAuthContext is the normalized, provider-neutral auth payload that
// the MCP adapter hands to module-local target dispatchers.
type ResolvedAuthContext struct {
	Name    string
	Mode    string
	Headers map[string]string
}

// DispatchRequest is the module-facing request contract for runtime-bound MCP
// targets such as handler and rulechain.
type DispatchRequest struct {
	Tool           types.McpToolDefinition
	Arguments      map[string]any
	AuthContext    ResolvedAuthContext
	EndpointConfig types.McpEndpointNodeConfiguration
}

// TargetDispatcher handles module-local target kinds that Matrix should not
// implement directly.
type TargetDispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) (types.McpToolResult, bool, error)
}

// TargetDispatcherFunc adapts a function to TargetDispatcher.
type TargetDispatcherFunc func(ctx context.Context, req DispatchRequest) (types.McpToolResult, bool, error)

func (f TargetDispatcherFunc) Dispatch(ctx context.Context, req DispatchRequest) (types.McpToolResult, bool, error) {
	return f(ctx, req)
}

// WithTargetDispatcher injects a module-local dispatcher for runtime-bound
// target kinds.
func WithTargetDispatcher(dispatcher TargetDispatcher) Option {
	return func(e *Endpoint) {
		if dispatcher != nil {
			e.dispatcher = dispatcher
		}
	}
}
