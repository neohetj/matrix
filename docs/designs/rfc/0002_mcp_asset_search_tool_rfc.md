---
uuid: "45780fc0-420b-4e79-a366-5c740f9e3bcb"
type: "RFC"
title: "需求：开发者工具增强之MCP资产搜索工具"
status: "Draft"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "mcp"
  - "developer-tooling"
  - "asset-search"
relations:
  - type: "relates_to"
    target_uuid: "423a1d0c-0f81-45bd-bcc1-7ae64d519f65"
    description: "本 RFC 仍然服务于“代码优先、优先复用”的开发原则。"
---

# RFC: 开发者工具增强之 MCP 资产搜索工具

## 1. 摘要

本 RFC 提议提供一个面向开发者和 AI Agent 的资产搜索工具，用来发现 Matrix 生态里可复用的：

- rulechain
- endpoint
- shared DSL
- 节点元数据
- 函数注册元数据

## 原始需求点总结

1. 建立统一搜索入口：开发者和 Agent 需要一个统一入口，快速知道当前工作区或生态里已经有哪些 rulechain、endpoint、shared 定义和函数能力可以复用。
2. 避免重复造轮子：原始需求核心不是“做一个搜索产品”本身，而是减少重复实现，让人先搜现有资产，再决定是否新增节点、DSL 或函数。
3. 搜索范围不能只限 DSL 文件名：除了 `dsl/*` 资产，还应能搜索节点元数据、函数注册信息、文档和技能线索，形成“能力目录”。
4. 为 AI Agent 提供检索能力：原始需求里一个关键动机，是让 Agent 能通过结构化检索理解现有能力，而不是只能靠大范围读代码。
5. 搜索能力应尽量复用已有边界：搜索源应优先建立在现有 DSL、registry 和文档图谱之上，而不是额外造一套与框架脱节的资产模型。

## 2. 当前状态

截至当前代码状态，这份 RFC 仍然是**提案**，尚未在仓库内看到对应实现：

- 没有 `search_matrix_assets` MCP tool
- 没有独立的资产索引服务
- 没有对应的 Matrix runtime 扩展点
- 没有 Morpheus / WhiteRoom / Matrix 内置的资产搜索后端

换句话说，它描述的是一个**外部开发者工具方向**，而不是已经存在的 Matrix 运行时能力。

## 3. 如果未来实施，建议遵守的边界

为了和当前实现保持一致，未来如果继续做这件事，建议把数据源限定为现有可稳定解析的资产，而不是重新发明一套“搜索专用模型”。

### 3.1 可作为搜索源的现有资产

- `dsl/rulechains`
- `dsl/endpoints`
- `dsl/shared`
- 已注册节点的 `NodeMetadata`
- 已注册函数的 `NodeFuncObject` / `FuncObject`
- skills 元数据与引用文档

### 3.2 不应直接耦合到运行时核心

这个工具更适合作为：

- MCP server
- CLI / indexer
- 构建时或离线索引服务

而不应强行把索引逻辑耦合进 Matrix runtime 的消息执行链。

## 4. 与当前实现的关系

当前仓库已经有一批可以复用的基础信息来源，例如：

- 组件发现路径
- shared / endpoint / rulechain 的 DSL 装载目录
- 节点与函数的静态元数据

但这些能力目前只服务于加载、校验和执行，并没有直接暴露成搜索产品。

## 5. 现阶段如果要推进，应复用哪些现成边界

为了避免未来实现再次和代码脱节，这份提案如果继续推进，建议直接建立在现有边界之上：

1. 以 DSL 目录扫描结果作为资产发现入口
2. 以 `NodeMetadata` / `NodeFuncObject` 作为节点与函数的静态索引来源
3. 以 `metadata.relations`、`ruleChain.attrs.imports` 等现有结构作为关联关系来源
4. 把搜索能力做成独立工具层，而不是嵌进 runtime 执行链

## 6. 结论

这份 RFC 继续保留为有效提案，但不能被误读为“已经有这项能力”。如果后续要推进，应把它作为**开发者工具链项目**来做，而不是作为“Matrix 核心已实现功能”来引用。
