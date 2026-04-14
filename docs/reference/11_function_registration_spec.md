---
# === Node Properties: 定义文档节点自身 ===
uuid: "dbf6fb21-5e26-44b6-b1bb-d94d13112ae9"
type: "Specification"
title: "参考-11: 函数开发与注册规范"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "reference"
  - "function"
  - "specification"
  - "registry"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_referenced_by"
    target_uuid: "60c07c47-df0e-4b76-9ed9-62fabe2e2add"
    description: "本文档为函数开发模式提供底层实现规范。"
  - type: "references"
    target_uuid: "f745eae6-f75c-4849-b7fb-407d6c439182"
    description: "函数的数据契约扩展了通用节点契约。"
---

# 参考-11: 函数开发与注册规范

本文档定义 Matrix 中函数开发与注册的当前规范。这里的“函数”指被 `type: "functions"` 节点调用的 `types.NodeFuncObject`，而不是一个独立的 NodeType。

## 1. 核心模型

函数注册的基本单元是：

```go
type NodeFuncObject struct {
    Func       NodeFunc
    FuncObject FuncObject
}
```

其中：

- `Func` 是 `func(ctx types.NodeCtx, msg types.RuleMsg)` 形态的运行时入口。
- `FuncObject` 是元数据与配置契约。

## 2. `FuncObject` 元数据

```go
type FuncObject struct {
    ID            string               `json:"id"`
    Name          string               `json:"name"`
    Desc          string               `json:"desc"`
    Dimension     string               `json:"dimension"`
    Tags          []string             `json:"tags"`
    Version       string               `json:"version"`
    Configuration FuncObjConfiguration `json:"configuration"`
}
```

要求：

1. `ID` 必须全局唯一。
2. `ID` 必须与 DSL 中 `configuration.functionName` 一致。
3. `Name` / `Desc` / `Tags` / `Version` 应能支撑 UI 展示和静态分析。

## 3. `FuncObjConfiguration`

函数的详细契约定义如下：

```go
type FuncObjConfiguration struct {
    Name     string               `json:"name"`
    FuncDesc string               `json:"funcDesc"`
    Business []DynamicConfigField `json:"business"`
    Inputs   []IOObject           `json:"inputs"`
    Outputs  []IOObject           `json:"outputs"`
    Errors   []*Fault             `json:"errors"`

    RoutingMode       FunctionRoutingMode `json:"routingMode,omitempty"`
    DeclaredRelations []string            `json:"declaredRelations,omitempty"`

    FuncReads  []ContractDef `json:"funcReads,omitempty"`
    FuncWrites []ContractDef `json:"funcWrites,omitempty"`
}
```

### 3.1. `Inputs` / `Outputs`

`Inputs` 和 `Outputs` 定义函数与 `DataT` 的逻辑参数契约：

```go
type IOObject struct {
    ParamName string `json:"paramName"`
    DefineSID string `json:"defineSid"`
    Desc      string `json:"desc"`
    Required  bool   `json:"required"`
}
```

它们描述的是**函数视角**的参数名，不是规则链里的 `objId`。真正的 `objId` 绑定发生在 DSL 节点上的 `inputs` / `outputs`：

```json
{
  "id": "enrichProfile",
  "type": "functions",
  "configuration": {
    "functionName": "EnrichUserProfile"
  },
  "inputs": {
    "userInput": {
      "objId": "userreq001",
      "defineSid": "UserEnrichmentRequestV1_0"
    }
  },
  "outputs": {
    "userProfile": {
      "objId": "userprof001",
      "defineSid": "UserProfileV1_0"
    }
  }
}
```

函数内部通过：

- `msg.DataT().GetByParam(ctx, "userInput")`
- `msg.DataT().NewItemByParam(ctx, "userProfile")`

读取的是逻辑参数名，运行时再根据 `NodeDef` 上的 `inputs` / `outputs` 解析到实际 `objId`。

### 3.2. `Business`

`Business` 用于声明 `configuration.business` 内允许出现哪些业务配置项：

```go
type DynamicConfigField struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Type        cnst.MType `json:"type"`
    Desc        string     `json:"description"`
    Required    bool       `json:"required"`
    Default     any        `json:"defaultValue,omitempty"`
    NotEditable bool       `json:"notEditable,omitempty"`
}
```

推荐规则：

1. `ID` 使用稳定键名，供 DSL / UI / 校验器共同依赖。
2. `Name` 用于展示。
3. 默认值、只读字段等约束在这里一并声明。

读取配置时，推荐使用 `asset` + `helper`：

```go
assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))
retryCount, _ := helper.GetConfigAsset[int](assetCtx, "retryCount")
sql, _ := helper.RenderConfigAsset[string](assetCtx, "query")
```

而不是继续依赖旧文档里那种把业务字段直接写在 `configuration` 顶层的做法。

### 3.3. `FuncReads` / `FuncWrites`

如果函数直接读写 `msg.Data()` 或 `msg.Metadata()`，应通过 URI 形式声明：

- `rulemsg://metadata/requestId`
- `rulemsg://data`
- `rulemsg://data/result?format=JSON`

`DataT` 的对象级读写仍优先通过 `Inputs` / `Outputs` 声明。

### 3.4. `RoutingMode` / `DeclaredRelations`

默认函数是 `standard` 模式，只应通过 `TellSuccess` / `TellFailure` 结束。

只有以“决策”为主要职责的函数，才应声明：

- `routingMode: decision`
- `declaredRelations: ["Hit", "Miss", "Retry"]`

这样运行时才能校验自定义 relation 是否合法。

## 4. 推荐开发流程

1. 定义输入输出 `CoreObj`。
2. 编写 `NodeFuncObject`，补齐 `Inputs` / `Outputs` / `Business` / `FuncReads` / `FuncWrites` / `Errors`。
3. 实现 `func Xxx(ctx types.NodeCtx, msg types.RuleMsg)`。
4. 使用 `msg.DataT().GetByParam()` / `NewItemByParam()` 处理结构化输入输出。
5. 使用 `asset.NewAssetContext(...)` + `helper.GetConfigAsset(...)` 读取 `configuration.business`。
6. 调用 `registry.Default.NodeFuncManager.Register(...)` 完成注册。
7. 在 DSL 中使用 `type: "functions"`，设置 `configuration.functionName`，并补齐节点级 `inputs` / `outputs`。

## 5. 完整示例

```go
var EnrichUserProfileFuncObj = &types.NodeFuncObject{
    Func: EnrichUserProfile,
    FuncObject: types.FuncObject{
        ID:   "EnrichUserProfile",
        Name: "Enrich User Profile",
        Configuration: types.FuncObjConfiguration{
            Inputs: []types.IOObject{
                {ParamName: "userInput", DefineSID: "UserEnrichmentRequestV1_0", Required: true},
            },
            Outputs: []types.IOObject{
                {ParamName: "userProfile", DefineSID: "UserProfileV1_0"},
            },
            Business: []types.DynamicConfigField{
                {ID: "timeoutMs", Name: "Timeout", Type: "int", Desc: "外部服务超时", Default: 3000},
            },
            FuncReads: []types.ContractDef{
                {URI: "rulemsg://metadata/requestId", Description: "请求链路 ID"},
            },
        },
    },
}

func EnrichUserProfile(ctx types.NodeCtx, msg types.RuleMsg) {
    input, err := msg.DataT().GetByParam(ctx, "userInput")
    if err != nil {
        ctx.HandleError(msg, err)
        return
    }

    assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))
    timeoutMs, _ := helper.GetConfigAsset[int](assetCtx, "timeoutMs")

    _ = input
    _ = timeoutMs

    out, err := msg.DataT().NewItemByParam(ctx, "userProfile")
    if err != nil {
        ctx.HandleError(msg, err)
        return
    }

    if err := out.SetBody(&UserProfile{}); err != nil {
        ctx.HandleError(msg, err)
        return
    }

    ctx.TellSuccess(msg)
}
```

<!-- qa_section_start -->
> **问：函数的业务配置应该写在 DSL 的哪里？**
> **答：** 写在 `configuration.business` 下面。`configuration` 顶层只保留 `functionName` 以及少量通用字段。

> **问：函数声明了 `Inputs` / `Outputs`，为什么 DSL 里还要再写 `inputs` / `outputs`？**
> **答：** 因为函数签名描述的是“逻辑参数”，DSL 负责把逻辑参数绑定到这条规则链里的实际 `objId`。两者缺一不可。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Ref-NodeSpec]: ./12_node_specification.md
