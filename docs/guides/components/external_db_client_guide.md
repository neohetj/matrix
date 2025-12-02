---
# === Node Properties: 定义文档节点自身 ===
uuid: "b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e"
type: "ComponentGuide"
title: "组件指南：数据库客户端共享节点 (external/dbClient)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "external"
  - "database"
  - "sql"
  - "shared"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的功能组件之一。"
  - type: "references"
    target_uuid: "a3b4c5d6-e7f8-9a0b-1c2d-3e4f5a6b7c8d"
    description: "本节点是共享节点的一种实现，遵循共享节点的设计模式。"
---

# 1. 功能概述 (Overview)

`dbClient` 是一个**[共享节点 (Shared Node)][Guide-SharedNode-7c8d]**，它的核心职责是创建并管理一个可供其他节点复用的数据库连接池 (`*sqlx.DB`)。

在规则链中，任何需要与数据库交互的节点（例如 `sqlQuery`）都不应该自己创建连接，而是应该通过引用一个 `dbClient` 节点的实例来获取连接。这种模式实现了连接管理的中心化，提高了资源利用率和系统的健壮性。

> **核心概念**: 在继续之前，请务必阅读 **[指南：理解和使用共享节点][Guide-SharedNode-7c8d]** 以了解共享节点的通用工作原理和生命周期。

# 2. 如何配置 (Configuration)

在规则链的DSL中，可以通过 `configuration` 字段为此节点配置以下参数：

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `driverName` | 驱动名称 | 数据库驱动的名称，例如 `mysql`, `postgres`。 | `string` | 是 | N/A |
| `dsn` | DSN | 数据库连接字符串 (Data Source Name)。 | `string` | 是 | N/A |
| `poolSize` | 连接池大小 | 数据库连接池的最大连接数。 | `int` | 否 | `0` (不限制) |

## 2.1. 配置示例 (Example)

```json
{
  "id": "shared_mysql_db",
  "type": "external/dbClient",
  "name": "共享MySQL连接池",
  "description": "为所有业务节点提供MySQL数据库连接",
  "configuration": {
    "driverName": "mysql",
    "dsn": "user:password@(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
    "poolSize": 20
  }
}
```

# 3. 如何引用 (HowToReference)

其他节点可以通过 `ref://<node_id>` 的格式来引用本节点。详细的引用机制请参见 **[共享节点指南][Guide-SharedNode-7c8d]**。

### 3.1. `sqlQuery` 引用示例

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

# 4. 数据契约 (DataContract)

`dbClient` 节点是一个资源管理节点，它**不参与**消息的数据流处理。它不会读取或修改 `RuleMsg` 的任何部分 (`Data`, `DataT`, `Metadata`)。

<!-- 链接定义区域 -->
[Guide-MatrixOverview]: ../../00_matrix_guide.md
[Guide-SharedNode-7c8d]: ../01_shared_node_guide.md
