---
uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
type: "ArchitectureOverview"
title: "Matrix 项目总览：README First"
status: "Stable"
owner: "@cline"
version: "2.0.0"
tags:
  - "matrix"
  - "readme"
  - "onboarding"
  - "architecture"

relations:

---

# 1. 理解Matrix的定位 (Understanding The Role)

**Matrix是Architect生态系统中的“核心能力层”。**

它的核心职责是实现构成业务的、可复用的原子能力（节点/组件），是所有上层业务逻辑的“弹药库”。它旨在提供一个稳定、高效、可扩展的底层框架，让业务开发可以聚焦于逻辑编排，而非底层实现。

# 2. 如何开始在Matrix中开发 (How to Start Development)

所有在`Matrix`项目中的开发工作，都由统一的 **[SOP-AI-01: AI辅助开发元工作流][SOP-AIDevWorkflow-77a10f]** 进行驱动和纳管。

根据您的具体开发场景，`SOP-AI-01`的路由表会将您引导至以下核心的“微观SOPs”：

-   **场景一：对Matrix框架进行核心变更**
    -   **遵循**: **[SOP-AI-07: AI辅助的Matrix框架开发][SOP-MatrixFrameworkDev-6bc2f0]**
    -   **适用情况**: 修改核心接口、调整核心机制、进行大范围重构等。

-   **场景二：开发一个新的、可复用的节点/组件**
    -   **遵循**: **[SOP-AI-08: AI辅助的Matrix节点开发][SOP-MatrixNodeDev-03f6dfd]**
    -   **适用情况**: 创建一个新的、独立的业务逻辑单元。
    -   **核心指南**: **[指南：Matrix函数节点开发][Guide-FuncDev-40f2bf5]** 提供了关于如何实现函数节点的详细技术规范和最佳实践。

<!-- qa_section_start -->
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-FuncDev-40f2bf5]: ./components/03_function_development_guide.md
[SOP-AIDevWorkflow-77a10f]: ../../../Architect/docs/sop/ai_01_assisted_development_workflow_sop.md
[SOP-MatrixFrameworkDev-6bc2f0]: ../../../docs/sop/ai_07_framework_development_sop.md
[SOP-MatrixNodeDev-03f6dfd]: ../../../docs/sop/ai_08_node_development_sop.md
