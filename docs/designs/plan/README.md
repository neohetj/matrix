---
# === Node Properties: 定义文档节点自身 ===
uuid: "fde86d18-74ce-4350-9137-553929ee1261"
type: "README"
title: "README: Matrix Plan 文档库"
status: "Stable"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "plan"
  - "design"
  - "readme"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本Plan库是Matrix项目设计文档体系的一部分。"
---

# 1. 目录概述 (Overview)

本目录 (`/plan`) 是 `Matrix` 项目的**实现计划 (Implementation Plan)** 文档库。

Plan文档是设计阶段的最终产出，它将RFC（需求）和ADR（决策）转化为具体的、可供工程师执行的开发蓝图。

由于Matrix专注于底层基础能力，一份好的Plan文档应该包含足够的技术细节，使得工程师可以基于它直接开始编码，其核心内容包括：
-   **公共接口定义**: 详细的Go接口、函数签名和参数说明。
-   **核心结构与接口定义**: 新增或修改的核心结构体（如`RuleMsg`, `DataT`的扩展）或框架层级的Go接口。
-   **组件交互流程**: 核心组件或节点之间交互的时序图或活动图。
-   **任务拆解**: 可供领取的、具体的模块或函数开发任务列表。

所有新的Plan都应使用 **[Matrix Plan模板][Tpl-PlanMatrix]** 进行编写。

# 2. 文档命名规范 (NamingConvention)

本目录下的所有文档命名应遵循 **[Architect通用语义化文档规范][Ref-SemanticDoc]** 中定义的文件命名规范。

*   **格式**: `NNNN-M_<description>_plan.md`
*   **`NNNN`**: 对应其父级 RFC 的4位**根ID**。
*   **`M`**: 针对该RFC的第 M 个实现计划，从1开始。
*   **`<description>`**: 对该计划内容的简短、小写、连字符分隔的描述 (kebab-case)。
*   **`_plan.md`**: 固定的后缀。

---
<!-- 链接定义区域 -->
[Ref-SemanticDoc]: ../../../../docs/reference/04_semantic_documentation_standard.md
[Tpl-PlanMatrix]: ../../templates/plan_template.md
