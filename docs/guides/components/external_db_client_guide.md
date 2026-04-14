---
# === Node Properties: 定义文档节点自身 ===
uuid: "b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e"
type: "ComponentGuide"
title: "组件指南：SQL客户端共享节点 (external/sqlClient)"
status: "Draft"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "component"
  - "external"
  - "database"
  - "sql"
  - "shared"
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本组件指南属于 Matrix 指南体系。"
---

# 1. 功能概述

`external/sqlClient` 是 Matrix core 内建的共享节点，用于提供可复用的 `*sqlx.DB` 连接实例。

它的职责只有两件事：

1. 根据配置创建数据库连接
2. 在引擎生命周期内复用该连接

它**不负责**执行业务查询；业务查询通常由宿主应用注册的函数来完成，并通过 `ref://` 引用这个共享资源。

# 2. 配置字段

| 字段 | 描述 |
| :--- | :--- |
| `driverName` | 数据库驱动名，例如 `postgres` |
| `uri` | 数据库连接 URI |
| `poolSize` | 最大连接数，`0` 表示不额外设置 |

示例：

```json
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
```

# 3. 如何引用

消费方应通过 `ref://<node-id>` 引用该共享节点：

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

这里的 `sqlQuery` 是宿主应用自己注册到 `NodeFuncManager` 的函数，`Matrix` core 只提供 `functions` 节点和 `external/sqlClient` 共享节点。

# 4. 数据契约

`external/sqlClient` 是资源提供节点，不参与消息读写：

- 不读取 `RuleMsg.Data`
- 不读取 `RuleMsg.Metadata`
- 不读取 `RuleMsg.DataT`
- 不修改任何消息内容

它的价值在于被其他节点或函数按需引用。
