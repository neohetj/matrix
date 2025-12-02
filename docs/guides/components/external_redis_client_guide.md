---
# === Node Properties: 定义文档节点自身 ===
uuid: "e5f6a7b8-c9d0-1e2f-3a4b-5c6d7e8f9a0b"
type: "ComponentGuide"
title: "组件指南：Redis客户端共享节点 (external/redisClient)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "external"
  - "redis"
  - "cache"
  - "shared"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的功能组件之一。"
  - type: "references"
    target_uuid: "a3b4c5d6-e7f8-9a0b-1c2d-3e4f5a6b7c8d"
    description: "本节点是共享节点的一种实现，遵循共享节点的设计模式。"
---

# 1. 功能概述 (Overview)

`redisClient` 是一个**[共享节点 (Shared Node)][Guide-SharedNode-7c8d]**，它的核心职责是创建并管理一个可供其他节点复用的 Redis 客户端连接 (`*redis.Client`)。

通过中心化管理 Redis 连接，可以极大地提高资源利用率，并简化规则链中需要与 Redis 交互的节点的配置。

> **核心概念**: 在继续之前，请务必阅读 **[指南：理解和使用共享节点][Guide-SharedNode-7c8d]** 以了解共享节点的通用工作原理和生命周期。

# 2. 如何配置 (Configuration)

在规则链的DSL中，可以通过 `configuration` 字段为此节点配置以下参数：

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `dsn` | DSN | Redis 连接字符串 (Data Source Name)，遵循标准URI格式。 | `string` | 是 | N/A |
| `poolSize` | 连接池大小 | Redis 连接池的大小。 | `int` | 否 | `0` (使用驱动默认值) |

## 2.1. 配置示例 (Example)

```json
{
  "id": "shared_redis_cache",
  "type": "external/redisClient",
  "name": "共享Redis缓存连接",
  "configuration": {
    "dsn": "redis://:password@hostname:port/db_number",
    "poolSize": 10
  }
}
```

# 3. 如何引用 (HowToReference)

其他节点可以通过 `ref://<node_id>` 的格式来引用本节点。详细的引用机制请参见 **[共享节点指南][Guide-SharedNode-7c8d]**。

### 3.1. `redisCommand` 引用示例

```json
{
  "id": "get_user_cache",
  "type": "functions/redisCommand",
  "name": "获取用户缓存",
  "configuration": {
    "redisDsn": "ref://shared_redis_cache",
    "commands": ["GET user:${metadata.userId}"]
  }
}
```

# 4. 数据契约 (DataContract)

`redisClient` 节点是一个资源管理节点，它**不参与**消息的数据流处理。它不会读取或修改 `RuleMsg` 的任何部分 (`Data`, `DataT`, `Metadata`)。

<!-- 链接定义区域 -->
[Guide-MatrixOverview]: ../../00_matrix_guide.md
[Guide-SharedNode-7c8d]: ../03_shared_node_guide.md
