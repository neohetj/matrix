package types

// NodeManager is the interface for managing components (nodes).
// It handles the registration, unregistration, and creation of component instances.
type NodeManager interface {
	// Register adds a new component to the manager.
	Register(node Node) error

	// Unregister removes a component by its type.
	Unregister(nodeType NodeType) error

	// NewNode creates a new instance of a node by its type.
	NewNode(nodeType NodeType) (Node, error)

	// Get retrieves a registered node prototype by its type.
	Get(nodeType NodeType) (Node, bool)

	// GetComponents returns a map of all registered components.
	GetComponents() map[NodeType]Node
}

// SharedNodeCtx is the context for a shared node.
type SharedNodeCtx interface {
	NodeCtx
	GetInstance() (any, error)
	GetNode() Node
}

// NodePool is the interface for a manager of shared node resources.
type NodePool interface {
	Load(dsl []byte, nodeMgr NodeManager) (NodePool, error)
	LoadFromRuleChainDef(def *RuleChainDef, nodeMgr NodeManager) (NodePool, error)
	NewFromNodeDef(def NodeDef, nodeMgr NodeManager) (SharedNodeCtx, error)
	Get(id string) (SharedNodeCtx, bool)
	GetInstance(id string) (any, error)
	Del(id string)
	Stop()
	GetAll() []NodeCtx
	// AddEndpoint adds a node instance identified as an endpoint to a dedicated internal list.
	AddEndpoint(endpoint Endpoint)
	// GetEndpoints returns all nodes that have been identified and added as endpoints.
	GetEndpoints() []Endpoint
}
