---
uuid: "8f7e6a5d-1b2c-4e3f-9a0b-1c2d3e4f5a6b"
type: "RFC"
title: "需求：Unified Error Handling"
status: "Accepted"
owner: "neohetj"
version: "2.1.0"
tags:
  - "rfc"
  - "design"
  - "error-handling"
relations:
  - type: "is_supported_by"
    target_uuid: "d1e2f3a4-b5c6-d7e8-f9a0-b1c2d3e4f5a6"
    description: "统一错误处理仍建立在 RuleMsg 与 metadata 传播模型之上。"
  - type: "is_explained_by"
    target_uuid: "ab800e51-847f-4523-b330-9f14e71caf29"
    description: "当前统一错误处理与 HTTP 错误映射的使用方式见指南文档。"
  - type: "references"
    target_uuid: "f5614284-7536-45c4-8342-b098f005394e"
    description: "HTTP endpoint 错误处理的当前实现细节以 Reference-10 为准。"
  - type: "is_specified_by"
    target_uuid: "a18a0c1b-d599-4f8d-a949-51a60182873d"
    description: "Matrix core 内部 Fault.Code 的 aabbbcccc 编码和分配规则以当前 Reference 为准。"
---

# RFC: Unified Error Handling

## 1. 摘要

这份 RFC 的核心模型已经在当前代码中落地：

1. `Fault`
2. `FailureInfo`
3. `ServiceError`
4. Endpoint 级错误码映射

但最终实现的包路径和少量类型细节与初稿不同。

## 原始需求点总结

1. 统一错误分层：原始需求想把“节点内部错误”“规则链运行时失败”和“对外协议响应错误”拆成明确层级，而不是都混成普通字符串 error。
2. 保留结构化失败上下文：当链路失败时，框架应能稳定记录失败节点、失败时间、错误码和错误消息，便于 endpoint 统一消费。
3. 给外部协议一个统一出口：无论内部怎么失败，对外最终都应能包装成稳定的服务错误对象，而不是每个 endpoint 自己拼响应。
4. 支持错误码映射：原始需求不是只想“打印错误”，而是希望内部 fault code 能被映射到具体协议层状态码。
5. 为多协议扩展打底：虽然当前主要落在 HTTP，但原始需求从一开始就是跨协议的，希望 gRPC、WebSocket 等后续边界也能接入同一模型。

## 2. 当前实现对齐

### 2.1 当前类型位置

当前错误模型主要位于：

- `pkg/types/logger.go`
- `pkg/types/http.go`

而不是 RFC 初稿里写的 `pkg/types/error.go` / `pkg/types/endpoint.go`。

### 2.2 当前已落地的结构

当前代码已经稳定存在：

- `types.Fault`
- `types.FailureInfo`
- `types.ServiceError`
- `types.ErrorMapping`

### 2.3 当前 endpoint 错误处理

`endpoint/http` 已经实现：

1. 从 metadata 重建 `FailureInfo`
2. 基于 `errorMappings` 计算对外响应码
3. 构造 `ServiceError`
4. 输出统一错误响应

对应实现位于：

- `internal/builtin/nodes/endpoint/http_endpoint.go`

## 3. 与原 RFC 的差异

原 RFC 中这些细节已经需要更新：

1. `Fault.Code` 不是简单的 `int32`，而是 `cnst.ErrCode`
2. 相关类型不在 `pkg/types/error.go`
3. 当前 first-class 落地的是 HTTP endpoint 错误处理，不是完整的“所有协议都已统一”
4. `HandleError` 与 runtime 传播细节已经在当前 runtime 实现里固定下来

## 4. 当前实现范围

### 4.1 已实现

- 节点用 `Fault` 包装错误
- runtime 将结构化错误写入 metadata
- HTTP endpoint 基于 metadata 生成 `ServiceError`
- `errorMappings` 将内部 fault code 映射到外部响应码

### 4.2 尚未统一到所有协议

当前文档不应暗示这些能力已经在所有边界完全统一：

- gRPC
- WebSocket
- 其他主动 / 被动 endpoint 协议

## 5. 结论

本 RFC 已被接受，并以当前错误模型实现落地。后续如果扩展到更多协议，应在现有 `Fault -> FailureInfo -> ServiceError` 模型上继续演进，而不是回到初稿里过时的包路径和类型签名。

## 6. 相关现行文档

1. [指南：统一错误处理与 HTTP 错误映射](../../guides/15_unified_error_handling_guide.md)
2. [参考-10：HttpEndpoint 节点深度解析](../../reference/10_http_endpoint_deep_dive.md)
3. [参考：Matrix 消息设计哲学 (RuleMsg)](../../reference/06_message_design_philosophy.md)
