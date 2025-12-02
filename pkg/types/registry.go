package types

// RegistryProvider defines the interface for the central component registry.
// This interface is used by the MatrixEngine to access various component managers and pools,
// decoupling the engine from the concrete implementation of the registry.
type RegistryProvider interface {
	// GetRuntimePool returns the pool that manages runtime instances.
	GetRuntimePool() RuntimePool
	// GetSharedNodePool returns the pool that manages shared node instances.
	GetSharedNodePool() NodePool
	// GetNodeManager returns the manager for node component lifecycles.
	GetNodeManager() NodeManager
	// GetNodeFuncManager returns the manager for function node registrations.
	GetNodeFuncManager() NodeFuncManager
	// GetCoreObjRegistry returns the registry for business object definitions.
	GetCoreObjRegistry() CoreObjRegistry
	// GetErrorRegistry returns the registry for predefined error objects.
	GetErrorRegistry() ErrorRegistry
}

// DefaultRegistry is a global variable that holds the default registry provider.
// It must be initialized by the registry package during its init phase.
var DefaultRegistry RegistryProvider
