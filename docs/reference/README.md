---
# === Node Properties: 定义文档节点自身 ===
uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
type: "Guide"
title: "指南：Matrix 参考文档库"
status: "Stable"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "reference"
  - "documentation"
  - "readme"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本参考库是Matrix项目文档体系的一部分。"
---

# 1. 目录概述 (Overview)

本目录 (`/reference`) 是 `Matrix` 项目的**核心参考文档库**。它旨在为开发者提供关于 `Matrix` 架构、核心概念、设计哲学和底层实现的深入解释和技术白皮书。

所有文档都应遵循 **[通用语义化文档规范][Ref-SemanticDoc]**。

### 文档索引 (Document Index)

*   **架构与设计哲学**:
    *   [`03_architecture_overview.md`](./03_architecture_overview.md): Matrix 整体架构概览。
    *   [`06_message_design_philosophy.md`](./06_message_design_philosophy.md): `RuleMsg` 消息体的核心设计理念。
    *   [`33_component_design_principles.md`](./33_component_design_principles.md): 组件设计的核心原则。
*   **核心概念**:
    *   [`09_core_objects.md`](./09_core_objects.md): `CoreObj` 和 `DataT` 的概念详解。
    *   [`12_node_specification.md`](./12_node_specification.md): 节点的标准规范和生命周期。
    *   [`15_shared_resource_management.md`](./15_shared_resource_management.md): 共享节点池的工作机制。
*   **开发与实现**:
    *   [`08_node_development_patterns.md`](./08_node_development_patterns.md): 节点开发的常见模式与最佳实践。
    *   [`10_http_endpoint_deep_dive.md`](./10_http_endpoint_deep_dive.md): HTTP Endpoint 节点的深度解析。
    *   [`11_function_registration_spec.md`](./11_function_registration_spec.md): 函数节点的注册规范。
    *   [`18_dsl_specification.md`](./18_dsl_specification.md): 规则链DSL的语法规范。
    *   [`21_component_catalog.md`](./21_component_catalog.md): 官方核心组件的功能清单。
    *   [`36_testing_strategy.md`](./36_testing_strategy.md): 框架的官方测试策略与最佳实践。

# 2. Reference vs. Guides: 定位差异 (ReferenceVsGuides)

为了确保文档的正确归类，必须理解 `reference` 和 `guides` 目录的核心定位差异：

| 维度 | `/reference` (本目录) | `/guides` |
| :--- | :--- | :--- |
| **核心目标** | **解释是什么 (What-Is)** | **教授如何做 (How-To)** |
| **内容形式** | 理论阐述、设计哲学、代码分析、API参考、架构决策的深入探讨。 | 任务导向、分步教程、最佳实践、端到端示例。 |
| **读者意图** | “我想要**理解一个概念的底层原理**，请给我详细的解释。” | “我想要**完成一个特定的任务**，请告诉我具体步骤。” |
| **示例** | `共享资源管理机制分析`、`RuleMsg消息设计哲学` | `如何使用共享节点`、`HTTP客户端节点使用指南` |

**简而言之, `reference` 是技术白皮书, `guides` 是操作手册。**

# 3. 文档命名规范 (NamingConvention)

本目录下的所有文档命名应遵循以下规范，以保持一致性和可读性：

*   **格式**: `NN_<topic_description>.md`
*   **`NN`**: 两位数字前缀，用于**定义文档的逻辑阅读顺序**。
    *   编号遵循从**宏观概览**到**核心概念**，再到**内部机制深潜**的递进关系。
    *   **编号应以3的倍数递增**（`03`, `06`, `09`...），以便在未来可以方便地在现有文档之间插入新的关联文档。
    *   **`NN` 编号必须唯一**，不应重复。
*   **`<topic_description>`**: 使用小写字母和下划线 `_` 描述文档的核心主题。
*   **同一主题下的细节说明**:
    *   与 `guides` 目录不同，本目录不使用次级编号 (`-M`)。
    *   对于同一宏观主题下的不同细节，应在**同一个文档内**，使用**二级或三级标题**进行组织和阐述，以保证概念的完整性和上下文的连续性。
*   **示例**:
    *   `03_architecture_overview.md` (最高层级的架构概览)
    *   `06_node_lifecycle.md` (对“节点”这一核心概念的生命周期进行阐述)
    *   `09_core_mechanisms_deep_dive.md` (对多个核心概念的内部工作机制进行深入剖析)

# 4. 贡献指南 (ContributionGuide)

我们欢迎对本参考库的任何贡献。在撰写或修改文档时，请遵循以下原则：
1.  **深度优先**: 内容应深入探讨“为什么”和“是什么”，而不仅仅是“怎么做”。
2.  **代码佐证**: 关键的设计决策和分析应附带相关的核心代码片段作为佐证。
3.  **保持更新**: 当底层设计发生重大变更时，应及时更新相关的参考文档。

<!-- 链接定义区域 -->
[Ref-SemanticDoc]: ../../../docs/reference/04_semantic_documentation_standard.md
