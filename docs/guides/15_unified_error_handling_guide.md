---
uuid: "ab800e51-847f-4523-b330-9f14e71caf29"
type: "Guide"
title: "指南：统一错误处理与 HTTP 错误映射"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
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

`ServiceError` 是面向外部协议的错误对象，当前主要包含：

1. `ResponseCode`
2. `UserMessage`
3. `Cause`
4. `FailureInfo`

它是 HTTP endpoint 最后返回给外部调用方的标准包装。

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

## 5. 当前推荐实践 (RecommendedPractices)

1. 新节点或新函数应尽量复用已有 `cnst.ErrCode`，避免随意发明字符串错误码。
2. 需要对外暴露稳定协议错误时，优先通过 `Fault -> FailureInfo -> ServiceError` 这条链路，不要在 endpoint 层零散拼字符串。
3. 如果某个 endpoint 需要对不同 fault code 返回不同 HTTP 状态，请把策略集中写在 `errorMappings`。
4. 如果只是内部排障，不建议直接把原始 `Cause` 文本透传给用户端。

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
