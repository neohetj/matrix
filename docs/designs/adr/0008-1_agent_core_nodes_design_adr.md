---
uuid: "e1f2a3b4-c5d6-e7f8-a9b0-c1d2e3f4a5b6"
type: "ADR"
title: "决策：采用通用节点与可插拔工具的Agent架构"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "adr"
  - "architecture"
  - "matrix"
  - "agent"
relations:
  - type: "realizes"
    target_uuid: "c2d3e4f5-a6b7-4c8d-9e0f-1a2b3c4d5e6f"
    description: "本ADR为RFC-0008中提出的Agent节点需求提供了具体的架构设计决策。"
---

# ADR: 采用通用节点与可插拔工具的Agent架构 (ADR-AgentCoreNodes)

## 1. 决策背景 (Context)

在 `RFC-0008` 中，我们提出了在Matrix中构建AI Agent能力的需求。核心的技术挑战是如何设计一套既能满足复杂Agent工作流（如`MacOS_Agent`的“观察-思考-行动”循环），又能保持Matrix框架通用性和可扩展性的节点架构。

备选的解决方案主要有两种：
1.  **方案A (整体式节点)**: 将Agent的所有逻辑（UI观察、LLM调用、动作执行）封装在一个或少数几个与特定平台（如macOS）紧耦合的节点中。
2.  **方案B (通用节点 + 可插拔工具)**: 将Agent的核心思考循环（控制、记忆、LLM调用）抽象为一套通用的、与平台无关的核心节点，而将所有与外部世界交互的感知（Observe）和行动（Act）能力，实现为可被动态调用的、独立的“工具”。

## 2. 核心设计决策 (Decision)

**我们决定，采用方案B：通用节点与可插拔工具的架构。**

具体设计如下：
1.  **实现4个通用的Agent核心节点**: `AgentControllerNode`, `LLMNode`, `ToolExecutorNode`, `MemoryNode`。
2.  **实现1个通用的状态持久化节点**: `StoreNode`，供`AgentControllerNode`等需要跨迭代保持状态的节点使用。
3.  **定义一套标准的工具（Tool）规范**: 所有与具体平台或业务相关的能力（如`获取macOS UI`、`点击鼠标`、`调用Web API`），都必须实现为符合此规范的Matrix函数或共享节点。
4.  **Agent的循环由`AgentControllerNode`在节点内部逻辑驱动**，每次迭代调用一次线性的、由核心节点和工具组成的规则链，而非通过规则链的环路实现。

## 3. 决策理由 (Rationale)

| 对比维度 | 方案A (整体式节点) | **方案B (通用节点 + 工具)** |
| :--- | :--- | :--- |
| **通用性** | 差。每个新平台（Web, Windows）都需要重写一套Agent节点。 | **优**。核心节点可完全复用，只需为新平台开发新的工具集。 |
| **可扩展性** | 差。为Agent增加新能力（如一个新的API调用）可能需要修改核心节点。 | **优**。只需实现一个新的工具函数并注册，即可被`ToolExecutorNode`动态调用。 |
| **复用性** | 差。`LLMNode`等通用能力被耦合在Agent逻辑中，无法被其他非Agent场景单独使用。 | **优**。`LLMNode`, `StoreNode`等都是独立的、高内聚的组件，可在任何规则链中使用。 |
| **复杂度** | 表面上简单，但长期维护成本高，容易形成巨型单体节点。 | **初期设计更复杂**，但长期来看架构更清晰，符合Matrix的设计哲学。 |

我们选择方案B，是因为它在通用性、可扩展性和复用性上具有压倒性优势，完全符合Matrix作为“核心能力层”的定位。虽然初期需要投入更多精力来设计工具规范，但这将为未来的功能扩展和跨平台支持打下坚实的基础。

## 4. 决策结果 (Consequences)

*   **优点 (Pros)**:
    *   Agent核心逻辑与平台实现完全解耦。
    *   极大地提高了Agent能力的可扩展性和跨平台迁移能力。
    *   产出的`LLMNode`, `StoreNode`等是高度可复用的基础组件。
*   **缺点 (Cons)**:
    *   需要额外定义一套清晰的`Tool`接口规范和注册机制。
    *   `AgentControllerNode`的内部状态机逻辑相对复杂。
*   **风险 (Risks)**:
    *   如果`Tool`规范设计不当，可能会限制未来工具的能力。

## 5. 未来工作 (Future Work)

*   需要为`Tool`的定义、注册和发现在`Plan`文档中进行详细设计。
*   需要为`StoreNode`设计其后端存储（如Redis）的实现方案。

<!-- qa_section_start -->
> **问：这个决策意味着我们不能有`MacAgent`这样的东西了吗？**
> **答：** 不是。我们仍然可以有一个`MacAgent`，但它不再是一个单一的节点，而是一个**规则链应用**。这个应用会使用通用的`AgentControllerNode`等核心节点，并为其配置一套macOS专属的工具集（如`MacUIObserverTool`, `MacClickTool`等）。
<!-- qa_section_end -->
