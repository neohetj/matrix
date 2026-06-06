---
uuid: "f80e7feb-c1dd-4d39-80a1-bb4797df4bd7"
type: "Reference"
title: "Reference: MCP Endpoint Adapter"
status: "Draft"
owner: "neohetj"
version: "0.1.0"
tags:
  - "matrix"
  - "mcp"
  - "endpoint"
  - "adapter"
relations:
  - type: "is_part_of"
    target_uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
    description: "本文档属于 Matrix 参考文档库。"
  - type: "implements"
    target_uuid: "18d7026c-61eb-4b39-b537-7d927df34921"
    description: "本文档记录 MCP 业务入口适配器 RFC 的当前实现事实。"
---

# MCP Endpoint Adapter

Matrix provides a transport-neutral MCP adapter for module-owned business tools. It does not own the transport host process and it does not own business tool catalogs.

## Runtime Shape

Implemented packages:

| Path | Responsibility |
| --- | --- |
| `pkg/types/mcp.go` | Public `McpEndpoint` interface and configuration/result structs. |
| `pkg/mcp` | JSON-RPC MCP protocol handling, `tools/list`, `tools/call`, stdio loop, HTTP handler, endpoint JSON loader, auth context resolution, target dispatch. |
| `internal/builtin/nodes/endpoint/mcp_endpoint.go` | Built-in `endpoint/mcp` node registration for Matrix shared node discovery. |

Transport host ownership:

1. WhiteRoom `cmd/mcp-server` owns stdio and Streamable HTTP process/listener lifecycle.
2. Matrix `pkg/mcp.Server` exposes the handler/stdio loop used by hosts.
3. Business modules own `code/dsl/endpoints/mcp/*.json` catalogs.

## Supported MCP Methods

| Method | Behavior |
| --- | --- |
| `initialize` | Returns MCP capabilities with `tools` enabled and server info. |
| `notifications/initialized` | Accepted as notification. |
| `ping` | Returns an empty result. |
| `tools/list` | Returns only module catalog tools after adapter validation. |
| `tools/call` | Dispatches one whitelisted tool to its configured target. |

## Target Dispatch

Current target support:

| `target.kind` | Status | Behavior |
| --- | --- | --- |
| `http_api` | Implemented | Calls an existing module HTTP API using configured base URL / method / path. |
| `external_http` | Implemented | Calls an approved external HTTP URL using the same HTTP dispatch path. |
| `rulechain` | Reserved | Returns a tool error until a runtime-bound adapter is designed and approved. |

The adapter rejects MVP tools whose `riskLevel` is not `read`.

## Security Boundary

MCP tool arguments are not trusted identity facts. The adapter rejects argument/schema fields such as:

- `user_id`
- `company_id`
- `permissions`
- `team_ids`
- `session_id`
- `internal_token`
- `authorization`
- `cookie`
- `x_identityx_*`

`dev_static_context` is resolved from host config or env and may inject static headers for local smoke. It does not bypass business module middleware.

## Validation

Focused Matrix tests:

```bash
go test ./pkg/mcp ./internal/builtin/nodes/endpoint
```
