---
# === Node Properties: 定义文档节点自身 ===
uuid: "f8c00454-f58f-4cea-9bf3-93c61f30294d"
type: "Plan"
title: "计划：运维基础组件与DSL扩展的实现"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "plan"
  - "implementation"
  - "ops"
  - "dsl"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_plan_for"
    target_uuid: "978c1b44-65eb-43ef-bcf8-793c1793a0b3" # -> 指向RFC-0006
    description: "本实现计划旨在将RFC-0006中定义的提案转化为可执行的开发步骤。"
  - type: "is_guided_by"
    target_uuid: "becf62ab-9a79-483a-96d9-243928855ff9" # -> 指向ADR-0006-1
    description: "本计划的实施将严格遵循ADR-0006-1中记录的架构决策。"
---

# Plan: 计划：运维基础组件与DSL扩展的实现 (OpsComponentsAndDslImplPlan)

## 1. 计划概述 (Overview)

本计划文档详细描述了实现“RFC-0006: 运维基础组件与DSL扩展”所需的具体开发步骤。计划分为三个主要阶段：创建基础节点、扩展DSL解析器、以及编写测试与文档。

## 2. 核心问题澄清 (Clarifications)

在执行前，我们澄清以下两个关键设计点：

### 2.1. 如何支持节点自定义图标 (NodeIconSupport)

**问题**: 不同的运维节点（如`ops/machine`, `ops/service`）在可视化渲染时，需要有不同的图标来直观地区分。

**方案**: 我们将在 `types.NodeDefinition` 结构体中增加一个可选的 `Icon` 字段。

```go
// In: Architect/matrix/pkg/types/node.go
type NodeDefinition struct {
    Type        string   `json:"type"`
    Name        string   `json:"name"`
    // ... (其他字段)
    Icon        string   `json:"icon,omitempty"` // <-- 新增字段
}
```

*   **实现**: 在每个运维节点的 `Definition()` 方法返回的 `NodeDefinition` 中，为其 `Icon` 字段赋一个标准化的值。例如，对于 `MachineNode`，可以设置为 `"server"`；对于 `ServiceNode`，可以设置为 `"cog"`。
*   **消费**: 前端可视化工具或拓扑图生成器在解析规则链DSL后，可以读取每个节点的 `Definition().Icon` 字段，并根据该值从图标库（如Font Awesome, Material Icons）中选择并渲染对应的图标。

### 2.2. Application 与 Service 节点的区别与关联 (ApplicationVsService)

**问题**: `ops/application` 节点和 `ops/service` 节点应如何区分，它们之间是什么关系？

**方案**: 我们从概念、实现和关联三个层面来精确定义它们：

*   **概念区别与实现**:
    *   **`Application` (应用)**: **它是一个纯粹的逻辑概念，不可执行。** 它代表一个面向最终用户的、完整的业务产品或解决方案，用于组织和聚合服务。因此，`ApplicationNode` 的 `OnMsg` 方法将是一个严格的空操作（no-op），它在任何执行流中都只应作为元数据容器和关系路由的起点。
    *   **`Service` (服务)**: 代表一个可独立部署和运行的**具体技术实现**。一个服务的功能可以由**一组规则链**共同定义，并且可以通过多个API端点暴露其能力。

*   **`ServiceNode` 的详细配置**:
    ```go
    // In ServiceNode's Configuration struct
    type ServiceNodeConfiguration struct {
        // ... 其他配置
        // 关联到实现该服务核心逻辑的一组规则链ID
        RuleChainRefs []string `json:"ruleChainRefs,omitempty"`
        // 描述该服务暴露的API端点
        Endpoints []ServiceEndpoint `json:"endpoints,omitempty"`
    }

    type ServiceEndpoint struct {
        // 关联到Matrix规则链中定义的endpoint节点的ID。
        // 如果此字段有值，表示这是一个Matrix原生端点。
        // 如果此字段为空，表示这是一个外部、非Matrix管理的端点。
        EndpointRef string `json:"endpointRef,omitempty"`

        // 以下字段用于描述端点。
        // 对于外部端点，这些是必需的定义信息。
        // 对于Matrix原生端点，这些是可选的、冗余的描述性信息。
        Type     string `json:"type"` // http, grpc
        Path     string `json:"path"` // e.g., /v1/users/login
        Method   string `json:"method,omitempty"` // e.g., POST
        // ... 其他接口相关元数据
    }
    ```

*   **关联关系**:
    *   一个 `Application` 在逻辑上由一个或多个 `Service` 组成。
    *   在我们的DSL中，这种关系将通过 `relations` 数组来表达。
    *   **示例**:
        ```json
        "relations": [
          { "from": "app_user_center", "to": "svc_auth", "label": "hasPart" },
          { "from": "app_user_center", "to": "svc_profile", "label": "hasPart" }
        ]
        ```
    *   `app_user_center` (`ops/application`) 通过 `hasPart` 关系，声明了它在逻辑上包含了 `svc_auth` 和 `svc_profile` 这两个 `ops/service` 节点。

## 3. 实现计划 (ImplementationPlan)

我们将分三个阶段完成此任务：

### 阶段一：创建运维（Ops）基础节点 (CreateOpsBaseNodes)
我将首先在 `Architect/matrix/pkg/components/` 下创建 `ops` 目录，然后逐一创建以下节点。每个节点的实现都将遵循 `endpoint_link_node.go` 的模式，并包含动静分离的设计和新增的 `Icon` 字段。
1.  `ops/machine` 节点
2.  `ops/service` 节点
3.  `ops/application` 节点
4.  `ops/database` 节点
5.  `ops/control` 节点 (用于流程图的Start/End)

### 阶段二：扩展DSL解析器以支持 `imports` 和 `relations` (ExtendDSLParser)
1.  **修改DSL结构体**: 我将定位并修改 `RuleChainDef` 结构体（可能在 `parser/dsl.go` 中），为其增加 `Metadata` 和 `Relations` 字段。
2.  **修改解析器逻辑**: 我将修改核心的解析函数（可能在 `parser/parser.go` 中），以实现对新字段的解析，并实现 `imports` 关键字的递归文件加载和合并逻辑。

### 阶段三：编写测试与文档 (WriteTestsAndDocumentation)
1.  **单元测试**: 为所有新创建的 `ops` 节点编写单元测试。
2.  **集成测试**: 创建一个使用了所有新特性（`imports`, `relations`, 新节点）的复杂DSL文件，并编写集成测试来验证其解析和（如果可执行）运行的正确性。
3.  **文档**: 为新节点和新的DSL特性编写或更新相应的使用指南和规范文档。
