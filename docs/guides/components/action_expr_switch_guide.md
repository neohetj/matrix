---
# === Node Properties: 定义文档节点自身 ===
uuid: "c5b4a398-f6e7-4b0d-8c6a-1e2f3a4b5c6d"
type: "ComponentGuide"
title: "组件指南：表达式路由 (action/exprSwitch)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "action"
  - "switch"
  - "routing"
  - "expression"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的动作组件之一。"
  - type: "references"
    target_uuid: "81080378-a3e9-41ee-86ed-807193d45bce"
    description: "本文档遵循语义化文档规范编写。"
---

# 1. 功能概述 (FunctionalOverview)

`action/exprSwitch` 是一个核心的**路由 (Routing)** 节点。它允许开发者根据消息的实时内容，通过一系列强大的**表达式**来动态地决定消息的流向。

当消息到达此节点时，节点会按顺序计算一系列布尔表达式。消息将被路由到**第一个**计算结果为 `true` 的表达式所对应的关系链路上。这为实现复杂的条件判断、分支逻辑和内容分发提供了强大的支持。

# 2. 如何配置 (Configuration)

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `cases` | 路由分支 | 一个定义了路由规则的 `map`。`key` 是关系类型（如 `"HighTemp"`），`value` 是一个返回布尔值的表达式。 | `map[string]string` | 是 | N/A |
| `defaultRelation` | 默认路由 | (可选) 一个关系类型字符串。当 `cases` 中所有表达式的计算结果都为 `false` 时，消息将被路由到此关系。 | `string` | 否 | `""` |

# 3. 核心概念：表达式 (`expr-lang`) (ExpressionConcept)

本节点使用 [expr](https://github.com/expr-lang/expr) 作为其表达式引擎。表达式可以直接访问 `RuleMsg` 中的数据。

### 3.1. 可用数据源

在表达式中，你可以直接通过以下顶级变量访问消息内容：

*   `metadata`: 访问消息的元数据。
*   `dataT`: 访问结构化的 `DataT` 对象容器。
*   `data`: 访问原始的 `Data` 字符串。

### 3.2. 表达式示例

| 表达式 | 描述 |
| :--- | :--- |
| `metadata.temp > 40.0` | 检查元数据中的 `temp` 字段是否大于40。 |
| `dataT.user.role == 'admin'` | 检查 `dataT` 中 `id` 为 `user` 的对象的 `role` 字段是否为 'admin'。 |
| `len(dataT.alerts) > 5` | 检查 `dataT` 中 `id` 为 `alerts` 的对象（应为数组或切片）的长度是否大于5。 |
| `metadata.deviceType in ['A', 'B']` | 检查 `metadata.deviceType` 是否为 'A' 或 'B'。 |
| `data matches '^ERROR'` | 检查原始 `Data` 字符串是否以 "ERROR" 开头。 |

# 4. 配置示例 (Example)

假设我们有一个处理设备遥测数据的规则链，需要根据温度将消息路由到不同的处理分支。

**输入消息**: `metadata` 中包含 `{"temperature": 55.0, "source": "sensor-01"}`。

**DSL 配置**:
```json
{
  "id": "node-check-temperature",
  "type": "action/exprSwitch",
  "name": "根据温度路由",
  "configuration": {
    "cases": {
      "HighTemp": "metadata.temperature > 50.0",
      "NormalTemp": "metadata.temperature >= 10.0 and metadata.temperature <= 50.0",
      "LowTemp": "metadata.temperature < 10.0"
    },
    "defaultRelation": "Unknown"
  }
}
```
**连接配置**:
```json
[
  { "fromId": "node-check-temperature", "toId": "node-high-temp-alert", "type": "HighTemp" },
  { "fromId": "node-check-temperature", "toId": "node-log-normal-temp", "type": "NormalTemp" },
  { "fromId": "node-check-temperature", "toId": "node-low-temp-alert", "type": "LowTemp" },
  { "fromId": "node-check-temperature", "toId": "node-log-unknown-temp", "type": "Unknown" }
]
```

**流程解析**:
1.  消息到达 `node-check-temperature` 节点。
2.  节点计算第一个表达式 `"metadata.temperature > 50.0"`。因为 `55.0 > 50.0`，结果为 `true`。
3.  节点立即将消息通过关系 `"HighTemp"` 发送到 `node-high-temp-alert` 节点，并停止计算。
4.  如果温度是 `25.0`，第一个表达式为 `false`，第二个为 `true`，消息将被路由到 `"NormalTemp"`。
5.  如果温度是 `-5.0`，前两个表达式为 `false`，第三个为 `true`，消息将被路由到 `"LowTemp"`。
6.  如果 `metadata.temperature` 字段不存在，所有 `cases` 表达式都会计算失败或返回 `false`，消息将被路由到默认的 `"Unknown"` 关系。

# 5. 数据契约 (DataContract)

*   **输入**: 任意 `RuleMsg`。节点会从 `metadata`, `dataT`, `data` 中读取数据用于表达式计算。
*   **输出**: **原始的、未被修改的 `RuleMsg`**。此节点是一个纯粹的路由组件，它**不会**以任何方式改变消息的内容。

# 6. 错误处理 (ErrorHandling)

*   **表达式编译失败**: 如果 `cases` 中的任何一个表达式语法不正确，会导致节点初始化失败或在运行时报错，并将消息路由到 `Failure` 链路。
*   **表达式执行失败**: 如果表达式在执行时出错（例如，访问了一个不存在的对象字段），会将消息路由到 `Failure` 链路。
*   **无匹配且无默认**: 如果所有 `cases` 都不匹配，并且没有配置 `defaultRelation`，消息将被路由到 `Failure` 链路，并附带 `ErrNoMatchCase` 错误。

<!-- 链接定义区域 -->
[Guide-MatrixOverview-2b3c4d]: ../00_matrix_guide.md
[Ref-SemanticDoc-d45bce]: ../../reference/04_semantic_documentation_standard.md
