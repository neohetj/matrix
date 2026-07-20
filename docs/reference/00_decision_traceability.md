---
uuid: "8a765249-ba84-4240-b3f2-c210c14bf285"
type: "Reference"
title: "参考：Matrix 决策追踪索引"
status: "Draft"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "decision-traceability"
  - "reference"
relations:
  - type: "is_part_of"
    target_uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
    description: "本文档属于 Matrix 参考文档库。"
  - type: "supports"
    target_uuid: "86e6d134-004f-4483-a97a-396d2ab79a37"
    description: "本文档追踪 Matrix core interface boundary refactor RFC 的后续闭环。"
  - type: "indexes"
    target_uuid: "a18a0c1b-d599-4f8d-a949-51a60182873d"
    description: "本文档登记 Matrix 内部错误码编码规范对 Unified Error Handling RFC 的事实承接。"
---

# Matrix 决策追踪索引

本文档只记录 Matrix repo-local RFC / ADR / Plan / Reference / Guide 的闭环追踪。它不承载 ADR 的决策理由，也不承载 Plan 的阶段执行细节。

当前文档是 Matrix 正式文档集的 seed 索引。历史 RFC / ADR / Plan 的完整回填需要单独治理任务承接；本次只登记新增 RFC，避免把架构 RFC 编写扩散为全量文档治理重构。

## 1. RFC 闭环

| RFC | 状态 | 设计承接 | 当前事实 / 操作承接 |
| --- | --- | --- | --- |
| [0010 Unified Error Handling](../designs/rfc/0010_unified_error_handling_rfc.md) | `Accepted` | `Fault -> FailureInfo -> ServiceError` 与 DSL `errorMappings` 已落地 | 当前错误分层与安全公开映射见 [10_http_endpoint_deep_dive.md](10_http_endpoint_deep_dive.md)，Matrix core 内部错误码见 [39_internal_error_code_specification.md](39_internal_error_code_specification.md)，操作规则见 [15_unified_error_handling_guide.md](../guides/15_unified_error_handling_guide.md)。 |
| [0015 Matrix Core Interface Boundary Refactor](../designs/rfc/0015_matrix_core_interface_boundary_refactor_rfc.md) | `Draft` | [Plan 0015-1](../designs/plan/0015-1_matrix_core_interface_boundary_refactor_plan.md) | 当前事实参考 [03_architecture_overview.md](03_architecture_overview.md)、[11_function_registration_spec.md](11_function_registration_spec.md)、[12_node_specification.md](12_node_specification.md)、[15_shared_resource_management.md](15_shared_resource_management.md)、[37_mcp_endpoint_adapter.md](37_mcp_endpoint_adapter.md)、[38_validation_inspection_schema.md](38_validation_inspection_schema.md) |

## 2. Plan 执行追踪

| Plan | 状态 | 当前阶段 | 验证 / 证据 |
| --- | --- | --- | --- |
| [0015-1 Matrix Core Interface Boundary Refactor](../designs/plan/0015-1_matrix_core_interface_boundary_refactor_plan.md) | `Implementing` | Stage 0.5 low-impact boundary slices 已记录 | 见 Plan `## 9. Stage 0 Baseline Snapshot`。Stage 0 记录了仓库状态、测试 baseline、文档审计结果、事实层分类和低影响优先队列；Stage 0.5 已记录 validation / inspection schema、loader report-only、CLI、DAG、function registry error-return contract、文档治理、endpoint catalog、runtime lifecycle owner contract 和 generic infrastructure identity audit。 |

## 3. 当前事实文档

| 文档 | 状态 | 权威范围 |
| --- | --- | --- |
| [03_architecture_overview.md](03_architecture_overview.md) | `Draft` | Matrix 当前整体架构、规则链、节点、runtime、registry 与 shared node pool 基本概念。 |
| [11_function_registration_spec.md](11_function_registration_spec.md) | `Stable` | 函数节点注册、routing mode、declared relations 和函数配置契约。 |
| [12_node_specification.md](12_node_specification.md) | `Draft` | 通用节点生命周期、数据契约和节点接口事实。 |
| [15_shared_resource_management.md](15_shared_resource_management.md) | `Stable` | 共享节点池、`ref://` 引用和共享资源管理语义。 |
| [37_mcp_endpoint_adapter.md](37_mcp_endpoint_adapter.md) | `Draft` | `endpoint/mcp` 与 Matrix MCP adapter core 的当前实现事实。 |
| [38_validation_inspection_schema.md](38_validation_inspection_schema.md) | `Draft` | Validation report 与 inspection snapshot 的当前输出模型。 |
| [39_internal_error_code_specification.md](39_internal_error_code_specification.md) | `Stable` | Matrix core `aabbbcccc` 内部错误码的格式、命名空间分配、传播与兼容边界。 |

## 4. 维护规则

1. 新增 Matrix repo-local RFC、ADR、Plan、Reference 或 Guide 时，必须同步更新本文档。
2. RFC 进入 `Accepted` 或 `Implementing` 后，必须能追踪到当前有效的 Reference 或 Guide。
3. Plan 标记为 `Stable` 前，必须确认当前事实已回写到 Reference，操作流程已回写到 Guide。
4. 本文只做索引，不写 rationale；rationale 属于 ADR，阶段实施属于 Plan。
