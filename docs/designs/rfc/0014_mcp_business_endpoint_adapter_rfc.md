---
uuid: "18d7026c-61eb-4b39-b537-7d927df34921"
type: "RFC"
title: "需求：MCP 业务入口适配器"
status: "Draft"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "mcp"
  - "endpoint"
  - "business-capability"
relations:
  - type: "is_part_of"
    target_uuid: "68b1646b-2238-48e0-b77d-c81ecfc4317d"
    description: "本文档属于 Matrix RFC 文档库。"
  - type: "is_refined_by"
    target_uuid: "edda57eb-b3a4-42ab-be66-8f7c7a97d717"
    description: "本 RFC 的阶段化实现计划由 Plan 文档承接。"
---

# RFC：MCP 业务入口适配器

## 原始需求点总结

1. 需要把已经通过 Matrix / WhiteRoom 实现的业务能力暴露为 MCP tools，而不是只暴露 Matrix 引擎资产搜索能力。
2. MCP adapter 应像 `endpoint/http` 一样成为可被宿主加载的入口节点，并能由 Gin、`net/http` 或其他 Go host 注册。
3. stdio MCP 不是 HTTP route，MVP 暂时只支持 WhiteRoom `cmd/mcp-server` 独立进程 host；但它应复用与 HTTP MCP 相同的 adapter core。
4. MCP tool 调用必须复用现有业务 rulechain / function / service 主路径，不能形成第二套长期业务实现。
5. MCP tool catalog 必须显式白名单化，并处理身份上下文、输出脱敏、错误映射和高风险 mutation 排除。

## 1. 背景

Matrix 当前已有 `types.Endpoint` 抽象、`types.HttpEndpoint` 接口和 `endpoint/http` 节点。模块 server 会扫描 shared node pool 中的 endpoint 节点，并把 HTTP endpoint 注册到宿主 mux。

MCP 业务入口需要类似能力，但 MCP 的协议语义不同于 REST：

- `tools/list` 用于发现工具。
- `tools/call` 用于调用工具。
- stdio transport 是本地子进程标准输入输出。
- Streamable HTTP transport 可以由 web server route 承载。

因此，本 RFC 不把 MCP 建模为 `endpoint/http` 的配置扩展，而是建议增加独立 `endpoint/mcp` 适配器模型。

## 2. 目标

1. 定义 Matrix / WhiteRoom 可复用的 MCP endpoint adapter 边界。
2. 支持 Streamable HTTP MCP endpoint 被 Gin、`net/http` 或模块 server 挂载。
3. 支持 stdio MCP server 复用同一 adapter core。
4. 将 MCP tool 映射到既有主路径：稳定 HTTP API contract、第三方 HTTP contract 或 Matrix rulechain。
5. 定义 tool catalog 白名单、身份上下文、错误映射和输出策略。
6. 保持 Matrix core 只承载框架级抽象，不直接硬编码 IdentityX、UsageX 或 TuriX 名称。

## 3. 非目标

1. 不在 Matrix core 内实现具体 IdentityX、UsageX 业务 tools。
2. 不自动扫描并暴露全部 `endpoint/http`。
3. 不改变现有 `endpoint/http` 语义。
4. 不实现 MCP marketplace、OAuth 产品化或 remote MCP hosting。
5. 不让 MCP tool 绕过模块现有 auth、permission、quota 或 billing 边界。

## 4. 设计方向

### 4.1 Endpoint 类型

建议新增节点类型：

```text
endpoint/mcp
```

如果同时支持 stdio，stdio host 不应伪装成 HTTP endpoint。host 由 WhiteRoom 可复用 `cmd/mcp-server` 承载，并复用 Matrix adapter core：

```text
WhiteRoom cmd/mcp-server
  -> mcp adapter core
  -> tool catalog
  -> runtime pool / module API client
```

### 4.2 Adapter Core

adapter core 应提供 transport 无关接口：

```text
ListTools(ctx, authContext) -> []ToolDefinition
CallTool(ctx, name, args, authContext) -> ToolResult
```

WhiteRoom `cmd/mcp-server` transport host 负责：

- stdio stream I/O
- HTTP request / response lifecycle
- process startup / shutdown
- connection shutdown

adapter core 负责：

- JSON-RPC envelope handling
- MCP method routing
- tool catalog lookup
- input schema validation
- auth context resolution
- rulechain / API / external HTTP dispatch
- error and output normalization

### 4.3 Tool Catalog

MCP tool catalog 应由模块显式声明，不从所有 HTTP endpoints 自动生成。建议配置字段：

```json
{
  "serverName": "identityx",
  "tools": [
    {
      "name": "identityx_get_me_access",
      "description": "获取当前用户访问上下文。",
      "target": {
        "kind": "http_api",
        "id": "GET /api/identityx/auth/me/access"
      },
      "authContext": "dev_static_context",
      "riskLevel": "read"
    }
  ]
}
```

首轮只允许 `riskLevel=read`。`write`、`admin_write`、`billing_sensitive` 必须走独立设计和审批。

### 4.4 身份上下文

MCP adapter 不信任模型传入的身份字段。以下字段不得直接作为可信输入：

- `user_id`
- `company_id`
- `permissions`
- `team_ids`
- `session_id`
- `internal_token`
- `authorization`
- `cookie`

允许的上下文来源：

1. MVP 暂时只允许本地开发专用 `dev_static_context`。
2. host 已验证的 HTTP session / header 是后续候选。
3. 短时 delegation token 是产品化候选。

### 4.5 调用现有主路径

调用模式按每个 tool 的既有主路径选择，不设置全局优先级：

1. `http_api`
   - adapter 作为 MCP server，内部调用模块现有 HTTP API。
   - 适用于原流程就是 Orchestrator 固定接口、模块 public API，或必须复用现有 middleware、error mapping 和 API contract 的能力。

2. `external_http`
   - adapter 调用受控第三方 HTTP 契约。
   - 适用于原能力本身就是第三方 HTTP 接口，模块侧没有更合适的 rulechain 主路径。

3. `rulechain`
   - adapter 通过 Matrix runtime pool 直接执行 rulechain。
   - 适用于原能力已经由 Matrix rulechain 承载，且可信身份上下文已在 host / adapter 边界建立的能力。
   - 风险是必须补齐 endpoint mapping、auth context 和 response normalization。

不允许在 adapter 内重写业务流程。

## 5. 分层落位

| 层 | 允许内容 | 禁止内容 |
| --- | --- | --- |
| Matrix | endpoint/mcp 接口、MCP JSON-RPC handler、transport-neutral adapter core、generic mapper / normalizer、runtime bridge、通用 helper | 业务模块名、产品路由、TuriX 绑定、具体 tool catalog |
| WhiteRoom common | 可复用 `cmd/mcp-server` transport host、模块注册辅助、scaffold template、common sync、domain manifest、catalog 约定和治理规则 | 模块私有工具清单、MCP protocol core |
| 业务模块 | tool catalog、module-local auth context mapping | MCP transport 框架复制 |
| WhiteRoom / host glue | `cmd/mcp-server` invocation、HTTP route registration、trust boundary handoff | 业务编排复制 |
| DSL / capability | 业务 rulechain、function、service | 协议 framing |

## 6. 验收标准

1. `endpoint/mcp` 或等价 adapter 能列出显式白名单工具。
2. MCP `tools/call` 能按 tool catalog 调用已有只读 rulechain、模块 HTTP API 或受控第三方 HTTP 契约。
3. WhiteRoom `cmd/mcp-server` 能以 stdio 和 Streamable HTTP 两种 transport 复用同一 Matrix adapter core。
4. tool schema 不暴露可信身份字段。
5. 输出和错误不泄露 token、env、header、cookie 或 raw provider body。
6. 不影响现有 `endpoint/http` 注册和执行。

## 7. 未解决问题

1. Matrix 内具体包路径采用 core package 还是 experimental adapter package。
2. rulechain direct execution 是否需要新的 endpoint IO packet 类型。
3. delegation token 的签发、撤销和审计由哪个模块拥有。
4. mutation tools 是否需要 MCP 侧二次 approval。
