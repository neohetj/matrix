---
uuid: "fde86d18-74ce-4350-9137-553929ee1261"
type: "README"
title: "README: Matrix Plan 文档库"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "plan"
  - "design"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "c4bccbec-f333-4f8c-bd21-1203cb04fdec"
    description: "本 Plan 库是 Matrix 设计文档体系的一部分。"
---

# Matrix Plan 文档库

本目录用于记录可执行的实现计划。共享规则见 [设计文档总览](../README.md) 和 [Matrix 文档总览](../../README.md)。

## 1. Plan 目录规则

1. 文件名必须遵循 `NNNN-M_<description>_plan.md`。
2. `status` 只允许使用 `Draft / Implementing / Stable / Superseded`。
3. 每份 Plan 都必须至少有一条 `type: "is_plan_for"` 且指向父 RFC 的关系。
4. 新 Plan 应从 [plan_template.md](../../templates/plan_template.md) 创建。

## 2. 文档索引

- [0006-1_ops-components-and-dsl-impl_plan.md](./0006-1_ops-components-and-dsl-impl_plan.md): 运维基础组件与 DSL 扩展的实现计划。
- [0012-1_topology-driven-deployment-platform_plan.md](./0012-1_topology-driven-deployment-platform_plan.md): 拓扑驱动部署平台的阶段化工程蓝图。
- [0014-1_mcp_business_endpoint_adapter_plan.md](./0014-1_mcp_business_endpoint_adapter_plan.md): MCP 业务入口适配器的阶段化实施方案。
- [0015-1_matrix_core_interface_boundary_refactor_plan.md](./0015-1_matrix_core_interface_boundary_refactor_plan.md): Matrix core 接口边界重构与跨仓同步计划。
