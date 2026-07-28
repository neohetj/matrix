---
uuid: "6c2ad10a-5f92-4832-b050-ec08026ffb42"
type: "Reference"
title: "README: Matrix ADR 文档库"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "adr"
  - "design"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "c4bccbec-f333-4f8c-bd21-1203cb04fdec"
    description: "本 ADR 库是 Matrix 设计文档体系的一部分。"
---

# Matrix ADR 文档库

本目录用于记录已经做出的架构决策。共享规则见 [设计文档总览](../README.md) 和 [Matrix 文档总览](../../README.md)。

## 1. ADR 目录规则

1. 文件名必须遵循 `NNNN-M_<description>_adr.md`。
2. `status` 只允许使用 `Draft / Accepted / Rejected / Superseded`。
3. 每份 ADR 都必须至少有一条 `type: "realizes"` 且指向父 RFC 的关系。
4. 新 ADR 应从 [adr_template.md](../../templates/adr_template.md) 创建。

## 2. 文档索引

- [0000-1_dag-vs-cyclic-graph_adr.md](./0000-1_dag-vs-cyclic-graph_adr.md): 记录 Matrix 规则链坚持 DAG 设计的正式决策。
- [0006-1_ops-modeling-and-dsl-extensions_adr.md](./0006-1_ops-modeling-and-dsl-extensions_adr.md): 记录运维实体建模与 DSL 扩展的关键决策。
- [0007-1_matrix_refactor_design_adr.md](./0007-1_matrix_refactor_design_adr.md): 细化 RFC-0007 的重构抽象层与适配器设计。
- [0008-1_agent_core_nodes_design_adr.md](./0008-1_agent_core_nodes_design_adr.md): 细化通用 Agent 核心节点的架构决策。
