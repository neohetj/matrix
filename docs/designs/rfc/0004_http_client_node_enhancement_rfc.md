---
uuid: "0b632717-cadd-439e-bc83-6fc641ddebb9"
type: "RFC"
title: "需求：HTTP Client 节点功能与重构"
status: "Superseded"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "matrix"
  - "http-client"
  - "history"
relations:
  - type: "references"
    target_uuid: "f5614284-7536-45c4-8342-b098f005394e"
    description: "当前 HTTP endpoint 映射模型的实现细节以 Reference-10 为准。"
  - type: "references"
    target_uuid: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
    description: "当前 `endpoint/http` 的配置与使用方式见组件指南。"
  - type: "references"
    target_uuid: "b1c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e"
    description: "当前动态对象转换与 packet 风险边界见 06 指南。"
  - type: "references"
    target_uuid: "f4e5d6c7-b8a9-0c1d-2e3f-4a5b6c7d8e9f"
    description: "当前 packet / DSL 风险说明可继续参考 SQL Query 函数组件指南中的相关章节。"
---

# RFC: HTTP Client 节点功能与重构

> Historical note: 这份 RFC 描述的是一版过渡期设计。当前实现并没有采用文中的 `HttpParam`、`HttpMapping`、`bodyFields`、`DataFormat()` 方案。

## 1. 原始目标

这份 RFC 的目标是把 `external/httpClient` 的入参和出参映射能力升级到接近 `endpoint/http` 的水平，并统一两者的映射体验。

这个目标已经实现了一部分，但实现结构和 RFC 初稿不同。

## 原始需求点总结

1. 统一映射心智模型：原始需求希望 `external/httpClient` 与 `endpoint/http` 不再各自维护一套完全不同的请求/响应映射思路，减少学习和维护成本。
2. 覆盖完整 HTTP 结构：调用方需要能够稳定映射请求头、查询参数、请求体、响应头、响应体和状态码，而不是只支持零散字段。
3. 减少样板转换代码：希望把请求拼装、响应提取和字段转换下沉到通用映射层，而不是在每个业务节点里手写大量 glue code。
4. 提升 DSL 表达能力：原始提案想让 DSL 能描述更复杂的 HTTP 入参与出参绑定关系，并让这些关系更容易做静态校验。
5. 降低 endpoint / client 能力割裂：从需求动机看，这份 RFC 不只是增强一个节点，而是希望形成一套更统一的 HTTP packet / mapping 抽象。

## 2. 当前实现对齐

当前 `external/httpClient` 的真实模型是：

- `types.HttpRequestMap`
- `types.HttpResponseMap`
- `types.EndpointIOPacket`
- `types.EndpointIOField`

对应代码主要位于：

- `pkg/types/http.go`
- `pkg/components/external/http_client_node.go`
- `pkg/helper/http_mapper.go`

## 3. 已落地的部分

当前实现已经具备这些能力：

1. 请求映射与响应映射统一走 helper
2. 请求 `headers/queryParams/body` 都使用 packet 结构
3. 响应 `headers/body/statusCodeTarget` 可回写到消息
4. `external/httpClient` 与 `endpoint/http` 共用一套 packet 处理思路

## 4. 未采用的旧设计

下列 RFC 内容已经明确不是现行实现：

1. `RuleMsg.DataFormat()` / `WithDataFormat()`
2. `HttpParam`
3. `HttpMapping`
4. `bodyFields`
5. `mapping.from` / `mapping.to` 风格的专用 HTTP 映射 DSL

当前实现统一使用：

- `bindPath`
- `mapAll`
- `fields`

## 5. 当前 DSL 关键字段

如果今天要配置 `external/httpClient` 或审查其实现，应该围绕这些实际字段理解：

1. `request.headers` / `request.queryParams` / `request.body`
2. `response.headers` / `response.body` / `response.statusCodeTarget`
3. `EndpointIOPacket.MapAll`
4. `EndpointIOPacket.Fields`
5. `EndpointIOField.BindPath`

## 6. 现行规范入口

当前 HTTP 映射规范请直接参考：

1. [参考-10：HttpEndpoint 节点深度解析](../../reference/10_http_endpoint_deep_dive.md)
2. [组件指南：HTTP Endpoint (endpoint/http)](../../guides/components/endpoint_http_guide.md)
3. [指南：动态对象转换](../../guides/dynamic-object-conversion-guide.md)
4. [组件指南：SQL Query 函数](../../guides/components/functions_sql_query_guide.md) 中对 packet / DSL 风险的说明

## 7. 结论

这份 RFC 不再是现行规范，只保留为 HTTP client 重构历史背景。任何新的 HTTP endpoint / httpClient 配置与实现，都不应再参考本文中的旧 API。
