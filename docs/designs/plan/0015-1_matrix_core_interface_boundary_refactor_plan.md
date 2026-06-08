---
uuid: "ce131041-d7a5-494c-b4f3-dace15eff644"
type: "Plan"
title: "计划：Matrix Core Interface Boundary Refactor"
status: "Implementing"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "architecture"
  - "interface-boundary"
  - "refactor"
  - "cross-repo"
relations:
  - type: "is_plan_for"
    target_uuid: "86e6d134-004f-4483-a97a-396d2ab79a37"
    description: "本计划用于落地 Matrix core interface boundary refactor RFC。"
  - type: "is_part_of"
    target_uuid: "fde86d18-74ce-4350-9137-553929ee1261"
    description: "本文档属于 Matrix Plan 文档库。"
---

# Plan：Matrix Core Interface Boundary Refactor

## 1. 计划概述

本计划用于把 Matrix core 中过宽的 engine、runtime、registry、endpoint、function catalog、shared resource 与 node context 边界收敛为显式小接口，并同步修复 WhiteRoom、Morpheus 和业务模块对旧接口的依赖。

本计划采用“分阶段稳健推进，但阶段内不保留长期历史兼容”的策略：每个阶段都有明确的编译、测试和文档门禁；一旦阶段完成，该阶段相关旧入口应删除或降级为测试专用，不在生产路径保留双实现。

## 2. 范围与目标

### 2.1 范围

1. Matrix core interface split：
   - `MatrixEngine`
   - `Runtime`
   - `RuntimePool`
   - `NodePool`
   - `NodeCtx`
   - `Endpoint`
   - `NodeFuncManager`
   - global registry / global factory
2. Matrix startup validation pipeline：
   - DAG validation
   - endpoint target validation
   - `ref://` shared resource validation
   - function relation validation
   - data contract / endpoint IO validation
   - loader parse failure / optional fallback / strict mode validation
   - validation report / inspection snapshot schema
3. WhiteRoom common / scaffold template 同步：
   - `platform/WhiteRoom/common/server`
   - `platform/WhiteRoom/common/executors`
   - `platform/WhiteRoom/cmd/module-scaffold/templates`
   - `platform/WhiteRoom/internal/mcpservercmd`
4. 业务模块同步：
   - `modules/identityx`
   - `modules/lens`
   - `modules/notifyx`
   - `modules/paymentx`
   - `modules/sellitx`
   - `modules/usagex`
5. Morpheus 同步：
   - rulechain / node / function handlers
   - graph dependency / flow / layout / ops packages
   - trace and single-node execution tools
6. 文档与治理同步：
   - Matrix Reference / Guide / ADR / Plan
   - WhiteRoom governance / scaffold docs
   - 业务模块本地 README / RFC / ADR / Plan，如模块边界发生变化
   - 跨仓决策追踪，如后续把本计划拆成正式 Evolution cross-repo initiative
   - 当前事实层基线审计，避免按过期 Reference / ADR 实施
7. generic infrastructure 身份边界：
   - Matrix / WhiteRoom / Morpheus 不硬编码业务模块、frontend shell、workspace 拓扑或产品身份
   - 身份来源必须是显式 config、manifest、registry、workspace tooling output 或 inspection metadata

### 2.2 非范围

1. 不新增业务功能。
2. 不扩大 MCP tool catalog。
3. 不把安全边界迁入普通 DSL 或 decision function。
4. 不新增长期兼容旧接口的 adapter。
5. 不重写业务模块的产品 API 语义。
6. 不把 Morpheus private source packages 作为 Matrix、WhiteRoom 或业务模块依赖。

### 2.3 可衡量目标

1. Matrix 生产代码不再依赖 `matrix.Registry` / `types.DefaultRegistry` 作为运行时主路径。
2. HTTP / MCP / active endpoint 不再通过 shared node pool 扫描注册；统一通过 endpoint catalog 或等价显式目录暴露。
3. 函数执行、函数契约、函数 schema、函数错误列表使用同一个 engine-scoped function catalog。
4. `SetRuntimePool(any)` 从生产 endpoint 接口中删除。
5. shared resource 解析通过显式 resolver 完成，不再由函数节点自行穿透 `ctx.GetRuntime().GetEngine().SharedNodePool()`。
6. Morpheus 只通过 inspection API 消费 Matrix 运行时事实，不直接依赖 Matrix internal pool 结构。
7. WhiteRoom scaffold 生成的新模块默认使用新接口。
8. 所有受影响 Go 仓库通过 focused tests；最终阶段通过全仓 `go test ./...` 或记录可解释 baseline。
9. Matrix validation / inspection 输出有稳定 schema，可被 Morpheus、CI、rulechain validator 和文档审计工具复用。
10. Reference / Guide / ADR / Plan 已完成事实基线校准，历史设计文档不再作为未验证的当前事实来源。
11. generic infrastructure 中不新增业务模块、frontend shell、workspace 拓扑或产品身份硬编码。

## 3. 当前影响面

| 仓库 / 目录 | 当前依赖形态 | 计划改动 |
| --- | --- | --- |
| `platform/Matrix` | 定义宽接口、全局 registry、endpoint/shared pool 混用 | 定义新核心接口并删除旧生产路径 |
| `platform/WhiteRoom/common/server` | 通过 `eng.SharedNodePool()` 扫描 `endpoint/http` | 改为 endpoint catalog / HTTP endpoint registry |
| `platform/WhiteRoom/common/executors` | 通过 global registry 或 runtime->engine 解析 shared resource | 改为 `SharedResourceResolver` / asset context resolver |
| `platform/WhiteRoom/cmd/module-scaffold/templates` | 生成旧 server、executor、MCP host 代码 | 同步模板，防止新模块回退旧接口 |
| `modules/*/common/server` | 复制 WhiteRoom 旧 server 逻辑 | 通过 sync-common 或手动同步新 server |
| `modules/*/common/executors` | 复制 WhiteRoom common executor 逻辑 | 同步 resolver / registry / function registration 改动 |
| `modules/*/code/executors` | 大量 `NodeFuncObject`、`NodeCtx`、global register | 改为新 function registry / node execution context |
| `modules/identityx/internal/mcpservercmd` | 持有 `MatrixEngine` 执行 rulechain target | 改为显式 `RuleChainExecutor` 和 target dispatcher |
| `platform/Morpheus/pkg/handlers` | 直接使用 runtime pool、shared pool、function manager | 改为 inspection API |
| `platform/Morpheus/pkg/graph` | 直接消费 `types.MatrixEngine` 的 pool 和 runtime internals | 改为 `RuntimeInspector` / `EndpointCatalog` / `FunctionCatalog` |
| Matrix Reference / Guide / ADR / Plan | 部分文档可能落后于当前代码事实 | Stage 0 先做事实基线，Stage 8 再完成最终事实闭环 |
| generic Matrix / WhiteRoom / Morpheus infrastructure | 可能存在模块、shell、workspace 拓扑或产品身份硬编码 | 改为显式 config、manifest、registry 或 inspection metadata |

## 4. 设计原则

1. **不做长期兼容**：每个阶段完成后删除该阶段旧生产入口。
2. **阶段内可编译**：每个阶段结束必须让阶段 scope 内代码编译或明确标记未进入验证 scope。
3. **先 contract 后迁移**：先在 Matrix 定义新接口，再迁移 WhiteRoom 和模块。
4. **先模板后模块**：先改 WhiteRoom common/template，再同步已生成模块，避免旧模板反复生成旧代码。
5. **先 host 后 Morpheus**：业务 runtime 和 host 先跑通，再改 Morpheus inspection 视图。
6. **validation 先报告后强制**：启动 validation 可以先 report-only，但最终合并前 strict 模式必须覆盖新接口路径。
7. **先低影响稳定基线**：优先修复文档事实层、report-only validation、schema 定义、registry error-return 这类影响面较小但能降低后续风险的问题，再进入跨仓生产路径替换。

## 5. 实施计划

### 阶段 0：基线与执行边界

目标：确认当前分支、测试 baseline、文档事实基线、跨仓改动入口和审批边界。

步骤：

1. 在 `platform/Matrix` 当前分支确认文档计划已提交或 staged 边界清晰。
2. 为 WhiteRoom、Morpheus、各业务模块分别记录当前分支、dirty state、go test baseline。
3. 记录当前 Matrix `go test ./...` 已知失败项，避免后续误判为本次引入。
4. 审计 Matrix 当前 Reference / Guide / ADR / Plan 与代码事实的差异，标记：
   - 可作为当前事实的文档
   - 只能作为历史设计输入的文档
   - 必须在本计划 Stage 8 关闭的事实漂移
5. 建立当前问题 inventory，并按低影响优先顺序标记：
   - 文档事实层和决策追踪
   - validation / inspection schema
   - loader failure report-only
   - function registry error-return
   - endpoint catalog descriptor
   - reload / lifecycle owner
   - generic infrastructure identity boundary
6. 建立跨仓验证清单：
   - `platform/Matrix`
   - `platform/WhiteRoom`
   - `platform/Morpheus`
   - `modules/identityx`
   - `modules/lens`
   - `modules/notifyx`
   - `modules/paymentx`
   - `modules/sellitx`
   - `modules/usagex`
7. 如果要形成正式跨仓 initiative，在 Evolution root 创建 `docs/cross-repo/matrix-core-interface-boundary-refactor/` 文档集。

验收：

1. 每个仓库的 dirty state 已记录。
2. 每个仓库的 baseline test 命令和结果已记录。
3. Matrix 文档事实层已完成基线分类，旧文档不会被直接当作当前事实使用。
4. 低影响优先队列已确认。
5. 未开始生产代码改动。

### 阶段 0.5：低影响边界优化

目标：先完成影响面较小、但能为后续全量替换降低不确定性的基础优化。

建议任务：

1. 定义 validation report / inspection snapshot 的字段、错误分类和 JSON 输出草案，不改变运行时行为。
2. 将 DSL / endpoint / shared resource loader failure 统一收集为 report-only 结果，不立即打开 strict。
3. 为 function registry 增加 error-return contract 和内建函数注册测试，暂不迁移跨仓业务模块。
4. 为 endpoint catalog 定义 descriptor model，先让 HTTP / MCP endpoint 能产出 descriptor，再替换 host 注册路径。
5. 为 runtime reload / stop / destroy 定义 owner contract，先限制新接口暴露面，再评估是否需要并发保护实现。
6. 增加 generic infrastructure identity 检查清单，禁止新代码硬编码模块、shell、workspace 拓扑或产品身份。

验收：

1. 上述任务不改变公开 DSL schema、HTTP / MCP 协议语义或业务模块行为。
2. 每项任务都有 focused test 或文档审计命令。
3. 完成后再进入 Matrix core 生产路径替换。

### 阶段 1：Matrix 新接口契约

目标：在 Matrix 中定义新接口和包边界，作为后续所有仓库迁移的唯一目标。

建议改动文件：

1. `pkg/types/runtime.go`
2. `pkg/types/registry.go`
3. `pkg/types/node.go`
4. `pkg/types/factory.go`
5. `pkg/asset/**`
6. `pkg/runtimebridge/**`
7. 新增 `pkg/inspection/**` 或等价包

新接口分组：

1. 执行：
   - `RuleChainExecutor`
   - `RuleChainAsyncExecutor`
2. 运行时只读：
   - `RuntimeCatalog`
   - `RuntimeInspector`
   - `RuleChainInspector`
3. 资源：
   - `SharedResourceResolver`
   - `SharedResourceCatalog`
4. Endpoint：
   - `EndpointCatalog`
   - `HttpEndpointDescriptor`
   - `HttpEndpointHandler`
   - `McpToolEndpoint`
   - `ActiveEndpointController`
5. 函数：
   - `FunctionRegistry`
   - `FunctionCatalog`
   - `FunctionContractCatalog`
6. 构造：
   - `MessageFactory`
   - `NodeContextFactory`
7. Validation / inspection：
   - `ValidationReport`
   - `ValidationIssue`
   - `InspectionSnapshot`
   - `RuntimeFactDescriptor`

删除或替换目标：

1. 生产路径不再调用 `matrix.Registry`。
2. 生产路径不再调用 `types.DefaultRegistry`。
3. 生产 endpoint 不再实现 `SetRuntimePool(any)`。
4. 新节点上下文不再通过 `GetRuntime().GetEngine()` 暴露全量 engine。

验收：

1. Matrix 新接口通过单元测试覆盖。
2. 旧接口引用在 Matrix 生产代码中只剩尚未迁移阶段明确列出的点。
3. 文档更新 `docs/reference/03_architecture_overview.md`、`docs/reference/12_node_specification.md` 草案。

### 阶段 2：Matrix core 内部迁移

目标：Matrix 自身 runtime、registry、endpoint、function node、shared resource 实现迁移到新接口。

建议改动文件：

1. `matrix.go`
2. `internal/registry/**`
3. `internal/runtime/**`
4. `internal/builtin/base/functions_node.go`
5. `internal/builtin/nodes/endpoint/http_endpoint.go`
6. `internal/builtin/nodes/endpoint/mcp_endpoint.go`
7. `internal/builtin/nodes/pipeline/**`
8. `internal/builtin/nodes/loop/**`
9. `internal/builder/**`

关键任务：

1. Engine 初始化独立 registry，不再默认共享全局 registry。
2. Runtime 构造接收 catalog / resolver / factory 快照。
3. Function node 的执行、错误、contract、schema 统一使用 engine-scoped catalog。
4. Shared resource 解析走 resolver。
5. Endpoint 加载后进入 endpoint catalog。
6. HTTP endpoint 通过 typed executor binding 获取 rulechain executor。
7. MCP endpoint 不再实现 runtime pool 空方法。
8. Pipeline endpoint 和 active endpoint lifecycle 从 shared pool 中剥离。

验收：

1. `go test ./pkg/mcp ./internal/builtin/nodes/endpoint ./internal/registry ./internal/runtime`
2. Matrix 生产代码不再依赖旧 global registry 主路径。
3. `endpoint/http` 和 `endpoint/mcp` 都通过 endpoint catalog 被发现。

### 阶段 3：Matrix startup validation

目标：建立启动 validation pipeline，最终替代当前 warning 后继续和运行时才失败的模式。

建议改动文件：

1. `internal/builder/**`
2. `internal/runtime/runtime.go`
3. `pkg/types/runtime.go`
4. 新增 `pkg/validation/**` 或 `internal/validation/**`
5. `skills/matrix-rulechain-validator/**`
6. `docs/reference/18_dsl_specification.md`

validation 覆盖：

1. DAG / cycle / dangling edge。
2. duplicate node ID / duplicate shared resource ID。
3. unknown node type / function name / endpoint target。
4. function routing relation。
5. shared resource ref exists。
6. endpoint IO mapping and data contract。
7. MCP target kind / auth context / forbidden trusted fields。
8. loader parse failure / missing target / optional fallback semantics。
9. validation report schema and inspection snapshot schema。

验收：

1. validation report 模式可输出结构化错误集合。
2. strict 模式能拒绝非法 DSL。
3. 当前历史 DSL 的失败点有迁移清单。
4. Morpheus、CI、rulechain validator 和文档审计工具能消费同一类 validation / inspection 输出。

### 阶段 4：WhiteRoom common 与 scaffold 模板

目标：让 WhiteRoom 作为新模块模板源先迁移，避免后续生成旧接口。

建议改动文件：

1. `platform/WhiteRoom/common/server/server.go`
2. `platform/WhiteRoom/common/server/runtime_helpers.go`
3. `platform/WhiteRoom/common/executors/data/**`
4. `platform/WhiteRoom/common/executors/observability/**`
5. `platform/WhiteRoom/common/executors/ai/**`
6. `platform/WhiteRoom/cmd/module-scaffold/templates/business/basic/project/cmd/server/main.go.tpl`
7. `platform/WhiteRoom/cmd/module-scaffold/templates/business/basic/project/internal/mcpservercmd/**`
8. `platform/WhiteRoom/cmd/module-scaffold/templates/business/basic/executors/**`
9. `platform/WhiteRoom/docs/reference/**`
10. `platform/WhiteRoom/skills/**`，如果操作协议变化

关键任务：

1. `RegisterDynamicEndpoints` 改为消费 Matrix endpoint catalog。
2. `CollectDynamicEndpointPatterns` 改为读取 HTTP endpoint descriptors。
3. common executors 改为通过 resolver 获取 shared resources。
4. common function registration 改为 error-return registry API。
5. scaffold 模板生成新接口代码。
6. common sync domain manifest 更新版本和 capability 说明。

验收：

1. `go test ./common/... ./cmd/module-scaffold/...`
2. scaffold 生成的新模块不包含 `SetRuntimePool(any)`、`matrix.Registry.GetSharedNodePool()`、`eng.SharedNodePool()` 注册 endpoint。
3. WhiteRoom 文档更新。

### 阶段 5：业务模块同步

目标：同步所有业务模块到新 WhiteRoom common 和 Matrix 接口，不保留旧 common 复制件。

模块顺序：

1. `modules/usagex`
2. `modules/identityx`
3. `modules/lens`
4. `modules/notifyx`
5. `modules/paymentx`
6. `modules/sellitx`

顺序理由：

1. `usagex` 和 `identityx` 是 MCP / admin / platform 当前重点路径，先修可以尽早发现接口问题。
2. `sellitx` 引用量最大，放在中后段，避免在核心接口未稳定时承受最大迁移成本。

每个模块固定步骤：

1. 同步 `common/server`。
2. 同步 `common/executors`。
3. 修改 `cmd/server/main.go` 中 Matrix engine / endpoint registration 调用。
4. 修改 `code/orchestrator/bootstrap` 中 `SetupContext.Engine` 类型。
5. 修改 module-local `code/executors/resources` shared nodes。
6. 修改 module-local `code/executors/funcs` function registration。
7. 修改 tests 中 mock engine / mock runtime / mock node ctx。
8. 跑模块 focused tests。

IdentityX 额外步骤：

1. 修改 `modules/identityx/internal/mcpservercmd/runtime.go`。
2. `identityRuleChainRunner` 改为依赖 `RuleChainExecutor`。
3. MCP target dispatcher 不持有 full `MatrixEngine`。

验收：

1. 每个模块 `go test ./...` 或 focused test 通过。
2. 每个模块不再出现旧 endpoint registration 生产路径。
3. 每个模块不再通过 global registry 解析 shared resource。

### 阶段 6：Morpheus inspection API 迁移

目标：Morpheus 不再直接消费 Matrix runtime/shared pool internals，而是通过 Matrix inspection API 读取图谱、节点、endpoint、function、trace 事实。

建议改动文件：

1. `platform/Morpheus/pkg/handlers/handlers_trigger.go`
2. `platform/Morpheus/pkg/handlers/handlers_function.go`
3. `platform/Morpheus/pkg/handlers/execute_node.go`
4. `platform/Morpheus/pkg/handlers/services.go`
5. `platform/Morpheus/pkg/graph/dependency/**`
6. `platform/Morpheus/pkg/graph/flow/**`
7. `platform/Morpheus/pkg/graph/layout/**`
8. `platform/Morpheus/pkg/graph/ops/**`
9. `platform/Morpheus/pkg/graph/testutils/**`

关键任务：

1. 新增 Morpheus-side adapter，将 Matrix inspection API 转为 graph package 所需模型。
2. `BuildDependencyGraph` 不再读取 `engine.SharedNodePool().GetAll()`。
3. `BuildFlowGraph` 不再直接依赖 `types.MatrixEngine.RuntimePool()`。
4. function handlers 使用 `FunctionCatalog`。
5. single-node execution 使用 `RuleChainExecutor` / `NodeExecutor` 专用接口。
6. graph tests 更新 mock 类型。

验收：

1. `go test ./pkg/graph/... ./pkg/handlers/...`
2. Morpheus graph explorer 能列出 rulechains、endpoints、functions。
3. 不引入业务模块或 WhiteRoom 对 Morpheus private package 的反向依赖。

### 阶段 7：删除旧入口与全量 strict

目标：不兼容收口，删除旧接口生产路径，打开 strict validation。

删除目标：

1. `types.Endpoint.SetRuntimePool(any)`。
2. 生产路径 `matrix.Registry`。
3. 生产路径 `types.DefaultRegistry`。
4. `ctx.GetRuntime().GetEngine()` 作为 shared resource / function catalog 查询入口。
5. shared node pool 中 endpoint list 的生产使用。
6. function registration panic-only API。

保留限制：

1. 测试 helper 可以保留 mock factory。
2. 文档可以提及旧接口作为迁移历史。
3. 如果需要 legacy symbol 暂存，必须标记为 internal test-only 或 deprecated compile-time failing stub，不允许生产代码调用。

验收：

1. `rg "matrix\\.Registry|types\\.DefaultRegistry|SetRuntimePool\\(|GetRuntime\\(\\)\\.GetEngine\\(|SharedNodePool\\(\\).*endpoint" platform/Matrix platform/WhiteRoom platform/Morpheus modules` 不命中生产路径。
2. Matrix strict validation 对新 DSL path 生效。
3. 历史 DSL 迁移问题已修复或在 plan 中关闭为可解释例外。

### 阶段 8：全仓验证与文档闭环

目标：确认可以合并主线。

验证命令：

```bash
git -C platform/Matrix diff --check
git -C platform/Matrix status --short --branch
go test ./...
```

对每个 Go 仓库执行：

```bash
go test ./...
```

重点 smoke：

1. Matrix HTTP endpoint dispatch。
2. Matrix MCP endpoint `tools/list` / `tools/call`。
3. WhiteRoom scaffold 生成模块。
4. IdentityX `cmd/server` startup smoke。
5. IdentityX `cmd/mcp-server` stdio / streamable-http smoke。
6. Morpheus graph explorer endpoint smoke。

文档闭环：

1. Matrix RFC 0015 status 进入 `Accepted` 或保持 `Draft` 并说明未进入实施。
2. Matrix Plan 0015-1 status 进入 `Implementing` 或 `Stable`。
3. 增加 ADR 记录：
   - core interface split
   - global registry removal
   - endpoint catalog ownership
   - Morpheus inspection API boundary
4. 更新 Reference：
   - architecture overview
   - node specification
   - function registration spec
   - shared resource management
   - DSL specification
   - MCP endpoint adapter
   - validation / inspection schema
   - runtime lifecycle ownership
5. 更新 Guide：
   - node development
   - shared node
   - endpoint development
   - WhiteRoom module development
6. `docs/reference/00_decision_traceability.md` 记录 RFC、Plan、ADR、Reference、Guide 和验证证据的闭环状态。
7. Stage 0 标记为事实漂移的文档项已更新、废弃或明确作为历史输入保留。

## 6. 风险评估

| 风险 | 可能性 | 影响 | 缓解 |
| --- | --- | --- | --- |
| Morpheus graph packages 依赖 Matrix internals 太深 | 高 | 高 | 单独阶段迁移到 inspection API；先建立 adapter 和 test fixtures。 |
| 业务模块 common 复制件漂移 | 高 | 高 | 先改 WhiteRoom common/template，再逐模块同步，禁止模块保留旧 common/server。 |
| strict validation 暴露大量历史 DSL 问题 | 中 | 高 | validation 先 report-only，阶段 7 前集中修复或记录明确例外。 |
| 删除 global registry 导致 init 顺序问题 | 高 | 中 | function / node / core object registration 改为 explicit registry builder，测试覆盖启动顺序。 |
| NodeCtx 拆分影响所有函数节点 | 高 | 高 | 先保留 `NodeFunc(ctx, msg)` 形态，但让 ctx 内部只暴露新 resolver；最后再判断是否改函数签名。 |
| WhiteRoom scaffold 继续生成旧接口 | 中 | 高 | 模板和 common sync 在业务模块迁移前完成。 |
| MCP host 和 HTTP host 对 executor 语义不一致 | 中 | 中 | 使用同一个 `RuleChainExecutor` 和 target dispatcher contract。 |
| 全仓测试 baseline 本身不干净 | 高 | 中 | 阶段 0 固化 baseline，最终只关闭本次引入的失败。 |
| 文档事实层不可信 | 高 | 中 | Stage 0 先做事实基线分类，Stage 8 关闭事实漂移和决策追踪。 |
| validation / inspection schema 不稳定 | 中 | 高 | Stage 0.5 先定义输出模型，再让 Morpheus、CI、validator 消费。 |
| reload / lifecycle owner 不清晰 | 中 | 中 | 先收敛到 `RuntimeLifecycle` owner，再迁移 endpoint/shared resource lifecycle。 |
| generic infrastructure 出现身份硬编码 | 中 | 高 | 在 WhiteRoom、Morpheus、业务模块迁移前建立 config / manifest / inspection metadata 边界检查。 |

## 7. 提交与合并策略

建议按阶段提交，不把所有改动压成一个提交：

1. `docs: plan Matrix core interface refactor`
2. `refactor: split Matrix core interfaces`
3. `refactor: migrate Matrix runtime catalogs`
4. `refactor: introduce Matrix endpoint catalog`
5. `refactor: migrate WhiteRoom Matrix boundaries`
6. `refactor: sync modules to Matrix boundary APIs`
7. `refactor: migrate Morpheus Matrix inspection APIs`
8. `test: close Matrix boundary validation coverage`
9. `docs: reconcile Matrix boundary references`

如果中途必须跨仓提交，保持每个仓库提交可独立说明，PR 描述用中文并记录验证证据。

## 8. 执行门禁

本计划是 Draft。开始生产代码改动前必须获得明确人工批准，例如：

```text
开始实施
```

或：

```text
按这个计划执行
```

如果实施中改变以下任一边界，必须更新本计划并重新等待批准：

1. 删除或改变公开 DSL schema。
2. 改变 HTTP / MCP 对外协议语义。
3. 改变认证、权限、quota、billing 或 trust boundary。
4. 改变 WhiteRoom scaffold 输出结构。
5. 改变 Morpheus 与 Matrix 的依赖方向。
6. Stage 0 文档事实基线发现 RFC / Plan / Reference / Guide 与当前实现存在重大冲突。

## 9. Stage 0 Baseline Snapshot

记录日期：2026-06-08。

本节记录 Stage 0 的当前基线。它只用于后续重构验收对照，不代表这些失败、脏状态或历史文档问题由本计划引入。

### 9.1 仓库状态

| 仓库 | 当前分支 | dirty / ahead 状态 | Stage 0 处理 |
| --- | --- | --- | --- |
| `platform/Matrix` | `codex/refactor/matrix-core-interface-boundary` | 有本 RFC / Plan / traceability 文档变更 | 本计划文档工作区 |
| `platform/WhiteRoom` | `main` | clean | 后续 Stage 4 迁移 |
| `platform/Morpheus` | `main` | clean | 后续 Stage 6 迁移 |
| `modules/identityx` | `main` | clean | 后续 Stage 5 迁移 |
| `modules/lens` | `main` | clean | 后续 Stage 5 迁移 |
| `modules/notifyx` | `main` | clean | 后续 Stage 5 迁移 |
| `modules/paymentx` | `main` | ahead 1 | 记录为既有状态，Stage 5 前需确认是否同步远端 |
| `modules/sellitx` | `main` | 有大量 DSL、function、common、docs 未提交改动 | 记录为既有状态，Stage 5 前必须单独确认归属和合并策略 |
| `modules/usagex` | `main` | clean | 后续 Stage 5 迁移 |

### 9.2 测试 baseline

| 仓库 | 命令 | 结果 | 观察 |
| --- | --- | --- | --- |
| `platform/Matrix` | `go test ./...` | 失败 | 失败包包括 `internal/registry`、`internal/runtime`、`pkg/message`、`pkg/utils`、`test/e2e_test/alert`。已观察到 `GoodSidV1_0` SID warning 断言失败、mock 调用次数失败、Decode 错误文案期望不一致、alert E2E nil pointer / HTTP 500。 |
| `platform/WhiteRoom` | `go test ./...` | 通过 | 当前 common / server / scaffold 相关测试可作为 Stage 4 前 baseline。 |
| `platform/Morpheus` | `go test ./...` | 失败 | `builtin/executors/funcs/rpa` build/vet 失败：`fmt.Errorf` 使用 non-constant format string。 |
| `modules/identityx` | `go test ./...` | 通过 | 可作为 Stage 5 的重点模块 baseline。 |
| `modules/lens` | `go test ./...` | 通过 | 可作为 Stage 5 baseline。 |
| `modules/notifyx` | `go test ./...` | 通过 | 可作为 Stage 5 baseline。 |
| `modules/paymentx` | `go test ./...` | 通过 | 分支 ahead 1，后续迁移前需确认本地提交归属。 |
| `modules/sellitx` | `go test ./...` | 通过 | 工作区 dirty，后续迁移前需确认未提交改动归属。 |
| `modules/usagex` | `go test ./...` | 失败 | `code/orchestrator/attachmentparser` 的 `TestParseAsyncAndJobEndpointFlow` 失败，现象为 `job did not complete`。 |

### 9.3 文档事实层 baseline

执行：

```bash
python3 platform/Matrix/skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py --root platform/Matrix/docs --scope . --strict-targets
```

结果：扫描 80 个文档，26 个 error，39 个 warning。

主要历史问题：

1. `docs/designs/rfc/0001_*` 到 `docs/designs/rfc/0013_*` 中多份历史 RFC 缺少 `## 原始需求点总结`。
2. 多份历史 RFC / ADR / Plan / Guide 使用 legacy relation type，例如 `is_formalized_by`、`is_explained_by`、`is_guided_by`、`documents`。
3. `docs/reference/18_dsl_specification.md` 存在未知 relation type：`defines_schema_for`、`is_related_to`。
4. 多份 Reference 文档的 frontmatter `type` 不是 `Reference`，例如 `03_architecture_overview.md`、`08_node_development_patterns.md`、`09_core_objects.md`、`11_function_registration_spec.md`、`12_node_specification.md`、`15_shared_resource_management.md`、`18_dsl_specification.md`、`21_component_catalog.md`、`33_component_design_principles.md`、`36_testing_strategy.md`。
5. `docs/guides/00_matrix_guide.md` 的 `type` 不是 `Guide`，多份 Guide 文件名不符合 auditor 当前规则。

事实层分类：

| 文档 | Stage 0 分类 | 说明 |
| --- | --- | --- |
| `docs/designs/adr/0000-1_dag-vs-cyclic-graph_adr.md` | 架构决策有效，运行时事实未闭环 | DAG 是正式决策，但当前 runtime validation 不强制 DAG。 |
| `docs/designs/adr/0007-1_matrix_refactor_design_adr.md` | 历史设计输入 | 文档仍包含 `SharedNodePool`、`registry.Default` 等旧迁移口径，不应作为新目标架构事实。 |
| `docs/reference/03_architecture_overview.md` | 部分当前事实，需重写 | 仍可说明基本概念，但 frontmatter 类型不合规，DAG / shared pool 事实需要与新 validation / endpoint catalog 方案对齐。 |
| `docs/reference/11_function_registration_spec.md` | 当前 legacy 行为事实，需升级 | 当前仍描述 `registry.Default.NodeFuncManager.Register(...)`，Stage 0.5/Stage 1 后需要补 engine-scoped catalog 和 error-return registry。 |
| `docs/reference/12_node_specification.md` | 部分当前事实，需升级 | 可作为节点生命周期历史事实，但 NodeCtx 最小能力和 runtime 穿透边界需要重写。 |
| `docs/reference/15_shared_resource_management.md` | 当前 legacy 行为事实，需升级 | 可说明当前 `SharedNodePool` / `ref://` 实现；Stage 2/4 后应迁移到 resolver / catalog 事实。 |
| `docs/reference/18_dsl_specification.md` | DSL 目标事实，结构治理不合格 | 声明 `connections` 必须是 DAG，但当前 runtime 未统一强制；relation type 也需治理。 |
| `docs/reference/37_mcp_endpoint_adapter.md` | 当前 MCP 边界事实 | 可作为 MCP adapter 当前事实输入，后续需补 inspection / endpoint catalog 边界。 |

### 9.4 当前问题 inventory

代码搜索确认以下旧路径仍存在，后续阶段必须逐步删除或收敛：

| 问题 | 代表性证据 | 后续阶段 |
| --- | --- | --- |
| Matrix 仍暴露 global registry / shared pool | `matrix.go` 中 `Registry = types.DefaultRegistry`、`MatrixEngine.SharedNodePool()`、fallback 到 `registry.Default` | Stage 1 / Stage 2 |
| endpoint 仍通过 `SetRuntimePool(any)` 绑定 runtime | `pkg/types/node.go`、`internal/builder/builder.go`、`endpoint/http`、`endpoint/mcp`、`pipeline_endpoint` | Stage 1 / Stage 2 |
| function execution / contract 仍会回到默认 registry | `internal/builtin/base/functions_node.go` 多处读取 `types.DefaultRegistry.GetNodeFuncManager()` | Stage 0.5 / Stage 2 |
| NodeCtx / asset 可穿透 runtime 和 engine | `pkg/asset/config_asset.go`、`action/flow_node.go`、pipeline push 相关代码 | Stage 1 / Stage 2 |
| reload / lifecycle owner 仍在宽 runtime 接口上 | `pkg/types/runtime.go`、`internal/runtime/runtime.go` | Stage 0.5 / Stage 2 |
| WhiteRoom common 仍消费 `eng.SharedNodePool()` 和 `matrix.Registry.GetSharedNodePool()` | `platform/WhiteRoom/common/server`、`platform/WhiteRoom/common/executors/**` | Stage 4 |
| 业务模块 common 复制件仍消费旧 shared pool / registry | `modules/*/common/server`、`modules/*/common/executors/**` | Stage 5 |
| Morpheus 仍直接消费 Matrix internals | `platform/Morpheus/pkg/handlers/**`、`platform/Morpheus/pkg/graph/**` 使用 `MatrixEngine`、`RuntimePool()`、`SharedNodePool()`、`NodeFuncManager()` | Stage 6 |
| generic infrastructure 身份边界需要检查 | 当前 Stage 0 只完成初步确认，后续需专门搜索模块名、shell 名、workspace 拓扑硬编码 | Stage 0.5 / Stage 4 / Stage 6 |

### 9.5 低影响优先队列

Stage 0 后建议按以下顺序进入 Stage 0.5：

1. 定义 `ValidationReport` / `ValidationIssue` / `InspectionSnapshot` / `RuntimeFactDescriptor` 草案和文档。
2. 把 loader parse failure / missing target / optional fallback 收敛为 report-only validation 输出。
3. 为 function registry 增加 error-return contract 和 focused tests，暂不迁移所有模块注册点。
4. 建立 endpoint descriptor catalog model，先产出 descriptor，再替换 host 注册路径。
5. 收敛 runtime reload / stop / destroy owner contract。
6. 增加 generic infrastructure identity hard-code 检查清单。

### 9.6 Stage 0 结论

Stage 0 的结论是：当前可以继续推进，但不能以全仓 `go test ./...` 绿色作为进入 Stage 0.5 的前置条件，因为 Matrix、Morpheus、usagex 已有 baseline 失败。后续验收应区分：

1. 本次重构引入的新失败必须关闭。
2. Stage 0 已记录的 baseline 失败可以作为历史风险保留，但最终合并主线前需要修复或形成明确例外。
3. `modules/paymentx` ahead 1 与 `modules/sellitx` dirty state 必须在 Stage 5 前重新确认，避免重构覆盖人工或其他任务改动。

### 9.7 Stage 0.5 Slice 1：Validation / Inspection Schema

记录日期：2026-06-08。

本切片完成 Stage 0.5 的第一项低影响优化：先新增稳定输出模型，不改变 runtime、loader、endpoint 或跨仓生产路径。

新增实现：

1. `pkg/validation`
   - `Report`
   - `Issue`
   - `Target`
   - `Scope`
   - `ModeReportOnly` / `ModeStrict`
   - severity 与 issue code 常量
2. `pkg/inspection`
   - `InspectionSnapshot`
   - `RuntimeFactDescriptor`
   - runtime / rulechain / endpoint / function / shared resource fact kind 常量
3. `docs/reference/38_validation_inspection_schema.md`
   - 记录 JSON 输出字段、当前用途和限制。

验证：

```bash
go test ./pkg/validation ./pkg/inspection ./pkg/runtimebridge
```

当前限制：

1. 本切片只定义 schema，不执行 DAG、loader、endpoint、shared ref 或 function relation 校验。
2. loader report-only 接入留到 Stage 0.5 后续切片。
3. Morpheus 迁移到 inspection API 留到 Stage 6。
