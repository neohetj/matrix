---
uuid: "9e38d28b-cd7f-49b6-b099-23fdee01329e"
type: "RFC"
title: "需求：重构Matrix框架以提升内聚性"
status: "Accepted"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "matrix"
  - "refactor"
  - "cohesion"
relations:
  - type: "is_explained_by"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "项目总览已经按当前统一入口与框架职责描述 Matrix。"
  - type: "references"
    target_uuid: "60c07c47-df0e-4b76-9ed9-62fabe2e2add"
    description: "当前节点与函数开发入口以 Reference-08 为统一阅读起点。"
  - type: "references"
    target_uuid: "f745eae6-f75c-4849-b7fb-407d6c439182"
    description: "当前通用节点生命周期、数据契约和共享资源规范以 Reference-12 为准。"
---

# RFC: 重构 Matrix 框架以提升内聚性

## 1. 摘要

这份 RFC 的主要目标已经落地：Matrix 现在拥有更完整的统一入口、组件发现、DSL 装载、shared / endpoint 加载和 runtime 初始化流程，不再需要依赖早期文档中描述的“宿主先组装好一切再交给 Matrix”模式。

## 原始需求点总结

1. 回收框架装载职责：原始需求希望把 loader、runtime、shared、endpoint、rulechain 等核心装载过程收回 Matrix 核心，而不是让宿主项目自己拼装。
2. 建立统一入口：Matrix 应提供一个稳定、清晰的统一初始化入口，让宿主不必知道过多内部创建细节。
3. 提升内聚性与可维护性：原始问题之一是框架能力散落在宿主和扩展层，导致边界模糊、排障困难、重构成本高。
4. 统一发现与注册机制：框架应自己负责发现 DSL 目录、shared 节点、endpoints 和 runtimes，并形成一致的生命周期。
5. 澄清 Matrix 与宿主职责：这份 RFC 原本也试图回答“哪些能力应由 Matrix 持有，哪些应留给宿主组合层”这一长期边界问题。

## 2. 当前实现对齐

### 2.1 统一入口

当前统一入口是：

- `matrix.New(cfg, opts...)`

对应实现位于：

- `matrix.go`

### 2.2 当前 `MatrixEngine` 已提供的能力

`MatrixEngine` 当前已经暴露：

- `RuntimePool()`
- `SharedNodePool()`
- `NodeManager()`
- `NodeFuncManager()`
- `TraceManager()`
- `BizConfig()`
- `Loader()`

### 2.3 当前初始化流程

当前 `New()` 内部已经会完成这些工作：

1. 根据配置创建 loader
2. 发现 rulechain / endpoint / shared 路径
3. 装载 rulechain 定义
4. 装载 shared nodes
5. 装载 endpoints
6. 创建并注册 runtimes

这说明 RFC 里强调的“把框架内聚回 Matrix 核心”的目标已经基本实现。

## 3. 与原 RFC 的差异

原 RFC 中有些路径和术语已经不再准确：

1. 文中仍然用大量 `Trinity` / `trinity/matrixext` 的未来式叙述
2. 文中假设有 `pkg/loader/` 作为核心实现落点，但当前 loader 创建能力主要落在 builder 流程里
3. 一些“待重构”的事项，在当前代码里已经不是待办，而是现状

## 4. 当前仍然不应过度承诺的部分

虽然内聚性重构主体已经完成，但这并不意味着：

- 所有宿主适配逻辑都已经完全内建
- 所有平台集成层都已经统一
- 所有上层产品都不再需要自己的组合层

也就是说，这份 RFC 已经完成的是“**核心装载与运行时入口内聚**”，而不是“所有外围平台完全归并”。

## 5. 结论

本 RFC 应视为**已接受且主要目标已实现**。后续如果继续讨论 Matrix 与宿主应用之间的职责边界，应在此基础上做增量设计，而不是继续沿用本文中的未来式表述。

## 6. 相关现行文档

1. [Matrix 项目总览：README First](../../guides/00_matrix_guide.md)
2. [参考-08：Matrix 节点开发模式](../../reference/08_node_development_patterns.md)
3. [参考-12：通用节点规范](../../reference/12_node_specification.md)
