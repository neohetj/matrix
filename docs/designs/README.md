---
uuid: "c4bccbec-f333-4f8c-bd21-1203cb04fdec"
type: "README"
title: "README: Matrix 设计文档总览"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "docs"
  - "designs"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "a422d409-4b02-431a-b14e-2dec8f75b506"
    description: "本目录是 Matrix 文档总览下的设计文档入口。"
---

# Matrix 设计文档总览

本目录承载 `Matrix` 的设计阶段产物。共享文档规则请先参考 [docs/README.md](../README.md)，本目录只补充设计文档家族的分工说明。

## 1. 子目录分工

- [rfc/README.md](./rfc/README.md): 记录需求提案、设计动机与高层方案。
- [adr/README.md](./adr/README.md): 记录架构决策、取舍和长期影响。
- [plan/README.md](./plan/README.md): 记录可执行的实现计划与阶段拆解。

## 2. 设计文档关系约定

1. `RFC` 描述“为什么做、打算怎么做”。
2. `ADR` 记录“关键设计最终怎么定，以及为什么这样定”。
3. `Plan` 记录“如何分阶段落地到工程实现”。

一个功能可以只有 RFC，也可以逐步补齐 ADR 和 Plan，但三类文档都应通过 `relations` 显式建立关系。
