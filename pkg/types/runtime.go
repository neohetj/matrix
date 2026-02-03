package types

import (
	"context"
	"encoding/json"
	"io/fs"

	"github.com/neohetj/matrix/pkg/cnst"
)

// ----------- Parser -----------

// Connection defines the link between two nodes in the rule chain.
type Connection struct {
	FromID string `json:"fromId"`
	ToID   string `json:"toId"`
	Type   string `json:"type"`
}

// Relation defines a logical link between two nodes for visualization and modeling.
type Relation struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// RuleChainAttrs holds attributes about the rule chain definition.
type RuleChainAttrs struct {
	Executable bool          `json:"executable"`
	ViewType   cnst.ViewType `json:"viewType,omitempty"`
	Imports    []string      `json:"imports,omitempty"`
}

// RuleChainData holds the core data of a rule chain.
type RuleChainData struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Configuration ConfigMap      `json:"configuration,omitempty"`
	Attrs         RuleChainAttrs `json:"attrs,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface to set default values for RuleChainData.
func (r *RuleChainData) UnmarshalJSON(data []byte) error {
	// To avoid recursion, we use a temporary type that does not have the UnmarshalJSON method.
	type Alias RuleChainData

	// Set default values before unmarshaling.
	temp := &Alias{
		Attrs: RuleChainAttrs{
			Executable: true, // Default value for Executable
		},
	}

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}

	*r = RuleChainData(*temp)
	return nil
}

// MetadataData holds the metadata of a rule chain, such as nodes and connections.
type MetadataData struct {
	Nodes       []NodeDef    `json:"nodes"`
	Connections []Connection `json:"connections"`
	Relations   []Relation   `json:"relations,omitempty"`
}

// RuleChainDef represents the definition of a rule chain.
type RuleChainDef struct {
	RuleChain RuleChainData `json:"ruleChain"`
	Metadata  MetadataData  `json:"metadata"`
}

// NodeDef represents the definition of a single node in the rule chain.
type NodeDef struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Configuration ConfigMap      `json:"configuration"`
	Inputs        map[string]any `json:"inputs,omitempty"`
	Outputs       map[string]any `json:"outputs,omitempty"`
}

// Parser is the interface for parsing the rule chain definition file (DSL).
// It allows for different DSL formats (e.g., JSON, YAML) to be used.
type Parser interface {
	// DecodeRuleChain parses a rule chain structure from a byte slice.
	DecodeRuleChain(dsl []byte) (*RuleChainDef, error)

	// DecodeNode parses a single node structure from a byte slice.
	DecodeNode(dsl []byte) (*NodeDef, error)

	// EncodeRuleChain converts a rule chain structure into a byte slice.
	EncodeRuleChain(def *RuleChainDef) ([]byte, error)

	// EncodeNode converts a single node structure into a byte slice.
	EncodeNode(def *NodeDef) ([]byte, error)
}

// ----------- Instance -----------

// Scheduler is the interface for a task scheduler.
// It is responsible for managing a pool of goroutines to execute tasks asynchronously.
type Scheduler interface {
	// Submit submits a task to the scheduler for execution.
	// It returns an error if the scheduler is closed or the task cannot be accepted.
	Submit(task func()) error

	// Stop gracefully shuts down the scheduler, waiting for all active tasks to complete.
	Stop()
}

// ChainInstance holds the live state of a running rule chain execution.
type ChainInstance interface {
	// GetNode returns the node instance for the given ID.
	GetNode(id string) (Node, bool)
	// GetNodeDef returns the node definition for the given ID.
	GetNodeDef(id string) (*NodeDef, bool)
	// GetConnections returns the outgoing connections for the given node ID.
	GetConnections(fromNodeID string) []Connection
	// Definition returns the rule chain definition.
	Definition() *RuleChainDef
	// GetRootNodeIDs returns the IDs of the root nodes (nodes with no incoming connections).
	GetRootNodeIDs() []string
	// GetAllNodes returns all nodes in the chain.
	GetAllNodes() map[string]Node
	// Destroy releases resources held by the chain instance.
	Destroy()
}

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

	// GetEngine returns the engine instance associated with this runtime.
	GetEngine() MatrixEngine

	// GetChainInstance returns the chain instance for accessing initialized nodes.
	GetChainInstance() ChainInstance
}

// Source indicates the origin of a loaded resource.
type Source int

const (
	// FromUnknown indicates an unknown or unspecified source.
	FromUnknown Source = iota
	// FromEmbed indicates the resource was loaded from the embedded filesystem.
	FromEmbed
	// FromExternal indicates the resource was loaded from the external filesystem.
	FromExternal
	// FromEtcd indicates the resource was loaded from etcd. (For future use)
	FromEtcd
)

// String returns the string representation of the Source.
func (s Source) String() string {
	switch s {
	case FromEmbed:
		return "embed"
	case FromExternal:
		return "external"
	case FromEtcd:
		return "etcd"
	default:
		return "unknown"
	}
}

// Resource holds the content of a file and its source.
type Resource struct {
	Content []byte
	Source  Source
}

// ResourceProvider defines a unified interface for reading files from various sources,
// such as an embedded filesystem or the local disk. This abstraction is key to the
// hybrid loading strategy. It embeds standard library interfaces for better composability.
type ResourceProvider interface {
	fs.ReadDirFS
	fs.StatFS

	// ReadFile reads the file named by name and returns its content and source.
	// This is a custom method not part of the standard fs interfaces.
	ReadFile(name string) (*Resource, error)
	// WalkDir walks the file tree rooted at root.
	WalkDir(root string, fn fs.WalkDirFunc) error
	// Name returns the name of the provider.
	Name() string
	// Priority returns the priority of the provider.
	Priority() int
}

// MatrixEngine represents the core interface for the Matrix engine.
// It provides access to global configuration and components.
type MatrixEngine interface {
	// GetEngineConfig retrieves a value from the global business configuration.
	GetEngineConfig(key string) (any, bool)
	// RuntimePool returns the runtime pool.
	RuntimePool() RuntimePool
	// SharedNodePool returns the shared node pool.
	SharedNodePool() NodePool
	// NodeManager returns the node manager.
	NodeManager() NodeManager
	// NodeFuncManager returns the node function manager.
	NodeFuncManager() NodeFuncManager
	// BizConfig returns the global business configuration.
	BizConfig() ConfigMap
	// Loader returns the resource loader.
	Loader() ResourceProvider
	// Logger returns the logger.
	Logger() Logger
}

// TriggerSource describes the origin of a trigger that initiates a rule chain execution.
type TriggerSource struct {
	SourceChainID string // ID of the chain containing the trigger node
	NodeID        string // ID of the trigger node
	NodeType      string // Type of the node (e.g., "action/flow", "endpoint/http")
	IsEndpoint    bool   // True if it's an endpoint (self-triggering)
}

// RuntimePool holds a collection of named Runtime instances.
type RuntimePool interface {
	// Get retrieves a runtime by its ID.
	Get(id string) (Runtime, bool)
	// Register adds a new runtime to the pool.
	Register(id string, runtime Runtime) error
	// Unregister removes a runtime from the pool.
	Unregister(id string)
	// ListIDs returns a list of all registered runtime IDs.
	ListIDs() []string
	// ListByViewType returns a list of runtimes that match the given view type.
	ListByViewType(viewType string) []Runtime

	// GetTriggers returns a list of triggers that can initiate the specified chain.
	GetTriggers(chainID string) []TriggerSource
	// RegisterTrigger registers a new trigger for a target chain.
	RegisterTrigger(targetChainID string, source TriggerSource)
	// UnregisterTrigger removes a registered trigger.
	UnregisterTrigger(targetChainID string, source TriggerSource)
}
