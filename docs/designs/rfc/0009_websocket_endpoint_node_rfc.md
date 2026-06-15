---
uuid: "d4a34d7e-a66d-47de-b2b6-146d84596467"
type: "RFC"
title: "需求：实现Matrix WebSocket Endpoint节点"
status: "Draft"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "matrix"
  - "websocket"
  - "endpoint"
  - "node"
relations: []
---

# RFC: 实现 Matrix WebSocket Endpoint 节点

## 1. 摘要

这份 RFC 继续保留为**未实现提案**。截至当前代码状态，Matrix 还没有 `endpoint/websocket` 这一节点类型。

## 原始需求点总结

1. 提供 WebSocket 被动入口：原始需求希望 Matrix 拥有和 `endpoint/http` 类似的长期连接型 endpoint，能够接住 WebSocket 会话。
2. 支持消息到 RuleMsg 的映射：来自 WebSocket 的事件、文本或结构化消息需要被转换成规则链可执行的 `RuleMsg`。
3. 支持回写与主动推送：不仅要能接收消息，还要能基于当前连接、会话或订阅上下文向客户端回推结果。
4. 管理连接生命周期：原始需求隐含地需要连接建立、断开、心跳、会话标识和上下文保持等基础能力。
5. 尽量复用既有 endpoint 边界：理想状态下，WebSocket endpoint 应复用现有 runtime 调用、错误传播与映射模型，而不是另起一套执行框架。

## 2. 当前状态

当前仓库中：

- 未发现 `endpoint/websocket` 注册
- 未发现 WebSocket endpoint 运行时实现
- 未发现连接管理、事件映射、反向推送等对应代码

因此本文描述的能力仍然是设计方向，而不是现行功能。

## 3. 与现有 endpoint 能力的边界

当前已落地的 endpoint 主体仍然是：

- `endpoint/http`
- `endpoint/pipeline`

如果后续继续做 WebSocket endpoint，建议复用现有能力边界：

1. endpoint 作为 shared / passive endpoint 的生命周期模型
2. packet / bindPath 风格的数据映射理念
3. 统一错误处理和 runtime 调用边界

## 4. 结论

本文继续保持 Draft。任何文档、技能或业务 DSL 都不应把 `endpoint/websocket` 当成当前已经存在的内建节点。
