---
# === Node Properties: 定义文档节点自身 ===
uuid: "d5e4f3c2-b1a0-4i9h-8j8k-7l6m5n4o3p2q"
type: "README"
title: "README: Matrix ADR 文档库"
status: "Stable"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "adr"
  - "design"
  - "readme"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本ADR库是Matrix项目设计文档体系的一部分。"
---

# 1. 目录概述 (Overview)

本目录 (`/adr`) 是 `Matrix` 项目的**架构决策记录 (Architecture Decision Record)** 文档库。

与RFC（提案）不同，ADR是**对已做出决策的记录**。它用于记录在项目演进过程中的关键技术选型、架构设计权衡及其背后的原因。它清晰地阐述了“为什么”选择某个方案，以及该决策带来的影响。

每个ADR都应该简明扼要，并清晰地记录以下内容：
-   **背景 (Context)**: 当时面临的问题或需要做的决策是什么。
-   **决策 (Decision)**: 最终做出了什么决定。
-   **后果 (Consequences)**: 这个决策带来了哪些积极和消极的影响。

所有新的ADR都应使用 **[Matrix ADR模板][Tpl-ADRMatrix]** 进行编写。

# 2. 文档命名规范 (NamingConvention)

本目录下的所有文档命名应遵循 **[Architect通用语义化文档规范][Ref-SemanticDoc]** 中定义的文件命名规范。

*   **格式**: `NNNN-M_<description>_adr.md`
*   **`NNNN`**: 对应其父级 RFC 的4位**根ID**。
*   **`M`**: 针对该RFC的第 M 个架构决策，从1开始。
*   **`<description>`**: 对该决策内容的简短、小写、连字符分隔的描述 (kebab-case)。
*   **`_adr.md`**: 固定的后缀。

---
<!-- 链接定义区域 -->
[Ref-SemanticDoc]: ../../../../docs/reference/04_semantic_documentation_standard.md
[Tpl-ADRMatrix]: ../../templates/adr_template.md
