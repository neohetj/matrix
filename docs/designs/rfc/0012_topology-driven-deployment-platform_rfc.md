---
uuid: "f1d4c850-1f6e-4dcb-8d51-1fe6a46701ae"
type: "RFC"
title: "需求：拓扑驱动的多视图部署与自动分发平台"
status: "Draft"
owner: "neohetj"
version: "2.0.0"
tags:
  - "rfc"
  - "design"
  - "ops"
  - "deployment"
  - "topology"
  - "morpheus"
relations:
  - type: "extends"
    target_uuid: "978c1b44-65eb-43ef-bcf8-793c1793a0b3"
    description: "本 RFC 建立在 ops 基础节点与 DSL 扩展之上。"
---

# RFC: 拓扑驱动的多视图部署与自动分发平台

## 1. 摘要

这份 RFC 仍然是**平台级 Draft**，但它依赖的若干基础能力已经在 Matrix 内部先落地了。

## 原始需求点总结

1. 以拓扑为单一事实源：原始需求希望把应用、服务、数据库、网络、Runner 等对象统一沉淀成可复用拓扑，而不是把部署信息散落在多个系统里。
2. 从同一拓扑投影出多视图：同一份拓扑数据需要同时服务于可视化、部署编排、巡检、生成器和资产管理等不同视角。
3. 打通部署闭环：不仅要描述拓扑，还希望进一步生成 bundle、分发到 runner，并形成自动部署/执行链。
4. 分离建模与执行：原始提案强调拓扑建模层与部署 controller、执行器、分发 DAG 应分层设计，避免把平台流程直接写死在 DSL 里。
5. 让 Morpheus 等上层有稳定基础：这份 RFC 的目标不是只改 Matrix 内核，而是为上层平台提供一个统一的部署与多视图底座。

## 2. 当前实现对齐

### 2.1 已具备的前置能力

当前仓库已经具备这些基础设施：

1. `ops/*` 拓扑节点
2. `ruleChain.attrs.viewType`
3. `ruleChain.attrs.imports`
4. `metadata.relations`
5. shared / endpoint / rulechain 多目录发现与加载
6. `RuntimePool.ListByViewType(...)`

这些能力为“单一拓扑源 + 多视图投影”提供了底座。

### 2.2 当前仍未看到的核心平台能力

截至当前代码状态，以下平台级能力仍未在 Matrix / Morpheus 中形成闭环：

1. 部署 controller
2. bundle generator
3. `deploy-runner` 执行面
4. 自动分发 DAG
5. Morpheus 中对应的完整部署视图与操作面

因此本文还不能被视为“已实现方案说明”。

## 3. 如何理解当前状态

更准确的说法是：

- `0006` 提供的基础 DSL / ops 建模能力已经部分落地
- `0012` 描述的平台闭环仍然主要停留在设计层

## 4. 当前引用建议

如果只讨论：

- ops 拓扑节点
- relations / imports / viewType
- 静态多视图投影前提

可以把它们当作当前 Matrix 已提供的基础能力。

如果讨论：

- 自动部署
- Runner 分发
- Morpheus 操作面
- 部署 bundle 生成

则仍然应将本文视作提案，而不是现有系统能力说明。

## 5. 结论

本文继续保持 Draft，但其“底座依赖”已经不再是空想。后续若推进实现，应把“已具备前置能力”和“尚未建设的平台闭环”严格区分开来。
