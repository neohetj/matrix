# Guidelines and Best Practices for Generic Node Implementation

This document summarizes the guidelines and best practices for developing generic nodes within the Matrix framework.

## 1. Directory Structure

*   **Grouping**: Nodes should be categorized into appropriate subdirectories under `internal/builtin/nodes` (e.g., `endpoint`, `loop`, `pipeline`, `action`).
*   **Independent Packages**: Each major component category should be a separate Go package.

## 2. Node Definition and Interfaces

### 2.1 Basic Interface
All nodes must implement the `types.Node` interface. Even for `Endpoint` type nodes, it is recommended to implement the `OnMsg` method to maintain compatibility with regular flows (e.g., being invoked as a standard node in a chain).

```go
type MyNode struct {
    types.BaseNode
    types.Instance
    // ...
}

// Enforce compile-time interface check
var _ types.Node = (*MyNode)(nil)

func (n *MyNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
    // Implementation logic
    ctx.TellSuccess(msg)
}
```

### 2.2 Endpoint Nodes
*   Must implement the `types.Endpoint` interface.
*   If active (e.g., listening on a port, running background tasks), must implement the `types.ActiveEndpoint` interface (including `Start` and `Stop` methods).
*   Add compile-time check: `var _ types.ActiveEndpoint = (*MyEndpoint)(nil)`.

### 2.3 Sub-Chain Trigger Nodes
*   Nodes that trigger a sub-chain execution must implement the `types.SubChainTrigger` interface.
*   Implement `GetInputMapping`, `GetOutputMapping`, and `GetTargetChainID` methods.
*   Add compile-time check: `var _ types.SubChainTrigger = (*MyTriggerNode)(nil)`. Since `SubChainTrigger` embeds `Node`, this also covers the node interface check.

## 3. Configuration Management

### 3.1 Static Configuration
*   **Define Struct**: Define a configuration struct for each node (e.g., `MyNodeConfiguration`).
*   **Init Initialization**: Use `utils.Decode` in the `Init` method to decode `types.ConfigMap` into the struct.
*   **Naming Convention**: Struct fields should use `json` tags, and the configuration field is recommended to be named `nodeConfig`.

```go
type MyNodeConfig struct {
    TargetID string `json:"targetId"`
    Timeout  int    `json:"timeout"`
}

type MyNode struct {
    // ...
    nodeConfig MyNodeConfig
}

func (n *MyNode) Init(config types.ConfigMap) error {
    if err := utils.Decode(config, &n.nodeConfig); err != nil {
        return types.InvalidConfiguration.Wrap(err)
    }
    // Set default values
    if n.nodeConfig.Timeout == 0 {
        n.nodeConfig.Timeout = 5000
    }
    return nil
}
```

### 3.2 Dynamic Configuration
If certain configuration items support dynamic expressions (e.g., `${metadata.key}`), keep them as `string` types in the configuration struct and resolve them at runtime in `OnMsg` using `helper.GetConfigAsset` or `helper.RenderConfigAsset`.

## 4. Error Handling

*   **Error Code Definition**: Define unique error codes in `pkg/cnst/constant.go` (following modular naming conventions).
*   **Fault Registration**: Register possible Faults in the node's `init()` function to the global `FaultRegistry`.

```go
func init() {
    registry.Default.GetNodeManager().Register(myNodePrototype)
    registry.Default.GetFaultRegistry().Register(
        DefMyCustomError,
    )
}
```

## 5. Testing Guidelines

*   **File Separation**: Split test code into multiple files, one for each major component (e.g., `manager_test.go`, `endpoint_test.go`, `node_test.go`).
*   **Setup Test**: Use `setup_test.go` for common test initialization (e.g., registering Mock factory methods, initializing global variables).
*   **Mocking**: Fully utilize Mock objects (e.g., `MockRuntime`, `MockNodeCtx`) to isolate dependencies. Avoid relying on real external services in unit tests.

## 6. General Design Principles

*   **Clear Data Flow**: For nodes involving data routing (like Pipeline), explicitly define input/output channels (`InputChannel`/`OutputChannel`) in the configuration rather than relying on implicit conventions.
*   **Type Safety**: When handling data transfer (e.g., `DataT`), prefer type assertions and factory methods (`types.NewDataT`), and handle type mismatch edge cases.
*   **Concurrency Safety**: For global managers or shared resources, ensure thread safety (use `sync.Mutex` or `sync.RWMutex`).
