---
uuid: "ca31ab15-9a80-4868-8073-9cbfff7ca948"
type: "MigrationGuide"
title: "迁移指南：适配节点数据契约"
status: "Active"
owner: "@cline"
version: "1.0.0"
tags:
  - "migration"
  - "data-contract"
  - "node-interface"
---

# 迁移指南：适配节点数据契约

## 1. 变更概述

为了增强框架的静态分析能力和UI工具链支持，我们对`types.Node`接口和相关定义进行了扩展，引入了统一的数据访问契约。本次变更的核心是在`types.Node`接口上增加了一个新方法`GetDataContract()`，并为`NodeMetadata`和`FuncObject`增加了用于声明数据依赖的字段。

虽然此变更是向后兼容的，但我们强烈建议所有节点开发者遵循本指南，为现有节点添加数据契约，以充分利用新功能带来的优势。

## 2. 受影响的范围

*   **`types.Node`接口**: 增加了一个新方法`GetDataContract() DataContract`。
*   **所有`types.Node`的实现**: 所有实现了`Node`接口的结构体都必须实现`GetDataContract`方法。
*   **非函数节点定义**: `types.NodeMetadata`结构体现已包含`ReadsData`, `ReadsMetadata`, `WritesMetadata`字段。
*   **函数节点定义**: `types.FuncObjectConfiguration`结构体现已包含`ReadsData`, `ReadsMetadata`, `WritesMetadata`字段。

## 3. 手动迁移与适配清单

### 3.1. 步骤一：实现`GetDataContract`接口方法

这是强制性的一步，以确保代码能够编译通过。

- [ ] **对于嵌入了`types.BaseNode`的节点**:
    - 无需任何操作。`BaseNode`已经提供了满足接口的默认实现。

- [ ] **对于未嵌入`types.BaseNode`或需要特殊逻辑的节点 (如`FunctionsNode`)**:
    - 必须手动为该节点实现`GetDataContract() types.DataContract`方法。
    - `FunctionsNode`的实现应从其持有的`FuncObject`中动态读取契约信息并返回。

### 3.2. 步骤二：为非函数节点添加契约

对于您开发的每一个非函数节点（如`action`, `filter`, `flow`等类型）：

- [ ] 审查节点`OnMsg`方法的实现逻辑。
- [ ] **如果**节点从`msg.Data()`中读取数据，请在`NodeMetadata`的`ReadsData`字段中声明所读取字段的路径列表（例如 `["user.id", "product.name"]`）。
- [ ] **如果**节点从`msg.Metadata()`中读取数据，请在`NodeMetadata`的`ReadsMetadata`字段中声明所读取的键。
- [ ] **如果**节点向`msg.Metadata()`中写入数据，请在`NodeMetadata`的`WritesMetadata`字段中声明所写入的键。

**示例 (`for_each_node.go`):**
```go
// 之前
var forEachNodePrototype = &ForEachNode{
	BaseNode: *types.NewBaseNode(ForEachNodeType, types.NodeMetadata{
		Name: "For Each",
		// ...
	}),
}

// 之后
var forEachNodePrototype = &ForEachNode{
	BaseNode: *types.NewBaseNode(ForEachNodeType, types.NodeMetadata{
		Name: "For Each",
		// ...
		WritesMetadata: []types.MetadataDef{
			{Key: MetadataKeyLoopIndex, Description: "..."},
			{Key: MetadataKeyIsLastItem, Description: "..."},
		},
	}),
}
```

### 3.3. 步骤三：为函数节点添加契约

对于您开发的每一个函数节点：

- [ ] 审查函数`Func`的实现逻辑。
- [ ] **如果**函数从`msg.Data()`中读取数据，请在`FuncObject.Configuration`的`ReadsData`字段中声明所读取字段的路径列表。
- [ ] **如果**函数从`msg.Metadata()`中读取数据，请在`FuncObject.Configuration`的`ReadsMetadata`字段中声明所读取的键。
- [ ] **如果**函数向`msg.Metadata()`中写入数据，请在`FuncObject.Configuration`的`WritesMetadata`字段中声明所写入的键。

## 4. 自动化迁移工具

本次变更不提供自动化迁移工具，需要开发者根据上述清单手动进行适配。
