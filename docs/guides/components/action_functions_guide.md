---
# === Node Properties: 定义文档节点自身 ===
uuid: "98c5b4a3-e7f6-4b0d-8c6a-1e2f3a4b5c6d"
type: "ComponentGuide"
title: "组件指南：通用函数 (functions)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "action"
  - "function"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix核心能力层的动作组件之一。"
  - type: "references"
    target_uuid: "81080378-a3e9-41ee-86ed-807193d45bce"
    description: "本文档遵循语义化文档规范编写。"
---

# 1. 功能概述 (FunctionalOverview)

`functions` 节点是一个**通用函数执行器**。它本身不包含任何具体的业务逻辑，而是扮演一个“代理”或“调度器”的角色。

它的核心功能是根据配置中提供的函数名，从一个全局的**函数注册表 (`NodeFuncManager`)** 中查找一个预先通过Go代码注册的函数，然后执行它。

这个节点是 `Matrix` 框架扩展性的关键，它允许开发者用Go编写自定义的、可复用的业务逻辑片段，然后在规则链中通过名字来灵活调用。

> **重要区别**: 此 `functions` 节点 (NodeType: `functions`) 不同于 `pkg/components/functions/` 目录下的具体功能节点 (如 `sql_query`, `redis_command`)。后者是具有独立 `NodeType` 的、功能固化的节点；而前者是一个通用的、可以执行**任何**已注册函数的“空壳”节点。

# 2. 如何配置 (Configuration)

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `functionName` | 函数名 | 要执行的、已在 `NodeFuncManager` 中注册的函数的唯一名称。 | `string` | 是 | N/A |

# 3. 核心概念：函数注册表 (FunctionRegistry)

`functions` 节点的所有能力都源于其背后的函数注册表 (`registry.Default.NodeFuncManager`)。

*   **注册 (Go)**: 开发者可以在Go代码中实现一个函数，该函数接收 `(types.NodeCtx, types.RuleMsg)` 作为参数，并将其封装在一个 `types.NodeFuncDef` 结构中，然后调用 `NodeFuncManager.Register()` 将其注册到一个全局唯一的名称下。
*   **调用 (DSL)**: 在规则链的DSL中，开发者可以放置一个 `functions` 节点，并将其 `functionName` 配置为注册时使用的那个唯一名称。
*   **解耦**: 这种机制将函数的**实现**（Go代码）与**调用**（DSL配置）完全解耦，使得在不修改DSL的情况下，可以热更新或替换函数的底层实现。

## 3.1. 如何开发并注册一个函数

以下是一个完整的示例，展示了如何开发一个新函数并将其注册到 `NodeFuncManager` 中，使其可以被 `functions` 节点调用。

**场景**: 创建一个名为 `EnrichUserProfile` 的函数，它接收一个包含 `UserID` 的 `CoreObj`，然后调用外部服务获取用户的详细信息，并将结果填充到一个新的 `UserProfile` 对象中。

**第一步：定义数据契约 (`CoreObj`)**

首先，我们需要定义函数的输入和输出数据结构。这遵循标准的 `CoreObj` 开发流程。

> 关于 `CoreObj` 的详细定义和最佳实践，请参阅 **[参考: 核心数据契约 (CoreObj)][Ref-CoreObj]**。

```go
// in: matrixext/nodes/user_service/types.go
package user_service

type UserEnrichmentRequest struct {
    UserID string `json:"user_id" required:"true"`
}

type UserProfile struct {
    UserID   string `json:"user_id" required:"true"`
    UserName string `json:"user_name" required:"true"`
    Status   string `json:"status" required:"true" enum:"active,inactive,pending"`
}

// in: matrixext/nodes/user_service/const.go
const (
    UserEnrichmentRequestV1_0_SID = "UserEnrichmentRequestV1_0"
    UserProfileV1_0_SID           = "UserProfileV1_0"
    EnrichUserProfileFuncID       = "EnrichUserProfile"
    ParamNameUserInput            = "userInput"
    ParamNameUserProfile          = "userProfile"
)

// in: matrixext/nodes/user_service/coreobj_defs.go
func init() {
    registry.Default.CoreObjRegistry.Register(
        types.NewCoreObjDef(&UserEnrichmentRequest{}, UserEnrichmentRequestV1_0_SID, "..."),
        types.NewCoreObjDef(&UserProfile{}, UserProfileV1_0_SID, "..."),
    )
}
```

**第二步：实现并注册函数**

```go
// in: matrixext/nodes/user_service/funcs.go
package user_service

import (
    "github.com/NeohetJ/Architect/matrix/pkg/registry"
    "github.com/NeohetJ/Architect/matrix/pkg/types"
)

func init() {
    registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
        Func: EnrichUserProfile, // 关联到下面的Go函数
        FuncObject: types.FuncObject{
            ID:   EnrichUserProfileFuncID,
            Name: "Enrich User Profile",
            Configuration: types.FuncObjConfiguration{
                Inputs: []types.IOObject{
                    {ParamName: ParamNameUserInput, DefineSID: UserEnrichmentRequestV1_0_SID},
                },
                Outputs: []types.IOObject{
                    {ParamName: ParamNameUserProfile, DefineSID: UserProfileV1_0_SID},
                },
            },
        },
    })
}

// EnrichUserProfile 是具体的业务逻辑实现
func EnrichUserProfile(ctx types.NodeCtx, msg types.RuleMsg) {
    // 1. 获取输入
    inputUntyped, err := msg.DataT().GetByParam(ctx, ParamNameUserInput)
    if err != nil {
        ctx.TellFailure(msg, err)
        return
    }
    req, _ := inputUntyped.Body().(*UserEnrichmentRequest)

    // 2. 执行业务逻辑 (伪代码)
    // userProfile := external_service.GetUserProfile(req.UserID)
    userProfile := &UserProfile{
        UserID:   req.UserID,
        UserName: "Mock User",
        Status:   "active",
    }

    // 3. 创建并设置输出
    // 3.1. 通过参数名创建一个新的、空的 CoreObj 实例
    newObj, err := msg.DataT().NewItemByParam(ctx, ParamNameUserProfile)
    if err != nil {
        ctx.TellFailure(msg, err)
        return
    }
    // 3.2. 为新的 CoreObj 实例设置 Body
    if err := newObj.SetBody(userProfile); err != nil {
        ctx.TellFailure(msg, err)
        return
    }
    
    // 4. 告诉框架处理成功
    ctx.TellSuccess(msg)
}
```

# 4. 配置示例 (Example)

以下DSL展示了如何在一个规则链中调用上面 `3.1` 节中定义的 `EnrichUserProfile` 函数。

**DSL 配置**:
```json
{
  "id": "node-enrich-profile",
  "type": "functions",
  "name": "丰富用户信息",
  "configuration": {
    "functionName": "EnrichUserProfile"
  },
  "inputs": {
    "userInput": {
      "objId": "userEnrichRequestObj",
      "defineSid": "UserEnrichmentRequestV1_0"
    }
  },
  "outputs": {
    "userProfile": {
      "objId": "enrichedUserProfileObj",
      "defineSid": "UserProfileV1_0"
    }
  }
}
```

**流程解析**:
1.  消息到达 `node-enrich-profile` 节点。假设此时 `DataT` 容器中已有一个 `objId` 为 `userEnrichRequestObj` 的对象。
2.  节点从 `configuration` 中读取 `functionName` 为 `"EnrichUserProfile"`。
3.  **数据绑定 (输入)**: 节点查看 `inputs` 块，它将函数在Go代码中定义的逻辑输入参数 `userInput` (即 `ParamNameUserInput`)，绑定到 `DataT` 中 `objId` 为 `userEnrichRequestObj` 的对象上。
4.  **执行**: 节点在 `NodeFuncManager` 中查找到 `EnrichUserProfile` 函数并执行。函数内部通过 `msg.DataT().GetByParam(ctx, "userInput")` 就能正确获取到 `userEnrichRequestObj` 对象。
5.  **数据绑定 (输出)**: 函数执行成功后，节点查看 `outputs` 块。它将函数在Go代码中定义的逻辑输出参数 `userProfile` (即 `ParamNameUserProfile`)，绑定到 `DataT` 中一个 `objId` 为 `enrichedUserProfileObj` 的新对象上。函数内部通过 `msg.DataT().NewItemByParam(ctx, "userProfile")` 创建的对象，其 `objId` 将被设置为 `enrichedUserProfileObj`。

# 5. 数据契约 (DataContract)

`functions` 节点的数据契约是**动态的**。它没有固定的输入输出。

当规则链被解析时，节点会根据其配置的 `functionName`，从注册的函数定义中动态地获取该函数预先声明的 `DataContract`。这意味着，即使 `functions` 节点是通用的，静态分析工具和UI编辑器也能准确地知道一个特定配置的 `functions` 节点将会读取和写入哪些数据。

# 6. 错误处理 (ErrorHandling)

*   **函数未找到 (`DefFuncNotFound`)**: 这是最常见的错误。如果配置的 `functionName` 在 `NodeFuncManager` 中不存在，节点会立即失败，并将消息路由到 `Failure` 链路。
*   **函数执行错误**: 被调用的函数在执行过程中遇到的任何错误，应由函数自身通过 `ctx.HandleError()` 或 `ctx.TellFailure()` 来处理。这些错误会正常地在规则链中传播。

<!-- 链接定义区域 -->
[Guide-MatrixOverview-2b3c4d]: ../00_matrix_guide.md
[Ref-SemanticDoc-d45bce]: ../../reference/04_semantic_documentation_standard.md
[Ref-CoreObj]: ../../reference/09_core_objects.md
