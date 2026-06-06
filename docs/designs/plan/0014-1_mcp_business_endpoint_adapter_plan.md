---
uuid: "edda57eb-b3a4-42ab-be66-8f7c7a97d717"
type: "Plan"
title: "计划：MCP 业务入口适配器实施方案"
status: "Draft"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "mcp"
  - "endpoint"
  - "implementation-plan"
relations:
  - type: "is_plan_for"
    target_uuid: "18d7026c-61eb-4b39-b537-7d927df34921"
    description: "本计划用于落地 MCP 业务入口适配器 RFC。"
  - type: "is_part_of"
    target_uuid: "fde86d18-74ce-4350-9137-553929ee1261"
    description: "本文档属于 Matrix Plan 文档库。"
---

# Plan：MCP 业务入口适配器实施方案

## 1. 计划概述

本计划用于将 MCP adapter 设计落地为 Matrix / WhiteRoom 可复用入口能力。2026-06-04 已收到明确实施指令：`按照规范和流程，开始实现这个MCP的需求`。后续如扩大 tool 范围、改变身份上下文、改变 host 形态或影响业务认证边界，必须另行更新计划并等待批准。

## 2. 当前事实

| 领域 | 当前事实 |
| --- | --- |
| Endpoint 抽象 | `types.Endpoint` 是共享节点，支持 `SetRuntimePool`。 |
| HTTP endpoint | `types.HttpEndpoint` 已定义 `HandleHttpRequest`、path、method、input/output mapping 和 target chain。 |
| Host 注册 | WhiteRoom / module common server 扫描 shared node pool，注册 `endpoint/http` 到 `net/http` mux。 |
| Active endpoint | `types.ActiveEndpoint` 可在 server 启动时 `Start`，适合非 HTTP 主动监听或后台入口。 |
| MCP | stdio 和 Streamable HTTP transport 语义不同；transport host 由 WhiteRoom `cmd/mcp-server` 承载，工具 discovery / call 语义共用 Matrix adapter core。 |
| MVP 决策 | 身份上下文暂时使用 `dev_static_context`；WhiteRoom `cmd/mcp-server` 暂时只支持独立进程形态。 |
| 当前实施事实 | Matrix 已实现 `pkg/types.McpEndpoint*`、`pkg/mcp` transport-neutral adapter / JSON-RPC handler / HTTP handler / stdio loop，以及 `internal/builtin/nodes/endpoint/mcp_endpoint.go`。 |

## 3. 实施切片

### M1：接口与配置草案

工作：

1. 定义 `McpEndpoint` 或等价接口草案。
2. 定义 `McpEndpointNodeConfiguration` 字段。
3. 明确 tool catalog schema、target、authContext、riskLevel 和 outputPolicy。
4. 明确 `endpoint/mcp` 与 `endpoint/http` 的注册差异。

验收：

1. 设计文档记录接口草案。
2. 不把 MCP 配置塞入 `HttpEndpointNodeConfiguration`。
3. tool catalog 不允许自动暴露全部 HTTP endpoints。

### M2：Adapter core 原型

工作：

1. 实现 transport 无关 adapter core。
2. 支持 `ListTools` 和 `CallTool`。
3. 支持按 tool catalog 分发 `http_api`、`external_http` 和 `rulechain` target。
4. 输出错误归一和 secret sanitizer。

验收：

1. 单元测试覆盖 tool catalog validation。
2. 单元测试覆盖 unknown tool、invalid args、secret redaction。
3. 不依赖 IdentityX / UsageX 私有包。
4. 不把 TuriX、IdentityX、UsageX 具体名称写入 Matrix 通用 adapter。

### M3：WhiteRoom Streamable HTTP host contract

工作：

1. 定义 Matrix adapter core 暴露给 WhiteRoom 独立进程 `cmd/mcp-server` 的 HTTP handler contract。
2. WhiteRoom `cmd/mcp-server` 以独立进程内的 Gin 或 `net/http` listener 挂载该 handler。
3. 支持 MCP `tools/list` 和 `tools/call` 基础请求。

验收：

1. HTTP handler 可通过本地 route smoke。
2. request 日志不污染 MCP response。
3. 错误输出符合 MCP tool result 语义。

### M4：WhiteRoom stdio host contract

工作：

1. 定义 Matrix adapter core 暴露给 WhiteRoom 独立进程 `cmd/mcp-server` 的 stdio loop contract。
2. stdout 只输出 MCP JSON-RPC。
3. stderr 承载日志。
4. 复用 M2 adapter core。

验收：

1. TuriX MCP Runtime 可启动 stdio server。
2. approval 后能 discovery tools。
3. `tools/call` 可调用只读工具。

### M5：模块接入试点

工作：

1. IdentityX 只选择 `identityx_get_me_access` 作为 MVP 只读工具。
2. UsageX 不进入 MVP。
3. MVP 身份上下文使用 `dev_static_context`，只允许本地 smoke。
4. 按 `identityx_get_me_access` 的既有主路径选择 target；若该能力以 HTTP handler / middleware 为信任边界，则使用 `http_api`。
5. 后续工具若原能力由 rulechain 承载，则直接使用 `rulechain` target，并补齐 endpoint IO mapping 设计。

验收：

1. `identityx_get_me_access` 能追溯现有 API 或 rulechain。
2. 不新增第二套业务实现。
3. 不暴露 mutation 或 billing-sensitive 能力。

## 4. 验证命令

文档阶段：

```bash
python3 platform/Matrix/skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py --root platform/Matrix/docs --scope designs/rfc
python3 platform/Matrix/skills/matrix-doc-graph-auditor/scripts/audit_matrix_docs.py --root platform/Matrix/docs --scope designs/plan
git diff --check
```

实现阶段按 touched area 追加：

```bash
go test ./pkg/mcp ./internal/builtin/nodes/endpoint
```

如果涉及业务模块 DSL，再运行对应模块的 Matrix rulechain validator。

## 5. 风险

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| `endpoint/mcp` 与 `endpoint/http` 混用 | 中 | 独立配置类型和接口 |
| 业务模块名进入 Matrix core | 高 | core 只定义协议抽象 |
| tool schema 暴露身份字段 | 高 | catalog validator 拒绝禁止字段 |
| target.kind 选择绕开既有主路径 | 高 | catalog 必须说明原能力主路径；Orchestrator 固定接口走 HTTP，rulechain 主路径走 rulechain |
| stdio stdout 被日志污染 | 中 | stdout 只写 JSON-RPC，日志走 stderr |
| MCP transport host 误入 Matrix | 中 | Matrix 文档只定义 adapter core 和 host contract，WhiteRoom 实现 `cmd/mcp-server` |

## 6. 下一阶段门槛

下一阶段继续扩大范围前必须确认：

1. 是否新增 `delegation_token` 或 `identityx_session`。
2. 是否启用 rulechain target 的 runtime-bound 直接执行。
3. 是否允许 WhiteRoom `cmd/mcp-server` 同进程嵌入模块 HTTP server。
4. 是否新增 UsageX 或 mutation / billing-sensitive 工具。
