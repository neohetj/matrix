---
# === Node Properties: 定义文档节点自身 ===
uuid: "f4e5d6c7-b8a9-0c1d-2e3f-4a5b6c7d8e9f"
type: "ComponentGuide"
title: "组件指南：SQL查询功能节点 (sqlQuery)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "function"
  - "sql"
  - "database"
  - "external"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的功能组件之一。"
---

# 1. 功能概述 (Overview)

`sqlQuery` 是一个外部交互功能节点，用于在规则链中执行SQL查询。它支持连接到任何兼容 `database/sql` 的数据库，并能执行 `SELECT`、`INSERT`、`UPDATE`、`DELETE` 等操作。

此节点的核心特性包括：
*   **安全查询模式**: 默认使用参数化查询，防止SQL注入。
*   **动态SQL模式**: 支持直接将占位符替换为SQL片段，用于动态表名/列名等场景（**有SQL注入风险，慎用**）。
*   **事务支持**: 可以参与由其他节点（如 `transaction` 节点）开启的事务。
*   **结果自动转换**: `SELECT` 查询的结果会自动转换为JSON数组字符串，并覆盖到 `msg.Data()` 字段，方便下游节点处理。

# 2. 如何配置 (Configuration)

在规则链的DSL中，可以通过 `configuration` 字段为此节点配置以下参数：

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `dsn` | DSN或引用 | 数据库连接字符串(DSN)，或一个指向共享DB节点的引用（格式为 `ref://<shared_node_id>`）。 | `string` | 是 | `"ref://default_db"` |
| `txContextKey` | 事务上下文键 | 如果要参与一个已存在的事务，需提供该事务在Go Context中的键。 | `string` | 否 | `""` |
| `query` | SQL查询模板 | 要执行的SQL语句。在安全模式下包含 `?` 占位符，在动态模式下包含 `${...}` 占位符。 | `string` | 是 | `""` |
| `params` | SQL参数 | 一个数组，提供了查询所需的参数。数组中的元素可以是字面量，也可以是 `${...}` 占位符。 | `[]interface{}` | 否 | `[]` |
| `isDynamicSql` | 是否为动态SQL | 如果为 `true`，则直接替换 `query` 字符串中的 `${...}` 占位符，`params` 字段将被忽略。**此模式有SQL注入风险。** | `bool` | 否 | `false` |

## 2.1. 占位符替换 (PlaceholderSubstitution)

占位符 `${...}` 的解析由 `helper.ExtractFromMsgByPath` 函数实现，支持从消息的任何部分提取数据。
*   **`${dataT.<objId>.<path>}`**: 从 `msg.DataT()` 中取值。
*   **`${metadata.<key>}`**: 从 `msg.Metadata()` 中取值。
*   **`${data.<path>}`**: 从 `msg.Data()` 解析出的JSON对象中取值。
*   **`'a string literal'`**: 使用单引号或双引号包裹的字符串字面量。

## 2.2. 配置示例 (Example)

### 2.2.1. 安全的参数化查询 (SecureParameterizedQuery)

假设 `msg.Metadata()` 为 `{"userId": 123}`。

```json
{
  "id": "node-get-user-orders",
  "type": "functions/sqlQuery",
  "name": "查询用户订单",
  "configuration": {
    "dsn": "ref://my_db",
    "query": "SELECT order_id, amount, created_at FROM orders WHERE user_id = ?",
    "params": ["${metadata.userId}"]
  }
}
```
**效果**: 节点将安全地执行 `SELECT ... WHERE user_id = 123`。执行完毕后，`msg.Data()` 会被覆盖为查询结果的JSON字符串，例如 `[{"order_id": "abc", "amount": 99.9, "created_at": "..."}]`。

### 2.2.2. 动态SQL查询 (DynamicSQLQuery)

**⚠️ 警告：此模式有SQL注入风险，仅在完全信任输入源且必须动态构建表名或列名时使用。**

假设 `msg.Data()` 为 `{"table_suffix": "2025_q4"}`。

```json
{
  "id": "node-get-archived-data",
  "type": "functions/sqlQuery",
  "name": "查询归档数据",
  "configuration": {
    "dsn": "ref://my_db",
    "query": "SELECT * FROM orders_${data.table_suffix} WHERE status = 'archived'",
    "isDynamicSql": true
  }
}
```
**执行的SQL**: `SELECT * FROM orders_2025_q4 WHERE status = 'archived'`。

# 3. 数据契约 (DataContract)

## 3.1. 输入 (Input)

*   **`msg.Data()` (string)**: 用于 `${data.<path>}` 占位符的替换。
*   **`msg.Metadata()` (Metadata)**: 用于 `${metadata.<key>}` 占位符的替换。
*   **`msg.DataT()` (DataT)**: 用于 `${dataT.<objId>.<path>}` 占位符的替换。

## 3.2. 输出 (Output)

*   **对于 `SELECT` 查询**:
    *   `msg.Data()` 字段会被**覆盖**为查询结果集的JSON数组字符串。
*   **对于 `INSERT`, `UPDATE`, `DELETE` 等操作**:
    *   `msg.Data()` 字段**不会被修改**。
*   `msg.Metadata()` 和 `msg.DataT()` **不会被修改**。

# 4. 错误处理 (ErrorHandling)

如果发生错误，节点将通过 `TellFailure` 关系传递错误信息。常见的错误包括：
*   **连接失败**: 无法根据 `dsn` 连接到数据库或找到引用的共享节点。
*   **参数提取失败**: `params` 中指定的占位符路径在消息中不存在。
*   **SQL执行失败**: 数据库返回了查询执行错误（例如，SQL语法错误、约束冲突）。

<!-- qa_section_start -->
> **问：`txContextKey` 是如何工作的？**
> **答：** `Matrix` 引擎支持通过一个专门的 `transaction` 节点来开启数据库事务。该节点会将创建的事务对象（`*sqlx.Tx`）存入Go的 `context.Context` 中，并使用一个特定的键。当 `sqlQuery` 节点配置了相同的 `txContextKey` 时，它会优先从 `context` 中获取该事务对象并使用它来执行查询，从而保证了多个 `sqlQuery` 节点可以在同一个事务中运行。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview]: ../00_matrix_guide.md
