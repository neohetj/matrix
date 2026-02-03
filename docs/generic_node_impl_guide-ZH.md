# 通用节点实现规范与最佳实践

本文档总结了在 Matrix 框架中开发通用节点（Generic Nodes）的规范和注意事项。

## 1. 目录结构

*   **分组管理**：节点应根据功能归类到 `internal/builtin/nodes` 下的相应子目录中（例如 `endpoint`, `loop`, `pipeline`, `action`）。
*   **独立包**：每个大的组件类别应作为一个独立的 Go package。

## 2. 节点定义与接口

### 2.1 基础接口
所有节点必须实现 `types.Node` 接口。即使是 `Endpoint` 类型的节点，也建议实现 `OnMsg` 方法，以保持与常规流程的兼容性（例如作为链中的普通节点被调用）。

```go
type MyNode struct {
    types.BaseNode
    types.Instance
    // ...
}

// 强制编译期接口检查
var _ types.Node = (*MyNode)(nil)

func (n *MyNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
    // 实现逻辑
    ctx.TellSuccess(msg)
}
```

### 2.2 Endpoint 节点
*   必须实现 `types.Endpoint` 接口。
*   如果是主动运行（如监听端口、后台任务），必须实现 `types.ActiveEndpoint` 接口（包含 `Start` 和 `Stop` 方法）。
*   添加编译期检查：`var _ types.ActiveEndpoint = (*MyEndpoint)(nil)`。

## 3. 配置管理

### 3.1 静态配置结构
*   **定义结构体**：为每个节点定义一个配置结构体（例如 `MyNodeConfiguration`）。
*   **Init 初始化**：在 `Init` 方法中使用 `utils.Decode` 将 `types.ConfigMap` 解码为结构体。
*   **命名规范**：结构体字段应使用 `json` 标签，配置字段建议存储在 `nodeConfig` 成员变量中。

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
    // 设置默认值
    if n.nodeConfig.Timeout == 0 {
        n.nodeConfig.Timeout = 5000
    }
    return nil
}
```

### 3.2 动态配置
如果某些配置项支持动态表达式（如 `${metadata.key}`），可以在配置结构体中保留为 `string` 类型，并在 `OnMsg` 运行时使用 `helper.GetConfigAsset` 或 `helper.RenderConfigAsset` 进行解析。

## 4. 错误处理

*   **错误码定义**：在 `pkg/cnst/constant.go` 中定义唯一的错误码（遵循模块化命名规范）。
*   **错误注册**：在节点的 `init()` 函数中，将可能的 Fault 注册到全局 `FaultRegistry`。

```go
func init() {
    registry.Default.GetNodeManager().Register(myNodePrototype)
    registry.Default.GetFaultRegistry().Register(
        DefMyCustomError,
    )
}
```

## 5. 测试规范

*   **文件分离**：将测试代码拆分为多个文件，每个主要组件一个测试文件（如 `manager_test.go`, `endpoint_test.go`, `node_test.go`）。
*   **Setup Test**：使用 `setup_test.go` 进行通用的测试初始化（如注册 Mock 工厂方法、初始化全局变量）。
*   **Mocking**：充分利用 Mock 对象（如 `MockRuntime`, `MockNodeCtx`）来隔离依赖。避免在单元测试中依赖真实的外部服务。

## 6. 通用设计原则

*   **数据流向明确**：对于涉及数据流转的节点（如 Pipeline），在配置中显式定义输入/输出通道（`InputChannel`/`OutputChannel`），而不是依赖隐式约定。
*   **类型安全**：在处理数据传递时（如 `DataT`），优先使用类型断言和工厂方法（`types.NewDataT`），并处理类型不匹配的边界情况。
*   **并发安全**：对于全局管理器（Manager）或共享资源，必须确保线程安全（使用 `sync.Mutex` 或 `sync.RWMutex`）。
    *   **RuleMsg 深拷贝**：`types.RuleMsg` 包含 `Metadata`（Map），非线程安全。当消息跨越协程边界（例如推送到 Channel、异步分发）时，**必须调用 `msg.DeepCopy()`**，确保接收方获得状态隔离的副本，防止竞态条件。

## 7. 共享资源与依赖管理

### 7.1 共享节点引用
*   **配置模式**：组件若依赖共享资源（如数据库连接、通道管理器），应在配置中接收资源的 URI 引用（例如 `channelManager: "ref://my-manager-id"`），而不是直接接收 ID。
*   **解析方式**：在运行时（`OnMsg` 或 `Start`）使用 `pkg/asset` 包进行解析。
    ```go
    // 解析共享资源
    ast := asset.Asset[*MyResource]{URI: config.ResourceURI}
    ctx := asset.NewAssetContext(asset.WithNodePool(pool)) // 获取 pool 的方式视上下文而定
    resource, err := ast.Resolve(ctx)
    ```

### 7.2 共享节点实现
*   实现 `types.SharedNode` 接口，提供 `GetInstance()` 方法返回具体的资源实例。
*   **接口定义**：为了支持跨包调用和测试，建议将资源的操作接口定义在 `pkg/types` 中（如 `types.ChannelManager`），而不是仅依赖 `internal` 中的具体结构体。

## 8. 测试注意事项

*   **避免 Internal 引用**：集成测试（如位于其他模块的测试）不应导入 `internal` 包。应依赖 `pkg/types` 中的公共接口。
*   **DSL 注册**：在测试中需要模拟内部共享节点时，可以通过 DSL 加载的方式将其注册到测试环境的 SharedNodePool 中，而不是手动实例化内部结构体。
    ```go
    // 通过 DSL 注册内部节点
    dsl := `{"metadata":{"nodes":[{"id":"test-res","type":"resource/my_resource"}]}}`
    matrix.Registry.GetSharedNodePool().Load([]byte(dsl), ...)
    // 通过接口获取实例
    inst, _ := pool.GetInstance("test-res")
    manager := inst.(types.MyResourceManager)
    ```
