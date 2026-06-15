---
uuid: "a4b5c6d7-e8f9-4a0b-1c2d-3e4f5a6b7c8d"
type: "RFC"
title: "需求：固化“函数”模式并重构其代码结构"
status: "Superseded"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "design"
  - "functions"
  - "architecture"
relations:
  - type: "superseded_by"
    target_uuid: "dbf6fb21-5e26-44b6-b1bb-d94d13112ae9"
    description: "函数模式的现行规范已收敛到 Reference-11。"
---

# RFC: 固化“函数”模式并重构其代码结构

> Historical note: 这份 RFC 的核心判断“函数模式需要被正式化”是对的，但正文里的“创建 `pkg/functions/` 并整体迁移源码”“新增 `0002_function_node_pattern.md` ADR”并没有按原方案落地。

## 1. 原始目标

本 RFC 试图解决两个问题：

1. 函数模式是隐性知识
2. 开发者容易把“通用组件节点”和“函数节点”混为一谈

这两个问题在当前实现里已经通过**统一函数注册模型 + 文档规范**基本解决，但不是通过文件系统迁移解决。

## 原始需求点总结

1. 把函数模式正式化：原始需求希望把“通过一个通用执行器节点调用注册函数”从实践技巧升级为框架正式能力，而不是继续靠口口相传。
2. 明确函数与节点的职责差异：函数应该更轻量、聚焦业务处理片段；通用组件节点则承担独立生命周期、复杂配置和更重的运行时职责。
3. 给函数建立统一注册与调用模型：需要有稳定的函数 ID、配置声明、输入输出描述和 DSL 调用方式，避免每个业务模块各玩一套。
4. 降低业务扩展门槛：相比写一个完整 Node，函数模式应让开发者更快地增加一段可复用业务逻辑，并便于测试与复审。
5. 提升工程结构清晰度：原始提案当时还希望通过目录和文档边界，把“平台通用函数能力”与“业务模块内逻辑”在工程层面拉开。

## 2. 当前实现对齐

### 2.1 核心执行器

当前函数节点的通用执行器是：

- `type: "functions"`

对应实现位于：

- `internal/builtin/base/functions_node.go`

### 2.2 注册模型

函数的注册单元是：

- `types.NodeFuncObject`

函数元数据和配置模型位于：

- `types.FuncObject`
- `types.FuncObjConfiguration`

### 2.3 当前推荐模式

当前实现和文档推荐的模式是：

1. 用 `NodeFuncObject` 注册函数
2. 在 DSL 中统一使用 `type: "functions"`
3. 用 `configuration.functionName` 指定函数 ID
4. 用 `Inputs` / `Outputs` / `Business` / `FuncReads` / `FuncWrites` 描述输入输出和配置

这套模式已经是当前稳定实现，不再依赖旧文档里假设的目录迁移。

## 3. 与原 RFC 的主要偏差

原 RFC 中这些内容不再成立：

1. “函数源码位于 `pkg/components/functions/`，需要整体迁移到 `pkg/functions/`”
2. “存在一个 `designs/adr/0002_function_node_pattern.md` 作为正式 ADR”
3. “通过物理目录隔离来定义函数模式”

当前仓库里：

- 没有 `pkg/components/functions`
- 也没有 `pkg/functions`
- 函数模式的稳定边界来自注册模型、运行时校验和 Reference 文档，而不是来自一个专用目录

## 4. 现行规范入口

函数模式的当前规范和辅助材料已经在这些地方稳定存在：

- `docs/reference/11_function_registration_spec.md`
- `docs/reference/12_node_specification.md`
- `docs/designs/rfc/0013_function_routing_constraints_rfc.md`
- `skills/matrix-function-node-creator/`

## 5. 结论

这份 RFC 作为“函数模式需要被正式化”的历史提案保留，但其具体实施方案已被后续实现替代。新的函数节点设计与改造，不应再参考本文中的目录迁移方案。
