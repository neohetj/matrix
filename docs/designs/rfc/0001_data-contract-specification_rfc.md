---
uuid: "589dab79-3b5c-4231-ada6-b2617d375abb"
type: "RFC"
title: "需求：为节点声明分层的数据访问契约"
status: "Superseded"
owner: "@cline"
version: "1.0.0"
tags:
  - "rfc"
  - "design"
  - "data-contract"
  - "static-analysis"
relations:
  - type: "is_formalized_by"
    target_uuid: "d8a3bfe2-0a7e-4b3a-9c1d-8e7f6a5b4c3d" # -> 07_data_contract_specification.md
    description: "The design proposed in this RFC has been implemented and is formally specified in the reference document."
---

# RFC: 为节点声明分层的数据访问契约 (HierarchicalDataContract)

## 1. 摘要 (Summary)

本RFC提议为Matrix框架引入一个分层的数据访问契约体系：为`function`节点扩展其独有的`FuncObject`以声明完整的`DataT/Data/Metadata`交互；为非`function`节点扩展其`NodeDefinition`以声明其对`Data/Metadata`的简单读写，从而为所有节点提供清晰、精准的静态数据契约。

## 2. 动机 (Motivation)

*   **当前存在的问题**: 节点的数据访问模式是隐式的。`function`节点虽有`DataT`的`Inputs/Outputs`定义，但缺乏对`Data/Metadata`的声明；非`function`节点则完全没有数据契约。这导致静态分析无法进行，UI工具也无法提供智能辅助。
*   **用例**: 静态分析工具可以校验`forEach`产生的`is_last_item`是否被下游节点正确声明读取。UI可以在用户配置`log`函数节点时，根据其契约提示它可以访问哪些`Metadata`和`DataT`对象。
*   **目标**: 为所有节点提供清晰、准确、与其能力匹配的数据访问契约，增强框架的可观测性和工具链支持。

## 3. 设计详解 (DetailedDesign)

*   **核心思路**:
    我们认识到`function`节点和非`function`节点在数据处理上的本质区别。因此，我们提议在它们各自的元数据定义中，声明相匹配的契约。

### 3.1. 非`function`节点的数据契约 (NonFunctionNodeDataContract)

此类节点（如`action/log`, `flow/forEach`）不与`DataT`交互。我们在其`NodeDefinition`中声明对`Data`和`Metadata`的访问。

*   **API变更: `matrix/pkg/types/node.go`**:
<!--
finetune_role: code_generation_example
finetune_instruction: "展示如何修改NodeDefinition结构以增加数据契约字段"
-->
    ```go
    // MetadataDef 描述了对一个元数据键的读或写
    type MetadataDef struct {
        Key         string `json:"key"`
        Description string `json:"description"`
    }

    // NodeDefinition 描述一个节点的静态元数据
    type NodeDefinition struct {
        Type        string   `json:"type"`
        Name        string   `json:"name"`
        // ... (其他字段)

        // ReadsData (可选) 声明节点从原始 RuleMsg.Data 中读取的字段路径列表。
        // 例如: ["customer.name", "order.id"]
        ReadsData []string `json:"readsData,omitempty"`
        // ReadsMetadata (可选) 声明节点读取的元数据键。
        ReadsMetadata []MetadataDef `json:"readsMetadata,omitempty"`
        // WritesMetadata (可选) 声明节点写入的元数据键。
        WritesMetadata []MetadataDef `json:"writesMetadata,omitempty"`
    }
    ```

### 3.2. `function`节点的数据契约 (FunctionNodeDataContract)

此类节点是`DataT`的主要消费者和生产者。我们在其具体的`FuncObject.Configuration`中声明所有数据访问。

*   **API变更: `matrix/pkg/types/func.go`**:
<!--
finetune_role: code_generation_example
finetune_instruction: "展示如何修改FuncObjConfiguration结构以增加数据契约字段"
-->
    ```go
    // FuncObjConfiguration holds the detailed configuration definition of a function node.
    type FuncObjConfiguration struct {
        // ... (已有字段)
        Inputs   []IOObject           `json:"inputs"`  // DataT 输入
        Outputs  []IOObject           `json:"outputs"` // DataT 输出

        // --- 新增字段 ---
        // ReadsData (可选) 声明函数从原始 RuleMsg.Data 中读取的字段路径列表。
        ReadsData []string `json:"readsData,omitempty"`
        // ReadsMetadata (可选) 声明函数读取的元数据键。
        ReadsMetadata []MetadataDef `json:"readsMetadata,omitempty"`
        // WritesMetadata (可选) 声明函数写入的元数据键。
        WritesMetadata []MetadataDef `json:"writesMetadata,omitempty"`
    }
    ```
    *(注: `MetadataDef`需定义在`node.go`或`types.go`以被`func.go`引用)*

*   **逻辑自洽**: 此方案解决了`function`节点共用`NodeDefinition`的矛盾。每个具体的函数（如`log`函数）都在其独有的`FuncObject`中定义自己的数据契约，而通用的`FunctionNode`的`NodeDefinition`则保持为空。

## 4. 缺点与风险 (DrawbacksAndRisks)

*   **契约与实现可能不一致**: 开发者可能会忘记更新`FuncObject`或`NodeDefinition`中的契约，需要通过代码审查来缓解。
*   **增加了少量开发负担**: 节点和函数开发者需要额外声明其数据依赖。

## 5. 备选方案 (Alternatives)

*   **在`NodeDefinition`中定义所有契约**: 此方案因无法解决`function`节点共用`NodeDefinition`的矛盾而被否决。将契约放在`FuncObject`中是更精确的层级。

## 6. 未解决的问题 (UnresolvedQuestions)

*   对于泛型节点或函数，其契约可能是动态的。这需要未来的RFC引入更高级的契约声明机制。

## 7. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：这个变更会破坏现有节点吗？**
> **答：** 不会。所有新字段都是可选的 (`omitempty`)。现有定义中没有这些字段，会被视为空。这是一个向后兼容的增量增强。

> **问：如何保证`function`节点的`NodeDefinition`不被错误地填充？**
> **答：** `FunctionNode`的`Definition()`方法实现应硬编码返回一个不包含`Reads/Writes`契约的`NodeDefinition`。所有契约信息都应从其持有的`FuncObject`中读取。
<!-- qa_section_end -->
