---
# === Node Properties: 定义文档节点自身 ===
uuid: "ref-func-registration-spec-20250911"
type: "Specification"
title: "参考-11: 函数开发与注册规范"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "reference"
  - "function"
  - "specification"
  - "registry"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_referenced_by"
    target_uuid: "dev-patterns-entrypoint-20250911" # -> 08_node_development_patterns.md
    description: "本文档为函数开发模式提供了底层的、详细的实现规范。"
  - type: "references"
    target_uuid: "node-spec-20250911" # -> 12_node_specification.md
    description: "函数的数据契约是对通用节点数据契约的扩展，本文档引用通用规范作为基础。"
---

# 参考-11: 函数开发与注册规范

本文档为开发者提供了在 `Matrix` 框架中开发和注册自定义**函数 (Function)** 的完整、权威的规范。

## 1. 核心理念：声明式的元数据

`Matrix` 中的函数不仅仅是一段Go代码，它是一个**包含了丰富元数据的、自描述的逻辑单元**。通过 `types.NodeFuncObject` 结构，开发者可以将Go函数 (`Func`) 与其完整的元数据 (`FuncObject`) 绑定在一起，然后注册到全局的 `NodeFuncManager` 中。

这种模式使得框架本身以及外部工具（如UI编辑器）都能在**运行时**和**静态分析时**，清晰地了解一个函数的能力、配置需求和数据依赖。

## 2. `NodeFuncObject`: 注册的基本单元

这是向 `registry.Default.NodeFuncManager` 注册一个函数的标准结构。

```go
// in: matrix/pkg/types/func.go
type NodeFuncObject struct {
	Func       NodeFunc   // 业务逻辑的Go函数实现
	FuncObject FuncObject // 函数的完整元数据和配置定义
}
```

## 3. `FuncObject`: 函数的元数据

`FuncObject` 定义了一个函数“是什么”。

```go
// in: matrix/pkg/types/func.go
type FuncObject struct {
	ID            string               `json:"id"`   // 全局唯一ID, e.g., "EnrichUserProfile"
	Name          string               `json:"name"` // 人类可读的名称, e.g., "Enrich User Profile"
	Desc          string               `json:"desc"`
	Dimension     string               `json:"dimension"`
	Tags          []string             `json:"tags"`
	Version       string               `json:"version"`
	Configuration FuncObjConfiguration `json:"configuration"` // [核心] 函数的配置与数据契约
}
```

## 4. `FuncObjConfiguration`: 配置与数据契约

这是函数定义中**最核心**的部分，它声明了函数“如何工作”以及“需要什么”。

### 4.1. 输入/输出 (`Inputs`/`Outputs`)

`Inputs` 和 `Outputs` 数组定义了函数与 `RuleMsg.DataT` 容器的交互契约。

-   **结构**: `[]IOObject`
-   **`IOObject`**:
    -   `ParamName`: 在 `DataT` 中存取 `CoreObj` 实例时使用的参数名（Key）。
    -   `DefineSID`: 该参数名对应的 `CoreObj` 的语义ID (SID)。
    -   `Desc`: 描述。
    -   `Required`: (仅用于`Inputs`) 标记该输入参数是否为必须。

**示例**:
```go
// 声明该函数需要一个名为 "userInput" 的输入参数，
// 其类型必须是 "UserEnrichmentRequestV1_0"。
// 它会产生一个名为 "userProfile" 的输出，
// 其类型为 "UserProfileV1_0"。
Inputs: []types.IOObject{
    {ParamName: "userInput", DefineSID: "UserEnrichmentRequestV1_0", Required: true},
},
Outputs: []types.IOObject{
    {ParamName: "userProfile", DefineSID: "UserProfileV1_0"},
},
```

#### 4.1.1. 深入理解：`DataT` 与参数解耦

`Inputs` 和 `Outputs` 声明的背后，是 `Matrix` 框架管理业务数据流的核心机制：`DataT` 容器与参数解耦。

`DataT` (`types.DataT`) 是 `RuleMsg` 的核心，它是一个以对象ID（`objId`）为键，存储业务对象（`CoreObj`）的线程安全容器。它的最关键特性是**写时复制 (Copy-on-Write)** 语义：
- **`DataT` 是浅拷贝的**：当流程产生分支时，新旧`RuleMsg`指向**同一个** `DataT` 实例。这使得在同一逻辑分支上的状态传递非常高效。
- **`Metadata` 是深拷贝的**：每个分支拥有独立的`Metadata`，适合传递分支隔离的控制信息。

**参数解耦 (P-Key -> Obj-ID)** 是 `DataT` 机制的精髓，它允许函数的代码与具体使用完全解耦。

当你在 `Inputs` 中声明 `ParamName: "userInput"`，并在函数代码中调用 `msg.DataT().GetByParam(ctx, "userInput")` 时，框架会自动执行以下查找：
1.  读取当前 `functions` 节点在规则链DSL中的 `configuration`。
2.  在配置中找到 `inputs` 映射，将函数的逻辑参数名 (P-Key) `"userInput"` 解析为规则链中定义的具体对象ID (Obj-ID)，例如 `"user_request_from_http"`。
3.  使用这个 Obj-ID 从 `DataT` 容器中获取对应的 `CoreObj` 实例并返回。

这个机制解决了**可复用性**和**可测试性**的核心问题。函数代码只关心“我需要一个`userInput`”，而具体这个输入是什么，则由使用它的规则链在JSON配置中动态决定。函数本身变得完全与上下文无关，从而实现了最大程度的复用。

### 4.2. 补充数据契约

作为对通用节点数据契约的扩展，函数除了 `Inputs/Outputs` 外，也可以声明其对 `RuleMsg` 的 `Data` 和 `Metadata` 层的直接访问。

其核心概念与API（`ReadsData`, `ReadsMetadata`, `WritesMetadata`）与通用节点完全一致。详情请参阅 **[参考-12: 通用节点规范][Ref-NodeSpec]** 中关于数据合约的定义。

### 4.3. 业务配置 (`Business`)

`Business` 数组允许函数声明自己需要从 `functions` 节点的 `configuration` 部分读取哪些**自定义配置项**。

-   **结构**: `[]DynamicConfigField`
-   **`DynamicConfigField`**:
    -   `Name`: 配置项的名称。
    -   `Type`: 配置项的期望类型 (`string`, `int`, `bool`, `float`, `map`, `array`)。
    -   `Desc`: 描述。
    -   `Required`: 是否为必须。

**示例**:
假设一个函数需要一个名为 `retryCount` 的数字配置。
```go
// 1. 在 FuncObjConfiguration 中声明
Business: []types.DynamicConfigField{
    {Name: "retryCount", Type: "int", Desc: "重试次数", Required: false},
},

// 2. 在 Go 函数中通过 NodeCtx 获取
func MyFunc(ctx types.NodeCtx, msg types.RuleMsg) {
    // GetConfig() 返回的是 `functions` 节点的 `configuration`
    retryCount, err := ctx.Config().GetInt("retryCount")
    if err != nil {
        // 如果不是必须的，则使用默认值
        retryCount = 3 
    }
    // ...
}

// 3. 在 DSL 中配置
{
    "id": "myNode",
    "type": "functions",
    "configuration": {
        "functionName": "MyFunc",
        "retryCount": 5 // 覆盖默认值
    }
}
```

## 5. 完整开发与注册流程

1.  **定义 `CoreObj`**: 为函数的输入和输出创建 `CoreObj` 定义。
2.  **实现 `NodeFunc`**: 编写 `func(ctx NodeCtx, msg RuleMsg)` 签名的Go函数。
    -   使用 `msg.DataT().GetByParam(ctx, ...)` 获取输入。
    -   使用 `msg.DataT().NewItemByParam(ctx, ...)` 结合 `coreObj.SetBody(...)` 来创建和设置输出。
    -   使用 `ctx.Config().Get...()` 获取业务配置。
    -   使用 `ctx.TellSuccess(msg)` 或 `ctx.TellFailure(msg, err)` 结束处理。
3.  **定义 `NodeFuncObject`**: 在 `init()` 函数中，创建一个 `NodeFuncObject` 实例，完整地填充其元数据和数据契约。
4.  **注册**: 调用 `registry.Default.NodeFuncManager.Register()` 将其注册。

<!-- 链接定义区域 -->
[Ref-NodeSpec]: ./12_node_specification.md
