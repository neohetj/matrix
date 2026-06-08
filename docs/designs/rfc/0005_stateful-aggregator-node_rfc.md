---
uuid: "a9214860-3b24-4140-96da-81af508c9ec3"
type: "RFC"
title: "需求：框架级并行任务聚合器节点"
status: "Accepted"
owner: "neohetj"
version: "2.1.0"
tags:
  - "rfc"
  - "design"
  - "aggregator"
  - "parallel"
  - "join"
  - "fan-in"
relations:
  - type: "is_formalized_by"
    target_uuid: "a6b7c8d9-e0f1-4a2b-8c3d-4e5f6a7b8c9d"
    description: "当前组件目录已收录 `action/aggregator` 为内建节点。"
  - type: "is_explained_by"
    target_uuid: "9f9f0df6-dbc9-4fcb-8f60-d39971809d7e"
    description: "当前 `action/aggregator` 的使用方式见对应组件指南。"
---

# RFC: 框架级并行任务聚合器节点 (ProposeStatefulAggregatorNode)

> Historical note: 本文保留原始 RFC 提案正文。由于该 RFC 已落地，当前实现差异统一补充在文末附录，原始需求点不得被覆盖。

## 1. 摘要 (Summary)

本RFC提议在Matrix框架中引入一个全新的、内置的 `aggregator/join` 节点类型，以提供一个声明式的、健壮的机制来同步和聚合多个并行执行任务的结果，解决当前框架在“扇入”（Fan-in）模式上的缺失。

## 原始需求点总结

1. 框架需要内建的 fan-in / join 能力，让多个并行分支在全部完成后只汇合触发一次，而不是重复触发下游。
2. 并行聚合场景不能再依赖业务函数手动在 `msg.Metadata` 中维护状态，这类同步复杂度应由框架统一承担。
3. 需要一个声明式节点来表达“等待哪些输入、超时如何处理、是否 fail-fast”这类聚合策略，而不是散落在自定义代码里。
4. 规则链在运维巡检、并行探测、批量收集结果这类场景里，需要更直观地表达扇出后再扇入的结构。
5. 新能力应尽量复用现有 RuleMsg 与运行时模型，不要求业务侧引入额外的外部状态管理系统。

## 2. 动机 (Motivation)

-   **当前存在的问题**: 当前的Matrix运行时支持并行启动多个根节点（扇出），但缺乏一个内置机制来等待所有并行分支都完成后再执行下一步。当一个节点有多个上游连接时，它会被每个上游分支独立、多次地触发，而不是等待所有分支完成后被触发一次。这使得实现“扇入”聚合逻辑变得非常困难。

-   **用例**: 在开发“统一服务健康巡检”功能时，需要并行探测MySQL、Redis等多个服务的状态，然后收集所有探测结果，生成一份最终的聚合报告。为了实现这一逻辑，开发者被迫在业务函数中手动实现复杂且容易出错的状态管理和同步逻辑（通过序列化状态到`msg.Metadata`），这污染了业务代码，并增加了开发负担。

-   **目标**:
    1.  为Matrix框架提供原生的、声明式的并行任务聚合能力。
    2.  将并发同步的复杂性从业务逻辑中剥离，封装到框架层面。
    3.  简化涉及并行聚合模式的规则链的设计和可读性。

## 3. 设计详解 (Detailed Design)

-   **核心思路**: 引入一个新的 `aggregator/join` 节点。该节点在内部维护一个状态机，用于跟踪其所有上游分支的执行状态。当所有预期的输入都已从上游节点成功接收后，它会将收集到的所有数据对象合并到一个`RuleMsg`中，然后触发其下游节点。

-   **组件交互**:
    ```mermaid
    sequenceDiagram
        participant Runtime as 运行时
        participant ProbeA as 探测任务A
        participant ProbeB as 探测任务B
        participant JoinNode as 聚合节点
        participant AggregatorFunc as 聚合函数

        Runtime->>+ProbeA: Execute
        Runtime->>+ProbeB: Execute
        ProbeA-->>-JoinNode: TellSuccess(MsgA)
        ProbeB-->>-JoinNode: TellSuccess(MsgB)
        JoinNode->>JoinNode: All inputs received, aggregate results
        JoinNode-->>+AggregatorFunc: TellSuccess(AggregatedMsg)
        AggregatorFunc-->>-Runtime: Final Result
    ```

-   **示例 (DSL)**:
    ```json
    {
      "id": "join_probes",
      "type": "aggregator/join",
      "name": "等待所有探测完成",
      "configuration": {
        "timeoutMs": 5000,
        "failFast": false,
        "expectedInputs": [
          { "objId": "mysql_status_obj" },
          { "objId": "redis_status_obj" },
          { "objId": "zlm_status_obj" }
        ]
      }
    }
    ```

## 4. 缺点与风险 (DrawbacksAndRisks)

-   **框架复杂性**: 引入一个新的有状态节点会增加框架本身的复杂度和维护成本。
-   **实现风险**: 需要仔细处理并发、超时和错误累积等问题，以确保节点的健壮性。

## 5. 备选方案 (Alternatives)

-   **有状态的业务函数**: 即在业务代码中通过`msg.Metadata`手动管理状态。已被证明过于复杂和笨拙，容易出错。
-   **扩展`functions`节点**: 为`functions`节点增加一个`executionMode: "join"`的配置。这会使`functions`节点的职责变得不单一，可能引起混淆。

## 6. 未解决的问题 (UnresolvedQuestions)

-   聚合后的`RuleMsg`中的`Metadata`应该如何合并？是简单地合并所有上游消息的元数据，还是有更复杂的策略？

## 7. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：这个变更会影响现有的规则链吗？**
> **答：** 不会。这是一个全新的节点类型，不会影响任何现有节点的行为。现有规则链将继续按原样工作。

> **问：为什么不直接在`default_runtime.go`中修改逻辑，让所有多输入节点都自动等待？**
> **答：** 因为这会破坏现有规则链的“或”逻辑（例如`alertGenerator`的例子）。框架需要同时支持“或”（任意上游触发即执行）和“与”（所有上游完成才执行）两种模式。引入一个明确的`join`节点是区分这两种模式的最清晰方法。
<!-- qa_section_end -->

## 8. 附录：当前实现对齐

### 8.1 当前落地形态

当前落地的节点类型是：

- `action/aggregator`

而不是草案中的：

- `aggregator/join`

### 8.2 当前实现语义

当前实现采用：

1. 根据当前节点的所有前驱节点判断是否收齐
2. 每收到一个上游消息就记录 `PreviousNodeID`
3. 所有前驱到齐后再继续向下游 `TellSuccess`
4. 超时则 `TellFailure`

### 8.3 当前与原 RFC 的主要差异

原始提案中这些内容没有按原样落地：

1. `timeoutMs`
2. `failFast`
3. `expectedInputs`
4. 专门的聚合结果合并 DSL

### 8.4 当前应该如何阅读本文

1. 第 `1-7` 节是原始扇入能力提案
2. 本附录解释它如何以 `action/aggregator` 形式落地
3. 后续示例与文档应以当前节点类型和配置为准

## 9. 相关现行文档

1. [组件指南：聚合器 (action/aggregator)](../../guides/components/action_aggregator_guide.md)
2. [学习 Matrix 组件目录](../../reference/21_component_catalog.md)
