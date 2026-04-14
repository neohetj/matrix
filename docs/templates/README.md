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

本目录存放 `RFC / ADR / Plan / ComponentGuide` 等文档模板。模板目录是 `docs/` 树中的特殊目录，允许出现占位值，但这些占位值只允许留在模板中。

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
