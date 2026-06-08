---
uuid: "589dab79-3b5c-4231-ada6-b2617d375abb"
type: "RFC"
title: "需求：为节点声明分层的数据访问契约"
status: "Superseded"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "design"
  - "data-contract"
  - "static-analysis"
relations:
  - type: "superseded_by"
    target_uuid: "f745eae6-f75c-4849-b7fb-407d6c439182"
    description: "当前节点契约规范已合并进 Reference-12。"
---

# RFC: 为节点声明分层的数据访问契约

> Historical note: 本 RFC 保留的是早期数据契约设计草案。文中 `ReadsData`、`ReadsMetadata`、`WritesMetadata` 方案没有按原样落地，当前实现请以 `NodeReads` / `NodeWrites`、`FuncReads` / `FuncWrites` 和 `DataContract()` 为准。

## 1. 原始目标

这份 RFC 当年的目标是把“节点和函数到底读写了什么”从隐性知识变成显式声明，让静态分析、UI 展示和 DSL 校验都能有统一依据。

这个目标已经实现，但实现方式和 RFC 初稿不同。

## 原始需求点总结

1. 显式声明读写范围：节点和函数不应再依赖“阅读源码才能知道它改了什么”的隐性知识，而应在元数据层声明自己会读取和写入哪些消息区域。
2. 区分不同数据层次：原始设想里，希望至少把 `Data`、`Metadata`、`DataT` 这几类访问边界分开表达，避免“全部都算读写消息”导致静态分析失真。
3. 服务 DSL 校验与编辑体验：规则链编辑器、DSL 校验器和代码审查工具需要基于这些契约判断上下游节点是否真的连得上，而不是只看节点类型。
4. 服务可视化与文档生成：希望 UI、组件目录和自动文档能直接展示“这个节点依赖哪些输入、会产出哪些副作用”，降低上手和排障成本。
5. 降低副作用不透明问题：原始需求强调，消息读写必须变成框架级约束，否则节点/函数会悄悄改 metadata 或 data，导致链路排查困难。

## 2. 当前实现对齐

当前 Matrix 的数据契约模型以 URI 为核心，而不是以 `ReadsData` / `ReadsMetadata` 这样的分栏字段为核心。

### 2.1 节点侧

- `types.NodeMetadata` 使用 `NodeReads` / `NodeWrites`
- 每一项都是 `ContractDef`
- `ContractDef.URI` 使用统一 URI 约定，例如：
  - `rulemsg://data/...`
  - `rulemsg://metadata/...`
  - `rulemsg://dataT/<objId>...`

### 2.2 函数侧

- `types.FuncObjConfiguration` 使用：
  - `Inputs`
  - `Outputs`
  - `Business`
  - `FuncReads`
  - `FuncWrites`
- 函数节点真正的注册单元是 `types.NodeFuncObject`

### 2.3 统一视图

- 所有节点仍通过 `DataContract()` 暴露统一读写视图
- 运行时、DSL 工具和文档现在都围绕 URI 契约工作，而不是围绕旧版分层字段工作

## 3. 与原 RFC 的主要差异

原 RFC 中这些内容已经过时：

1. `NodeMetadata.ReadsData` / `ReadsMetadata` / `WritesMetadata`
2. `FuncObjConfiguration.ReadsData` / `ReadsMetadata` / `WritesMetadata`
3. “非函数节点只读 `Data/Metadata`、函数节点单独声明 DataT 交互”的拆分方式

当前代码采用的是更统一的 URI 契约模型，因此：

- 不再强调“Data / Metadata / DataT”三套字段各自独立声明
- 更强调“读取什么 URI、写入什么 URI”
- 更容易和 HTTP packet、object mapper、rulechain projection 等能力对齐

## 4. 现行规范入口

当前规范性内容已经迁移到这些文档：

- `docs/reference/12_node_specification.md`
- `docs/reference/11_function_registration_spec.md`
- `docs/migration/20250723_data_contract_adoption_guide.md`

## 5. 结论

这份 RFC 不再是现行规范，只保留为历史背景材料。任何新的节点、函数、DSL 校验或工具开发，都不应再参考本文中的旧字段设计。
