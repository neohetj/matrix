---
uuid: "affcadae-091e-4f63-9072-c3c8216e8f40"
type: "Reference"
title: "README: Matrix 迁移指南库"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "migration"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "a422d409-4b02-431a-b14e-2dec8f75b506"
    description: "本目录是 Matrix 文档树中的迁移指南入口。"
---

# Matrix 迁移指南库

本目录用于记录版本迁移、兼容性调整和不兼容变更的落地说明。共享规则见 [Matrix 文档总览](../README.md)。

## 1. 编写要求

当框架发生由 RFC、ADR 或实现落地触发的破坏性变更时，应在本目录补充迁移指南，并明确：

1. 变更概述与影响范围。
2. 需要调整的 API、DSL、配置或目录结构。
3. 手动迁移步骤与验证方式。
4. 若存在自动迁移脚本，应说明入口与限制。

## 2. 文档索引

- [20250723_data_contract_adoption_guide.md](./20250723_data_contract_adoption_guide.md): 数据契约迁移指南。
