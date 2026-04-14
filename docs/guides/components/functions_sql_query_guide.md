---
# === Node Properties: 定义文档节点自身 ===
uuid: "f4e5d6c7-b8a9-0c1d-2e3f-4a5b6c7d8e9f"
type: "ComponentGuide"
title: "示例指南：宿主注册的 SQL 查询函数 (sqlQuery)"
status: "Draft"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "component"
  - "function"
  - "sql"
  - "database"
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本文档描述宿主应用中常见的函数型组件模式。"
---

# 1. 先说明边界

`sqlQuery` **不是 Matrix core 内建函数**。本文档描述的是一种常见的宿主应用函数模式：宿主把 `sqlQuery` 注册到 `NodeFuncManager`，再通过通用 `functions` 节点调用它。

如果你的宿主应用没有注册这个函数，这份 DSL 示例不能直接运行。

# 2. 推荐调用形态

当前推荐 DSL 形态如下：

```json
{
  "id": "node-get-user-orders",
  "type": "functions",
  "name": "查询用户订单",
  "configuration": {
    "functionName": "sqlQuery",
    "business": {
      "dsn": "ref://shared_sql_client",
      "query": "SELECT order_id, amount, created_at FROM orders WHERE user_id = ?",
      "params": ["${metadata.userId}"]
    }
  }
}
```

注意点：

1. `type` 是 `functions`，不是 `functions/sqlQuery`
2. `functionName` 放在 `configuration.functionName`
3. 业务字段放在 `configuration.business`
4. 共享数据库连接通过 `ref://shared_sql_client` 引用 `external/sqlClient`

# 3. 常见配置约定

不同项目里的 `sqlQuery` 函数实现可能不同，但常见约定包括：

| 业务字段 | 含义 |
| :--- | :--- |
| `dsn` | SQL 连接 URI 或 `ref://` 共享资源引用 |
| `query` | SQL 模板 |
| `params` | 参数数组 |
| `isDynamicSql` | 是否启用动态 SQL 拼接 |
| `txContextKey` | 事务上下文键 |

这些字段是否存在、如何解释，最终取决于宿主注册的函数实现。

# 4. 与共享资源的关系

推荐把数据库连接交给 `external/sqlClient` 管理，再在 `sqlQuery` 中通过 `ref://` 消费：

```json
{
  "id": "shared_sql_client",
  "type": "external/sqlClient",
  "configuration": {
    "driverName": "postgres",
    "uri": "postgres://user:pass@localhost:5432/app?sslmode=disable"
  }
}
```

这样可以避免每次函数执行时重复建连。

# 5. 设计建议

如果你在宿主应用里实现 `sqlQuery`，建议：

1. 用 `NodeFuncObject` 注册，而不是单独定义新 NodeType
2. 通过 `configuration.business` 读取业务配置
3. 通过 `FuncReads` / `FuncWrites`、`Inputs` / `Outputs` 声明契约
4. 通过 `ref://` 消费共享 SQL 客户端
5. 把事务相关能力设计成额外业务字段或独立函数，而不是硬编码在 DSL 之外

<!-- qa_section_start -->
> **问：为什么文档不再推荐 `type: "functions/sqlQuery"`？**
> **答：** 因为当前 Matrix core 的函数模型是“通用 `functions` 节点 + 宿主注册函数 ID”，而不是为每个函数各自提供一个独立 NodeType。
<!-- qa_section_end -->
