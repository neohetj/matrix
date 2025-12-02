---
uuid: "c2d3e4f5-a6b7-4c8d-9e0f-1a2b3c4d5e6f"
type: "RFC"
title: "需求：Matrix通用Agent与LLM核心节点"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "rfc"
  - "design"
  - "matrix"
  - "agent"
  - "llm"
relations:
  - type: "realizes"
    target_uuid: "f47ac10b-58cc-4372-a567-0e02b2c3d479"
    description: "本RFC是任务“在Matrix中实现大模型与Agent相关节点”的核心需求定义。"
---

# RFC: Matrix通用Agent与LLM核心节点 (Title)

## 1. 摘要 (Summary)

本RFC提议在Matrix框架中引入一套通用的、与平台无关的AI Agent核心节点，包括LLM调用、工具执行、记忆管理和循环控制。这将使Matrix具备原生的Agent编排能力，同时将特定平台的交互（如macOS UI操作）解耦为可插拔的工具。

## 2. 动机 (Motivation)

*   **当前存在的问题**: Matrix缺乏构建Agent应用所需的标准组件。开发者需要手动处理LLM调用、工具管理和状态循环，导致复用性差且开发效率低。
*   **用例**: 将一个Python实现的桌面Agent (`MacOS_Agent`) 迁移到Matrix平台。我们需要一个通用的Agent框架，能够调用平台相关的工具（如UI观察、鼠标点击）来完成任务。
*   **目标**:
    *   建立一套通用的、平台无关的Agent核心节点。
    *   定义标准的工具（Tool）接口，并实现一个通用的工具执行器节点。
    *   将Agent的“思考”循环与“感知/行动”工具解耦。

## 3. 设计详解 (DetailedDesign)

*   **核心思路**: Agent的核心能力（思考、记忆、控制）应是通用的，而与外部世界的交互（感知、行动）应通过专门的工具（Tools）来实现。我们将创建4个核心Agent节点，并通过一个`ToolExecutorNode`来调用各种平台相关的工具。

    <!--
    finetune_role: "diagram_generation"
    finetune_instruction: "绘制一个Mermaid流程图，展示AgentControllerNode如何通过读写StoreNode来维护状态，并在线性规则链中调用其他节点。"
    -->
    ```mermaid
    flowchart TD
        subgraph "Agent单次迭代规则链 (DAG)"
            A[Input: Task] --> controller("AgentControllerNode<br/>(状态机与循环逻辑)")
            controller -- "读/写循环状态" --> store("StoreNode<br/>(状态持久化)")
            controller -- "Observe" --> observer_tool["Tool: ObserveState"]
            observer_tool -- "State" --> memory_tool["Tool: RetrieveMemory"]
            memory_tool -- "State + Memory" --> llm["LLMNode"]
            llm -- "Action" --> executor["ToolExecutorNode"]
            executor -- "Result" --> controller
            controller --> B[Output: Final Result / Loop]
        end
    ```
    **循环机制说明**: Agent的循环并非通过规则链的环路实现，而是由`AgentControllerNode`在**节点内部**通过状态机逻辑来驱动。每次循环，控制器都会重新触发一次线性的规则链执行，并将上一步的结果作为下一步的输入，从而在更高维度上实现了循环。

*   **核心节点定义**:
    1.  **`AgentControllerNode`**: 驱动Agent的“思考-行动”循环，管理状态机和任务流程。通过`StoreNode`来持久化和恢复循环状态。
    2.  **`LLMNode`**: 封装与大模型的交互，负责生成思考过程和工具调用指令。支持结构化输出。
    3.  **`ToolExecutorNode`**: 接收`LLMNode`的指令，查找并执行在Matrix中注册的工具（函数或共享节点），并返回结果。
    4.  **`MemoryNode`**: 为Agent提供上下文记忆（如对话历史）的读写能力。
    5.  **`StoreNode` (新增)**: 提供一个通用的键值存储接口，用于持久化节点状态。其后端可以是内存、Redis或数据库，通过配置指定。

*   **工具定义 (示例)**:
    *   工具将作为标准的Matrix函数或共享节点实现。例如，可以创建一个`MacUIObserver`函数，它返回UI树，并被`ToolExecutorNode`调用。

## 4. 缺点与风险 (DrawbacksAndRisks)

*   **工具接口设计**: 需要设计一个足够灵活且标准的工具接口，以适应未来各种工具的接入。
*   **LLM的工具调用能力**: 方案的成功依赖于LLM能够稳定、准确地生成工具调用指令。

## 5. 备选方案 (Alternatives)

*   **为每个平台开发一套完整的Agent节点**: 例如`MacAgentControllerNode`, `WebAgentControllerNode`。
    *   **放弃原因**: 导致大量逻辑重复，违背了通用性和复用性的设计目标。

## 6. 未解决的问题 (UnresolvedQuestions)

*   Matrix中工具（Tool）的具体注册、发现和权限管理机制需要进一步设计。
*   如何对Agent的执行过程进行有效的追踪（Tracing）和调试？
*   `StoreNode` 的后端实现（如Redis, DB）需要作为共享节点被开发。

## 7. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：`StateObserverNode`去哪了？**
> **答：** 它被降级并重新设计为一个平台相关的**工具**（例如`MacUIObserverTool`）。Agent的核心循环不再直接依赖它，而是通过通用的`ToolExecutorNode`来按需调用它，从而实现了核心逻辑与平台实现的解耦。

> **问：这个设计如何支持Web Agent？**
> **答：** 非常简单。我们只需额外实现一套Web环境的工具（如`WebPageReaderTool`, `DOMClickTool`），并在规则链中让`ToolExecutorNode`加载这些Web工具即可。Agent的核心控制和思考逻辑完全不需要改变。
<!-- qa_section_end -->
