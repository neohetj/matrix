---
uuid: "4d1b7352-8e0d-4df0-92ab-99f87927b6a1"
type: "RFC"
title: "需求：Function Routing Constraints"
status: "Accepted"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "function"
  - "routing"
  - "validation"
relations:
  - type: "is_formalized_by"
    target_uuid: "dbf6fb21-5e26-44b6-b1bb-d94d13112ae9"
    description: "当前规范性内容已并入 Reference-11。"
  - type: "is_explained_by"
    target_uuid: "98c5b4a3-e7f6-4b0d-8c6a-1e2f3a4b5c6d"
    description: "当前 `functions` 节点与决策函数的使用方式见组件指南。"
---

# RFC: Function Routing Constraints

## 1. 摘要

本 RFC 已实现，当前 Matrix 已对 `functions` 节点的路由模式施加明确约束：

1. 普通函数默认只允许 `Success` / `Failure`
2. 只有 `decision` 函数允许显式自定义 relation
3. 决策函数必须声明 `DeclaredRelations`
4. 运行时会在加载规则链时做校验

## 原始需求点总结

1. 给 `functions` 节点建立清晰路由边界：原始需求希望明确“普通函数负责处理，路由函数负责决策”，避免所有函数都能随意把控制流藏进代码里。
2. 把默认行为收紧：没有显式声明的普通函数，应只走 `Success / Failure`，这样 DSL 图和函数实现的语义才一致。
3. 允许受控的决策函数：框架仍然需要保留少量 `TellNext(customRelation)` 场景，但这些 relation 必须显式声明，不能靠约定俗成。
4. 让注册、规范和运行时一致：原始需求不只是写文档，而是希望函数元数据、注册校验、运行时装载和 DSL 连线使用同一套规则。
5. 降低可视化与排障成本：这份 RFC 背后的核心痛点，是函数里隐藏了过多控制流后，规则链图会失真，代码评审和排障都变得更难。

## 2. 当前实现对齐

这份 RFC 已对应到当前实现，主要落地内容包括：

1. `functions` 节点默认路由模式已收紧为 `standard`
2. `decision` 函数的 relation 声明与运行时校验已经落地
3. 规范性定义已并入 Reference-11，使用方式已写入组件指南

当前实现入口主要位于：

- `pkg/types/node.go`
- `internal/registry/func_manager.go`
- `internal/runtime/runtime.go`

## 3. 当前有效规则

### 3.1 Standard Function

- `routingMode` 省略时默认为 `standard`
- 不允许声明自定义 relation
- DSL 出边只能是 `Success` / `Failure`

### 3.2 Decision Function

- 必须显式声明 `routingMode: decision`
- 必须声明 `declaredRelations`
- 运行时只允许连接到已声明 relation 以及标准 `Success` / `Failure`

## 4. 当前校验发生在哪些阶段

当前约束不是“文档约定”，而是会在两个层面实际生效：

1. 函数注册与元数据构建阶段，会固化 `routingMode` / `DeclaredRelations`
2. 规则链装载与 runtime 校验阶段，会拒绝不符合路由约束的连线

这也是为什么它已经可以被视为当前实现的一部分，而不是停留在提案层。

## 5. 与现行文档的关系

这份 RFC 仍保留决策背景和迁移原因，但规范性内容已经收敛到：

- `docs/reference/11_function_registration_spec.md`

## 6. 结论

本文已与当前实现保持一致，可继续保留为“设计决策记录 + 迁移背景”，而真正的开发规范以 Reference-11 为准。

## 7. 相关现行文档

1. [组件指南：通用函数 (functions)](../../guides/components/action_functions_guide.md)
2. [参考-11：函数开发与注册规范](../../reference/11_function_registration_spec.md)
