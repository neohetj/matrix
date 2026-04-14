---
# === Node Properties: 定义文档节点自身 ===
uuid: "a3b4c5d6-e7f8-9a0b-1c2d-3e4f5a6b7c8d"
type: "Guide"
title: "指南：理解和使用共享节点 (Shared Node)"
status: "Draft"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "guide"
  - "shared-node"
  - "architecture"
  - "resource-management"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "共享节点是 Matrix 核心架构的重要概念之一。"
---

# 1. 什么是共享节点？

共享节点是 Matrix 中专门用于**提供可复用资源实例**的节点类型。它们通常在引擎初始化阶段就被加载进 `SharedNodePool`，而不是等某条规则链运行到它时才临时创建。

典型示例：

- `external/sqlClient`
- `external/redisClient`
- `external/mongoClient`
- `resource/channel_manager`

# 2. 为什么需要共享节点？

共享节点主要解决两个问题：

1. **避免重复建连**：数据库、Redis、Mongo 这类连接代价高，重复初始化没有意义。
2. **统一配置入口**：连接串、连接池大小、TLS 策略等集中管理，不需要散落到每条业务链上。

# 3. 如何定义共享节点？

推荐把共享节点放到专门的 shared DSL 文件中：

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

这个文件会在引擎启动时被 `LoadSharedNodes(...)` 扫描并装载。

# 4. 如何引用共享节点？

业务节点应通过 `ref://<shared-node-id>` 引用共享资源：

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

这里的 `sqlQuery` 代表宿主应用注册的函数。`Matrix` 核心本身只提供通用 `functions` 节点和共享资源解析能力。

# 5. 业务节点如何拿到资源？

推荐通过 `asset.Asset` 或封装好的 helper 解析：

```go
func resolveResource(pool types.NodePool, uri string) (any, error) {
    ast := asset.Asset[any]{URI: uri}
    assetCtx := asset.NewAssetContext(asset.WithNodePool(pool))
    return ast.Resolve(assetCtx)
}
```

这样可以统一处理：

- `ref://` 解析
- 类型校验
- 模板渲染
- 未来扩展的新 scheme

# 6. 常见误区

1. 把共享节点直接写进业务规则链节点列表里，然后当成普通节点执行。
2. 在每个业务节点里重复手写 `NodePool.GetInstance(...)`。
3. 把共享资源字段写成裸 ID，而不是 URI。
4. 把消费共享资源的业务逻辑塞回共享节点自身。

<!-- qa_section_start -->
> **问：共享节点可以处理消息吗？**
> **答：** 理论上可以，但通常不应承担业务消息处理职责。共享节点的核心责任是管理资源实例。

> **问：为什么推荐单独的 shared DSL 文件？**
> **答：** 因为这样最容易区分“引擎级共享资源”和“某条业务规则链的普通节点”，也更符合当前引擎装载流程。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Ref-SharedResource]: ../reference/15_shared_resource_management.md
