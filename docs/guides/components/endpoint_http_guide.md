---
# === Node Properties: 定义文档节点自身 ===
uuid: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
type: "ComponentGuide"
title: "组件指南：HTTP端点 (endpoint/http)"
status: "Draft"
owner: "@cline"
version: "2.0.0"
tags:
  - "matrix"
  - "component"
  - "endpoint"
  - "http"
  - "rest"
  - "api"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix规则链的入口点之一。"
  - type: "references"
    target_uuid: "81080378-a3e9-41ee-86ed-807193d45bce"
    description: "本文档遵循语义化文档规范编写。"
---

# 1. 功能概述 (FunctionalOverview)

`endpoint/http` 是 `Matrix` 规则链的**核心入口点 (Entrypoint)** 之一。它的主要职责是监听一个特定的HTTP路径和方法，接收传入的HTTP请求，然后将其**声明式地**转换为一个标准的 `RuleMsg` 消息，并以此消息为起点，触发并执行一个指定的规则链。

> 关于本节点内部工作机制的深度解析，请参阅 **[参考-10: HttpEndpoint 节点深度解析][Ref-HttpEndpointDeepDive]**。

当规则链执行完毕后，它还会负责将最终的 `RuleMsg` 消息转换回一个标准的HTTP响应（包含状态码、头和JSON响应体），并返回给调用方。

# 2. 如何配置 (Configuration)

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `ruleChainId` | 规则链ID | 指定此端点要触发的规则链的ID。 | `string` | 是 | N/A |
| `startNodeId` | 起始节点ID | (可选) 指定规则链从哪个节点开始执行。**如果为空，则会从所有入度为0的节点并行开始执行**。<br/><br/>**⚠️ 警告**: 并行执行可能引发资源竞争问题（如多个分支同时修改同一`DataT`对象），在没有充分测试的情况下，**强烈建议总是明确指定一个起始节点**。 | `string` | 否 | `""` |
| `httpMethod` | HTTP方法 | 监听的HTTP方法，例如 `GET`, `POST`, `PUT`, `DELETE`。 | `string` | 是 | N/A |
| `httpPath` | HTTP路径 | 监听的HTTP路径，支持 `httprouter` 风格的路径参数，例如 `/api/v1/users/:userId`。 | `string` | 是 | N/A |
| `description` | 描述 | 对该端点功能的简短描述。 | `string` | 否 | `""` |
| `endpointDefinition` | 端点定义 | **[核心]** 定义了HTTP请求与 `RuleMsg` 之间的详细双向映射规则。 | `EndpointDefinitionObj` | 是 | N/A |

# 3. 核心概念：端点定义 (`EndpointDefinitionObj`) (EndpointDefinition)

这是 `http` 端点节点最核心、最强大的部分。它通过声明式的JSON结构，精确定义了数据如何在HTTP请求和 `RuleMsg` 之间流动。

`endpointDefinition` 包含两个主要部分：`request` 和 `response`。

## 3.1. 请求映射 (`request`) (RequestToMessage)

`request` 部分定义了如何将HTTP请求的四个部分（路径参数、查询参数、头、请求体）映射到 `RuleMsg` 的 `Metadata` 和 `DataT` 中。

每个部分都由一个 `HttpParam` 数组来定义。

### `HttpParam` 结构

| 字段 | 描述 | 示例 |
| :--- | :--- | :--- |
| `name` | 要从HTTP请求中提取的参数名。 | `"userId"`, `"X-Request-Id"`, `"user.name"` |
| `type` | 期望将参数值转换成的数据类型。 | `"string"`, `"int"`, `"bool"`, `"float"`, `"string[]"` |
| `required` | 该参数是否为必须。如果为 `true` 且请求中未找到，则请求会失败。 | `true` / `false` |
| `mapping.to` | **[核心]** 指定将提取并转换后的值写入 `RuleMsg` 的目标路径。 | `"metadata.userId"`, `"dataT.userObj.name"` |
| `mapping.defineSid` | **[核心]** 当 `mapping.to` 指向一个 `dataT` 对象时，此字段指定了该对象的SID。如果目标对象不存在，Matrix会使用此SID自动创建它。<br/><br/>关于SID及其背后的`CoreObj`数据契约的详细定义，请参阅 **[参考: 核心数据契约 (CoreObj)][Ref-CoreObj]**。 | `"User"` |

### 映射源

*   `pathParams`: 映射URL路径中的参数 (e.g., `/users/:userId` 中的 `userId`)。
*   `queryParams`: 映射URL查询字符串中的参数 (e.g., `/items?page=1` 中的 `page`)。
*   `headers`: 映射HTTP请求头。
*   `bodyFields`: 映射JSON请求体中的字段，支持使用点 `.` 进行嵌套访问。

## 3.2. 响应映射 (`response`) (MessageToResponse)

`response` 部分定义了如何从规则链执行完毕后的最终 `RuleMsg` 中提取数据，并构建成HTTP响应。

*   `successCode`: 定义了业务成功时返回的HTTP状态码，默认为 `200`。
*   `bodyFields`: 一个 `HttpParam` 数组，定义了如何从 `RuleMsg` 中提取数据并填充到JSON响应体中。
*   `headers`: 一个 `HttpParam` 数组，定义了如何从 `RuleMsg` 中提取数据并设置为HTTP响应头。

# 4. 配置示例 (Example)

假设我们要定义一个 `POST /api/v1/device/:deviceId/telemetry` 接口，用于接收设备遥测数据。

```json
{
  "id": "ep-post-telemetry",
  "type": "endpoint/http",
  "name": "接收设备遥测数据",
  "configuration": {
    "ruleChainId": "rc-telemetry-processing",
    "httpMethod": "POST",
    "httpPath": "/api/v1/device/:deviceId/telemetry",
    "description": "接收并处理来自设备的遥测数据",
    "endpointDefinition": {
      "request": {
        "pathParams": [
          {
            "name": "deviceId",
            "type": "string",
            "required": true,
            "mapping": { "to": "metadata.deviceId" }
          }
        ],
        "headers": [
          {
            "name": "X-Timestamp",
            "type": "int",
            "required": false,
            "mapping": { "to": "metadata.timestamp" }
          }
        ],
        "bodyFields": [
          {
            "name": "temperature",
            "type": "float",
            "required": true,
            "mapping": { "to": "dataT.telemetry.temp", "defineSid": "TelemetryData" }
          },
          {
            "name": "humidity",
            "type": "float",
            "required": true,
            "mapping": { "to": "dataT.telemetry.hum", "defineSid": "TelemetryData" }
          }
        ]
      },
      "response": {
        "successCode": 202,
        "bodyFields": [
          {
            "name": "status",
            "mapping": { "to": "'ok'" }
          },
          {
            "name": "processedAt",
            "mapping": { "to": "metadata.processedTimestamp" }
          }
        ]
      }
    }
  }
}
```
**流程解析**:
1.  当一个 `POST` 请求到达 `/api/v1/device/SN-001/telemetry` 时：
2.  `deviceId` (`"SN-001"`) 被提取并写入 `metadata.deviceId`。
3.  `X-Timestamp` 请求头的值被提取并写入 `metadata.timestamp`。
4.  请求体 `{"temperature": 25.5, "humidity": 60.0}` 被解析。
5.  `temperature` 的值 `25.5` 被写入 `dataT` 中 `id` 为 `telemetry` 的对象的 `temp` 字段。由于 `defineSid` 的存在，如果 `telemetry` 对象不存在，会先用 `TelemetryData` 这个SID创建它。
6.  `humidity` 的值 `60.0` 被写入同一个对象的 `hum` 字段。
7.  这个构建好的 `RuleMsg` 被送入 `rc-telemetry-processing` 规则链执行。
8.  规则链执行完毕后，假设最终的 `RuleMsg` 的 `metadata` 中包含了 `{"processedTimestamp": 1678886400000}`。
9.  节点会构建一个HTTP响应体 `{"status": "ok", "processedAt": 1678886400000}`，并以 `202 Accepted` 状态码返回。

# 5. 错误处理 (ErrorHandling)

节点在请求转换阶段可能会因为不匹配 `endpointDefinition` 的定义而失败。

| 错误对象名 | 错误码 | 描述 |
| :--- | :--- | :--- |
| `ErrRequestDecodingFailed` | `104001` | 解析请求体失败（例如，非法的JSON）。 |
| `ErrRequiredFieldMissing` | `104002` | 请求中缺少 `required: true` 的字段。 |
| `ErrFieldConversionFailed` | `104003` | 字段值无法转换为 `type` 定义的目标类型。 |
| `ErrInvalidMappingFormat` | `104004` | `mapping.to` 的路径格式不正确。 |
| `ErrDataTItemCreationFailed`| `104005` | 使用 `defineSid` 创建新的 `DataT` 对象失败。 |

这些错误会直接导致请求失败，并返回一个标准的HTTP错误响应，而不会触发规则链。

# 6. 问答环节 (FrequentlyAskedQuestions)
<!-- qa_section_start -->
> **问：`http` 端点和 `httpClient` 节点有什么区别？**
> **答：** 它们是规则链数据流动的“起点”和“中间点”。`http` 端点是**被动**的，它**接收**外部的HTTP请求并发起一次规则链执行。`httpClient` 节点是**主动**的，它在规则链**执行过程中**，向外部发起一次HTTP请求。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview-2b3c4d]: ../00_matrix_guide.md
[Ref-SemanticDoc-d45bce]: ../../reference/04_semantic_documentation_standard.md
[Ref-CoreObj]: ../../reference/09_core_objects.md
[Ref-HttpEndpointDeepDive]: ../../reference/10_http_endpoint_deep_dive.md
