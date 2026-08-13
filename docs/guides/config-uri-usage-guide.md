---
uuid: "788737cf-d58e-4931-8bbf-37841c6de80c"
type: "Guide"
title: "指南：Config URI 协议与统一配置读取"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "guide"
  - "config"
  - "uri"
  - "asset"
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "Config URI 是 Matrix 当前配置读取体系的一部分。"
  - type: "references"
    target_uuid: "7144507b-2c98-41e1-aeaa-34d4ccad476a"
    description: "本文档给出 RFC-0011 中 `config://` 协议已落地部分的实际用法。"
  - type: "references"
    target_uuid: "5a8b3c7e-9f0d-4a1b-8c2d-6e5f4a3b2c1d"
    description: "Config URI 在 DSL 中的使用仍应与 Reference-18 的当前约定保持一致。"
  - type: "references"
    target_uuid: "dbf6fb21-5e26-44b6-b1bb-d94d13112ae9"
    description: "函数节点侧通常通过 helper.GetConfigAsset / RenderConfigAsset 读取配置。"
---

# 1. 功能概述 (Overview)

`config://` 是 Matrix 当前已经落地的**统一配置读取协议**。它把节点业务配置、节点静态配置、引擎配置和环境变量收敛到同一套 URI 读取入口里，避免每个节点自己散写查配置逻辑。

对应的设计背景见 [RFC-0011](../designs/rfc/0011_config_uri_and_manager_rfc.md)。需要注意的是：**当前真正落地的是协议层与回退逻辑，不是“统一配置视图 UI”**。

## 2. 基本语法 (Syntax)

推荐写法：

```text
config:///some.key
config:///some.key?scope=engine,env
config:///some.key?default=30
config:///some.key?scope=node&default=foo
```

当前语法要点：

1. 推荐使用 `config:///path.to.key`，也就是把 key 放在 URI 的 path 中。
2. `scope` 用逗号分隔多个查找层级。
3. `default` 用于配置缺失时的兜底值。
4. `type` 查询参数当前主要用于上层工具或 helper 的扩展场景，核心读取逻辑不会因为它而改变查找顺序。

## 3. 默认查找顺序 (ResolutionOrder)

如果没有显式指定 `scope`，当前默认顺序是：

1. `business`
2. `node`
3. `engine`
4. `env`

这意味着同一个 key 会优先从节点的 `configuration.business` 中找，再到当前节点配置，再到引擎配置，最后才读环境变量。

### 3.1 `scope` 可选值

| scope | 查找位置 |
| :--- | :--- |
| `business` | `configuration.business` |
| `node` | 当前节点完整配置 |
| `engine` | 引擎 / business config / engine config |
| `env` | 环境变量；同时支持把点号转成下划线并转大写后的 key |

### 3.2 环境变量回退

如果 `config:///api.key` 走到 `env` scope：

1. 先查 `api.key`
2. 再查 `API_KEY`

## 4. 在 DSL 中怎么写 (DSLExamples)

### 4.1 函数节点业务配置

```json
{
  "id": "query_users",
  "type": "functions",
  "configuration": {
    "functionName": "sqlQuery",
    "business": {
      "dsn": "${config:///database.read.dsn?scope=business,engine,env}",
      "timeoutMs": "${config:///timeouts.sql?default=3000}"
    }
  }
}
```

### 4.2 只从引擎和环境变量读取

```text
config:///service.endpoint?scope=engine,env
```

### 4.3 完全禁止继续回退

如果某次解析后需要构造“剩余 scope”继续传递，底层会使用 `scope=-` 表示不再继续向下回退。这个行为更多是 helper / asset 内部机制，业务 DSL 一般不需要手写。

### 4.4 主动端点的启用开关

`endpoint/redis_stream` 和 `endpoint/pipeline` 的 `enabled` 字段也接受 `config://` 模板，用来决定引擎启不启动这个端点：

```json
{
  "configuration": {
    "enabled": "${config:///feature.consumer_enabled?scope=engine,env&default=false}"
  }
}
```

这条解析发生在引擎启动阶段，和上面几种用法有两点不同：

- **只有 `engine` 和 `env` 两个作用域可用。** 此时还没有 `NodeCtx` 与 `RuleMsg`，`business` 和 `node` 作用域读不到东西。引擎会把 `MatrixConfig.Business` 作为 `engine` 作用域喂入，所以 `business:` 配置树里的键仍然读得到。
- **解析失败不会回落成默认行为。** 键不存在又没写 `default`，或结果不是布尔值，引擎直接启动失败。

语义细节见 [Redis Stream Endpoint 可靠消费](../reference/40_redis_stream_endpoint_reliability.md) 第 2.1 节。

## 5. 在 Go 代码中怎么读 (GoUsage)

当前推荐通过 `assetCtx + helper` 读取，而不是手动拆 URI。

### 5.1 读取普通值

```go
assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))
timeoutMs, err := helper.GetConfigAsset[int](assetCtx, "timeoutMs")
```

### 5.2 读取并渲染模板值

```go
assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))
sql, err := helper.RenderConfigAsset[string](assetCtx, "query")
```

这种写法的好处是：

1. 配置项可以直接是常量。
2. 配置项也可以包含 `${config:///...}` 这样的嵌套引用。
3. helper 会统一处理解析、类型转换和错误传播。

## 6. 当前边界 (CurrentScope)

当前可以把 `config://` 理解成：

1. 一个**运行时配置解析协议**
2. 一个**统一读取入口**
3. 一个**scope 回退机制**

但不要把它误解成已经具备这些能力：

1. 统一配置总览 UI
2. 规则链级配置扫描页面
3. 面向业务人员的集中编辑视图

这些仍属于 RFC-0011 中尚未落地的部分。

## 7. 相关现行文档 (RelatedDocs)

1. [RFC-0011：Config URI 协议与规则链统一配置视图](../designs/rfc/0011_config_uri_and_manager_rfc.md)
2. [参考-11：函数开发与注册规范](../reference/11_function_registration_spec.md)
3. [学习 Matrix DSL 规范](../reference/18_dsl_specification.md)
