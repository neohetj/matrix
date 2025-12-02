---
# === Node Properties: 定义文档节点自身 ===
uuid: "a398c5b4-f6e7-4b0d-8c6a-1e2f3a4b5c6d"
type: "ComponentGuide"
title: "组件指南：循环 (action/forEach)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "action"
  - "loop"
  - "iterator"
  - "for-each"

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

`action/forEach` 是一个**循环控制**节点。它的核心功能是遍历一个来自输入消息的数组或切片，并为其中的**每一个元素**执行一个指定的子规则链。

这个节点是处理集合数据的关键，提供了同步/异步执行、错误处理和灵活的消息映射机制，适用于批量处理、数据扇出等多种场景。

# 2. 如何配置 (Configuration)

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `itemsExpression` | 遍历项表达式 | 一个路径表达式，用于从输入消息中提取要遍历的数组或切片。 | `string` | 是 | N/A |
| `chainId` | 子链ID | 为每个元素执行的子规则链的ID。 | `string` | 是 | N/A |
| `async` | 异步执行 | (可选) 是否异步（并行）执行所有子链。 | `boolean` | 否 | `false` |
| `continueOnError` | 出错时继续 | (可选) **仅在同步模式下有效**。如果为`true`，即使某个子链执行失败，循环也会继续。 | `boolean` | 否 | `false` |
| `messageScope` | 消息作用域 | (可选) 定义消息在循环迭代中的生命周期。 | `string` | 否 | `"INDEPENDENT"` |
| `inputMappings` | 输入映射 | **[核心]** 定义如何构建每个子链的输入消息。 | `InputMappingConfig[]` | 是 | `[]` |

# 3. 核心概念 (CoreConcepts)

## 3.1. 执行模式 (`async`) (ExecutionModes)

*   **同步模式 (`async: false`)**: 节点会按顺序为列表中的每个元素执行子链，并**等待**每一个子链执行完成后再开始下一个。如果 `continueOnError` 为 `false`，任何一个子链的失败都会导致整个 `forEach` 节点失败。
*   **异步模式 (`async: true`)**: 节点会**立即**为列表中的所有元素启动并行的子链执行，而**不会等待**它们完成。在这种模式下，`forEach` 节点本身总是会立即成功，并将原始消息传递给下一个节点。子链的执行结果（成功或失败）不会影响主链的流程。

## 3.2. 消息作用域 (`messageScope`) (MessageScopes)

*   **`INDEPENDENT` (独立作用域)**: 这是默认行为。在每次循环迭代开始时，都会创建一个全新的、干净的 `RuleMsg`。每次迭代都是完全隔离的。
*   **`SHARED` (共享作用域)**: 在循环开始前只创建一个 `RuleMsg`。这个消息会在所有的迭代中被**共享和复用**。这意味着，前一次迭代对消息的修改（例如，向 `dataT` 对象中添加一个值）对后续的迭代是可见的。这对于聚合计算（如累加、汇总）非常有用。

## 3.3. 输入映射 (`inputMappings`) (InputMappings)

这是 `forEach` 节点最强大的功能之一，它定义了如何为子链准备输入消息。

*   **`from`**: 定义数据来源。可以是父消息的任意路径（如 `metadata.key`, `dataT.obj.field`），也可以是一个特殊的关键字 `_item`，它代表当前正在遍历的那个元素。
*   **`to`**: 定义数据要写入子链消息的目标路径（如 `metadata.newKey`, `dataT.newItem`, `data`）。
*   **`defineSid`**: 当 `to` 指向一个 `dataT` 对象时，用于在对象不存在时自动创建它。

# 4. 配置示例 (Examples)

## 4.1. 示例1：同步处理设备列表 (SyncProcessing)

**场景**: 从 `dataT.deviceList` 中获取一个设备ID列表，然后为每个ID同步调用一个子链来获取设备详情。

```json
{
  "id": "node-get-device-details",
  "type": "action/forEach",
  "name": "遍历获取设备详情",
  "configuration": {
    "itemsExpression": "dataT.deviceList.ids",
    "chainId": "chain-get-single-device-detail",
    "async": false,
    "inputMappings": [
      {
        "from": "_item",
        "to": "metadata.deviceId"
      }
    ]
  }
}
```
**流程解析**:
1.  节点从 `dataT.deviceList.ids` 获取到一个数组，例如 `["id-001", "id-002"]`。
2.  **第一次迭代**: 创建一个新消息，将 `"id-001"` 写入其 `metadata.deviceId`，然后调用 `chain-get-single-device-detail` 并等待其完成。
3.  **第二次迭代**: 创建另一个新消息，将 `"id-002"` 写入其 `metadata.deviceId`，然后调用子链并等待。
4.  所有迭代成功后，`forEach` 节点将原始消息传递到 `Success` 链路。

## 4.2. 示例2：使用共享作用域进行聚合 (SharedScopeAggregation)

**场景**: 遍历一个数字列表，并调用一个子链来计算它们的总和。

**子链 (`chain-accumulator`)** 的核心节点是一个 `script` 节点，其代码为: `dataT.result.sum = dataT.result.sum + dataT.currentItem.value`。

```json
{
  "id": "node-sum-numbers",
  "type": "action/forEach",
  "name": "累加数字列表",
  "configuration": {
    "itemsExpression": "dataT.numberList.values",
    "chainId": "chain-accumulator",
    "async": false,
    "messageScope": "SHARED",
    "inputMappings": [
      {
        "from": "_item",
        "to": "dataT.currentItem.value",
        "defineSid": "NumberItem"
      },
      {
        "from": "{'sum': 0}",
        "to": "dataT.result",
        "defineSid": "SumResult"
      }
    ]
  }
}
```
**流程解析**:
1.  循环开始前，创建一个**共享消息**。`inputMappings` 被处理：`dataT.result` 对象被创建并初始化为 `{"sum": 0}`。
2.  **第一次迭代**: `_item` (例如 `10`) 被映射到 `dataT.currentItem.value`。子链执行，`dataT.result.sum` 变为 `10`。
3.  **第二次迭代**: `_item` (例如 `25`) 被映射到 `dataT.currentItem.value`。子链在**同一个共享消息**上执行，`dataT.result.sum` 变为 `35`。
4.  循环结束后，`forEach` 节点的输出消息（即那个共享消息）的 `dataT.result` 中将包含最终的累加结果。

# 5. 数据契约 (DataContract)

*   **输入**: 任意 `RuleMsg`，但 `itemsExpression` 必须能从中提取出一个数组或切片。
*   **子链输入**: 根据 `inputMappings` 构建的 `RuleMsg`。此外，子链消息的 `Metadata` 会被自动注入两个额外的键：
    *   `_loopIndex`: `string` 类型，当前迭代的索引（从 "0" 开始）。
    *   `is_last_item`: `string` 类型 (`"true"` 或 `"false"`)，表示当前是否为最后一次迭代。
*   **输出**: 原始的、未被修改的 `RuleMsg`。子链的执行结果**不会**直接合并回主链的消息中（除非使用 `SHARED` 作用域，此时输出消息即为那个被修改过的共享消息）。

# 6. 错误处理 (ErrorHandling)

*   **配置错误**: `itemsExpression` 或 `chainId` 未指定，或 `itemsExpression` 未返回一个切片，都会导致节点失败。
*   **同步模式**: 如果 `continueOnError: false`，任何一个子链的失败都会使整个 `forEach` 节点失败。
*   **异步模式**: 子链的失败只会被记录为错误日志，**不会**影响主链的执行流程。

<!-- 链接定义区域 -->
[Guide-MatrixOverview-2b3c4d]: ../00_matrix_guide.md
[Ref-SemanticDoc-d45bce]: ../../reference/04_semantic_documentation_standard.md
