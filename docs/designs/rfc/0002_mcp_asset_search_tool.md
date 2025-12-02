---
uuid: "45780fc0-420b-4e79-a366-5c740f9e3bcb"
type: "RFC"
title: "需求：开发者工具增强之MCP资产搜索工具"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "RFC"
  - "MCP"
  - "开发者工具"
  - "资产搜索"
relations:
  - type: "relates_to"
    target_uuid: "423a1d0c-0f81-45bd-bcc1-7ae64d519f65" # -> 指向《规则链设计原则》
    description: "本RFC旨在为《规则链设计原则》中“代码优先”原则提供关键工具支持。"

---

# RFC: 开发者工具增强之MCP资产搜索工具 (McpAssetSearchTool)
> **上下文**: 本提案是继已批准并实现的 [数据契约规范](./0001_data-contract-specification_rfc.md) 之后，对开发者工具生态的进一步增强。

## 1. 概要 (Summary)

本RFC提议设计并实现一个基于MCP（Model Context Protocol）的资产搜索工具。该工具旨在解决当前开发流程中的一个核心痛点：开发者难以高效、准确地发现和复用系统中已存在的业务节点和规则链。通过将资产搜索能力封装为标准的MCP工具，我们可以将其无缝集成到开发者的工作流中，尤其是与AI Agent的交互过程中，从而严格执行“代码优先，复用为王”的设计原则。

## 2. 动机 (Motivation)

随着Matrix系统业务逻辑的不断增长，节点和规则链的数量急剧增加。目前，开发者依赖于手动的、基于约定的文件目录浏览或文本搜索来寻找可复用的资产，这种方式存在以下弊端：

*   **效率低下**: 手动搜索耗时耗力，且容易遗漏。
*   **信息不全**: 简单的文件名或文本搜索无法提供节点的详细元数据，如功能描述、输入输出、版本等。
*   **原则难以落地**: “代码优先”原则因缺少有效的工具支持而难以被严格遵守，导致不必要的重复开发。
*   **AI Agent无法利用**: AI Agent在进行辅助设计时，无法程序化地访问和理解现有资产，其潜力受到极大限制。

一个专用的MCP资产搜索工具将从根本上解决这些问题。

## 3. 设计思路 (DesignProposal)

### 3.1. MCP工具定义 (MCPToolDefinition)

我们将创建一个名为 `asset_search` 的MCP工具，其输入和输出模式如下：

**工具名称**: `search_matrix_assets`

**输入参数 (Arguments)**:
```json
{
  "query": "string", // (必填) 语义化搜索查询，如“用户登录验证”、“订单状态更新”
  "asset_type": "string", // (可选) 资产类型, 受控词汇表: ["node", "rulechain", "all"], 默认为 "all"
  "tags": ["string"], // (可选) 基于标签进行过滤
  "limit": "integer" // (可选) 返回结果数量，默认为 5
}
```

**输出结果 (Output)**:
```json
{
  "assets": [
    {
      "uuid": "string",
      "name": "string",
      "type": "string", // "node" or "rulechain"
      "description": "string", // 节点的详细功能描述
      "path": "string", // 资产定义文件的路径
      "version": "string",
      "relevance_score": "float" // 查询相关度得分
    }
    // ... more assets
  ]
}
```

### 3.2. 实现方案 (ImplementationPlan)

1.  **资产索引**:
    *   开发一个后台服务或CLI工具，定期扫描 `trinity/` 和 `matrix/` 目录下的所有规则链和节点定义文件（如Go代码文件中的注释、JSON配置文件等）。
    *   提取每个资产的元数据（名称、描述、配置参数、代码注释等），并将其构建成一个可被快速搜索的索引（如使用Bleve或Elasticsearch）。
    *   索引应支持语义化搜索，即将用户的自然语言查询转换为向量，并与索引中的资产描述向量进行相似度匹配。

2.  **MCP Server实现**:
    *   创建一个MCP Server，它将提供 `search_matrix_assets` 工具。
    *   该Server接收到请求后，查询后台的资产索引，并按照定义的输出格式返回结果。

## 4. 预期收益 (ExpectedBenefits)

*   **提升开发效率**: 开发者和AI Agent可以秒级发现相关资产，极大缩短调研时间。
*   **提高代码复用率**: 显著降低重复造轮子的概率，提升系统整体质量。
*   **强化设计原则**: 为“代码优先”原则提供了强有力的技术保障。
*   **赋能AI Agent**: 使AI Agent能够基于对现有系统的深刻理解来进行更智能的设计和编码，是实现高级“AI辅助开发”的基石。

<!-- qa_section_start -->
> **问：这个工具的维护成本如何？**
> **答：** 初期需要投入开发资源构建索引器和MCP Server。但一旦建成，其维护成本相对较低。索引过程可以完全自动化，通过CI/CD流水线在代码提交后触发。其带来的长期收益（开发效率提升、代码质量提高）将远超其微维护成本。
<!-- qa_section_end -->
