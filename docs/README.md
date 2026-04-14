---
uuid: "a422d409-4b02-431a-b14e-2dec8f75b506"
type: "README"
title: "README: Matrix 文档总览"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "docs"
  - "readme"
  - "governance"
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本总览文档是 Matrix 项目文档体系的统一入口。"
---

# Matrix 文档总览

本目录是 `Matrix` 仓库的文档根目录。`docs/` 下的所有内容都应遵循同一套共享规则，再由各子目录的 `README.md` 叠加目录级约束。

## 1. 共享规则

除 `docs/templates/` 内的模板外，所有 Markdown 文档都必须满足：

1. 具有 YAML frontmatter，且至少包含 `uuid`、`type`、`title`、`status`、`owner`、`version`、`tags`、`relations`。
2. `uuid` 与所有 `target_uuid` 都必须使用标准小写连字符 UUID。
3. `relations` 可以为空数组，但不能保留空的 `target_uuid`。
4. 指向仓外实体时，`target_uuid` 仍需是 UUID，且 `description` 必须明确包含 `external`。
5. 每个 `docs/` 子目录都必须有自己的 `README.md`，并完整索引该目录下的直接子文件或子目录。

## 2. 一级目录索引

- [designs/README.md](./designs/README.md): RFC、ADR、Plan 等设计文档入口。
- [guides/README.md](./guides/README.md): 面向使用和实现的操作指南。
- [latest/README.md](./latest/README.md): 特殊目录，用于保留最新导出物或留空占位说明。
- [migration/README.md](./migration/README.md): 版本迁移与兼容性说明。
- [reference/README.md](./reference/README.md): 架构、规范与原理说明。
- [templates/README.md](./templates/README.md): 文档模板与占位符规则。

## 3. 根目录独立文档

- [generic_node_impl_guide-EN.md](./generic_node_impl_guide-EN.md): Generic node implementation guidelines in English.
- [generic_node_impl_guide-ZH.md](./generic_node_impl_guide-ZH.md): 通用节点实现规范与最佳实践（中文）。
- [how_to_add_basic_sid-EN.md](./how_to_add_basic_sid-EN.md): How to add a basic SID.
- [how_to_add_basic_sid-ZH.md](./how_to_add_basic_sid-ZH.md): 如何增加基础 SID。
