---
# === Node Properties: 定义文档节点自身 ===
uuid: "d2c3b4a5-f6e7-8a9b-0c1d-2e3f4a5b6c7d"
type: "ComponentGuide"
title: "组件指南：解析与验证功能节点 (parseValidate)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "function"
  - "parse"
  - "json"
  - "transformation"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的功能组件之一。"
---

# 1. 功能概述 (Overview)

> **⚠️ 注意：此组件尚在开发设计中，功能未完全实现，暂不建议在生产环境中使用。**

`parseValidate` 是一个核心的数据转换功能节点。它的主要职责是将 `RuleMsg` 的 `Data` 字段中存储的原始JSON字符串，解析（反序列化）为一个在系统中已注册的、强类型的业务对象（`CoreObj`），并将其存入 `DataT` 容器中。

这个节点通常位于规则链的起始位置，扮演着“数据入口守卫”的角色，负责将外部非结构化的数据转化为引擎内部可高效处理的结构化数据。

**注意**: 当前版本的节点只实现了“解析”功能，“验证”功能为未来扩展保留。

# 2. 如何配置 (Configuration)

在规则链的DSL中，可以通过 `configuration` 字段为此节点配置以下参数：

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `targetCoreObjSid` | 目标CoreObj的SID | 要解析成的目标业务对象的系统ID (SID)。当前格式为对象类型名（如 `UserInfoV1`），未来可能扩展为完整的URI格式（如 `sid://my_app/types.UserInfoV1`）。 | `string` | 是 | `""` |
| `targetObjId` | 目标CoreObj的Key | 解析成功后，该业务对象在 `DataT` 容器中存储的键 (Key)。 | `string` | 是 | `""` |

## 2.1. 配置示例 (Example)

假设我们有一个已注册的业务对象 `UserInfoV1`，其SID当前定义为 `"UserInfoV1"`。

```json
{
  "id": "node-parse-user-info",
  "type": "functions/parseValidate",
  "name": "解析用户信息",
  "description": "将输入的JSON解析为UserInfoV1对象",
  "configuration": {
    "targetCoreObjSid": "UserInfoV1",
    "targetObjId": "mainUserInfo"
  }
}
```

# 3. 数据契约 (DataContract)

本节点是连接 `Data` 和 `DataT` 的桥梁。

## 3.1. 输入 (Input)

*   **`msg.Data()` (string)**:
    *   节点从 `RuleMsg` 的 `Data` 字段读取一个JSON字符串。这个字符串应该是 `targetCoreObjSid` 对应结构体的JSON表示。

## 3.2. 输出 (Output)

*   **`msg.DataT()` (DataT)**:
    *   节点**不会修改** `msg.Data()` 字段。
    *   节点会将解析出的业务对象实例，通过 `configuration` 中指定的 `targetObjId` 作为键，存入 `msg.DataT()` 容器。
    *   执行此节点后，下游节点就可以通过 `msg.DataT().Get("mainUserInfo")` 来获取这个强类型的 `UserInfoV1` 对象实例。

# 4. 错误处理 (ErrorHandling)

如果解析过程中发生错误，节点将通过 `TellFailure` 关系传递错误信息。常见的错误包括：
*   **配置缺失**: `targetCoreObjSid` 或 `targetObjId` 未在 `configuration` 中提供。
*   **SID无效**: 提供的 `targetCoreObjSid` 未在系统的 `CoreObjRegistry` 中注册。
*   **JSON反序列化失败**: `msg.Data()` 中的字符串不是一个有效的JSON，或者其结构与 `targetCoreObjSid` 对应的Go结构体不匹配。

<!-- qa_section_start -->
> **问：这个节点和直接在业务节点里 `json.Unmarshal` 有什么区别？**
> **答：** 主要区别在于**职责分离**和**性能**。通过使用 `parseValidate` 节点，我们将“数据格式转换”这一通用职责从具体的业务逻辑中剥离出来，使得业务节点可以专注于处理已经验证和结构化的数据。此外，这种模式避免了在多个后续节点中重复进行反序列化操作，提升了整个规则链的执行效率。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview]: ../00_matrix_guide.md
