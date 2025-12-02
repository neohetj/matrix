---
# === Node Properties: 定义文档节点自身 ===
uuid: "dev-patterns-entrypoint-20250911"
type: "ArchitectureOverview"
title: "参考-08: Matrix节点开发模式"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "reference"
  - "development-pattern"
  - "architecture"
  - "node"
  - "function"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "supersedes"
    target_uuid: "c1d2e3f4-a5b6-4c7d-8e9f-0a1b2c3d4e5f" # -> 27_node_development_patterns.md
    description: "本文档整合并取代了旧的节点开发模式文档。"
  - type: "references"
    target_uuid: "ref-func-registration-spec-20250911" # -> 11_function_registration_spec.md
    description: "本文档引导开发者在需要时，查阅更详细的函数开发规范。"
  - type: "references"
    target_uuid: "node-spec-20250911" # -> 12_node_specification.md (placeholder)
    description: "本文档引导开发者在需要时，查阅更详细的通用节点规范。"
---

# 参考-08: Matrix节点开发模式

本文档是 `Matrix` 生态系统中进行扩展开发的**唯一官方入口**。它旨在帮助开发者根据其目标，自顶向下地做出正确的技术选型，并找到对应的详细开发规范。

## 1. 决策第一步：明确你的开发目标 (DefineYourGoal)

在编写任何代码之前，必须首先回答这个核心问题：**你的代码是服务于“平台”还是“应用”？**

| 对比维度 | 模式A：开发平台级通用能力 | 模式B：开发应用级业务逻辑 |
| :--- | :--- | :--- |
| **核心目标** | 为 **Matrix 平台**贡献一个**可被所有业务项目复用**的通用能力。 | 为 **Trinity 应用**实现一个**特定的业务需求**。 |
| **思维模式** | “我正在为框架添加一个基础构建块。” | “我正在用框架的构建块来实现一个业务功能。” |
| **产出物示例** | 通用的`HTTP请求`节点、`Redis`缓存操作函数、`数据格式转换`节点。 | `创建数据集`的业务流程、`校验用户权限`的函数、`同步订单到CRM`的函数。 |
| **代码位置** | `Architect/matrix/pkg/` | `Architect/trinity/matrixext/` |
| **下一步** | 继续阅读 `## 2. 决策第二步` | 跳转至 `Trinity` 相关开发文档。 |

## 2. 决策第二步：选择正确的实现模式 (ChooseImplementation)

如果你已确定目标是为 `Matrix` 平台开发通用能力（模式A），你还需要在两种实现方式中做出选择：**重量级的“节点”** 还是 **轻量级的“函数”**？

| | 通用组件节点 (Component Node) | “函数” (Function) |
| :--- | :--- | :--- |
| **核心职责** | 提供一个**平台级**的、通用的、可复用的能力。 | 提供一个**轻量级**的、具体的、可被调用的业务逻辑片段。 |
| **实现方式** | **重量级**。需完整实现 `types.Node` 接口，拥有独立的生命周期（`Init/Destroy`）。 | **轻量级**。只需实现一个特定的函数签名（`types.NodeFunc`），**它本身不是节点**。 |
| **调用方式** | 作为独立的节点方块，在规则链DSL中通过其`type`直接引用。 | **被**一个通用的 `functions` 节点根据`functionName`配置项动态调用。 |
| **何时选择** | 当你的能力需要复杂的配置、独立的生命周期管理或复杂的、多阶段的内部逻辑时。 | 当你的能力是一个相对简单的、可以被概括为“输入-处理-输出”的原子操作时。 |
| **设计模式** | 独立的构建块。 | “通用调用器 + 可注册插件”模式。 |
| **下一步** | 阅读 **[参考-12: 通用节点规范][Ref-NodeSpec]** | 阅读 **[参考-11: 函数开发与注册规范][Ref-FuncSpec]** |

## 3. 决策第三步：理清你的依赖类型 (ClarifyDependencies)

在进入具体实现之前，最后一个关键决策是理清你的节点/函数需要处理的依赖类型。`Matrix` 框架严格区分了两种依赖，并提供了完全不同的机制来处理它们。

| | 业务数据依赖 | 共享资源依赖 |
| :--- | :--- | :--- |
| **依赖的是什么？** | 在规则链中**流动**的、每个消息都可能不同的**业务对象**。 | 在应用生命周期内**静态**的、昂贵的、可被多方**共享的资源实例**。 |
| **示例** | 当前处理的用户订单、上一步生成的报告、需要被更新的数据库记录。 | 数据库连接池、Redis客户端、HTTP客户端。 |
| **核心机制** | `RuleMsg` + `DataT` | `SharedNode` + `NodePool` |
| **声明方式** | 在节点/函数的`Inputs/Outputs`元数据中声明**逻辑参数名 (P-Key)**。 | 在节点/函数的`Business`配置中提供资源的**引用路径 (ref://)**。 |
| **获取方式** | `msg.DataT().GetByParam(ctx, "p-key")` | `connectors.GetDBConnection(pool, "ref://...")` |
| **深入学习** | 见 **[参考-12: 通用节点规范][Ref-NodeSpec]** 中关于 `DataT` 和参数解耦的章节。 | 见 **[参考-12: 通用节点规范][Ref-NodeSpec]** 中关于 `SharedNode` 和 `ref://` 协议的章节。 |

<!-- qa_section_start -->
> **问：我应该在Trinity中开发“业务节点”还是“业务函数”？**
> **答：** **两者皆可，但强烈推荐优先选择“函数”模式。** Trinity的实践表明，“函数”模式是实现具体业务步骤的首选，因为它更轻量、更易于测试和复用。只有当你的业务逻辑非常复杂，需要管理自己的生命周期或内部状态时，才应考虑实现一个完整的“业务节点”。

> **问：这两种模式的选择是互斥的吗？**
> **答：** 不是。一个复杂的“业务节点”（模式B）在其内部实现中，完全可以调用一个或多个“函数”（模式A）来完成其工作。例如，一个“同步订单”的业务节点，可能会在其`OnMsg`方法中，调用一个通用的`sqlQuery`函数来查询数据库，再调用一个`httpRequest`函数来将结果推送到外部系统。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Ref-FuncSpec]: ./11_function_registration_spec.md
[Ref-NodeSpec]: ./12_node_specification.md
