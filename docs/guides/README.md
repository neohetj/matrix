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

## 2. 命名规则

本目录下的非组件指南使用 `<slug>-guide.md`，与 Evolution 文档治理规范一致（审计器按该规则检查）：

1. `slug` 使用小写字母、数字和连字符，描述指南主题。
2. 不使用数字前缀排序；阅读入口与顺序由本 README 的文档索引给出。
3. 目录入口类总览命名为 `matrix-guide.md`。

`reference/` 目录仍保留 `NN_<topic>.md` 编号，因为其编号被正文与 relation 以 `Reference-NN` 形式稳定引用；guides 不存在这类引用，故不编号。

## 3. 文档索引

- [matrix-guide.md](./matrix-guide.md): Matrix 开发路径与学习入口。
- [shared-node-guide.md](./shared-node-guide.md): 共享节点的概念与使用方法。
- [dynamic-object-conversion-guide.md](./dynamic-object-conversion-guide.md): 动态数据处理技巧。
- [e2e-alert-processing-guide.md](./e2e-alert-processing-guide.md): 端到端告警处理案例。
- [config-uri-usage-guide.md](./config-uri-usage-guide.md): `config://` 协议、scope 回退与 helper 读取方式。
- [unified-error-handling-guide.md](./unified-error-handling-guide.md): `Fault -> FailureInfo -> ServiceError` 与 HTTP 错误映射。
- [components/README.md](./components/README.md): 组件级使用指南与编写规范。
