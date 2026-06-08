---
# === Node Properties: 定义文档节点自身 ===
uuid: "68b1646b-2238-48e0-b77d-c81ecfc4317d"
type: "README"
title: "README: Matrix RFC 文档库"
status: "Stable"
owner: "neohetj"
version: "2.0.0"
tags:
  - "matrix"
  - "rfc"
  - "design"
  - "readme"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本 RFC 库是 Matrix 项目设计文档体系的一部分。"
---

# Matrix RFC 文档库

本目录用于保存 Matrix 的重大需求提案、架构调整背景和实现前后的设计记录。

需要额外强调的是：**RFC 的首要职责是保留原始需求点和原始设计提案**。即使 RFC 后续被接受、部分落地或被替代，也不应直接把正文改写成“当前实现说明”，否则就会失去需求档案价值。

需要特别注意的是：**RFC 不等于现行规范**。当前目录中的文档已经分成三类：

1. **Accepted / Implementing**: 与当前实现直接相关，但仍需结合 reference 文档阅读
2. **Draft**: 尚未实现或仅有部分前置能力
3. **Superseded**: 历史提案，只保留背景价值，不应再作为现行实现依据

## 当前索引

| RFC | 状态 | 当前含义 |
| :--- | :--- | :--- |
| [`0001_data-contract-specification_rfc.md`](./0001_data-contract-specification_rfc.md) | `Superseded` | 历史数据契约草案，现行规范已转向 URI 契约与 `NodeReads/FuncReads` |
| [`0002_mcp_asset_search_tool_rfc.md`](./0002_mcp_asset_search_tool_rfc.md) | `Draft` | 开发者工具提案，仓库中尚无实现 |
| [`0003_solidify-function-pattern_rfc.md`](./0003_solidify-function-pattern_rfc.md) | `Superseded` | 函数模式已正式化，但未按原文中的目录迁移方案落地 |
| [`0004_http_client_node_enhancement_rfc.md`](./0004_http_client_node_enhancement_rfc.md) | `Superseded` | HTTP client 旧设计，现行实现已采用 packet / bindPath 体系 |
| [`0005_stateful-aggregator-node_rfc.md`](./0005_stateful-aggregator-node_rfc.md) | `Accepted` | 扇入能力已落地，现行为 `action/aggregator` |
| [`0006_ops-foundation-components-and-dsl-extensions_rfc.md`](./0006_ops-foundation-components-and-dsl-extensions_rfc.md) | `Implementing` | ops 基础节点与 DSL 扩展已部分落地 |
| [`0007_matrix_cohesion_refactor_rfc.md`](./0007_matrix_cohesion_refactor_rfc.md) | `Accepted` | Matrix 统一入口与内聚性重构主体已完成 |
| [`0008_generic_agent_nodes_rfc.md`](./0008_generic_agent_nodes_rfc.md) | `Draft` | Agent / LLM 核心节点仍是提案 |
| [`0009_websocket_endpoint_node_rfc.md`](./0009_websocket_endpoint_node_rfc.md) | `Draft` | WebSocket endpoint 仍未实现 |
| [`0010_unified_error_handling_rfc.md`](./0010_unified_error_handling_rfc.md) | `Accepted` | Fault / FailureInfo / ServiceError 模型已落地 |
| [`0011_config_uri_and_manager_rfc.md`](./0011_config_uri_and_manager_rfc.md) | `Implementing` | `config://` 协议已实现，统一配置视图未实现 |
| [`0012_topology-driven-deployment-platform_rfc.md`](./0012_topology-driven-deployment-platform_rfc.md) | `Draft` | 部署平台闭环仍是提案，底层拓扑能力已具备部分前置实现 |
| [`0013_function_routing_constraints_rfc.md`](./0013_function_routing_constraints_rfc.md) | `Accepted` | 函数路由约束已在运行时和注册阶段实现 |
| [`0014_mcp_business_endpoint_adapter_rfc.md`](./0014_mcp_business_endpoint_adapter_rfc.md) | `Draft` | MCP 业务入口适配器提案，用于把业务模块能力白名单暴露为 MCP tools |
| [`0015_matrix_core_interface_boundary_refactor_rfc.md`](./0015_matrix_core_interface_boundary_refactor_rfc.md) | `Draft` | Matrix core 运行时、注册表、节点上下文、endpoint 与函数目录接口边界收敛提案 |

## 阅读顺序建议

如果你关心**当前代码怎么工作**，优先看：

1. `Accepted`
2. `Implementing`
3. 对应 `docs/reference/*`

如果你关心**为什么当初这么设计**，再回头看：

1. `Superseded`
2. `Draft`

## 与 Reference / ADR 的关系

- RFC 负责记录需求、动机、设计方向和实施边界
- ADR 负责记录已确认的架构决策
- Reference 负责记录当前实现的规范性说明

当 RFC 与代码不一致时，应优先以：

1. 代码
2. Reference
3. 已接受且明确回写过状态的 RFC

为准。

## 维护规则

1. `Summary / Motivation / DetailedDesign / Drawbacks / Alternatives / UnresolvedQuestions / FAQ` 这些原始 RFC 章节应尽量保真保留。
2. 如果需要补“当前实现对齐”“历史注记”“已落地范围”，请追加到正文后部，作为附录或新增章节，不要覆盖原始需求点。
3. 如果原始提案已经明显过时，也应通过 `Historical note` 或“附录：当前实现对齐”解释差异，而不是删除原始提案内容。
4. `Reference` 负责现行规范，`ADR` 负责已确认的架构决策，`RFC` 负责保留“为什么提出、最初想做什么、当时怎么设计”。
5. `Accepted` 或 `Implementing` 的 RFC 必须至少引用一份当前使用文档；优先链接 `docs/guides/` 或 `docs/migration/` 下的非 README 文档。
6. 如果 RFC 已经落地为新的节点、协议、配置模式或错误处理机制，但当前仓库还没有对应使用文档，应先补 guide，再把 guide 写回 RFC 的 `relations` 和正文“相关现行文档”部分。
7. 每份 RFC 都应在正文前部包含一个明确命名为 `原始需求点总结` 的小节，用 3-6 个要点总结最初的痛点、核心目标、边界或约束；它可以是对原始提案的压缩提炼，但不能被“当前实现说明”替代。
8. 当 RFC 进入 `Accepted` 或 `Implementing` 状态后，必须补充 `当前实现对齐`（或 `附录：当前实现对齐`）章节，明确说明：已落地范围、尚未落地或偏离原提案的部分、当前应以哪些 guide / reference / migration 文档为准。
9. 如果某份 RFC 的实现状态从“未实现”变为“部分实现”或“已实现”，应同步更新该 RFC 的 `status`、`relations`、“当前实现对齐”章节和“相关现行文档”章节，而不是只改代码不回写文档。
