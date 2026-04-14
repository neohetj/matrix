---
uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
type: "README"
title: "README: Matrix 参考文档库"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "reference"
  - "documentation"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "a422d409-4b02-431a-b14e-2dec8f75b506"
    description: "本参考库是 Matrix 项目文档体系的一部分。"
---

# Matrix 参考文档库

本目录用于解释 `Matrix` 的架构、核心概念、设计哲学与底层实现。共享规则见 [Matrix 文档总览](../README.md)。

## 1. 目录定位

- `reference/` 回答“是什么、为什么这样设计”。
- `guides/` 回答“如何落地使用或实现”。
- 当一个主题需要沉淀为稳定规范时，应优先放在 `reference/`。

## 2. 文件命名规则

本目录下的正式文档应使用 `NN_<topic_description>.md`，其中：

1. `NN` 为两位数字，表示推荐阅读顺序。
2. 主题描述使用小写加下划线。
3. 不在本目录内引入 `-M` 次级编号；更细粒度拆分应通过正文标题组织。

## 3. 文档索引

- [03_architecture_overview.md](./03_architecture_overview.md): Matrix 整体架构概览。
- [06_message_design_philosophy.md](./06_message_design_philosophy.md): `RuleMsg` 的设计哲学。
- [08_node_development_patterns.md](./08_node_development_patterns.md): 平台级扩展开发的官方入口。
- [09_core_objects.md](./09_core_objects.md): `CoreObj` 与 `DataT` 的核心数据契约。
- [10_http_endpoint_deep_dive.md](./10_http_endpoint_deep_dive.md): `endpoint/http` 的深度解析。
- [11_function_registration_spec.md](./11_function_registration_spec.md): 函数节点的开发与注册规范。
- [12_node_specification.md](./12_node_specification.md): 通用节点生命周期、数据契约与共享资源规范。
- [15_shared_resource_management.md](./15_shared_resource_management.md): 共享节点池与 `ref://` 资源管理。
- [18_dsl_specification.md](./18_dsl_specification.md): 规则链 DSL 语法规范。
- [21_component_catalog.md](./21_component_catalog.md): 当前内建组件目录。
- [33_component_design_principles.md](./33_component_design_principles.md): 组件设计原则。
- [36_testing_strategy.md](./36_testing_strategy.md): 测试策略与验证建议。
