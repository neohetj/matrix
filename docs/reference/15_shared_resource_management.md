---
uuid: "a1b3d5e7-c4a0-4b1e-9c3a-1f8e6d2c5b4a"
type: "Reference"
title: "学习Matrix共享资源管理"
status: "Stable"
owner: "neohetj"
version: "2.3.0"
tags:
  - "shared-resource"
  - "dependency-injection"
  - "decoupling"
  - "SharedNode"
  - "NodePool"
relations:
  - type: "is_part_of"
    target_uuid: "e4a9b8c1-3d2a-4b1e-8c5d-7a4b9c2d8e1f"
    description: "共享资源管理是 Matrix 架构总览的一部分。"
---

# 如何管理和使用共享资源 (SharedResourceManagement)

本文档解释 Matrix 中共享资源（Shareable Resource）的当前实现机制。共享资源包括 SQL 客户端、Redis 客户端、Mongo 客户端、通道管理器等长期存活、可跨规则链复用的实例。

## 1. 核心概念

共享资源机制由两个核心组件构成：

1. **`SharedNode`**：资源提供方节点。
2. **`NodePool`**：引擎级共享池，用于保存已实例化的共享节点；“共享”不代表跨 Engine 使用进程全局池。

接口定义如下：

```go
type SharedNode interface {
    Node
    GetInstance() (any, error)
}
```

`GetInstance()` 返回该节点所管理的真实资源实例，例如 `*sqlx.DB`、`*redis.Client` 或 `*mongo.Client`。

## 2. 共享节点何时创建

共享节点不是在某条规则链运行时临时创建的。标准流程是：

```mermaid
flowchart TD
    discover["1. Discover shared DSL paths"] --> load["2. builder.LoadSharedNodes(...)"]
    load --> pool["3. NodePool.NewFromNodeDef(...)"]
    pool --> inject["4. Init 前注入所属 Engine 的 NodePool / ConfigReader"]
    inject --> ready["5. SharedNodePool 持有实例"]
    ready --> runtime["6. 节点或 endpoint 从所属实例按需引用"]
    load -->|已启用模块配置且资源装载失败| abort["终止 Engine 创建并返回资源文件与错误原因"]
```

对应代码入口：

- `matrix.New(...)`
- `builder.LoadSharedNodes(...)`
- `NodePool.LoadFromRuleChainDef(...)`
- `NodePool.NewFromNodeDef(...)`

shared DSL 文件通常本身是一个 `RuleChainDef` 容器，但主要用途是承载 `metadata.nodes` 中的共享节点定义。

使用 `WithModuleConfig` 的 Engine 为共享资源和规则链建立私有池。其选中加载的 shared 文件若读取、解析或节点初始化失败，创建过程直接返回错误，不能记录 warning 后丢弃资源继续启动。未声明或不存在的可选 shared 目录仍允许跳过。未启用模块配置的旧入口保留原装载策略；配置感知节点的初始化错误仍必须向上传播。

这项检查不替业务模块判断哪些业务配置必填，也不提前调用所有惰性资源的 `GetInstance()`；条件必需项由业务启动检查负责。

## 3. 如何实现共享节点

当前仓库推荐通过嵌入 `internal/builtin/base.Shareable[T]` 来实现：

```go
type SQLClientNode struct {
    types.BaseNode
    types.Instance
    base.Shareable[*sqlx.DB]
    nodeConfig SQLClientNodeConfiguration
    client     *sqlx.DB
}

func (n *SQLClientNode) Init(cfg types.ConfigMap) error {
    if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
        return err
    }

    initFunc := func() (*sqlx.DB, error) {
        if n.client != nil {
            return n.client, nil
        }
        db, err := sqlx.Connect(n.nodeConfig.DriverName, n.nodeConfig.URI)
        if err != nil {
            return nil, err
        }
        n.client = db
        return n.client, nil
    }

    return n.Shareable.Init(nil, n.nodeConfig.URI, initFunc)
}
```

几点说明：

1. 共享节点本身负责提供资源实例，而不是消费 `ref://`。
2. `Shareable[T]` 负责缓存实例和延迟初始化。
3. `Destroy()` 负责在节点生命周期结束时释放底层连接。

## 4. 如何消费共享节点

消费方推荐把共享资源字段设计成 URI，再通过 `asset.Asset[T]` 或 helper 解析，而不是在每个节点里直接散写 `NodePool.GetInstance(...)`。

### 4.1. 推荐模式

```go
func resolveClient(pool types.NodePool, resourceURI string) (any, error) {
    ast := asset.Asset[any]{URI: resourceURI}
    assetCtx := asset.NewAssetContext(asset.WithNodePool(pool))
    return ast.Resolve(assetCtx)
}
```

优点：

1. `ref://`、模板值和其他 scheme 都走统一入口。
2. 解析层承担类型校验和错误包装。
3. 消费方代码可以保持简单，不必关心 `NodePool` 内部细节。

### 4.2. 什么时候直接用 `NodePool`

只有在以下场景才建议直接操作 `NodePool`：

- 你正在实现更底层的 helper / connector。
- 你需要绕过 `asset.Asset` 做特殊生命周期管理。
- 你明确知道返回类型，并且不希望引入额外包装。

### 4.3. 实例隔离边界

- 需要长期持有资源池的节点实现 `types.NodePoolAware`，由共享节点或规则链装配路径在 `Init` 前注入。Pipeline 端点、Channel Push、Redis Stream 端点使用此方式；直接手工构造节点的调用者也必须显式注入池。
- 已注入的资源池查不到 `ref://` 时直接报错，禁止改查全局同名资源。
- Redis Stream 与 Pipeline 端点已注入 `RuntimePool` 后，只在该池查找目标规则链；缺失时返回未找到。仅未注入运行时池的旧端点调用方式保留全局查找。
- `action/forEach` 从当前 `NodeCtx → Runtime → Engine → RuntimePool` 查找子链；执行上下文或目标链缺失时返回错误。
- 节点类型原型的全局注册不属于运行态资源查找，继续由 NodeManager 管理。

回归验证包括同名资源在私有池和全局池同时存在、本地缺失而全局存在，以及 `matrix.New(...WithModuleConfig...)` 的 Pipeline 启动路径。对应测试为 `module_config_pipeline_test.go`、`module_config_shared_loading_test.go` 和 endpoint / pipeline / loop 包内的实例隔离测试。

## 5. DSL 示例

### 5.1. shared DSL 中定义资源

```json
{
  "ruleChain": {
    "id": "shared_clients"
  },
  "metadata": {
    "nodes": [
      {
        "id": "shared_sql_client",
        "type": "external/sqlClient",
        "name": "共享 SQL 客户端",
        "configuration": {
          "driverName": "postgres",
          "uri": "postgres://user:pass@localhost:5432/app?sslmode=disable",
          "poolSize": 20
        }
      }
    ],
    "connections": []
  }
}
```

### 5.2. 业务节点中引用资源

```json
{
  "id": "queryUsers",
  "type": "functions",
  "configuration": {
    "functionName": "sqlQuery",
    "business": {
      "dsn": "ref://shared_sql_client",
      "query": "SELECT * FROM users WHERE id = ?",
      "params": ["${metadata.userId}"]
    }
  }
}
```

这里的 `sqlQuery` 是宿主应用注册到 `NodeFuncManager` 的函数；`Matrix` 核心只负责提供通用 `functions` 节点和 `ref://` 解析能力。

## 6. FAQ

<!-- qa_section_start -->
> **问：为什么推荐 `asset.Asset` 而不是每个节点都直接 `GetInstance()`？**
> **答：** 因为它能统一处理 URI、模板、类型转换和错误包装，让消费方节点更简洁，也更容易做复用。

> **问：共享节点能处理消息吗？**
> **答：** 可以，但通常不应该承担业务消息处理职责。共享节点的主要责任是提供和管理资源实例。

> **问：共享节点必须放在单独的 shared DSL 文件里吗？**
> **答：** 不是语法强制，但这是当前工程里最推荐、最清晰的装载方式。这样能明确区分“引擎级共享资源”和“某条规则链内部节点”。
<!-- qa_section_end -->
