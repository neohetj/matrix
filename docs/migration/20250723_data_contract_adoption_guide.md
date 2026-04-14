---
uuid: "ca31ab15-9a80-4868-8073-9cbfff7ca948"
type: "MigrationGuide"
title: "迁移指南：适配节点数据契约"
status: "Active"
owner: "neohetj"
version: "1.1.0"
tags:
  - "migration"
  - "data-contract"
  - "node-interface"
relations:
  - type: "is_part_of"
    target_uuid: "affcadae-091e-4f63-9072-c3c8216e8f40"
    description: "本迁移指南属于 migration 目录的正式内容。"
---

# 迁移指南：适配节点数据契约

## 1. 变更概述

Matrix 当前使用统一的 URI 契约来描述节点和函数的读写行为。早期文档里提到的 `GetDataContract()`、`ReadsData`、`ReadsMetadata`、`WritesMetadata` 已经不是现行实现。

当前代码的关键点是：

1. `types.Node` 对外暴露 `DataContract() DataContract`。
2. 通用节点通过 `NodeMetadata.NodeReads` / `NodeWrites` 声明契约。
3. 函数通过 `FuncObject.Configuration.FuncReads` / `FuncWrites` 声明补充契约。
4. `functions` 节点的 `DataT` 读写还会叠加 DSL 中的 `inputs` / `outputs`。

## 2. 受影响范围

- **`types.Node` 接口**：需要满足 `DataContract() DataContract`。
- **嵌入 `types.BaseNode` 的普通节点**：通常无需额外实现 `DataContract()`，但要补齐 `NodeReads` / `NodeWrites`。
- **特殊节点**：如 `FunctionsNode`、需要动态构造契约的节点，必须自己实现 `DataContract()`。
- **函数定义**：应补齐 `FuncReads` / `FuncWrites`、`Inputs` / `Outputs`、`Business`。

## 3. 迁移清单

### 3.1. 普通节点

- [ ] 审查 `OnMsg()`、`Start()`、`Stop()` 等实现，识别真实读写位置。
- [ ] 把 `msg.Metadata()` 的读写迁移成 `rulemsg://metadata/...` URI。
- [ ] 把 `msg.Data()` 的读写迁移成 `rulemsg://data...` URI。
- [ ] 把结构化对象读写迁移成 `rulemsg://dataT/...` URI。
- [ ] 在 `NodeMetadata.NodeReads` / `NodeWrites` 中声明这些 URI。

示例：

```go
BaseNode: *types.NewBaseNode(ForEachNodeType, types.NodeMetadata{
    Name: "For Each",
    NodeReads: []types.ContractDef{
        {URI: "rulemsg://dataT/items", Description: "待遍历集合"},
    },
    NodeWrites: []types.ContractDef{
        {URI: "rulemsg://metadata/loop.index", Description: "当前索引"},
        {URI: "rulemsg://metadata/loop.isLast", Description: "是否最后一项"},
    },
}),
```

### 3.2. 函数节点

- [ ] 审查函数实现是否直接读取 `msg.Data()` / `msg.Metadata()`。
- [ ] 若有，写入 `FuncReads` / `FuncWrites`。
- [ ] 如果函数读写 `DataT`，继续通过 `Inputs` / `Outputs` 声明逻辑参数。
- [ ] 检查 DSL 上的 `inputs` / `outputs` 是否与函数签名一致。
- [ ] 检查 `defineSid` 是否与 `IOObject.DefineSID` 保持一致。

示例：

```go
Configuration: types.FuncObjConfiguration{
    Inputs: []types.IOObject{
        {ParamName: "pagination", DefineSID: "JsonDBPagination", Required: true},
    },
    Outputs: []types.IOObject{
        {ParamName: "result", DefineSID: "WindowListResponse"},
    },
    FuncReads: []types.ContractDef{
        {URI: "rulemsg://metadata/requestId", Description: "链路追踪 ID"},
    },
}
```

### 3.3. `functions` DSL 接线

迁移时需要一并核对 DSL：

- [ ] `type` 必须是 `functions`。
- [ ] `configuration.functionName` 必须与 `FuncObject.ID` 一致。
- [ ] 业务配置应收敛到 `configuration.business`。
- [ ] 节点级 `inputs` / `outputs` 必须和函数签名一致，不能夹带伪输入。

## 4. 推荐验证步骤

1. 运行 DSL / 函数签名静态校验，确认 `defineSid`、必填输入、输出映射没有漂移。
2. 抽样跑一条真实规则链，验证运行时生成的契约视图仍然正确。
3. 若改动包含 endpoint / object mapper / shared resource，再补一轮映射和资源解析测试。

## 5. 常见迁移误区

- 把旧文档中的 `ReadsMetadata` 直接抄回代码。
- 只改函数签名，不同步 DSL `inputs` / `outputs`。
- 继续把函数业务配置平铺在 `configuration` 顶层，而不是放进 `configuration.business`。
- 只补静态契约，不补最小规则链验证，导致文档和运行时继续漂移。
