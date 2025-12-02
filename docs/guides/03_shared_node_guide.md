---
# === Node Properties: 定义文档节点自身 ===
uuid: "a3b4c5d6-e7f8-9a0b-1c2d-3e4f5a6b7c8d"
type: "Guide"
title: "指南：理解和使用共享节点 (Shared Node)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
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
    description: "共享节点是Matrix核心架构的关键概念之一。"
---

# 1. 什么是共享节点？ (WhatIsASharedNode)

在 `Matrix` 引擎中，**共享节点 (Shared Node)** 是一种特殊的节点类型，其核心职责是**创建和管理可被多个规则链复用的资源实例**。

与生命周期与单条规则链绑定的普通节点不同，共享节点的实例在 `Matrix` 引擎启动时就会被创建，并在整个引擎的生命周期内保持存在。

典型的共享节点包括：
*   数据库连接池 (`external/dbClient`)
*   Redis客户端 (`external/redisClient`)
*   HTTP客户端 (`external/httpClient`)

# 2. 为什么需要共享节点？ (WhyUseSharedNodes)

共享节点解决了资源管理中的两个核心问题：

1.  **性能与效率**: 像数据库连接、Redis连接这类资源，其初始化和建立连接的开销通常很大。如果每个规则链、甚至每个节点都创建自己的连接，将会造成巨大的性能浪费。通过共享节点，所有规则链可以复用同一个连接池，极大地提高了资源利用率。
2.  **中心化配置**: 将资源（如数据库）的配置（如DSN、PoolSize）集中在一个共享节点中进行管理，使得配置变更更加方便和安全，避免了在多个规则链中维护重复的配置信息。

# 3. 如何定义和使用共享节点 (HowToDefineAndUse)

## 3.1. 在DSL中定义共享节点 (DefiningInDSL)

共享节点通常在规则链定义的 `metadata.nodes` 数组中被定义，就像普通节点一样。关键在于要为其设置一个在当前规则链中唯一的 `id`。

### 示例：定义一个DB客户端共享节点

```json
{
  "id": "shared_mysql_db",
  "type": "external/dbClient",
  "name": "共享MySQL连接池",
  "configuration": {
    "driverName": "mysql",
    "dsn": "user:password@tcp(host:port)/dbname"
  }
}
```

## 3.2. 在其他节点中引用共享节点 (ReferencingFromOtherNodes)

任何需要使用该共享资源的节点，都可以在其 `configuration` 中通过 `ref://<shared_node_id>` 的特殊URI格式来引用它。

### 示例：`sqlQuery` 节点引用 `dbClient`

```json
{
  "id": "get_user_data",
  "type": "functions/sqlQuery",
  "name": "获取用户数据",
  "configuration": {
    "dsn": "ref://shared_mysql_db",
    "query": "SELECT * FROM users WHERE id = ?",
    "params": ["${metadata.userId}"]
  }
}
```
`Matrix` 引擎在解析 `sqlQuery` 节点的配置时，如果发现 `dsn` 字段的值是 `ref://` 前缀，它会自动去查找 `id` 为 `shared_mysql_db` 的节点，并将其管理的数据库连接实例注入到 `sqlQuery` 节点中。

<!-- qa_section_start -->
> **问：共享节点可以处理消息（`OnMsg`）吗？**
> **答：** 理论上可以，但通常不这么做。共享节点的核心职责是资源管理，而非数据处理。因此，它们的 `OnMsg` 方法通常是一个空操作（No-op）。数据处理的逻辑应该由引用它们的功能节点来完成。

> **问：共享节点的底层实现是怎样的？**
> **答：** 共享节点的实现基于 `base.Shareable[T]` 泛型工具，它封装了资源池化和懒加载的逻辑。更深入的实现细节和设计哲学，请参阅 **[参考：共享资源管理机制][Ref-SharedResource]**。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview]: ./00_matrix_guide.md
[Ref-SharedResource]: ../reference/15_shared_resource_management.md
