---
uuid: "GENERATED_UUID" # (必填) 创建新文档时，AI Agent应自动生成一个符合RFC 4122标准的小写UUID；人类开发者可使用`uuidgen | tr '[:upper:]' '[:lower:]'`等工具生成。
type: "RFC"
# [重要] 请确保本文档的文件名遵循 `NNNN_<description>_rfc.md` 格式，并与下方的标题保持一致。
# [重要] `<description>` 应为小写主题描述，允许使用 `_` 或 `-` 分隔单词。
title: "需求：[一个简洁且描述性的标题]"
status: "Draft" # -> InReview -> Accepted/Rejected/Superseded
owner: "neohetj"
version: "1.0.0"
tags:
  - "rfc"
  - "design"
  - "[feature-tag]"
relations:
  - type: "relates_to"
    target_uuid: "[UUID of related doc]"
    description: "Describes the relationship."
# 如果暂时没有关联文档，允许改为 `relations: []`，但不要保留空的 target_uuid。
# 当 RFC 进入 Accepted / Implementing 阶段后，请至少补一条指向 `docs/guides/` 或 `docs/migration/`
# 当前使用文档的 relation；如果还没有对应 guide，应先创建 guide 再回写 RFC。
---

<!--
维护约定：
1. 本模板生成的第 1-7 节应视为“原始 RFC 正文”。
2. 后续如果需要回写实现状态，请在文末追加“附录：当前实现对齐”或“附录：历史注记”。
3. 不要直接把第 1-7 节改写成“当前实现说明”，否则会丢失原始需求点。
4. 当 RFC 进入 Accepted / Implementing 阶段后，应在文末补“相关现行文档”章节，链接当前 guide / migration / reference。
5. 正文前部必须包含一个明确命名为 `原始需求点总结` 的小节，用分点方式保留最初需求，而不是只保留后来的实现说明。
6. 当 RFC 进入 Accepted / Implementing 阶段后，必须补 `当前实现对齐`（或 `附录：当前实现对齐`）章节，说明已落地范围、偏差、剩余缺口与当前入口文档。
-->

# RFC: [一个简洁且描述性的标题] (Title)

## 1. 摘要 (Summary)

*（用一到两句话高度概括这个RFC的核心提议。读者应该能通过摘要快速了解这个变更的目的。）*

### 原始需求点总结

*（用 3-6 个要点总结最初的需求点。建议覆盖：当前痛点、典型用例、核心目标、关键边界/约束。这里应是对原始需求的提炼，而不是“当前实现状态”的摘要。）*

## 2. 动机 (Motivation)

*（详细阐述“为什么”需要这个变更。这里应该包含：）*
*   **当前存在的问题**: 描述当前框架或流程中遇到的具体问题、限制或痛点。
*   **用例**: 提供一到两个具体的场景或用例，来说明当前的问题是如何影响开发效率或系统能力的。
*   **目标**: 清晰地列出此RFC希望达成的目标。

## 3. 设计详解 (DetailedDesign)

*（这是RFC的核心部分，详细描述“如何”解决问题。）*
*   **核心思路**: 介绍你的解决方案的核心思想和架构。
*   **API变更**: 如果涉及到对公共API、接口或数据结构的修改，请使用代码块清晰地列出变更前和变更后的对比。
    <!-- 
        [Agent/Author Guide] 
        为下面的代码块添加finetune指令，以供AI模型微调。
        finetune_role: code_generation_example
        finetune_instruction: "展示如何修改[某个struct]的API定义"
    -->
    ```go
    // 示例代码
    ```
*   **组件交互**: 如果涉及到多个组件或模块的交互变更，建议使用Mermaid图（序列图或流程图）来可视化地展示新的交互流程。
*   **示例**: 提供一个或多个代码示例，来展示新功能或新API将如何被使用。

## 4. 缺点与风险 (DrawbacksAndRisks)

*（诚实地列出这个设计可能带来的缺点、风险或需要权衡的地方。）*
*   为什么这个设计不是完美的？
*   它可能会给哪些方面带来新的复杂性？
*   在实施过程中可能存在哪些风险？

## 5. 备选方案 (Alternatives)

*（简要描述你曾考虑过的其他解决方案，并解释为什么你最终没有选择它们。这表明你已经对问题进行了深入的思考。）*

## 6. 未解决的问题 (UnresolvedQuestions)

*（列出在这个设计中仍然存在的一些悬而未决的问题，或者需要进一步讨论和设计的点。）*

## 7. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：这个变更会影响现有的规则链吗？是否需要数据迁移？**
> **答：** ...

> **问：这个功能的性能如何？是否做过基准测试？**
> **答：** ...
<!-- qa_section_end -->

## 8. 附录：当前实现对齐 (进入 Accepted / Implementing 后必填)

*（当 RFC 进入 Accepted / Implementing 阶段后，本节必须补充。至少说明：哪些能力已落地、哪些部分仍未落地或与原提案不同、当前应参考哪些 guide / reference / migration 文档。）*

## 9. 相关现行文档 (进入 Accepted / Implementing 后补)

*（当 RFC 进入 Accepted / Implementing 阶段后，请在这里补充当前 guide / migration / reference 链接；如果涉及新节点或新协议但没有 guide，应先创建对应 guide。）*
