---
uuid: "86e6d134-004f-4483-a97a-396d2ab79a37"
type: "RFC"
title: "需求：Matrix Core Interface Boundary Refactor"
status: "Draft"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "rfc"
  - "architecture"
  - "interface-boundary"
  - "refactor"
relations:
  - type: "is_part_of"
    target_uuid: "68b1646b-2238-48e0-b77d-c81ecfc4317d"
    description: "本文档属于 Matrix RFC 文档库。"
  - type: "references"
    target_uuid: "9e38d28b-cd7f-49b6-b099-23fdee01329e"
    description: "本 RFC 延续 Matrix 内聚性重构 RFC 中关于框架职责回收与宿主边界澄清的目标。"
  - type: "references"
    target_uuid: "e4a9b8c1-3d2a-4b1e-8c5d-7a4b9c2d8e1f"
    description: "当前整体架构事实参考 Matrix 架构概览。"
  - type: "references"
    target_uuid: "f745eae6-f75c-4849-b7fb-407d6c439182"
    description: "当前通用节点生命周期与节点接口事实参考通用节点规范。"
  - type: "references"
    target_uuid: "a1b3d5e7-c4a0-4b1e-9c3a-1f8e6d2c5b4a"
    description: "共享资源边界与 ref:// 资源管理参考共享资源管理文档。"
  - type: "references"
    target_uuid: "dbf6fb21-5e26-44b6-b1bb-d94d13112ae9"
    description: "函数目录、函数注册与函数节点契约参考函数开发与注册规范。"
  - type: "references"
    target_uuid: "18d7026c-61eb-4b39-b537-7d927df34921"
    description: "endpoint/mcp 暴露出的协议适配边界问题参考 MCP 业务入口适配器 RFC。"
  - type: "is_refined_by"
    target_uuid: "ce131041-d7a5-494c-b4f3-dace15eff644"
    description: "本 RFC 的分阶段重构执行方案由 Plan 0015-1 承接。"
---

# RFC: Matrix Core Interface Boundary Refactor

## 1. 摘要

本 RFC 提议收窄 Matrix core 中过宽的运行时、注册表、节点上下文、共享资源池、endpoint 与函数目录抽象，避免核心接口继续以全局 service locator 的方式暴露内部能力。

目标不是一次性重写 Matrix，而是为后续分阶段重构定义稳定边界：DSL 仍然负责业务编排，代码负责能力实现；节点只获得自己声明需要的能力，宿主和协议 adapter 通过显式接口与 Matrix 交互。

## 原始需求点总结

1. 当前 Matrix 的 `MatrixEngine`、`Runtime`、`NodeCtx` 暴露面过宽，节点可以通过上下文穿透到 runtime、engine、registry、loader、shared pool 和 function manager，依赖关系难以声明和验证。
2. `NodePool` 同时承担 DSL 加载、共享资源创建、endpoint 收集、生命周期管理和资源实例解析，抽象职责混杂，阻碍 endpoint registry 与 shared resource resolver 独立演进。
3. `Endpoint` 把 HTTP、MCP、pipeline / active endpoint 等不同协议形态压进同一个接口，并通过 `SetRuntimePool(any)` 注入依赖，说明协议边界和运行时依赖边界尚未建模清楚。
4. 函数节点执行、函数注册、函数契约 introspection 之间仍存在全局 registry 与 engine-local registry 混用，可能导致执行路径和契约生成路径不一致。
5. 启动校验、DSL 装载、函数注册、HTTP handler、MCP tool result 等边界的错误语义不统一，部分路径 panic、部分路径 warning 后继续、部分路径写 response 后返回 nil。
6. 全局 factory 与默认 registry 降低多实例隔离能力，也让测试、动态加载、宿主组合和未来协议 adapter 更容易互相污染。
7. 当前 Reference / Guide / ADR / Plan 与运行时事实存在漂移风险，不能直接把历史文档当作重构事实来源，实施前需要先做文档事实基线和决策追踪闭环。
8. runtime reload、destroy、endpoint lifecycle、shared resource lifecycle 的所有权和并发边界不清晰，后续不应继续把 reload 暴露给执行期节点或非 owner 对象。
9. DSL loader、shared resource loader、endpoint loader 的 parse failure / missing target / optional fallback 语义不一致，应该形成可报告、可审计、可切换 strict 的启动失败模型。
10. Matrix / WhiteRoom / Morpheus 的 generic infrastructure 不应硬编码业务模块、frontend shell、workspace 拓扑或产品身份，相关身份只能来自显式配置、manifest、registry 或 inspection metadata。
11. Morpheus、CI、文档审计和 rulechain validator 需要稳定的 validation / inspection 输出模型；否则即使接口拆分完成，也会继续依赖 Matrix 内部结构。

## 2. 动机

Matrix 的长期目标是成为 DSL-first 的业务规则链引擎。按照当前工作区治理模型，业务编排应落在 `code/dsl/**`，能力实现应落在函数、服务和共享资源，宿主只承担 trust boundary、启动生命周期和协议接入。

当前核心接口已经完成了“统一入口”和“框架内聚”的第一阶段，但也留下了新的边界问题：

1. 统一入口把能力收回 Matrix 后，核心对象开始暴露过多内部容器。
2. 节点实现可以通过 `NodeCtx.GetRuntime().GetEngine()` 获取远超自身职责的能力。
3. 共享资源、endpoint、runtime、function catalog 使用全局默认 registry 作为隐式后备路径。
4. 协议 endpoint 与共享节点复用同一资源池抽象，导致 transport adapter、runtime dispatcher 和 shared resource resolver 的职责相互交叉。
5. DSL 规范声称规则链是 DAG，但运行时结构校验、依赖声明和契约校验尚未形成统一的启动 validation pipeline。
6. 当前文档事实层没有完成治理闭环，旧 RFC / ADR / Reference 只能作为历史输入，不能替代代码事实和新的验收结果。
7. reload / lifecycle 仍暴露在宽 runtime 接口里，容易让执行、资源生命周期、endpoint 启停和宿主 reload owner 混在一起。

这些问题短期内不一定表现为单个 bug，但会提高以下工作的成本：

- 多 Matrix 实例在同一宿主进程内并行运行。
- WhiteRoom / Morpheus / 业务模块通过显式配置组合 Matrix，而不是依赖隐藏全局状态。
- 新增 MCP、WebSocket、stream、scheduler 等协议入口时保持一个业务主路径。
- 将 DSL 图、函数契约、数据契约、endpoint IO mapping 做成可验证的静态模型。
- 在启动阶段准确失败，而不是在运行时才暴露隐式依赖缺失。

## 3. 设计详解

### 3.1 目标架构原则

本 RFC 采用以下原则约束后续重构：

1. **能力显式注入**：节点、endpoint、函数只获得自己声明需要的能力。
2. **接口按角色拆分**：执行、生命周期、资源解析、目录查询、协议适配分成小接口。
3. **全局状态只作为兼容层**：`registry.Default`、`types.DefaultRegistry`、全局 factory 只能作为迁移桥，不作为新代码依赖入口。
4. **启动先校验再运行**：DSL、节点配置、函数路由、endpoint target、共享资源引用和图结构必须进入统一 validation pipeline。
5. **协议 adapter 不复制业务流程**：HTTP、MCP、stream、scheduler 只负责协议 framing、上下文解析和结果映射，业务主路径仍是 DSL / capability / service。

### 3.2 拆分 MatrixEngine 能力面

当前 `MatrixEngine` 直接暴露 runtime pool、shared node pool、node manager、function manager、loader、biz config 和 logger。后续应收敛为面向宿主的只读 facade，并将内部依赖分成显式角色。

建议新增或沉淀以下小接口：

```go
type RuleChainExecutor interface {
    Execute(ctx context.Context, chainID string, input RuleMsg) (RuleMsg, error)
    ExecuteAsync(ctx context.Context, chainID string, input RuleMsg, onEnd ExecutionCallback) error
}

type RuntimeCatalog interface {
    GetRuntime(chainID string) (RuntimeHandle, bool)
    ListRuntimeIDs() []string
}

type SharedResourceResolver interface {
    Resolve(ctx context.Context, ref ResourceRef) (any, error)
}

type FunctionCatalog interface {
    GetFunction(id string) (*NodeFuncObject, bool)
    ListFunctions() []*NodeFuncObject
}

type ConfigResolver interface {
    GetConfig(ctx context.Context, key string) (any, bool)
}
```

`MatrixEngine` 可以继续作为组合入口存在，但不应成为节点执行期的默认能力入口。

### 3.3 拆分 Runtime 接口

当前 `Runtime` 同时包含执行、同步等待、热重载、销毁、定义读取、node pool、engine、chain instance 等能力。应拆成：

| 接口 | 面向对象 | 允许能力 |
| --- | --- | --- |
| `RuleChainExecutor` | endpoint / subchain trigger / host dispatcher | 执行业务规则链 |
| `RuntimeLifecycle` | Matrix engine / runtime pool | reload、destroy、stop |
| `RuntimeInspector` | 调试、trace、开发工具 | 读取 definition、chain instance、节点列表 |
| `RuntimeHandle` | runtime pool 内部句柄 | 聚合 executor + inspector 的最小组合 |

节点执行时默认只能拿到执行上下文和声明的 resolver，不应拿到 `RuntimeLifecycle` 或 engine 全量能力。

### 3.4 拆分 NodeCtx 能力

当前 `NodeCtx` 同时提供规则链上下文、日志、路由、消息创建、节点定义、runtime 访问和 completion callback。后续应拆为更小角色：

```go
type NodeExecutionContext interface {
    context.Context
    NodeID() string
    PreviousNodeID() string
    ChainID() string
    Logger() Logger
}

type NodeRouter interface {
    TellSuccess(msg RuleMsg)
    TellFailure(msg RuleMsg, err error)
    TellNext(msg RuleMsg, relationTypes ...string)
    HandleError(msg RuleMsg, err error)
}

type NodeDependencyContext interface {
    SharedResources() SharedResourceResolver
    Functions() FunctionCatalog
    Config() ConfigResolver
}
```

具体节点通过接口组合声明依赖。普通纯处理节点不应因为需要 `TellSuccess` 而自动获得 runtime、engine、loader 或 registry。

### 3.5 拆分 NodePool / Shared Resource / Endpoint Catalog

当前 `NodePool` 混合了 DSL load、shared node create、resource instance resolve、endpoint list 和生命周期。建议拆为：

| 新边界 | 职责 |
| --- | --- |
| `SharedNodeLoader` | 从 DSL / NodeDef 创建 shared node |
| `SharedResourceRegistry` | 保存和解析 `ref://` 资源 |
| `EndpointCatalog` | 保存可被宿主发现的 endpoint descriptors |
| `SharedResourceLifecycle` | stop / destroy shared resources |

endpoint 被发现后应进入 `EndpointCatalog`，而不是作为 shared node pool 的副产物被收集。shared node 仍可作为 endpoint 的底层资源，但 endpoint 目录不应由资源池拥有。

### 3.6 重建 Endpoint 协议抽象

`Endpoint` 不应强制同时继承 `Node`、`SharedNode` 并暴露 `SetRuntimePool(any)`。建议将 endpoint 分为协议形态和运行时依赖两个维度：

```go
type EndpointDescriptor interface {
    ID() string
    Protocol() string
    Target() EndpointTarget
}

type RuntimeBoundEndpoint interface {
    BindExecutor(executor RuleChainExecutor) error
}

type HttpEndpoint interface {
    EndpointDescriptor
    HandleHTTP(w http.ResponseWriter, r *http.Request) error
}

type McpToolEndpoint interface {
    EndpointDescriptor
    ListTools(ctx context.Context) ([]McpToolDefinition, error)
    CallTool(ctx context.Context, name string, arguments map[string]any) (McpToolResult, error)
}

type ActiveEndpoint interface {
    EndpointDescriptor
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

HTTP endpoint、MCP endpoint、stream endpoint 可以共享 target mapping、auth context 和 result normalizer，但不应被同一个 coarse interface 约束生命周期。

### 3.7 函数目录与函数契约快照

函数注册应区分“可变注册器”和“运行时只读目录”：

```go
type FunctionRegistry interface {
    Register(funcs ...*NodeFuncObject) error
    Freeze() FunctionCatalog
}

type FunctionCatalog interface {
    Get(id string) (*NodeFuncObject, bool)
    List() []*NodeFuncObject
    Contract(id string, bindings NodeBinding) (DataContract, error)
}
```

后续约束：

1. 注册阶段遇到非法 routing mode、重复 ID、非法 schema、缺少默认值等问题时返回 error，不 panic。
2. runtime 创建时使用同一个 `FunctionCatalog` 快照执行函数和生成契约。
3. `functions` 节点的 `Errors`、`DataContract`、`ConfigSchema` 不能绕回全局默认 registry。
4. 函数元数据应成为 DSL validation 的输入，而不是只在执行时查询。

### 3.8 统一启动 Validation Pipeline

建议将 Matrix 启动分为四个阶段：

```text
Load
  -> Parse
  -> Validate
  -> Build
  -> Run
```

Validation 至少覆盖：

1. rulechain graph 是 DAG，不存在环、悬空连接、重复 node ID、无效 relation。
2. endpoint target 存在且 target kind 有明确 dispatcher。
3. `ref://` shared resource 引用可解析，除非节点显式声明可选 fallback。
4. function name 存在，函数路由 relation 与 DSL 连线一致。
5. `NodeReads` / `FuncReads` / endpoint IO mapping 与 `RuleMsg` / `DataT` 绑定一致。
6. loader 发现的 DSL、endpoint、shared node parse 失败时按配置决定 strict / compatibility 行为，默认新代码走 strict。

Validation 失败应返回结构化错误集合，供 CLI、host、CI 和文档审计工具复用。

### 3.9 统一错误语义

建议明确四类错误边界：

| 边界 | 推荐语义 |
| --- | --- |
| 启动 / 注册 / DSL validation | 返回 `error` 或 error collection，不 panic、不 warning 后默认继续 |
| 节点执行 | 使用 `FailureInfo` / `Fault` / `HandleError` 进入规则链错误语义 |
| HTTP handler | handler 返回 protocol-neutral error，adapter 统一写 response |
| MCP tool | 协议允许的 tool-level error 使用 `McpToolResult.IsError`，transport / protocol error 使用 `error` |

协议 handler 不应同时“写 response 后返回 nil”和“返回未写出的 ServiceError”两种语义混用。

### 3.10 全局 factory 与默认 registry 的迁移边界

`types.NewMsg`、`types.NewNodeCtx`、`types.NewDataT` 等全局 factory 可以保留为兼容层，但新增代码应优先从 engine-scoped factory 获取构造能力：

```go
type MessageFactory interface {
    NewMsg(msgType string, data string, metadata Metadata, dataT DataT) RuleMsg
    CloneMsgWithDataT(msg RuleMsg, dataT DataT) RuleMsg
    NewSubMsg(parent RuleMsg, subChainID string) RuleMsg
}
```

默认 registry 同理：

1. 新 engine 创建时默认生成独立 registry。
2. 显式传入 shared registry 时才共享组件目录。
3. 旧的 `registry.Default` 只作为无配置启动和历史节点的 fallback。
4. 测试应能构造隔离 registry，不污染其他测试或宿主进程。

### 3.11 问题清单与优先级

后续 Plan 应把问题按影响面和依赖顺序拆分，不把所有高风险改动压在同一阶段。优先处理影响不是特别大、但能降低后续重构不确定性的事项：

| 优先级 | 问题 | 处理方式 |
| --- | --- | --- |
| P0 | 文档事实层不可信 | 先做 Reference / Guide / ADR / Plan 基线审计，明确哪些文档是历史输入、哪些是当前事实 |
| P0 | validation / inspection 输出没有稳定模型 | 先定义 error collection、inspection snapshot、validator report 的 schema，再迁移消费者 |
| P1 | loader failure 语义不统一 | 先把 parse failure、missing target、optional fallback 统一成 report-only 结果，再切 strict |
| P1 | function registry panic-only | 先引入 error-return 注册契约并覆盖内建函数注册测试，再删除 panic-only 生产入口 |
| P1 | endpoint catalog 与 shared pool 混用 | 先建立 endpoint descriptor catalog，再迁移 HTTP / MCP / active endpoint dispatcher |
| P2 | runtime reload / lifecycle owner 不清晰 | 先把 reload / destroy / stop 收敛到 `RuntimeLifecycle` owner，禁止节点执行期穿透触发 |
| P2 | generic infrastructure 身份硬编码风险 | 先补 repo / manifest / config contract 检查，再迁移 WhiteRoom、Morpheus、业务模块调用点 |

### 3.12 建议迁移顺序

本 RFC 不直接定义实施计划，但建议后续 Plan 按以下风险顺序拆分：

1. 增加小接口和 adapter，不删除旧接口。
2. 将 endpoint runtime pool 注入从 `any` 改为 typed dependency，并保留旧方法兼容。
3. 给 function registry 增加 `Register(... ) error` 新 API，旧 API 逐步转为兼容包装。
4. 建立 validation pipeline，并先以 warning / report 模式运行。
5. 将 `FunctionsNode` contract / schema / errors 改为 engine-scoped catalog。
6. 将 endpoint catalog 从 shared node pool 中拆出。
7. 将默认 registry 和全局 factory 降级为 legacy fallback。

## 4. 缺点与风险

1. 接口拆分会增加短期适配层数量，旧节点和新节点会在一段时间内共存。
2. 严格 validation 可能暴露历史 DSL 的隐性问题，必须提供兼容模式和迁移报告。
3. 运行时、endpoint、shared resource、function catalog 的边界拆分可能影响宿主初始化流程，需要与 WhiteRoom scaffold 和模块 server 同步。
4. 如果只增加新接口但不清理旧穿透路径，系统会变成“双抽象”而不是边界收敛。
5. function registry 从 panic 改成 error 后，需要重新约定内建组件注册失败时的启动处理策略。
6. 如果文档事实层未先校准，后续实现可能按过期 Reference 或历史 ADR 重构，造成新的架构漂移。
7. 如果 reload / lifecycle owner 没有先收敛，endpoint catalog 和 shared resource lifecycle 拆分后仍可能保留隐式并发风险。

## 5. 备选方案

### 5.1 继续保持当前宽接口

优点是改动最小，现有节点无需迁移。缺点是 service locator 问题会继续扩大，MCP、WebSocket、stream endpoint 和多实例场景都会继续依赖隐式全局状态。

### 5.2 只修具体 bug，不调整接口

例如只修 DAG validation、reload、function contract registry 混用、endpoint error handling。这样能短期降低风险，但不能解决抽象层让这些问题反复出现的根因。

### 5.3 一次性重写 runtime 和 registry

一次性替换所有核心接口可以得到更干净的结果，但风险过高，也不符合当前 DSL-first 模块渐进演进的治理方式。Matrix 已有大量内建节点、测试和模块依赖，应采用增量兼容迁移。

### 5.4 将所有能力都下沉为 DSL 配置

这会把复杂 domain rule、协议细节、事务、性能敏感逻辑和 host lifecycle 过度 DSL 化，违背当前工作区治理规则。DSL 应负责编排，接口抽象应负责让编排边界可验证，而不是把所有实现细节都写进 JSON。

## 6. 未解决的问题

1. `RuleChainExecutor` 是否应以 chain ID 为入口，还是以 runtime handle 为入口。
2. `EndpointCatalog` 应存放 endpoint node 实例、descriptor，还是 transport-neutral adapter。
3. `FunctionRegistry.Register` 的兼容 API 如何命名，是否保留 panic 版本给内建组件 init 使用。
4. validation pipeline 的 error collection 是否需要稳定 JSON schema，供 CLI、CI 和文档审计复用。
5. `SharedResourceResolver` 是否应支持 typed resolver，例如 `Resolve[T]`，还是保持 Go 1.x 接口兼容的非泛型返回。
6. `NodeCtx` 拆分后，旧节点如何最小成本适配。
7. 默认 registry 何时可以从新代码中完全禁用。

## 7. 常见问题与解答

<!-- qa_section_start -->
> **问：这个 RFC 是否要求立刻修改所有节点？**
> **答：** 不要求。它定义的是后续边界收敛方向。实施时应先增加小接口和兼容 adapter，再逐步迁移高风险路径。

> **问：这会不会破坏现有 DSL？**
> **答：** RFC 本身不改变 DSL。后续 validation pipeline 可能发现历史 DSL 问题，但应先以报告模式运行，再按 Plan 决定 strict 切换点。

> **问：为什么不直接把 `MatrixEngine` 保持成统一入口？**
> **答：** `MatrixEngine` 可以继续作为宿主组合入口，但节点执行期不应把它当成全量能力容器。统一入口和执行期最小依赖是两个不同问题。

> **问：MCP endpoint 为什么要纳入这个 RFC？**
> **答：** MCP 暴露出当前 endpoint 抽象过宽的问题：它是 endpoint，但不一定需要 HTTP runtime pool，也不应被 `SetRuntimePool(any)` 这种接口约束。这个问题适用于未来所有非 HTTP 协议入口。
<!-- qa_section_end -->

## 8. 附录：当前观察到的接口问题

本节记录促成本 RFC 的当前代码观察点。它不是最终实现规范，后续应由 Reference / ADR / Plan 承接。

| 主题 | 当前观察 |
| --- | --- |
| Engine 能力面 | `pkg/types/runtime.go` 中 `MatrixEngine` 暴露 registry、runtime pool、shared pool、loader、config、logger 等全量能力 |
| Runtime 能力面 | `Runtime` 同时负责执行、reload、destroy、definition、node pool、engine、chain instance |
| NodeCtx 穿透 | `NodeCtx.GetRuntime()` 允许节点在执行期进一步访问 engine 和 registry |
| NodePool 职责 | `NodePool` 同时 load DSL、创建 shared node、解析 instance、保存 endpoint list、stop resources |
| Endpoint 抽象 | `Endpoint` 继承 `Node` 和 `SharedNode`，并用 `SetRuntimePool(any)` 传入运行时依赖 |
| MCP 空实现 | `endpoint/mcp` 不需要 runtime pool，但仍被迫实现 `SetRuntimePool` |
| 函数契约 | `FunctionsNode.OnMsg` 可走 engine-local manager，但 `Errors` / `DataContract` / `ConfigSchema` 仍读取默认 registry |
| 函数注册 | `NodeFuncManager.Register` 无 error 返回，非法配置会 panic |
| 全局 factory | `types.NewMsg`、`types.NewNodeCtx`、`types.NewDataT` 等是可替换全局变量 |
| 启动校验 | DAG、endpoint target、shared ref、函数 relation、IO contract 尚未形成统一 validation pipeline |
| 文档事实层 | Reference / Guide / ADR / Plan 需要先与当前代码事实和最终验收结果对齐 |
| loader failure | DSL、endpoint、shared resource parse failure 与 optional fallback 语义未形成统一报告模型 |
| reload 生命周期 | reload、destroy、stop、endpoint lifecycle、shared resource lifecycle 的 owner 边界仍需收敛 |
| generic identity | generic infrastructure 需要避免硬编码业务模块、frontend shell、workspace 拓扑或产品身份 |
| inspection 输出 | Morpheus、CI、文档审计、validator 需要稳定 validation / inspection schema，避免继续依赖内部 pool 结构 |

## 9. 后续文档承接

如果本 RFC 被接受，建议补充：

1. ADR：确认 core interface split 和默认 registry 降级策略。
2. Plan：定义兼容迁移阶段、验收标准和 validation 切换策略。
3. Reference：更新 Matrix 架构图、runtime / registry / endpoint / function catalog 当前事实。
4. Guide：更新节点开发、endpoint 开发和函数注册的操作说明。
