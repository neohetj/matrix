---
# === Node Properties: 定义文档节点自身 ===
uuid: "e3d4c5b6-a7b8-9c0d-1e2f-3a4b5c6d7e8f"
type: "ComponentGuide"
title: "组件指南：Redis命令功能节点 (redisCommand)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "function"
  - "redis"
  - "cache"
  - "external"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的功能组件之一。"
---

# 1. 功能概述 (Overview)

`redisCommand` 是一个外部交互功能节点，用于在规则链中执行一个或多个Redis命令。它提供了与Redis服务器进行通用交互的能力，支持字符串占位符替换，使得命令可以动态地根据传入消息的内容和元数据进行构建。

此节点的核心应用场景包括：
*   缓存读写（`GET`, `SET`）
*   发布/订阅（`PUBLISH`）
*   流处理（`XADD`）
*   执行复杂的业务逻辑脚本（`EVAL`）

# 2. 如何配置 (Configuration)

在规则链的DSL中，可以通过 `configuration` 字段为此节点配置以下参数：

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `redisDsn` | Redis DSN或引用 | Redis连接字符串(DSN)，或一个指向共享节点的引用（格式为 `ref://<shared_node_id>`）。 | `string` | 是 | `"ref://default_redis"` |
| `commands` | 命令列表 | 一个字符串数组，包含了要按顺序执行的Redis命令模板。 | `[]string` | 是 | `[]` |
| `propagateMeta` | 传播元数据 | 如果为 `true`，则会将消息的元数据注入到特定的Redis命令中（目前支持 `XADD` 和 `PUBLISH`）。 | `bool` | 否 | `false` |
| `propagateKeys` | 需传播的键 | 当 `propagateMeta` 为 `true` 时，定义哪些元数据键需要被传播。空数组默认只传播 `ExecutionID`，`['*']` 表示传播所有。 | `[]string` | 否 | `[]` |
| `wrapPayload` | 包装PUBLISH载荷 | 仅当命令为 `PUBLISH` 且 `propagateMeta` 为 `true` 时生效。如果为 `true`，会将原始消息和元数据包装成一个标准的JSON对象再发布。 | `bool` | 否 | `false` |

## 2.1. 占位符替换 (PlaceholderSubstitution)

`commands` 数组中的每个命令字符串都支持占位符替换。占位符的格式为 `${<source>.<path>}`。该替换逻辑由 `utils.ReplacePlaceholders` 函数实现，其数据源结构如下：

*   **`${data.<path>}`**: 从 `msg.Data()` 解析出的JSON对象中取值。
*   **`${metadata.<key>}`**: 从 `msg.Metadata()` 中取值。
*   **`${dataT.<objId>.<path>}`**: 从 `msg.DataT()` 中特定 `CoreObj` 的 `Body()` 中取值。

`<path>` 和 `<key>` 支持使用点 `.` 语法进行嵌套访问。

## 2.2. 配置示例 (Example)

### 2.2.1. 简单缓存设置 (SimpleCacheSet)

假设 `msg.Metadata()` 为 `{"userId": "123"}`，`msg.Data()` 为 `{"name": "Alice", "email": "alice@example.com"}`。

```json
{
  "id": "node-cache-user",
  "type": "functions/redisCommand",
  "name": "缓存用户信息",
  "configuration": {
    "redisDsn": "ref://my_redis_instance",
    "commands": [
      "SET user:${metadata.userId} ${data}"
    ]
  }
}
```
**执行的命令**: `SET user:123 {"name":"Alice","email":"alice@example.com"}`

### 2.2.2. 发布事件并传播元数据 (PublishEventWithMetadata)

假设 `msg.DataT()` 中包含一个 `objId` 为 `orderInfo` 的对象，其内容为 `{"orderId": "xyz", "amount": 99}`。

```json
{
  "id": "node-publish-event",
  "type": "functions/redisCommand",
  "name": "发布订单创建事件",
  "configuration": {
    "redisDsn": "redis://localhost:6379/0",
    "commands": [
      "PUBLISH order_events ${dataT.orderInfo}"
    ],
    "propagateMeta": true,
    "wrapPayload": true
  }
}
```
**效果**: 最终发布到 `user_events` 频道的消息会是类似这样的JSON字符串：
```json
{
  "__payload__": "{\"userId\": 123, \"action\": \"created\"}",
  "__matrix_metadata__": {
    "traceId": "abc-123",
    "executionId": "xyz-456"
  }
}
```

# 3. 数据契约 (DataContract)

本节点主要通过 `configuration` 和占位符与消息进行交互，不直接读写 `DataT`。

## 3.1. 输入 (Input)

*   **`msg.Data()` (string)**: 用于 `${data.<path>}` 占位符的替换。如果使用，必须是一个有效的JSON字符串。
*   **`msg.Metadata()` (Metadata)**: 用于 `${metadata.<key>}` 占位符的替换，以及元数据传播功能。
*   **`msg.DataT()` (DataT)**: 用于 `${dataT.<objId>.<path>}` 占位符的替换。

## 3.2. 输出 (Output)

*   本节点**不会修改**传入的 `RuleMsg`。它将原始消息原封不动地传递给下游节点。

# 4. 错误处理 (ErrorHandling)

如果发生错误，节点将通过 `TellFailure` 关系传递错误信息。常见的错误包括：
*   **连接失败**: 无法根据 `redisDsn` 连接到Redis服务器或找到引用的共享节点。
*   **命令执行失败**: Redis服务器返回了命令执行错误（例如，语法错误、权限问题）。

<!-- qa_section_start -->
> **问：`parseCommand` 函数的作用是什么？**
> **答：** `redisCommand` 节点需要将一个完整的命令字符串（如 `SET mykey "hello world"`）拆分成一个命令数组（`["SET", "mykey", "hello world"]`）才能发送给Redis客户端。`parseCommand` 函数负责这个拆分工作，并且能正确处理包含空格的带引号的参数。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview]: ../00_matrix_guide.md
