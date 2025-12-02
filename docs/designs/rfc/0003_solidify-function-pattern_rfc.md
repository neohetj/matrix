---
uuid: "a4b5c6d7-e8f9-4a0b-1c2d-3e4f5a6b7c8d"
type: "RFC"
title: "需求：固化“函数”模式并重构其代码结构"
status: "Draft"
owner: "@cline-agent"
version: "1.0.0"
tags:
  - "rfc"
  - "design"
  - "refactoring"
  - "functions"
relations:
  - type: "based_on"
    target_uuid: "c1d2e3f4-a5b6-4c7d-8e9f-0a1b2c3d4e5f" # -> 08_node_development_patterns.md
    description: "This RFC is a formal proposal based on the analysis in the node development patterns guide."
  - type: "based_on"
    target_uuid: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d" # -> 09_core_mechanisms_deep_dive.md
    description: "This RFC is a formal proposal based on the analysis in the core mechanisms deep dive."
---

# RFC: 固化“函数”模式并重构其代码结构 (SolidifyFunctionPatternAndRefactor)

## 1. 摘要 (Summary)

本RFC提议通过创建一份架构决策记录（ADR）和一次代码结构重构，来正式固化Matrix框架中的“通用调用器+可注册函数”设计模式。此举旨在将该模式从隐性知识转变为官方架构，并使其代码的物理结构与逻辑结构保持一致，从而为开发者提供更清晰的扩展指引。

## 2. 动机 (Motivation)

*   **当前存在的问题**:
    1.  **隐性知识**: “函数”作为一种轻量级的节点扩展模式，在`Trinity`的实践中被广泛使用，但其设计理念、适用场景和实现细节并未在任何官方架构文档中被定义。它是一种“隐性”的知识，依赖开发者通过阅读源码自行领悟。
    2.  **代码结构混淆**: 当前，所有“函数”的源码都存放在 `pkg/components/functions/` 目录下。这使得它们在物理结构上与“通用组件节点”（位于`pkg/components/action/`等目录）过于接近，容易让新开发者混淆这两种完全不同的扩展模式。

*   **用例**: 一位新开发者（或AI Agent）在接到一个新功能需求时，查阅`sop/02_node_development_sop.md`后，可能会模仿`log`节点去创建一个完整的“组件节点”，而实际上一个更轻量的“函数”才是该场景的最佳选择。这种决策困难和潜在的实现不一致性，降低了开发效率。

*   **目标**:
    1.  将“函数”模式的设计理念、优缺点和实现约定，通过ADR文档进行正式化、权威化。
    2.  在代码文件系统上，将“函数”的实现与“组件节点”的实现进行物理隔离，使代码结构更清晰。
    3.  为所有开发者提供关于何时以及如何使用“函数”模式的明确指引。

## 3. 设计详解 (Detailed Design)

本提案的核心思路是：**先通过文档定义来固化架构，再通过重构来对齐实现。**

### 3.1 已完成的准备工作 (Completed Work)

**状态: 已完成**

在提出本RFC之前，已完成了对该模式的深入分析，并创建了以下参考文档，它们是本RFC的理论基础：
-   **`reference/08_node_development_patterns.md`**: 清晰地对比了“组件节点”与“函数”模式，以及“平台”与“应用”的开发场景。
-   **`reference/09_core_mechanisms_deep_dive.md`**: 深入剖析了支撑“函数”模式的底层机制，如`DataT`和共享资源管理。

### 3.2 待办事项1：架构文档正式化 (ADR Creation)

**状态: 已完成**

我们已创建一份新的架构决策记录（ADR）来正式定义“函数”模式。

-   **路径**: `designs/adr/0002_function_node_pattern.md`
-   **内容**: 该ADR详细阐述了：
    -   **背景**: 为什么需要一种比“组件节点”更轻量的扩展机制。
    -   **决策**: 正式采纳“通用`functions`调用器节点 + 可注册的轻量级函数”的设计模式。
    -   **后果**: 分析该模式的优缺点（例如，降低开发成本 vs. 配置相对复杂）。
    -   **实现约定**: 明确“函数”需要实现的接口、注册方式以及与`NodeCtx`和`RuleMsg`的交互模式。

### 3.3 待办事项2：代码结构重构 (Code Refactoring)

**状态: 待实现**

我们将对现有代码库进行一次结构性重构，以实现“函数”与“组件节点”的物理隔离。

1.  **创建新目录**: 在`pkg/`下创建一个新的顶级目录 `functions/`。
    -   `Architect/matrix/pkg/functions/`
2.  **迁移代码**: 将现有 `Architect/matrix/pkg/components/functions/` 目录下的**所有内容**（例如 `redis_command_func.go`）移动到新的 `Architect/matrix/pkg/functions/` 目录下。
3.  **更新引用**: 调整所有受影响的`import`路径，确保代码能够正常编译和运行。

## 4. 缺点与风险 (Drawbacks & Risks)

*   **代码迁移风险**: 移动文件并更新导入路径是一个全局性的变更。虽然风险较低，但需要确保所有相关项目（包括`trinity`）都同步更新了对这些函数的引用。

## 5. 备选方案 (Alternatives)

*   **仅创建文档，不重构代码**: 我们可以只创建ADR和参考文档，而不移动代码。但这治标不治本，代码结构的混淆问题依然存在，新开发者仍可能被误导。
*   **保持现状**: 完全不处理。这将导致“隐性知识”问题持续存在，影响长期代码的可维护性和开发效率。

## 6. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：这个变更会影响现有的规则链DSL（JSON文件）吗？**
> **答：** **完全不会**。本次变更只涉及源码的物理位置和架构文档的补充。函数的注册ID（如`"redisCommand"`）和`functions`节点的调用方式都保持不变，因此所有现有的规则链JSON文件都无需任何修改。

> **问：执行这个RFC后，我应该如何开发一个新的“函数”？**
> **答：t答：** 你应该在新的 `pkg/functions/` 目录下，根据你的功能创建一个新的Go文件，实现`types.NodeFunc`，并通过`init()`函数注册它。同时，你应该阅读新的ADR文档来理解其设计约束。
<!-- qa_section_end -->
