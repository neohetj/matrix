---
uuid: "4231ef33-0963-4b9c-87c9-76277c7e8472"
type: "ComponentGuide"
title: "组件指南：运维拓扑节点 (ops/*)"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "ops"
  - "topology"
  - "deployment"
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "ops 节点是 Matrix 当前拓扑建模能力的一部分。"
  - type: "references"
    target_uuid: "a2b1c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
    description: "本文档遵循 components 目录的组件指南规范。"
  - type: "references"
    target_uuid: "978c1b44-65eb-43ef-bcf8-793c1793a0b3"
    description: "本文档给出 RFC-0006 当前已落地 ops 节点与 DSL 扩展的使用入口。"
  - type: "references"
    target_uuid: "5a8b3c7e-9f0d-4a1b-8c2d-6e5f4a3b2c1d"
    description: "ops 建模与 relations/imports 的当前 DSL 语义以 Reference-18 为准。"
  - type: "references"
    target_uuid: "a6b7c8d9-e0f1-4a2b-8c3d-4e5f6a7b8c9d"
    description: "组件目录收录了当前所有内建 ops 节点。"
---

# 1. 功能概述 (Overview)

`ops/*` 是 Matrix 当前用于**静态拓扑建模**的一组内建节点。它们的目标不是在运行时执行业务逻辑，而是把“应用、服务、数据库、网络、运行器、机器”等基础设施对象编码进 DSL，供以下场景复用：

1. 静态拓扑建模
2. 可视化渲染
3. 后续工作流通过 `imports` 复用拓扑定义
4. 面向部署/巡检/生成器的上层工具读取

它们对应 [RFC-0006](../../designs/rfc/0006_ops-foundation-components-and-dsl-extensions_rfc.md) 的当前已落地部分，但需要特别注意：**当前实现仍以静态配置承载为主，并没有把 RFC 中“消息驱动探测 + 动态 state”完整做完。**

## 2. 当前已支持的节点类型 (SupportedNodeTypes)

当前代码里已经注册的类型如下：

| NodeType | 说明 | 主要配置结构 |
| :--- | :--- | :--- |
| `ops/application` | 逻辑应用边界 | `ApplicationConfig` |
| `ops/service` | 可部署服务单元 | `ServiceConfig` |
| `ops/database` | 数据库依赖 | `DatabaseConfig` |
| `ops/message_queue` | 消息队列依赖 | `MessageQueueConfig` |
| `ops/network` | 网络资源 | `NetworkConfig` |
| `ops/volume` | 存储卷资源 | `VolumeConfig` |
| `ops/runner` | 执行器 / 宿主环境 | `RunnerConfig` |
| `ops/machine` | 机器 / 主机资源 | `MachineConfig` |

说明：

1. 当前实现中的消息队列节点类型是 **`ops/message_queue`**。
2. 早期提案或旧文档里如果出现 `ops/messageQueue`，应以当前代码中的下划线命名为准。

## 3. 常见配置字段 (CommonFields)

### 3.1 可部署节点

`ops/service`、`ops/database`、`ops/message_queue` 这几类节点都会复用 `DeploymentSpec`，常用字段包括：

| 字段 | 含义 |
| :--- | :--- |
| `runtimeProfile` | 运行时画像，例如 `matrix-service`、`compose-service` |
| `deployAdapter` | 部署适配器，例如 `docker-compose` |
| `deployable` | 是否可部署 |
| `runnerRef` | 指向运行器或宿主环境 |
| `artifactRef` / `image` | 制品或镜像标识 |
| `ports` | 暴露端口 |
| `envRefs` / `secretRefs` | 依赖的配置或密钥引用 |
| `networkRefs` / `volumeRefs` | 依赖的网络或卷 |
| `dependsOn` | 拓扑层面的依赖项 |
| `endpointRefs` / `ruleChainRefs` | 关联的入口或规则链 |

### 3.2 非部署类节点

其他节点更偏资源描述：

1. `ops/application`：`domain`、`environment`、`owners`
2. `ops/network`：`driver`、`scope`
3. `ops/volume`：`driver`、`scope`、`mountPath`、`persistent`
4. `ops/runner`：`executorType`、`environment`、`address`
5. `ops/machine`：`hostname`、`address`、`operatingSystem`、`architecture`、`labels`

## 4. 推荐使用方式 (HowToUse)

### 4.1 作为不可执行拓扑定义

最推荐的方式，是把 `ops/*` 节点放在一个单独的、不可执行的拓扑 DSL 中：

```json
{
  "ruleChain": {
    "id": "base_topology",
    "attrs": {
      "executable": false,
      "viewType": "static-topology"
    }
  },
  "metadata": {
    "nodes": [
      {
        "id": "app_orders",
        "type": "ops/application",
        "name": "Orders App",
        "configuration": {
          "domain": "orders",
          "environment": "prod"
        }
      },
      {
        "id": "svc_orders_api",
        "type": "ops/service",
        "name": "Orders API",
        "configuration": {
          "serviceType": "http",
          "language": "go",
          "deployment": {
            "deployable": true,
            "runtimeProfile": "matrix-service",
            "image": "registry.example/orders-api:1.0.0",
            "runnerRef": "runner_prod",
            "networkRefs": ["network_prod"],
            "volumeRefs": ["volume_logs"]
          }
        }
      },
      {
        "id": "db_orders",
        "type": "ops/database",
        "name": "Orders DB",
        "configuration": {
          "engine": "postgres",
          "version": "16",
          "deployment": {
            "deployable": true,
            "runtimeProfile": "compose-database",
            "runnerRef": "runner_prod"
          }
        }
      }
    ],
    "relations": [
      { "source": "app_orders", "target": "svc_orders_api", "label": "hasPart" },
      { "source": "svc_orders_api", "target": "db_orders", "label": "dependsOn" }
    ]
  }
}
```

### 4.2 由工作流通过 `imports` 复用

当你需要在巡检、部署、生成器等规则链中复用这些节点时，建议通过 `ruleChain.attrs.imports` 引入拓扑定义，而不是在每条链里重新复制一遍节点。

## 5. 当前限制 (CurrentLimitations)

当前 `ops/*` 节点需要这样理解：

1. 它们主要负责**承载静态配置**。
2. 当前节点实现基本只做配置解码，不负责复杂的 `OnMsg` 处理。
3. RFC 原文里“收到 `PROBE` 消息后探测并更新内部 state”的能力，目前还不是现状。
4. 如果你的目标是部署生成、拓扑展示或关系分析，这套实现已经够用；如果你的目标是动态探测执行，还需要在此之上继续扩展。

## 6. 相关现行文档 (RelatedDocs)

1. [RFC-0006：运维基础组件与 DSL 扩展](../../designs/rfc/0006_ops-foundation-components-and-dsl-extensions_rfc.md)
2. [学习 Matrix DSL 规范](../../reference/18_dsl_specification.md)
3. [学习 Matrix 组件目录](../../reference/21_component_catalog.md)
