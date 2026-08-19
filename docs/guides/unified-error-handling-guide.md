---
uuid: "ab800e51-847f-4523-b330-9f14e71caf29"
type: "Guide"
title: "指南：统一错误处理与 HTTP 错误映射"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "guide"
  - "error-handling"
  - "fault"
  - "http"
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "统一错误处理是 Matrix 当前运行时和 endpoint 行为的一部分。"
  - type: "references"
    target_uuid: "8f7e6a5d-1b2c-4e3f-9a0b-1c2d3e4f5a6b"
    description: "本文档给出 RFC-0010 当前已落地错误模型的使用方式。"
  - type: "references"
    target_uuid: "f5614284-7536-45c4-8342-b098f005394e"
    description: "HTTP 错误映射的请求/响应细节可继续参考 Reference-10。"
  - type: "references"
    target_uuid: "d1e2f3a4-b5c6-d7e8-f9a0-b1c2d3e4f5a6"
    description: "错误元数据的传播仍建立在 RuleMsg / metadata 模型之上。"
---

# 1. 功能概述 (Overview)

Matrix 当前已经形成一套可落地的统一错误处理链路：

1. 节点或函数内部用 `Fault` 表示结构化错误
2. runtime 把错误上下文写进 `RuleMsg.metadata`
3. endpoint/http 从 metadata 或执行错误中重建 `FailureInfo`
4. 再对外包装成 `ServiceError`
5. 最后根据 `errorMappings` 决定 HTTP 响应码
6. 通过统一公开呈现层输出安全 `message`，不序列化内部 `Cause`

它对应 [RFC-0010](../designs/rfc/0010_unified_error_handling_rfc.md) 的当前实现范围，但需要明确：**目前稳定落地的协议边界主要是 HTTP endpoint。**

## 2. 三层错误对象怎么理解 (CoreModel)

### 2.1 `Fault`

`Fault` 是开发时定义的**结构化错误规范**，重点是：

1. 有稳定错误码
2. 有明确消息
3. 可以继续包装底层 Go error

适合在节点、函数和 helper 中作为第一手错误表达。

### 2.2 `FailureInfo`

`FailureInfo` 是运行时失败快照，携带：

1. `error`
2. `error_node_id`
3. `error_node_name`
4. `error_timestamp`
5. `error_code`

它更接近“这次执行里哪一个节点、在什么时间、以什么错误码失败了”。

### 2.3 `ServiceError`

`ServiceError` 是协议边界的错误映射对象，当前主要包含：

1. `ResponseCode`
2. `UserMessage`
3. `Cause`
4. `FailureInfo`

其中只有 `UserMessage` 被声明为可公开展示。`Cause`、`FailureInfo.Error`、节点标识和原始输入属于内部诊断信息，供错误映射以及宿主显式写入日志或 Trace 使用，不能把整个 `ServiceError.Error()` 直接写入 HTTP body。

## 3. 节点 / 函数侧应该怎么写 (ProducerSide)

### 3.1 定义稳定 Fault

```go
var DefInvalidTenantID = &types.Fault{
    Code:    cnst.CodeInvalidParams,
    Message: "invalid tenant id",
}
```

### 3.2 传播错误

推荐模式：

1. 业务校验失败时返回或包装 `Fault`
2. 让节点通过 `ctx.HandleError(...)`、`ctx.TellFailure(...)` 或标准失败路径往上抛

这样 runtime 才能把错误码和节点上下文完整写入 metadata。

## 4. HTTP Endpoint 如何映射 (HttpEndpointMapping)

`endpoint/http` 当前会从两类来源构造 `ServiceError`：

1. 规则链最终返回的执行错误
2. 最终消息 metadata 中的错误上下文

然后按 `errorMappings` 做响应码映射。

### 4.1 DSL 示例

```json
{
  "id": "create_user_endpoint",
  "type": "endpoint/http",
  "configuration": {
    "httpMethod": "POST",
    "httpPath": "/users",
    "ruleChainId": "create_user_chain",
    "errorMappings": {
      "400": ["40004000", "40005000"],
      "404": ["40404000"],
      "409": ["40901000"]
    }
  }
}
```

解释：

1. key 是返回给客户端的 HTTP status code
2. value 是内部 `Fault.Code` 映射到的错误码列表
3. 如果没有命中映射，就使用 endpoint 定义里的默认错误码，或者回退到 `500`

### 4.2 统一公开错误映射

当前 HTTP 错误出口统一遵循以下优先级：

1. `Fault.Code` / `FailureInfo.Code` 先通过 DSL `errorMappings` 映射 HTTP status。
2. 宿主如果注入 `ServiceErrorAspect`，可按结构化错误码把 status 和 `UserMessage` 映射为产品可展示结果。
3. 未配置产品映射时，Matrix 按最终 HTTP status 使用安全兜底文案。
4. HTTP writer 只读取 `ServiceError.UserMessage`，并可从错误链的 `PublicErrorDetails() any` 读取显式公开的结构化 `details`；普通 `error.Error()`、`Cause` 和 `FailureInfo.Error` 不进入响应。

默认兜底映射保持协议级、与具体业务无关：

| HTTP status | 默认公开文案 |
| --- | --- |
| `400` / `422` | `invalid request` |
| `401` | `authentication required` |
| `403` | `permission denied` |
| `404` | `resource not found` |
| `409` | `request conflict` |
| `429` | `too many requests` |
| `502` / `503` | `service unavailable` |
| 其他 `4xx` | `request failed` |
| 其他 `5xx` | `internal server error` |

Matrix 默认响应结构保持兼容：

```json
{
  "code": 400,
  "message": "invalid request",
  "details": {
    "reason_code": "EXPLICIT_PUBLIC_REASON"
  }
}
```

`details` 为可选字段；上例只适用于错误类型主动实现 public-details provider 的场景。没有该接口时响应仍只有 `code/message`。

例如请求体里的 Product URL 解码失败时，`FailureInfo` 仍保留 `202501004` 和原始解码链路，公开响应只返回安全文案，不再出现字段路径、SID、URL 原值或重复的 `cause`。

### 4.3 产品层映射边界

Matrix 只负责通用协议级兜底，不硬编码 SellItX 等产品名称、中文文案或业务错误码。产品宿主应在 HTTP 边界按结构化错误码映射：

1. 通过 `ServiceErrorAspect` 生成安全、可本地化的 `UserMessage` 和必要的 HTTP status。
2. 如果产品采用 `{data, error, execution_id}` 业务 envelope，由模块或 server adapter 生成稳定的业务 `error.code`。
3. `details` / `field_errors` 只能来自显式声明为 public 的校验结果，禁止从 `Cause` 或错误字符串自动提取。
4. 字段级即时提示优先在表单 schema 中实现；后端仍保留相同约束作为信任边界。

## 5. 当前推荐实践 (RecommendedPractices)

1. 新节点或新函数应尽量复用已有 `cnst.ErrCode`，避免随意发明字符串错误码。
2. 需要对外暴露稳定协议错误时，优先通过 `Fault -> FailureInfo -> ServiceError` 这条链路，不要在 endpoint 层零散拼字符串。
3. 如果某个 endpoint 需要对不同 fault code 返回不同 HTTP 状态，请把策略集中写在 `errorMappings`。
4. 原始 `Cause`、`FailureInfo.Error`、节点路径和请求值不得透传给用户端；宿主应按需结合 execution ID 写入受控日志或 Trace。
5. `ServiceErrorAspect` 必须基于结构化错误码映射，禁止用正则解析内部错误文案。
6. `types.ServiceErrorAspect` 是 HTTP `ServiceError` 映射的唯一公开扩展契约；不在 endpoint 内或宿主层并行定义 `ErrorConverter` 之类同义接口。

## 6. 当前边界 (CurrentScope)

这套机制当前已经稳定覆盖：

1. 节点 / 函数内的 `Fault`
2. runtime 侧 metadata 传播
3. HTTP endpoint 的错误码映射与 `ServiceError` 输出

但不应默认认为这些边界已经完全统一：

1. gRPC
2. WebSocket
3. 其他未来主动 / 被动协议入口

## 7. 相关现行文档 (RelatedDocs)

1. [RFC-0010：Unified Error Handling](../designs/rfc/0010_unified_error_handling_rfc.md)
2. [参考-10：HttpEndpoint 节点深度解析](../reference/10_http_endpoint_deep_dive.md)
3. [参考：Matrix 消息设计哲学 (RuleMsg)](../reference/06_message_design_philosophy.md)
