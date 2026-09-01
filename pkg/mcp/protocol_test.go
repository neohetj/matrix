package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

func TestServerHandlesInitializeListAndCall(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer target.Close()

	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		HTTP:       types.McpHTTPConfiguration{BaseURL: target.URL},
		Tools: []types.McpToolDefinition{
			{
				Name:        "identityx_get_me_access",
				Description: "Read access context",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
				Target:    types.McpTargetSpec{Kind: TargetKindHTTPAPI, ID: "GET /api/identityx/auth/me/access"},
				RiskLevel: "read",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}
	server, err := NewServer([]ToolProvider{endpoint})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	initResp, ok := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`))
	if !ok {
		t.Fatal("expected initialize response")
	}
	var initEnvelope map[string]any
	if err := json.Unmarshal(initResp, &initEnvelope); err != nil {
		t.Fatalf("decode initialize response: %v", err)
	}
	result := initEnvelope["result"].(map[string]any)
	if result["protocolVersion"] != "2025-06-18" {
		t.Fatalf("unexpected protocolVersion: %v", result["protocolVersion"])
	}

	listResp, ok := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":"tools","method":"tools/list"}`))
	if !ok {
		t.Fatal("expected tools/list response")
	}
	if !strings.Contains(string(listResp), `"identityx_get_me_access"`) {
		t.Fatalf("tools/list missing tool: %s", listResp)
	}

	callResp, ok := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"identityx_get_me_access","arguments":{}}}`))
	if !ok {
		t.Fatal("expected tools/call response")
	}
	if !strings.Contains(string(callResp), `\"ok\":true`) {
		t.Fatalf("tools/call missing target response: %s", callResp)
	}
}

func TestServerToolsListPublishesRiskAnnotations(t *testing.T) {
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "annotated-tools",
		AuthContexts: map[string]types.McpAuthContext{
			"writer": {Mode: AuthModeDevStaticContext},
		},
		Tools: []types.McpToolDefinition{
			{
				Name:      "read_run",
				Target:    types.McpTargetSpec{Kind: TargetKindHTTPAPI, URL: "http://127.0.0.1:1/runs/1"},
				RiskLevel: "read",
			},
			{
				Name:        "create_run",
				Target:      types.McpTargetSpec{Kind: TargetKindHTTPAPI, Method: http.MethodPost, URL: "http://127.0.0.1:1/runs"},
				RiskLevel:   "write",
				AuthContext: "writer",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer([]ToolProvider{endpoint})
	if err != nil {
		t.Fatal(err)
	}

	response, ok := server.HandleMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if !ok {
		t.Fatal("expected tools/list response")
	}
	var envelope struct {
		Result struct {
			Tools []struct {
				Name        string                  `json:"name"`
				Annotations protocolToolAnnotations `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Result.Tools) != 2 {
		t.Fatalf("tools/list returned %d tools", len(envelope.Result.Tools))
	}
	if got := envelope.Result.Tools[0]; got.Name != "read_run" || !got.Annotations.ReadOnlyHint || got.Annotations.DestructiveHint || !got.Annotations.IdempotentHint {
		t.Fatalf("read annotations = %+v", got)
	}
	if got := envelope.Result.Tools[1]; got.Name != "create_run" || got.Annotations.ReadOnlyHint || !got.Annotations.DestructiveHint || got.Annotations.IdempotentHint {
		t.Fatalf("write annotations = %+v", got)
	}
}

func TestServerServeHTTPHandlesJSONRPCPost(t *testing.T) {
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		Tools: []types.McpToolDefinition{
			{
				Name:      "identityx_get_me_access",
				Target:    types.McpTargetSpec{Kind: TargetKindHTTPAPI, URL: "http://127.0.0.1:1"},
				RiskLevel: "read",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}
	server, err := NewServer([]ToolProvider{endpoint})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp/identityx", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"identityx_get_me_access"`) {
		t.Fatalf("tools/list missing tool: %s", rec.Body.String())
	}
}

func TestServerServeHTTPPropagatesIncomingHeadersToGatewayAssertionContext(t *testing.T) {
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		AuthContexts: map[string]types.McpAuthContext{
			"gateway_assertion_context": {
				Mode: AuthModeGatewayAssertion,
				Headers: map[string]string{
					"X-IdentityX-User-Id": "X-IdentityX-User-Id",
				},
			},
		},
		Tools: []types.McpToolDefinition{
			{
				Name:        "identityx_get_me_runtime_stats",
				Description: "Read runtime stats",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
				Target:      types.McpTargetSpec{Kind: TargetKindRuleChain, ID: "identityx/rc-auth-get-me-runtime-stats"},
				RiskLevel:   "read",
				AuthContext: "gateway_assertion_context",
			},
		},
	}, WithTargetDispatcher(TargetDispatcherFunc(func(ctx context.Context, req DispatchRequest) (types.McpToolResult, bool, error) {
		return NewTextToolResult(req.AuthContext.Headers["X-IdentityX-User-Id"]), true, nil
	})))
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}
	server, err := NewServer([]ToolProvider{endpoint})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp/identityx", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"identityx_get_me_runtime_stats","arguments":{}}}`))
	req.Header.Set("X-IdentityX-User-Id", "gateway-user")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `gateway-user`) {
		t.Fatalf("tools/call missing propagated gateway header: %s", rec.Body.String())
	}
}

func TestServerServeHTTPPreservesStructuredToolContent(t *testing.T) {
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "knowledge",
		Tools: []types.McpToolDefinition{{
			Name:      "kb_register_citations",
			Target:    types.McpTargetSpec{Kind: TargetKindHandler, ID: "knowledge/register-citations"},
			RiskLevel: "read",
		}},
	}, WithTargetDispatcher(TargetDispatcherFunc(func(context.Context, DispatchRequest) (types.McpToolResult, bool, error) {
		return types.McpToolResult{
			Content: []types.McpToolContent{{Type: "text", Text: "citations registered"}},
			StructuredContent: map[string]any{
				"presentation": map[string]any{
					"sources": []any{map[string]any{"id": "doc-1", "title": "Policy"}},
				},
			},
		}, true, nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer([]ToolProvider{endpoint})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp/knowledge", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"kb_register_citations","arguments":{}}}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"structuredContent":{"presentation":{"sources":[{"id":"doc-1","title":"Policy"}]}}`) {
		t.Fatalf("tools/call dropped structured content: %s", rec.Body.String())
	}
}
