# Matrix

Matrix 是一个用 Go 实现的 DSL-first 业务流程编排引擎。它负责加载 JSON
规则链 DSL、实例化节点、让 `RuleMsg` 在有向无环图中流转，并把 HTTP、MCP、
调度器、主动 endpoint 等入口交给宿主应用承载。

Matrix 的定位是可嵌入的编排内核，不是一个独立产品服务。产品模块和平台宿主
负责鉴权、传输协议生命周期、部署和用户可见 API；Matrix 负责规则链执行、
节点契约、共享资源、endpoint descriptor、validation 和 inspection 基础能力。

## README 一般应该写什么

一个有用的仓库 README 通常要回答这些问题：

1. 这个项目是什么，什么时候应该使用它。
2. 如何最快跑起或验证一个最小可用路径。
3. 日常开发有哪些安全命令。
4. 仓库目录如何组织。
5. 核心架构边界是什么。
6. RFC、ADR、Plan、Reference、Guide 等长期文档在哪里。
7. 当前哪些能力仍是 legacy、未完成或迁移中。

本文档只做仓库入口导航。长期架构事实、阶段计划和决策记录以 `docs/` 下的正式
文档为准。

## 快速开始

Matrix 是一个 Go module：

```bash
go list ./...
```

当前 interface-boundary 重构阶段常用的 focused validation：

```bash
go test -count=1 ./pkg/validation ./pkg/types ./pkg/inspection ./cmd/matrix-validate
go test -run '^$' ./internal/builtin/base ./internal/runtime ./pkg/helper ./pkg/validation ./cmd/matrix-validate
```

不启动业务模块，仅静态扫描模块 DSL：

```bash
go run ./cmd/matrix-validate --module-root ../../modules/identityx
go run ./cmd/matrix-validate --module-root ../../modules/sellitx
```

`matrix-validate` 是 report-only 工具。它读取 DSL 文件并输出结构化 JSON，
不会实例化 node、启动 endpoint 或改变 runtime 状态。

## 常用命令

| 命令 | 用途 |
| --- | --- |
| `go test -count=1 ./pkg/validation ./pkg/types ./pkg/inspection ./cmd/matrix-validate` | 验证 validation / inspection 契约。 |
| `go test -run '^$' ./internal/builtin/base ./internal/runtime ./pkg/helper ./pkg/validation ./cmd/matrix-validate` | 对边界重构触达包做编译检查。 |
| `go run ./cmd/matrix-validate --module-root <module-root>` | 对业务模块输出 report-only `ValidationReport`。 |
| `python3 skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py --root docs --scope reference --strict-targets` | 审计 Matrix Reference 文档。 |
| `python3 skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py --root docs --scope designs/plan --strict-targets` | 审计 Matrix Plan 文档。 |
| `git diff --check` | 提交前检查 whitespace。 |

只有在明确要检查全仓 baseline 时才运行 `go test ./...`。当前重构计划已经记录
历史 baseline 失败项，全仓结果需要结合计划文档判断，不能直接等同于本次改动回归。

## 仓库目录

| 路径 | 职责 |
| --- | --- |
| `matrix.go` | Matrix engine public facade 与高层装配入口。 |
| `cmd/matrix-validate` | report-only DSL validation CLI。 |
| `internal/builder` | rulechain、endpoint、shared node 的内部加载与构建逻辑。 |
| `internal/runtime` | rulechain runtime 执行实现。 |
| `internal/builtin` | Matrix 内置节点实现与注册。 |
| `pkg/types` | 对外契约：node、runtime、registry、data、lifecycle、DSL model。 |
| `pkg/validation` | validation report、endpoint catalog、loader scanner 与静态契约检查。 |
| `pkg/inspection` | runtime / validation fact snapshot model。 |
| `pkg/asset` | `rulemsg://`、`ref://`、config 等 asset URI 处理。 |
| `pkg/mcp` | MCP endpoint adapter 与 target dispatch contract。 |
| `docs/` | RFC、ADR、Plan、Reference、Guide、migration、template 等正式文档。 |
| `skills/` | Matrix 专用 Codex skills，用于 DSL、函数节点、rulechain 和文档审计。 |

## 核心概念

| 概念 | 含义 |
| --- | --- |
| Rulechain | JSON DSL 定义的有向无环图，由多个 node instance 组成。 |
| Node | 具备明确类型和契约的执行单元，承载业务或基础设施能力。 |
| RuleMsg | 规则链中的消息载体，包含原始数据、强类型 `DataT` 和 metadata。 |
| Shared resource | 通过 `ref://...` 显式引用的可复用能力。 |

业务模块默认使用四层模型：

1. `orchestrator`：宿主生命周期、鉴权、callback、scheduler、websocket 或传输入口。
2. `workflow`：DSL endpoint 和 rulechain，决定做什么以及按什么顺序做。
3. `capability`：函数节点、service、repository、provider 和 shared resource。
4. `common`：只有出现真实第二模块复用后才提升的公共能力。

业务编排应保持 DSL-first。复杂协议细节、事务、宿主生命周期和性能敏感实现应留在
代码里，不应过度 DSL 化。

## Validation 与 Inspection

当前 report-only validation 覆盖：

| 范围 | issue 示例 |
| --- | --- |
| Loader 输入 | `loader_failure` |
| Graph / DAG | `duplicate_node_id`、`dangling_connection`、`cycle_detected` |
| Endpoint target | `missing_endpoint_target` |
| Shared resource | `missing_shared_ref`、`optional_fallback` |
| 可选 catalog | `unknown_node_type`、`unknown_function`、`invalid_function_relation` |
| HTTP endpoint IO | `invalid_endpoint_io` |

`ValidationReport` 可以携带 `EndpointCatalog`，静态描述 HTTP、MCP、pipeline
endpoint。这个 catalog 是后续 host registration 重构的显式发现模型；当前 host
endpoint 注册路径尚未完全迁移到 catalog 消费模式。

当前 schema 事实见
[docs/reference/38_validation_inspection_schema.md](docs/reference/38_validation_inspection_schema.md)。

## 文档入口

| 文档 | 用途 |
| --- | --- |
| [docs/README.md](docs/README.md) | Matrix 文档系统总览。 |
| [docs/reference/00_decision_traceability.md](docs/reference/00_decision_traceability.md) | RFC / Plan / Reference 决策追踪索引。 |
| [docs/reference/03_architecture_overview.md](docs/reference/03_architecture_overview.md) | 架构概览和核心概念。 |
| [docs/reference/11_function_registration_spec.md](docs/reference/11_function_registration_spec.md) | 函数节点注册和 routing contract。 |
| [docs/reference/15_shared_resource_management.md](docs/reference/15_shared_resource_management.md) | shared resource 与 `ref://` 行为。 |
| [docs/reference/37_mcp_endpoint_adapter.md](docs/reference/37_mcp_endpoint_adapter.md) | MCP endpoint adapter 当前事实。 |
| [docs/reference/38_validation_inspection_schema.md](docs/reference/38_validation_inspection_schema.md) | validation、inspection、endpoint catalog 与 lifecycle contract 当前事实。 |
| [docs/designs/rfc/0015_matrix_core_interface_boundary_refactor_rfc.md](docs/designs/rfc/0015_matrix_core_interface_boundary_refactor_rfc.md) | core interface boundary refactor 需求和意图。 |
| [docs/designs/plan/0015-1_matrix_core_interface_boundary_refactor_plan.md](docs/designs/plan/0015-1_matrix_core_interface_boundary_refactor_plan.md) | 当前分阶段重构计划和验证证据。 |

## 当前重构状态

当前架构重构仍在分阶段推进：

1. Stage 0.5 已新增 report-only validation、endpoint catalog descriptor、
   lifecycle owner contract，并完成部分文档治理。
2. strict validation 已有 helper contract，但尚未接入 startup。
3. endpoint catalog descriptor 已存在，但 host endpoint registration 仍有 legacy path。
4. global registry / shared pool fallback 仍有历史路径，后续阶段会继续收敛。
5. generic infrastructure identity hard-code 已完成审计和记录，跨仓实际修复属于后续阶段。

当实现改变稳定行为时，需要在同一变更中更新相关 Reference 或 Guide，并保持
`docs/reference/00_decision_traceability.md` 对齐。

## 交付前检查

提交或交付 Matrix 改动前：

1. 运行 `git status --short --branch`，确认没有混入无关本地改动。
2. 运行 `git diff --check`。
3. 针对触达包运行 focused Go validation。
4. 如果改动 `docs/` 下正式文档，运行 Matrix doc graph auditor。
5. 保持提交只覆盖一个清晰的行为、阶段或文档变更。
6. 不提交本地 runtime 文件、生成草稿、IDE 状态、日志或无关模块改动。
