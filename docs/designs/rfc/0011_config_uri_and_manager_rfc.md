---
uuid: "7144507b-2c98-41e1-aeaa-34d4ccad476a"
type: "RFC"
title: "需求：Config URI 协议与规则链统一配置视图"
status: "Implementing"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "config"
  - "uri"
  - "design"
relations:
  - type: "has_guide"
    target_uuid: "788737cf-d58e-4931-8bbf-37841c6de80c"
    description: "当前 `config://` 协议与 helper 读取方式见对应指南。"
  - type: "is_specified_by"
    target_uuid: "5a8b3c7e-9f0d-4a1b-8c2d-6e5f4a3b2c1d"
    description: "当前 DSL 侧 URI 使用约定以 Reference-18 为准。"
---

# RFC: Config URI 协议与规则链统一配置视图

## 1. 摘要

这份 RFC 已经**部分实现**：

1. `config://` 资产协议已实现
2. scope / default / env 回退逻辑已实现
3. 规则链“统一配置视图”与对应管理 API 仍未实现

## 原始需求点总结

1. 统一配置读取入口：原始需求希望节点、函数和 helper 不再各写一套取配置逻辑，而是通过统一的 `config://` 协议读取配置。
2. 支持分层回退：配置读取应支持业务级、节点级、引擎级和环境变量等多层来源，并定义稳定的查找顺序。
3. 支持默认值与缺省兜底：配置协议应让 DSL 和代码能统一表达“如果没配就用什么值”，而不是在每个函数里散写默认值逻辑。
4. 形成统一配置视图：除了解析协议，原始需求还希望最终能够把规则链的配置项集中展示、扫描和编辑。
5. 降低配置分散问题：这份 RFC 背后的核心痛点，是配置散落在 engine、node、business 和 env 之间，缺乏统一心智模型和治理入口。

## 2. 当前实现对齐

### 2.1 协议实现位置

当前 `config://` 协议实现位于：

- `pkg/asset/config_asset.go`

### 2.2 当前支持的能力

当前已经支持：

- `config:///path`
- `scope=...`
- `default=...`
- 环境变量兜底
- engine / node / business / env 多层回退

### 2.3 当前默认 scope

当前默认查找顺序不是 RFC 初稿里的 `node,engine`，而是：

- `business,node,engine,env`

这是一个重要差异。

## 3. 与原 RFC 的主要差异

### 3.1 URI 推荐写法

当前更推荐的规范写法是：

- `config:///some.key`

而不是把 path 写在 host 部分的 `config://some.key` 形式。

### 3.2 节点作用域

当前 node scope 并不依赖一个统一强制的 `Variables` 字段模型，而是优先从当前节点配置 map 中读取；如果有 `business` 区，还会先查 `business`。

这意味着原 RFC 里“统一增加 `variables` 字段”的说法，不是当前实现的硬性事实。

### 3.3 已实现的是协议，不是配置管理 UI

当前代码里没有看到这些能力：

- 规则链配置概览 API
- config key 聚合扫描页面
- 节点级覆盖的统一编辑视图

所以本文中“统一配置视图”的部分，仍应视为未来工作。

## 4. 当前已实现范围

### 4.1 已实现

- config asset 解析
- 多 scope 查询
- engine / env fallback
- `default=` 默认值
- 环境变量 key 规范化（点号转下划线、大写）

### 4.2 未实现

- UI 配置总览
- 配置提取 API
- 面向规则链的集中配置编辑体验

## 5. 结论

本 RFC 当前应被理解为“**协议层已落地，管理视图层仍未落地**”。任何文档如果只讨论运行时配置解析，应引用已实现部分；如果讨论统一配置视图，则仍然属于提案范围。

## 6. 相关现行文档

1. [指南：Config URI 协议与统一配置读取](../../guides/config-uri-usage-guide.md)
2. [学习 Matrix DSL 规范](../../reference/18_dsl_specification.md)
3. [参考-11：函数开发与注册规范](../../reference/11_function_registration_spec.md)
