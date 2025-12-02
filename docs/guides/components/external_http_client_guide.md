---
# === Node Properties: 定义文档节点自身 ===
uuid: "d2c1b0a9-6e7a-4b8c-9d0e-1f2a3b4c5d6e"
type: "ComponentGuide"
title: "组件指南：HTTP客户端 (external/httpClient)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "external"
  - "http"
  - "rest"
  - "api"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的功能组件之一。"
  - type: "references"
    target_uuid: "81080378-a3e9-41ee-86ed-807193d45bce"
    description: "本文档遵循语义化文档规范编写。"
---

# 1. 功能概述 (FunctionalOverview)

> **⚠️ 注意：此组件的设计尚未完全稳定，其配置结构未来可能会有变更，暂不建议在生产环境中使用。**

`external/httpClient` 是一个功能强大的外部交互节点，用于在规则链中发起高度可配置的HTTP/HTTPS请求。

它的核心是一个**声明式映射引擎**。开发者通过在节点配置中定义 `request` 和 `response` 的映射规则，可以精确地控制 `RuleMsg` (包含 `Data`, `DataT`, `Metadata`) 与 `http.Request` / `http.Response` 之间的双向数据转换，实现了业务数据与HTTP协议细节的完全解耦。

# 2. 如何配置 (Configuration)

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `defaultTimeout` | 默认超时 | 请求的默认超时时间，例如 `"10s"`, `"500ms"`。 | `duration` | 否 | `"30s"` |
| `proxyUrl` | 代理URL | 如果需要通过HTTP代理发送请求，在此处配置代理服务器的URL。 | `string` | 否 | `""` |
| `request` | 请求映射 | 定义了如何从输入的 `RuleMsg` 构建 `http.Request`。 | `HttpRequestMap` | 是 | N/A |
| `response` | 响应映射 | 定义了如何将 `http.Response` 解析并写回到输出的 `RuleMsg` 中。 | `HttpResponseMap` | 是 | N/A |

## 2.1. 请求映射 (`HttpRequestMap`) (RequestMap)

| 字段 | 描述 |
| :--- | :--- |
| `url` | 请求的目标URL。支持使用 `${...}` 占位符动态替换来自 `metadata` 或 `dataT` 的值。 |
| `method` | HTTP方法 (e.g., `GET`, `POST`)。同样支持 `${...}` 占位符。 |
| `headers` | 定义请求头的映射规则。 |
| `queryParams` | 定义URL查询参数的映射规则。 |
| `body` | 定义请求体的映射规则。 |
| `propagateMeta` | **(✓ 已实现)** 是否自动将 `RuleMsg` 中符合W3C Trace Context规范的元数据（如 `traceparent`）作为请求头传播，以实现分布式链路追踪。 |
| `propagateKeys` | **(✓ 已实现)** 当 `propagateMeta` 为 `true` 时，可以额外指定需要强制传播的元数据键列表。 |

## 2.2. 响应映射 (`HttpResponseMap`) (ResponseMap)

| 字段 | 描述 |
| :--- | :--- |
| `statusCodeTarget` | (可选) 一个 `metadata` 键，用于存储响应的HTTP状态码。 |
| `latencyMsTarget` | (可选) 一个 `metadata` 键，用于存储请求的端到端延迟（毫秒）。 |
| `errorTarget` | (可选) 一个 `metadata` 键，用于存储网络或连接错误信息。 |
| `startTimeMsTarget` | (可选) 一个 `metadata` 键，用于存储请求开始时间的Unix毫秒时间戳。 |
| `endTimeMsTarget` | (可选) 一个 `metadata` 键，用于存储请求结束时间的Unix毫秒时间戳。 |
| `headers` | 定义响应头的映射规则。 |
| `body` | 定义响应体的映射规则。 |

# 3. 核心概念：映射源 (`HttpMappingSource`) (MappingSource)

`HttpMappingSource` 是 `httpClient` 节点配置的灵魂，它被用于 `headers`, `queryParams`, 和 `body` 的映射中，定义了数据“从哪里来，到哪里去”。它包含两种模式，可以独立或组合使用。

### 3.1. 动态映射 (`from`) (DynamicMapping)

动态映射用于将一个完整的 `map` 或 `struct` 对象整体映射为HTTP的一部分。

*   **配置**: `{"from": {"path": "dataT.myObj"}}`
*   **工作原理**:
    *   当用于 `headers` 或 `queryParams` 时，`dataT.myObj` 必须是一个 `map[string]interface{}` 结构，它的每个键值对会被转换为一个Header或Query Parameter。
    *   当用于 `body` 时，`dataT.myObj` 会被整个序列化为JSON作为请求体。
    *   **特殊路径 `data`**: 如果 `path` 设置为 `"data"`，节点会直接将 `msg.Data()` 的内容作为请求体，并根据 `msg.DataFormat()` 自动设置 `Content-Type`。

### 3.2. 静态映射 (`params`) (StaticMapping)

静态映射用于逐个字段地进行精确映射，提供了更细粒度的控制。

*   **配置**:
    ```json
    "params": [
      {
        "name": "X-Auth-Token",
        "mapping": { "from": "metadata.authToken" }
      },
      {
        "name": "userId",
        "mapping": { "from": "dataT.userInfo.id" }
      }
    ]
    ```
*   **工作原理**:
    *   `name`: 定义了目标HTTP元素的名称（Header名, Query Parameter名, 或JSON Body的字段名）。
    *   `mapping.from`: 定义了数据来源的路径。

### 3.3. 组合使用 (CombinedUsage)

当 `from` 和 `params` 同时存在时，节点会先处理 `from` 的动态映射，然后用 `params` 的静态映射进行补充或覆盖。这在“发送一个通用对象，但动态修改其中一两个字段”的场景中非常有用。

### 3.4. 数据来源路径 (`from` 路径) 语法 (SourcePathSyntax)

| 路径示例 | 描述 |
| :--- | :--- |
| `metadata.myKey` | 从消息元数据中读取 `myKey` 的值。 |
| `dataT.myObj.field` | 从 `dataT` 中 `id` 为 `myObj` 的对象的 `field` 字段读取值。 |
| `'a literal string'` | 使用单引号包裹的字符串字面量。 |
| `data` | **(仅用于请求体)** 直接使用 `msg.Data()` 的原始字符串内容。 |

# 4. 配置示例 (Examples)

## 4.1. 示例1：POST JSON对象并接收JSON响应 (ExamplePostJson)

假设输入消息 `dataT` 中有一个 `id` 为 `userInfo` 的对象，内容为 `{"userId": 123, "userName": "Alice"}`。

```json
{
  "id": "node-call-user-api",
  "type": "external/httpClient",
  "name": "调用用户服务API",
  "configuration": {
    "request": {
      "url": "http://api.example.com/users/${dataT.userInfo.userId}",
      "method": "POST",
      "headers": {
        "params": [
          { "name": "Content-Type", "mapping": { "from": "'application/json'" } }
        ]
      },
      "body": {
        "from": { "path": "dataT.userInfo" }
      }
    },
    "response": {
      "statusCodeTarget": "httpStatusCode",
      "body": {
        "from": { "path": "dataT.apiResult", "defineSid": "map_string_interface" }
      }
    }
  }
}
```
**流程解析**:
1.  **构建请求**:
    *   URL中的 `${dataT.userInfo.userId}` 被替换为 `123`，最终URL为 `http://api.example.com/users/123`。
    *   `Content-Type` 请求头被设置为 `application/json`。
    *   `dataT.userInfo` 对象被序列化为 `{"userId": 123, "userName": "Alice"}` 作为请求体。
2.  **处理响应**:
    *   HTTP状态码被写入 `metadata.httpStatusCode`。
    *   完整的JSON响应体被解析并存入 `dataT` 中一个 `id` 为 `apiResult` 的新对象中。`defineSid` 确保了即使 `apiResult` 对象不存在，也会被自动创建。

## 4.2. 示例2：GET 请求与查询参数映射 (ExampleGetWithQuery)

假设输入消息 `dataT` 中有一个 `id` 为 `queryParams` 的对象，内容为 `{"page": 1, "pageSize": 10}`。

```json
{
  "id": "node-get-list",
  "type": "external/httpClient",
  "configuration": {
    "request": {
      "url": "http://api.example.com/items",
      "method": "GET",
      "queryParams": {
        "from": { "path": "dataT.queryParams" }
      }
    },
    "response": {
      "body": {
        "from": { "path": "dataT.itemsList", "defineSid": "map_string_interface" }
      }
    }
  }
}
```
**流程解析**:
*   `dataT.queryParams` 对象被展开为URL查询参数，最终请求URL为 `http://api.example.com/items?page=1&pageSize=10`。

## 4.3. 示例3：提取响应中的特定字段 (ExampleExtractFields)

假设API返回的JSON为 `{"data": {"user": {"id": 456}}, "error_code": 0}`。

```json
{
  "id": "node-extract-fields",
  "type": "external/httpClient",
  "configuration": {
    "request": { "...": "..." },
    "response": {
      "body": {
        "params": [
          { "name": "data.user.id", "mapping": { "to": "metadata.extractedUserId" } },
          { "name": "error_code", "mapping": { "to": "dataT.apiResult.errorCode", "defineSid": "MapStringInterfaceV1_0" } }
        ]
      }
    }
  }
}
```
**流程解析**:
*   从响应体中提取 `data.user.id` 的值 (`456`)，并写入 `metadata.extractedUserId`。
*   从响应体中提取 `error_code` 的值 (`0`)，并写入 `dataT` 中 `apiResult` 对象的 `errorCode` 字段。如果 `apiResult` 对象不存在，将使用 `MapStringInterfaceV1_0` 的定义来创建它。

## 4.4. 示例4：组合映射请求体 (ExampleCombineRequestBody)

假设 `dataT.baseInfo` 为 `{"common": "value"}`，同时 `metadata.dynamicId` 为 `"xyz"`。

```json
{
  "id": "node-combine-body",
  "type": "external/httpClient",
  "configuration": {
    "request": {
      "url": "http://api.example.com/complex",
      "method": "POST",
      "body": {
        "from": { "path": "dataT.baseInfo" },
        "params": [
          { "name": "dynamic_field", "mapping": { "from": "metadata.dynamicId" } },
          { "name": "static_field", "mapping": { "from": "'static_value'" } }
        ]
      }
    },
    "response": { "...": "..." }
  }
}
```
**流程解析**:
*   节点首先将 `dataT.baseInfo` 的内容 (`{"common": "value"}`) 作为基础请求体。
*   然后，它会添加或覆盖 `params` 中定义的字段。
*   最终发送的请求体为: `{"common":"value", "dynamic_field":"xyz", "static_field":"static_value"}`。

## 4.5. 示例5：映射所有响应元信息 (ExampleMapAllMetadata)

```json
{
  "id": "node-map-all-meta",
  "type": "external/httpClient",
  "configuration": {
    "request": {
      "url": "http://api.example.com/status/200",
      "method": "GET"
    },
    "response": {
      "statusCodeTarget": "http.status_code",
      "latencyMsTarget": "http.latency_ms",
      "errorTarget": "http.error",
      "startTimeMsTarget": "http.start_time_ms",
      "endTimeMsTarget": "http.end_time_ms"
    }
  }
}
```
**流程解析**:
*   请求成功后，`metadata` 中会包含类似如下的键值对:
    *   `http.status_code`: `"200"`
    *   `http.latency_ms`: `"53"` (示例值)
    *   `http.start_time_ms`: `"1678886400000"` (示例值)
    *   `http.end_time_ms`: `"1678886400053"` (示例值)
*   如果请求发生网络错误，`metadata.http.error` 字段将被填充，例如 `connect: connection refused`。

# 5. 数据契约 (DataContract)

*   **输入**: 节点通过 `request` 配置从 `RuleMsg` 的 `Data`, `DataT`, `Metadata` 中读取数据。
*   **输出**: 节点**总是创建一个新的 `RuleMsg` 副本**作为输出，以避免对原始消息的副作用，这对于并行处理分支至关重要。它通过 `response` 配置将HTTP响应的各个部分写回到这个新消息的 `Data`, `DataT`, `Metadata` 中。

# 6. 错误处理 (ErrorHandling)

节点会将操作结果通过 `Success` 或 `Failure` 链路传递下去。

| 错误对象名 | 错误码 | 描述 |
| :--- | :--- | :--- |
| `ErrInvalidParams` | (N/A) | 构建请求时发生错误（如URL占位符无法替换、代理URL格式错误）。 |
| `ErrHttpSendFailed` | `202503002` | 发生网络层错误（如DNS解析失败、连接被拒）。错误详情会被记录在 `response.errorTarget` 指定的 `metadata` 字段中。 |
| `ErrInternal` | (N/A) | 映射响应回 `RuleMsg` 时发生严重错误。 |

**注意**: 即使HTTP响应状态码是 `4xx` 或 `5xx`，只要请求成功发出并收到响应，节点也会走向 `Success` 链路。状态码会记录在 `response.statusCodeTarget` 指定的 `metadata` 字段中，由后续节点判断和处理。

# 7. 问答环节 (FrequentlyAskedQuestions)
<!-- qa_section_start -->
> **问：为什么需要 `defineSid` 字段？**
> **答：** 当你希望将HTTP响应的一部分（如整个Body或所有Headers）映射到 `dataT` 中一个**可能尚不存在**的对象时，`defineSid` 告诉Matrix引擎：“如果目标对象（如 `dataT.apiResult`）不存在，请使用这个SID（Schema ID）来创建一个新的空对象，然后再填入数据。” 这避免了因对象不存在而导致的映射失败。
>
> **重要提示**: `defineSid` 必须引用一个已在系统中通过 `CoreObjRegistry` 注册的 `CoreObj` 定义。对于通用的JSON数据，通常需要一个代表 `map[string]interface{}` 的 `CoreObj` 被注册，例如使用 `MapStringInterfaceV1_0` 作为其 `SID`。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview-2b3c4d]: ../00_matrix_guide.md
[Ref-SemanticDoc-d45bce]: ../../reference/04_semantic_documentation_standard.md
