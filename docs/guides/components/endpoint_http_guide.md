---
# === Node Properties: 定义文档节点自身 ===
uuid: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
type: "ComponentGuide"
title: "组件指南：HTTP端点 (endpoint/http)"
status: "Draft"
owner: "neohetj"
version: "2.1.0"
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
    description: "本节点是 Matrix 规则链的核心入口点之一。"
  - type: "references"
    target_uuid: "a2b1c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
    description: "本文档遵循 guides/components 目录的组件指南规范。"
---

# 1. 功能概述

`endpoint/http` 是 Matrix 的被动 HTTP 入口。它负责：

1. 监听指定的 `httpMethod + httpPath`
2. 按 `endpointDefinition.request` 把 HTTP 请求映射为 `RuleMsg`
3. 执行目标规则链
4. 按 `endpointDefinition.response` 把最终 `RuleMsg` 映射回 HTTP 响应

> 深入实现细节见 **[参考-10: HttpEndpoint 节点深度解析][Ref-HttpEndpointDeepDive]**。

# 2. 顶层配置

| 配置键 | 描述 |
| :--- | :--- |
| `ruleChainId` | 要触发的规则链 ID |
| `startNodeId` | 可选起始节点 ID |
| `httpMethod` | 监听的 HTTP 方法 |
| `httpPath` | 监听路径，支持 `:param` |
| `domain` | 可选业务领域名称，用于 Morpheus 等宿主界面分组 |
| `description` | 描述 |
| `summary` | OpenAPI 摘要 |
| `tags` | OpenAPI / UI 标签 |
| `async` | 是否异步执行 |
| `endpointDefinition` | 请求/响应映射定义 |
| `errorMappings` | HTTP 状态与内部 Fault 的映射 |

`domain` 是 endpoint 的业务配置，宿主界面应优先使用它进行领域分组；缺失时宿主可以继续按 endpoint ID 命名约定兜底推导。

```json
{
  "id": "ep-identityx-admin-users-list",
  "type": "endpoint/http",
  "name": "IdentityX Admin List Users Endpoint",
  "configuration": {
    "domain": "Admin",
    "httpMethod": "GET",
    "httpPath": "/api/identityx/admin/users"
  }
}
```

# 3. `endpointDefinition` 结构

当前实现使用的是 `EndpointIOField` / `EndpointIOPacket`，而不是旧版 `mapping.to` / `mapping.defineSid` 结构。

## 3.1. 请求侧

`request` 对应 `types.HttpRequestDef`：

```go
type HttpRequestDef struct {
    PathParams  []EndpointIOField `json:"pathParams,omitempty"`
    QueryParams EndpointIOPacket  `json:"queryParams,omitempty"`
    Headers     EndpointIOPacket  `json:"headers,omitempty"`
    Body        EndpointIOPacket  `json:"body,omitempty"`
}
```

### `EndpointIOField`

| 字段 | 描述 | 示例 |
| :--- | :--- | :--- |
| `name` | 外部协议中的字段名 | `deviceId`, `X-Request-Id`, `data.temperature` |
| `bindPath` | 写入 `RuleMsg` 的 URI | `rulemsg://metadata/deviceId`, `rulemsg://dataT/telemetry.temp?sid=TelemetryData` |
| `type` | 类型转换目标 | `string`, `int`, `float`, `bool`, `object`, `string[]` |
| `required` | 是否必填 | `true` |
| `defaultValue` | 缺省值 | `"ok"` |
| `description` | 描述 | `"设备 ID"` |

### `EndpointIOPacket`

| 字段 | 描述 |
| :--- | :--- |
| `mapAll` | 把整个 packet 映射到一个 `RuleMsg` URI |
| `fields` | 对 packet 内部字段逐项映射 |

`MapAll` 和 `Fields` 可以同时存在：先整体映射，再用字段映射做补充或覆盖。

## 3.2. 响应侧

`response` 对应 `types.HttpResponseDef`：

```go
type HttpResponseDef struct {
    SuccessCode     int              `json:"successCode,omitempty"`
    ErrorStatusCode int              `json:"errorStatusCode,omitempty"`
    Body            EndpointIOPacket `json:"body,omitempty"`
    Headers         EndpointIOPacket `json:"headers,omitempty"`
}
```

响应映射会从最终 `RuleMsg` 提取数据，再构造 HTTP Body / Headers。

# 4. 配置示例

下面是一个当前实现可识别的示例：

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
            "bindPath": "rulemsg://metadata/deviceId"
          }
        ],
        "headers": {
          "fields": [
            {
              "name": "X-Timestamp",
              "type": "int",
              "bindPath": "rulemsg://metadata/timestamp"
            }
          ]
        },
        "body": {
          "fields": [
            {
              "name": "temperature",
              "type": "float",
              "required": true,
              "bindPath": "rulemsg://dataT/telemetry.temp?sid=TelemetryData"
            },
            {
              "name": "humidity",
              "type": "float",
              "required": true,
              "bindPath": "rulemsg://dataT/telemetry.hum?sid=TelemetryData"
            }
          ]
        }
      },
      "response": {
        "successCode": 202,
        "body": {
          "fields": [
            {
              "name": "status",
              "defaultValue": "ok"
            },
            {
              "name": "processedAt",
              "bindPath": "rulemsg://metadata/processedTimestamp"
            }
          ]
        }
      }
    }
  }
}
```

流程解析：

1. `deviceId` 从路径参数写入 `metadata.deviceId`
2. `X-Timestamp` 写入 `metadata.timestamp`
3. 请求体字段写入 `DataT.telemetry`
4. 规则链执行完成后，从 `metadata.processedTimestamp` 回填响应体

# 5. 当前实现中的几个重要约束

1. 请求和响应映射都基于 `helper.ProcessInbound` / `ProcessOutbound`。
2. `bindPath` 必须是合法的 `rulemsg://...` URI，或者留空配合 `defaultValue` 使用。
3. 当 `bindPath` 指向 `DataT` 且对象不存在时，通常需要在 URI 中携带 `sid`，否则无法自动创建对象。
4. 如果 `MapAll` 指向非对象值，而同时又声明了 `fields`，运行时会报错。

# 6. 常见错误

| 错误现象 | 常见原因 |
| :--- | :--- |
| 请求字段无法写入 `RuleMsg` | `bindPath` 非法或 URI 缺少 `sid` |
| 响应体构造失败 | `MapAll` 指向非对象，但仍定义了 `fields` |
| 类型转换失败 | `type` 与实际值不匹配 |
| 规则链未执行 | 请求阶段映射失败，HTTP 入口提前返回错误 |

<!-- qa_section_start -->
> **问：`endpoint/http` 和 `external/httpClient` 有什么区别？**
> **答：** `endpoint/http` 是入口，负责把外部请求转成 `RuleMsg`；`external/httpClient` 是链内节点，负责从 `RuleMsg` 主动构造对外请求。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Ref-HttpEndpointDeepDive]: ../../reference/10_http_endpoint_deep_dive.md
