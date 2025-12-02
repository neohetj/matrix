---
uuid: "978c1b44-65eb-43ef-bcf8-793c1793a0b3"
type: "RFC"
title: "需求：运维基础组件与DSL扩展"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "rfc"
  - "design"
  - "ops"
  - "dsl"
  - "node"

relations:
  - type: "is_based_on"
    target_uuid: "task-0004-uuid" # -> 指向我们创建的任务定义
    description: "本RFC是为实现任务0004中定义的目标而创建的。"
---

# RFC: 运维基础组件与DSL扩展 (OpsFoundationComponentsAndDslExtensions)

## 1. 摘要 (Summary)

本RFC提议为Matrix规则引擎框架引入一套可复用的运维（Ops）基础组件，并对现有的规则链DSL进行向上兼容的扩展。目标是建立一个灵活、可维护的运维自动化模型，该模型能够将静态的拓扑定义与动态的、可执行的工作流分离开来，同时支持多视角的建模和分析。

## 2. 动机 (Motivation)

*   **当前存在的问题**: Matrix框架缺乏专门用于表示运维实体（如机器、服务）的节点类型。这使得在规则链中进行运维场景（如健康检查、自动化部署、拓扑可视化）的建模变得繁琐且不直观。此外，现有DSL缺乏对逻辑拓扑和执行流程的明确区分，难以实现配置的复用和“数据与行为分离”的设计模式。
*   **用例**:
    1.  **拓扑即数据**: 用户可以定义一个不可执行的DSL文件来描述一个应用的所有组件及其部署关系，作为单一事实来源。
    2.  **流程复用拓扑**: 用户可以创建多个不同的、可执行的工作流（如健康检查、日志收集），这些工作流通过 `imports` 关键字复用同一个基础拓扑定义，而无需重复声明节点。
    3.  **多视角可视化**: 工具可以解析同一个DSL文件，根据不同的关系类型（`connections` vs. `relations`）生成不同的视图，如执行流程图或物理部署图。
*   **目标**: 增强Matrix框架在AIOps场景下的建模能力，提高运维自动化工作流的可维护性、可复用性和可观测性。

## 3. 设计详解 (Detailed Design)

### 3.1. 运维基础节点 (Ops Foundation Nodes)

我们将创建一系列新的、实现了 `types.Node` 接口的组件，存放于 `Architect/matrix/pkg/components/ops/` 目录下。

*   **节点类型**: `ops/machine`, `ops/service`, `ops/application`, `ops/database` 等。
*   **实现模式**: 遵循 `endpoint_link_node.go` 的标准模式（原型、注册、配置结构体、`New/Init/OnMsg`方法）。
*   **动静分离**:
    *   **Configuration**: 只包含节点的静态配置，如 `Address`, `CredentialSecretName`。
    *   **Internal State**: 节点内部维护一个 `State` 结构体，用于缓存动态探测到的信息（如OS, CPU）。
    *   **消息驱动探测**: 节点 `OnMsg` 方法将响应特定消息（如 `{"action": "PROBE"}`）来触发动态探测，并将结果更新到内部State。

### 3.2. DSL扩展

我们将对规则链DSL的JSON结构进行以下扩展：

*   **`metadata` 对象**: 在DSL根级别增加 `metadata` 对象，用于存放整个规则链的元数据。
    *   `executable` (bool, a must): 明确标识该DSL文件是否可被 `Matrix` 运行时执行。`false` 表示其仅为定义或拓扑。
    *   `viewType` (string, optional): 为可视化工具提供渲染提示，如 `static-topology`, `execution-flow`, `hybrid`。
    *   `imports` (array of strings, optional): 见下文。

*   **`relations` 数组**: 在DSL根级别增加 `relations` 数组，与现有的 `connections` 并行。
    *   `connections`: 维持原义，表示**可执行的连接**，必须构成有向无环图（DAG）。`Matrix` 运行时只处理此数组。
    *   `relations`: 表示**纯逻辑的关联**，用于数据建模和可视化，**可以有环**。可视化工具应主要渲染此数组来展示拓扑。

### 3.3. `imports` 机制

这是实现“数据与行为分离”的核心。

*   **功能**: 允许一个DSL文件通过 `metadata.imports` 数组，导入一个或多个其他DSL文件。
*   **实现**: 需要修改 `Matrix` 的DSL解析器 (`/pkg/matrix/parser`)。
    1.  当解析器遇到 `imports` 字段时，它会递归地加载并解析所有被导入的文件。
    2.  解析器将所有导入文件中的 `nodes` 和 `relations` 与当前文件中的 `nodes` 和 `connections` **合并**，形成一个完整的、内存中的 `RuleChainDef` 对象。
    3.  这个合并后的对象将被传递给 `Matrix` 运行时进行实例化。
*   **效果**: 允许用户创建一个不可执行的 `base_topology.json` 作为模型，然后创建多个可执行的 `workflow_*.json` 作为控制器，每个控制器都导入并复用同一个模型。

## 4. 缺点与风险 (Drawbacks & Risks)

*   **解析器复杂性增加**: 修改DSL解析器以支持 `imports` 和合并逻辑，是本次开发的核心风险点，需要详尽的测试。
*   **循环导入**: 解析器需要有能力检测并阻止 `imports` 的循环依赖。
*   **节点ID冲突**: 在合并DSL时，需要定义清晰的节点ID冲突解决策略（例如，默认覆盖，或提供选项）。

## 5. 备选方案 (Alternatives)

*   **仅使用 `SharedNodePool`**: 我们曾讨论过仅使用现有的 `SharedNodePool` 机制来复用节点。此方案被否决，因为它解决的是“运行时实例共享”而非“定义时配置复用”的问题，无法实现拓扑定义与工作流定义的分离。

## 6. 未解决的问题 (UnresolvedQuestions)

*   `relations` 数组中 `label` 字段的具体格式和解析方式，需要进一步的设计。
*   合并 `nodes` 时，对于同ID节点的配置（`configuration` 字段）是应该合并还是覆盖，需要确定一个明确的规则。

## 7. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：这个变更是向后兼容的吗？**
> **答：** 是的。所有新增的DSL字段（`metadata`, `relations`）都是可选的。不包含这些字段的现有DSL文件，其行为将和以前完全一样。

> **问：`Matrix` 运行时会如何处理 `relations`？**
> **答：** `Matrix` 运行时将**完全忽略** `relations` 数组。它的存在纯粹是为了外部工具（如拓扑图生成器、静态分析器）使用。这确保了对核心执行引擎的零影响。
<!-- qa_section_end -->
