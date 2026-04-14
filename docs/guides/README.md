---
uuid: "b4c5d6e7-f8a9-0b1c-2d3e-4f5a6b7c8d9e"
type: "README"
title: "README: Matrix 指南文档库"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "guide"
  - "documentation"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "a422d409-4b02-431a-b14e-2dec8f75b506"
    description: "本指南库是 Matrix 项目文档体系的一部分。"
---

# Matrix 指南文档库

本目录承载面向任务的使用与实现指南。共享规则见 [Matrix 文档总览](../README.md)。

## 1. 目录定位

- `guides/` 用于解释“如何做”。
- `reference/` 用于解释“是什么”与“为什么”。
- 组件级指南统一放在 [components/README.md](./components/README.md) 所管理的子目录中。

## 2. 命名建议

本目录下的非组件指南通常使用 `NN[-M]_<topic_description>.md`，其中：

1. `00` 预留为目录入口类总览。
2. 顶级主题使用两位数字排序。
3. 次级变体可使用 `-M` 编号。

## 3. 文档索引

- [00_matrix_guide.md](./00_matrix_guide.md): Matrix 开发路径与学习入口。
- [03_shared_node_guide.md](./03_shared_node_guide.md): 共享节点的概念与使用方法。
- [06_dynamic_object_conversion_guide.md](./06_dynamic_object_conversion_guide.md): 动态数据处理技巧。
- [09_e2e_alert_processing_guide.md](./09_e2e_alert_processing_guide.md): 端到端告警处理案例。
- [12_config_uri_usage_guide.md](./12_config_uri_usage_guide.md): `config://` 协议、scope 回退与 helper 读取方式。
- [15_unified_error_handling_guide.md](./15_unified_error_handling_guide.md): `Fault -> FailureInfo -> ServiceError` 与 HTTP 错误映射。
- [components/README.md](./components/README.md): 组件级使用指南与编写规范。
