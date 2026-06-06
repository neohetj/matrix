---
uuid: "a6b7c8d9-e0f1-4a2b-8c3d-4e5f6a7b8c9d"
type: "ComponentGuide"
title: "学习Matrix组件目录"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "component"
  - "catalog"
  - "reference"
relations:
  - type: "is_referenced_by"
    target_uuid: "60c07c47-df0e-4b76-9ed9-62fabe2e2add"
    description: "参考-08 会引导开发者查阅当前内建组件目录。"
---

# 学习 Matrix 组件目录

本文档列出当前仓库中能直接从源码定位到的核心内建节点。对 `functions` 节点来说，**Matrix core 只内建通用执行器**，具体函数 ID 由宿主应用或下游模块注册。

## 1. Action

| 源码锚点 | 说明 |
| :--- | :--- |
| `action/log` | 打印日志 |
| `action/exprSwitch` | 按表达式路由 |
| `action/flow` | 同步执行子规则链 |
| `action/aggregator` | 聚合分支输出 |
| `action/channel_push` | 向 pipeline channel 推送消息 |
| `functions` | 通用函数执行器 |

## 2. Loop

| 源码锚点 | 说明 |
| :--- | :--- |
| `loop/forEach` | 遍历集合并执行子规则链 |
| `loop/break` | 中断 forEach / 循环链路 |

## 3. Endpoint

| 源码锚点 | 说明 |
| :--- | :--- |
| `endpoint/http` | HTTP 入口 |
| `endpoint/mcp` | MCP tool 入口，transport host 由 WhiteRoom 等宿主承载 |
| `endpoint/pipeline` | Pipeline 入口 |

## 4. Pipeline / Shared Resources

| 源码锚点 | 说明 |
| :--- | :--- |
| `resource/channel_manager` | 通道管理器共享节点 |

## 5. Transform

| 源码锚点 | 说明 |
| :--- | :--- |
| `transform/object_mapper` | 基于 `EndpointIOPacket` 的对象映射节点 |

## 6. External

| 源码锚点 | 说明 |
| :--- | :--- |
| `external/httpClient` | 声明式 HTTP Client |
| `external/sqlClient` | SQL 共享客户端 |
| `external/redisClient` | Redis 共享客户端 |
| `external/mongoClient` | MongoDB 共享客户端 |

## 7. Ops Modeling

| 源码锚点 | 说明 |
| :--- | :--- |
| `ops/application` | 应用拓扑节点 |
| `ops/service` | 服务拓扑节点 |
| `ops/database` | 数据库拓扑节点 |
| `ops/message_queue` | 消息队列拓扑节点 |
| `ops/network` | 网络拓扑节点 |
| `ops/volume` | 存储卷拓扑节点 |
| `ops/runner` | 运行器拓扑节点 |
| `ops/machine` | 机器拓扑节点 |

## 8. 关于函数

`functions` 节点的具体 `functionName` 不在这个目录里硬编码列举，因为它们不是 Matrix core 的内建 NodeType。要查看某个工作区实际注册了哪些函数，应到宿主模块的 `NodeFuncManager.Register(...)` 调用点查找。

<!-- qa_section_start -->
> **问：我该如何调用某个函数？**
> **答：** 创建一个 `type: "functions"` 的节点，在 `configuration.functionName` 中填入宿主应用已注册的函数 ID，并在需要时补齐 `configuration.business` 与节点级 `inputs` / `outputs`。
<!-- qa_section_end -->
