---
uuid: "245d6065-8675-4614-8d7a-d3c97afb72f8"
type: "README"
title: "README: Matrix 文档模板库"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "docs"
  - "templates"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "a422d409-4b02-431a-b14e-2dec8f75b506"
    description: "本目录是 Matrix 文档模板与占位符规范的统一入口。"
---

# Matrix 文档模板库

本目录存放 Matrix repo-local 的 `RFC / ADR / Plan / ComponentGuide` 等文档模板。Matrix 模板扩展 Evolution 基础模板库 `docs/templates/`：当 Matrix 本地模板没有覆盖某类文档时，应使用 Evolution 基础模板；Matrix 本地模板只能增加 Matrix 组件、节点和校验相关约束，不能放宽 Evolution 的 frontmatter、命名、relation、生命周期和决策追踪规则。

## 1. 模板占位规则

以下占位值只允许出现在 `docs/templates/` 中：

- `GENERATED_UUID`
- `[UUID of parent RFC]`
- `[UUID of related doc]`

当模板被复制到其他目录后，上述占位值都必须被真实 UUID 替换，否则文档审计会报错。

## 2. 模板索引

- [adr_template.md](./adr_template.md): ADR 模板，要求 `realizes -> parent RFC`。
- [node_guide_template.md](./node_guide_template.md): 组件指南模板，适用于 `guides/components`。
- [plan_template.md](./plan_template.md): Plan 模板，要求 `is_plan_for -> parent RFC`。
- [rfc_template.md](./rfc_template.md): RFC 模板，适用于新提案。
