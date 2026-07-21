---
uuid: "e8d7c6b5-a4f9-4993-8262-3c1d0a9b8e7f"
type: "ComponentGuide"
title: "组件指南：Redis Stream Endpoint"
status: "Stable"
owner: "neohetj"
version: "2.0.0"
tags:
  - "matrix"
  - "component"
  - "endpoint"
  - "redis-stream"
relations:
  - type: "is_part_of"
    target_uuid: "a2b1c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
    description: "本文档属于 Matrix 组件指南。"
  - type: "uses_reference"
    target_uuid: "7eb3bd9a-0bdd-42d1-aa8c-3c45c14284ac"
    description: "可靠消费、ACK、pending recovery 与 DLQ 语义以该 Reference 为准。"
---

# 如何使用 `endpoint/redis_stream`

## 1. 功能概述

`endpoint/redis_stream` 使用 Redis consumer group 持续接收事件，并为每条投递同步执行指定规则链。它既可只消费新消息，也可显式启用 pending recovery，让其他实例接管崩溃消费者遗留的消息。

## 2. 推荐配置

```json
{
  "id": "event-consumer",
  "type": "endpoint/redis_stream",
  "name": "消费领域事件",
  "configuration": {
    "redisClient": "ref://shared_redis",
    "stream": "domain.events",
    "group": "domain-projector",
    "ruleChainId": "project-domain-event",
    "startNodeId": "validate-event",
    "count": 10,
    "blockMs": 5000,
    "concurrency": 2,
    "autoCreateGroup": true,
    "groupStartId": "0",
    "processingTimeoutMs": 30000,
    "pendingRecovery": {
      "enabled": true,
      "minIdleMs": 60000,
      "intervalMs": 10000,
      "count": 10,
      "maxDeliveries": 5,
      "deadLetterStream": "domain.events.dlq"
    }
  }
}
```

多实例部署时推荐省略 `consumer`。Matrix 会按节点、主机、进程和 worker 自动生成唯一名称。只有部署系统能为每个实例注入唯一值时，才显式设置 `consumer`。

完整字段和默认值见 [Redis Stream Endpoint 可靠消费](../../reference/40_redis_stream_endpoint_reliability.md)。

## 3. 输入数据契约

每条 Redis 消息会转换成一个 JSON `RuleMsg`：

- `Data` 包含原消息所有字段，并增加 `redis_stream` 和 `redis_message_id`。
- `Metadata` 包含 `redis_stream`、`redis_message_id` 和 `redis_group`。
- `DataFormat` 为 `JSON`。
- 配置 `input` 映射时，Matrix 会通过通用 inbound mapping 把 Redis 字段映射进 `DataT`。

业务规则链应在入口处完成事件 schema 校验，并用事件 ID、idempotency key 或领域唯一键保证重复投递安全。revision 顺序和业务事务也属于规则链调用的业务 Handler，不属于 Endpoint。

## 4. 启用可靠恢复

1. 将 `ackOnFailure` 保持为 `false`。
2. 设置业务处理的 `processingTimeoutMs`。
3. 启用 `pendingRecovery.enabled`，并确保 `minIdleMs > processingTimeoutMs`。
4. 根据允许的最长故障恢复时间设置 `intervalMs`。
5. 需要隔离毒消息时设置正数 `maxDeliveries` 和 `deadLetterStream`。
6. 确认 Redis 版本至少为 6.2。

不要把 `minIdleMs` 设得接近正常 P99 处理耗时。它是故障接管阈值，不是业务超时；过小会让另一个实例重领仍在执行的消息。

## 5. 验证方式

### 5.1 正常消费

向源 Stream 写入一条合法事件，验证：

- 规则链只产生一次预期业务结果；
- consumer group 的 lag 回落；
- 成功消息不再出现在 PEL。

### 5.2 崩溃恢复

1. 让实例 A 读到消息但不要 ACK。
2. 停止实例 A。
3. 等待超过 `minIdleMs`。
4. 启动或保留实例 B。
5. 验证实例 B 通过 `XAUTOCLAIM` 接管消息，并通过同一个 Handler 完成处理和 ACK。

### 5.3 重试与 DLQ

持续让 Handler 返回错误，验证：

- 未达到 `maxDeliveries` 时消息仍在 PEL；
- 达到上限时 DLQ 先出现包含 `matrix_original_message_id` 的记录；
- DLQ 写入成功后原消息才从 PEL 移除；
- DLQ 不可写时原消息仍保留，未被错误 ACK。

常用 Redis 检查命令：

```bash
redis-cli XINFO GROUPS domain.events
redis-cli XPENDING domain.events domain-projector
redis-cli XRANGE domain.events.dlq - + COUNT 20
```

## 6. 排障

| 现象 | 检查项 | 处理 |
| --- | --- | --- |
| lag 为 `0` 但业务状态没有更新 | `XPENDING` 是否有消息 | 检查 Handler 错误；启用 pending recovery 接管遗留消息。 |
| 同一消息被两个实例同时处理 | consumer 是否固定复用；`minIdleMs` 是否过小 | 使用唯一 consumer，并让 `minIdleMs` 大于处理超时和正常处理窗口。 |
| 消息反复失败 | `RetryCount` 与 DLQ 配置 | 修复业务错误，或设置有限投递和 DLQ。 |
| 关闭服务一直等待 | 业务节点是否响应 context | 为外部调用设置超时并传播 Matrix context。 |
| 启动时报配置错误 | `ackOnFailure`、超时与 DLQ 组合 | recovery 下关闭 `ackOnFailure`；保证 `minIdleMs` 更大；配置有限次数时提供 DLQ。 |

## 7. 交接清单

- [ ] stream、group 和 group 起始位置已确认。
- [ ] 多实例没有共享固定 consumer 名称。
- [ ] Handler 已实现业务幂等和 revision 防倒退。
- [ ] `processingTimeoutMs`、`minIdleMs` 和重试上限已按生产耗时设定。
- [ ] DLQ 已有监控、查询和人工/自动重放流程。
- [ ] 已完成正常消费、进程崩溃、毒消息和 Redis 短暂不可用测试。

<!-- qa_section_start -->
> **问：Matrix 的 pending recovery 能替代业务幂等吗？**
> **答：** 不能。Redis Stream 与 DLQ 都是 at-least-once 路径，超时、进程崩溃以及 DLQ 写入后 ACK 失败都可能造成重复投递。业务 Handler 必须用稳定事件键自行幂等。
<!-- qa_section_end -->
