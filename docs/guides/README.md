---
# === Node Properties: 定义文档节点自身 ===
uuid: "b4c5d6e7-f8a9-0b1c-2d3e-4f5a6b7c8d9e"
type: "Guide"
title: "指南：Matrix 指南文档库"
status: "Stable"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "guide"
  - "documentation"
  - "readme"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本指南库是Matrix项目文档体系的一部分。"
---

# 1. 目录概述 (Overview)

本目录 (`/guides`) 是 `Matrix` 项目的**核心指南文档库**。它旨在为开发者提供关于 `Matrix` 架构、核心概念、开发实践和组件使用的详细说明和最佳实践。

所有文档都应遵循 **[通用语义化文档规范][Ref-SemanticDoc]**。

# 2. 目录结构 (DirectoryStructure)

*   **/guides/README.md**: (本文件) 提供本目录的总体介绍和规范。
*   **/guides/00_matrix_guide.md**: **[必读]** `Matrix` 项目的最高入口，提供了项目的核心定位和架构概览。
*   **/guides/shared_node_guide.md**: 解释了 `Matrix` 中共享节点的概念、工作原理和使用方法。
*   **/guides/components/**: 存放所有独立组件（节点）的详细使用指南。
    *   **/guides/components/README.md**: 定义了组件指南的编写规范。
    *   **/guides/components/<category>_<component_name>_guide.md**: 具体的组件指南文档。

# 3. Guides vs. Reference: 定位差异 (GuidesVsReference)

为了确保文档的正确归类，必须理解 `guides` 和 `reference` 目录的核心定位差异：

| 维度 | `/guides` (本目录) | `/reference` |
| :--- | :--- | :--- |
| **核心目标** | **教授如何做 (How-To)** | **解释是什么 (What-Is)** |
| **内容形式** | 任务导向、分步教程、最佳实践、端到端示例。 | 理论阐述、设计哲学、代码分析、API参考、架构决策的深入探讨。 |
| **读者意图** | “我想要**完成一个特定的任务**，请告诉我具体步骤。” | “我想要**理解一个概念的底层原理**，请给我详细的解释。” |
| **示例** | `如何使用共享节点`、`HTTP客户端节点使用指南` | `共享资源管理机制分析`、`RuleMsg消息设计哲学` |

**简而言之, `guides` 是操作手册, `reference` 是技术白皮书。**

# 4. 文档命名规范 (NamingConvention)

本目录下的所有文档命名应遵循以下规范，以保持一致性和可读性：

*   **格式**: `NN[-M]_<topic_description>.md`
*   **`NN`**: 两位数字前缀，用于**顶级主题的逻辑分组和排序**。
    *   `00` 固定为本目录的入口文档（如 `00_matrix_guide.md`）。
    *   数字越大，代表越深入或越具体的主题。
    *   **顶级主题的 `NN` 必须唯一**。
*   **`-M`**: (可选) 一位数字的次级编号，用于表示**同一顶级主题下的不同分支或实现**。
*   **`<topic_description>`**: 使用小写字母和下划线 `_` 描述文档的核心主题。
*   **特例 - 组件指南**:
    *   `/guides/components/` 目录下的组件指南**不使用**数字前缀，而是遵循其 `README.md` 中定义的 `<component_type>_<component_name>_guide.md` 格式，以便于按类型和名称进行字母排序查找。
*   **示例**:
    *   `00_matrix_guide.md` (顶级入口)
    *   `01_shared_node_guide.md` (顶级主题：共享节点)
    *   `02_dsl_specification.md` (顶级主题：DSL规范)
    *   `components/functions_sql_query_guide.md` (组件指南特例)

# 5. 贡献指南 (ContributionGuide)

我们欢迎对本指南库的任何贡献。在撰写或修改文档时，请遵循以下原则：
1.  **代码优先**: 所有指南都必须以最新的代码实现为准。
2.  **结构清晰**: 优先创建独立的、主题明确的指南文档，而不是将所有内容都堆砌在一个庞大的文档中。
3.  **交叉引用**: 积极使用引用式链接，将相关的概念和指南链接起来，构建一个网状的知识结构。

<!-- 链接定义区域 -->
[Ref-SemanticDoc]: ../../../docs/reference/04_semantic_documentation_standard.md
