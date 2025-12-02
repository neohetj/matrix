package types

import (
	"context"
)

// Runtime is the interface for the rule chain execution engine.
// It takes a parsed rule chain definition and a message, and orchestrates the execution flow.
type Runtime interface {
	// Execute runs the rule chain with the given message, starting from a specific node.
	Execute(ctx context.Context, fromNodeID string, msg RuleMsg, onEnd func(msg RuleMsg, err error)) error

	// ExecuteAndWait runs the rule chain synchronously and waits for its completion.
	// It accepts an optional onEnd callback that is executed before the function returns.
	ExecuteAndWait(ctx context.Context, fromNodeID string, msg RuleMsg, onEnd func(msg RuleMsg, err error)) (RuleMsg, error)

	// Reload atomically replaces the underlying rule chain instance with a new one.
	Reload(newChainDef *RuleChainDef) error

	// Destroy releases all resources held by the runtime and its nodes.
	Destroy()

	// Definition returns the underlying rule chain definition that this runtime instance is executing.
	Definition() *RuleChainDef

	// GetNodePool returns the node pool associated with this runtime.
	GetNodePool() NodePool
}

// RuntimePool holds a collection of named Runtime instances.
type RuntimePool interface {
	Get(id string) (Runtime, bool)
	Register(id string, runtime Runtime) error
	Unregister(id string)
	ListIDs() []string
	ListByViewType(viewType string) []Runtime
}
