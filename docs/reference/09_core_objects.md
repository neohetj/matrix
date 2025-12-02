---
# === Node Properties: 定义文档节点自身 ===
uuid: "b8c1d4e1-7b3e-4c2a-8f5d-9e1b3c4d5a6b"
type: "Specification"
title: "参考: 核心数据契约 (CoreObj)"
status: "Stable"
owner: "@cline"
version: "2.0.0"
tags:
  - "core-objects"
  - "specification"
  - "data-contract"
  - "coreobj"
  - "schema"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "e4a9b8c1-3d2a-4b1e-8c5d-7a4b9c2d8e1f"
    description: "本规范详细定义了Matrix架构总览中提到的核心数据对象。"
---

# 参考: 核心数据契约 (CoreObj)

本文档提供了对Matrix框架中**最核心的数据交换单元 `CoreObj`** 的权威定义和使用规范。理解 `CoreObj` 是进行任何 Matrix 业务开发的前提。

本文档内容基于 `matrix/pkg/types/coreobj.go` 源码。

## 1. 核心理念：自描述的数据契约

`CoreObj` (Core Object) 不仅仅是一个简单的Go结构体，它是一种**自描述的、包含标准数据规约的数据契约**。

-   **数据与规约合一**: 每个 `CoreObj` 的定义都包含了其实例（Go结构体）和一个自动生成的 **OpenAPI v3 Schema**。
-   **类型安全**: 通过全局唯一的**语义ID (SID)** 进行标识和查找，确保了在动态的规则链中数据交换的类型安全。
-   **自动生成**: 开发者只需定义一个标准的Go结构体并为其添加`tags`，框架便能通过反射自动生成其完整的OpenAPI Schema。

### 1.1. 语义ID (SID) 规范

SID (Semantic ID) 是 `CoreObj` 的全局唯一标识符，其命名必须遵循严格的规范以保证系统的可维护性。

-   **格式**: `<ObjectName>V<Major>_<Minor>_SID`
-   **`<ObjectName>`**: 对象名称，使用大驼峰命名法 (UpperCamelCase)。例如 `UserProfile`。
-   **`V<Major>_<Minor>`**: 版本号。
    -   **`<Major>`**: 主版本号。当发生不兼容的API变更时（如删除字段、修改字段类型），主版本号必须增加。
    -   **`<Minor>`**: 次版本号。当发生向后兼容的变更时（如增加可选字段），次版本号必须增加。
-   **`_SID`**: 固定的后缀，以明确该常量是一个SID。

**示例**:
- `UserProfileV1_0_SID`: 用户配置文件的1.0版本。
- `UserProfileV1_1_SID`: 为 `UserProfile` 增加了一个可选字段后的版本。
- `UserProfileV2_0_SID`: 对 `UserProfile` 的某个字段进行了破坏性修改后的版本。

> **核心原则**: 永远不要修改一个已经发布的 `CoreObjDef`。当需要变更时，应通过提升版本号来创建一个新的 `CoreObjDef`。这确保了向后兼容性和系统的稳定性。

## 2. 两个关键实体：定义与实例

在Matrix中，`CoreObj`体系由两个核心实体构成：

| 实体 | Go类型 | 职责 |
| :--- | :--- | :--- |
| **核心对象定义** | `CoreObjDef` | **类型的元数据**。它代表一个`CoreObj`的“类”，包含了SID、描述和OpenAPI Schema。它被注册在全局的`CoreObjRegistry`中。 |
| **核心对象实例** | `CoreObj` | **类型的实例数据**。它是在规则链中实际流动的数据容器，包含了对`CoreObjDef`的引用和具体的Go结构体实例（`Body`）。 |

## 3. `CoreObjDef` 详解：从Go结构体到Schema

`CoreObjDef` 是通过工厂函数 `NewCoreObjDef` 创建的。这个函数是实现“自描述”的关键。

<!--
finetune_role: "code_explanation"
finetune_instruction: "解释NewCoreObjDef函数如何通过Go原型和反射，自动生成包含OpenAPI Schema的CoreObjDef。"
-->
```go
// NewCoreObjDef creates a new CoreObjDef from a prototype instance.
// It uses reflection to generate the OpenAPI schema.
func NewCoreObjDef(prototype any, sid, desc string) *CoreObjDef {
    // ...
    // 1. 获取Go类型信息
    objType := reflect.TypeOf(prototype)
    
    // 2. 通过反射遍历struct fields，解析struct tags
    schema := schemaFromStruct(typForFields)
    schema.Description = desc

    // 3. 序列化schema为JSON字符串并缓存
    schemaBytes, err := json.Marshal(schema)
    // ...
    
    // 4. 返回包含Go类型和Schema的CoreObjDef
    return def
}
```

### 3.1. 支持的Struct Tags

`schemaFromStruct` 函数在解析Go结构体时，会识别以下`struct tags`来构建OpenAPI Schema：

| Tag | 作用 | 示例 |
| :--- | :--- | :--- |
| **`json:"<name>"`** | 定义字段在JSON序列化时的名称。这是**必需的**。 | `json:"user_name"` |
| **`description:"<desc>"`** | 为字段提供一个人类可读的描述。 | `description:"用户的唯一标识符"` |
| **`enum:"<v1>,<v2>"`** | 将字段约束为一组预定义的枚举值。 | `enum:"active,inactive,pending"` |
| **`required:"true"`** | 标记该字段为必填项。 | `required:"true"` |

## 4. 最佳实践：定义一个完整的`CoreObj`

以下是一个定义 `CoreObj` 的完整示例，展示了如何组织代码和使用所有支持的`struct tags`。

**1. `types.go`: 定义Go结构体**
```go
// in: matrixext/nodes/user_service/types.go
package user_service

// UserProfile represents a user's profile information.
type UserProfile struct {
    // UserID is the unique identifier for the user.
    UserID string `json:"user_id" required:"true" description:"用户的唯一标识符"`
    
    // UserName is the display name of the user.
    UserName string `json:"user_name" required:"true" description:"用户的显示名称"`
    
    // Status represents the current status of the user account.
    Status string `json:"status" required:"true" enum:"active,inactive,pending" description:"用户账户状态"`
    
    // Email is the optional email address of the user.
    Email string `json:"email,omitempty" description:"用户的可选邮箱地址"`
}
```

**2. `const.go`: 定义SID和常量**
```go
// in: matrixext/nodes/user_service/const.go
package user_service

const (
    // UserProfileV1_0_SID is the semantic ID for UserProfile version 1.0.
    UserProfileV1_0_SID = "UserProfileV1_0"
)
```

**3. `coreobj_defs.go`: 注册`CoreObjDef`**
```go
// in: matrixext/nodes/user_service/coreobj_defs.go
package user_service

import (
    "github.com/NeohetJ/Architect/matrix/pkg/registry"
    "github.com/NeohetJ/Architect/matrix/pkg/types"
)

func init() {
    registry.Default.CoreObjRegistry.Register(
        types.NewCoreObjDef(
            &UserProfile{}, // 传入一个结构体原型实例
            UserProfileV1_0_SID,
            "Defines the structure of a user profile",
        ),
    )
}
```

**4. 生成的Schema (示意)**

当上述代码被执行后，`CoreObjRegistry` 中 `UserProfileV1_0_SID` 对应的 `CoreObjDef` 将会包含类似如下的OpenAPI Schema：

```json
{
  "type": "object",
  "description": "Defines the structure of a user profile",
  "properties": {
    "user_id": {
      "type": "string",
      "description": "用户的唯一标识符"
    },
    "user_name": {
      "type": "string",
      "description": "用户的显示名称"
    },
    "status": {
      "type": "string",
      "description": "用户账户状态",
      "enum": ["active", "inactive", "pending"]
    },
    "email": {
      "type": "string",
      "description": "用户的可选邮箱地址"
    }
  },
  "required": ["user_id", "user_name", "status"]
}
```
