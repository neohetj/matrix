---
# === Node Properties: 定义文档节点自身 ===
uuid: "a2b1c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
type: "Guide"
title: "指南：Matrix组件文档编写规范"
status: "Stable"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "documentation"
  - "guide"
  - "sop"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本规范是Matrix项目文档体系的一部分。"
  - type: "references"
    target_uuid: "ad53f899-a8c1-47b2-bd5a-2e666c292bef"
    description: "所有组件指南都必须基于通用语义化文档模板创建。"

---

# 1. 引言 (Introduction)

本目录 (`/components`) 存放了 `Matrix` 核心能力层中所有可复用组件（节点）的详细使用指南。每一份指南都是一个独立的、自包含的文档，旨在为开发者提供关于特定组件的全面信息。

本文档定义了编写这些组件指南的**强制性规范**，以确保所有文档的质量、一致性和可用性。所有贡献者在编写新的组件指南前，都**必须**阅读并遵循此规范。

# 2. 核心原则 (CorePrinciples)

*   **代码即真理 (CodeIsTheTruth)**: 所有文档内容都必须严格基于组件的最新代码实现。当代码与文档不一致时，应优先更新文档。
*   **用户导向 (UserOriented)**: 文档的编写应始终站在组件使用者的角度，清晰地解释“它是什么”、“它能做什么”以及“如何使用它”。
*   **自包含性 (SelfContained)**: 每份指南应包含理解和使用该组件所需的所有信息，避免让读者在多个文档之间频繁跳转。

# 3. 组件指南特定规范 (ComponentGuideSpecifications)

除了遵循全局的 **[语义化文档规范][Ref-SemanticDoc]** 之外，本目录下的所有组件指南还必须遵循以下特定规范。

## 3.1. 文件命名规范 (FileNamingConvention)
<span id="file-naming-convention"></span>

*   **格式**: `<component_type>_<component_name>_guide.md`
*   **规则**:
    *   全部使用小写字母。
    *   使用下划线 `_` 分隔单词。
    *   `<component_type>`: 组件的类型，与代码目录结构保持一致。例如 `functions`, `endpoint`, `action` 等。
    *   `<component_name>`: 组件的具体名称，应与其代码中的ID或核心文件名保持一致。例如 `http_client`, `sql_query`。
*   **示例**:
    *   `functions_http_client_guide.md`
    *   `endpoint_http_guide.md`

## 3.2. 文档结构规范 (StructureSpecification)
<span id="structure-specification"></span>

每份组件指南都**必须**使用 **[通用语义化文档模板][Tpl-SemanticDoc]** 作为起点，并**必须**包含以下核心章节，每个章节的标题都**必须**附带驼峰式英文锚点：

1.  **功能概述 (Overview)**
    *   清晰地描述组件的核心功能和应用场景。
    *   说明它在规则链中扮演的角色。

2.  **如何配置 (Configuration)**
    *   使用Markdown表格，详细列出该组件在DSL的 `configuration` 块中支持的所有参数。
    *   表格应包含：**配置键 (ID)**, **名称**, **描述**, **类型**, **是否必须**, **默认值**。
    *   提供一个完整的、可直接复制使用的JSON配置示例。

3.  **数据契约 (DataContract)**
    *   明确定义组件的输入和输出。
    *   **对于功能节点 (Function Node)**:
        *   详细描述其通过 `DataT` 容器交换的输入/输出对象。
        *   应包含：**参数名**, **对象SID**, **是否必须**, **描述**, 以及对象的**Go结构体定义**。
    *   **对于其他节点**:
        *   描述其如何读取和修改 `RuleMsg` 的 `Data` 和 `Metadata` 字段。

4.  **错误处理 (ErrorHandling)**
    *   使用Markdown表格，列出该组件在执行过程中可能抛出的所有特定错误。
    *   表格应包含：**错误对象名**, **错误码**, **描述**。
    *   说明这些错误应如何通过 `TellFailure` 等关系进行捕获。

5.  **问答环节 (FAQ)**
    *   （可选，但强烈推荐）
    *   在 `<!-- qa_section_start -->` 和 `<!-- qa_section_end -->` 块中，预设并回答一些开发者可能遇到的常见问题或设计决策的思考。

## 3.3. 元数据 (Metadata)
<span id="metadata-specification"></span>

*   `uuid`: **必须**为每个新文档生成一个唯一的、符合RFC 4122标准的小写UUID。
*   `type`: **必须**为 `ComponentGuide`。
*   `title`: **必须**遵循格式 `组件指南：<组件可读名称> (<组件ID>)`。
*   `tags`: **必须**至少包含 `matrix`, `component`, 组件类型 (e.g., `function`) 和功能关键词 (e.g., `http`)。
*   `relations`: **必须**包含一个 `is_part_of` 关系，指向 **[Matrix 项目总览][Guide-MatrixOverview]**。

<!-- 链接定义区域 -->
[Ref-SemanticDoc]: ../../../../docs/reference/04_semantic_documentation_standard.md
[Tpl-SemanticDoc]: ../../../../docs/templates/semantic_doc_template.md
[Guide-MatrixOverview]: ../00_matrix_guide.md