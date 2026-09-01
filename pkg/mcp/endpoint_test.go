package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

func TestEndpointRejectsWriteToolWithoutTrustedAuthContext(t *testing.T) {
	_, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "keymaker",
		Tools: []types.McpToolDefinition{{
			Name:      "create_spec_run",
			RiskLevel: "write",
			Target: types.McpTargetSpec{
				Kind: TargetKindHTTPAPI,
				ID:   "POST /api/keymaker/runs",
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "trusted authContext") {
		t.Fatalf("expected write tool authContext rejection, got %v", err)
	}
}

func TestEndpointCallsWriteToolWithTrustedContext(t *testing.T) {
	var gotBody map[string]any
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/keymaker/runs" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-IdentityX-User-Id") != "mcp-operator" {
			t.Fatalf("operator header = %q", r.Header.Get("X-IdentityX-User-Id"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"run_id":"run-1","status":"pending"}}`))
	}))
	defer target.Close()

	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "keymaker",
		HTTP:       types.McpHTTPConfiguration{BaseURL: target.URL},
		AuthContexts: map[string]types.McpAuthContext{
			"operator": {
				Mode: AuthModeDevStaticContext,
				Headers: map[string]string{
					"X-IdentityX-User-Id": "mcp-operator",
				},
			},
		},
		Tools: []types.McpToolDefinition{{
			Name:        "create_spec_run",
			RiskLevel:   "write",
			AuthContext: "operator",
			Target: types.McpTargetSpec{
				Kind: TargetKindHTTPAPI,
				ID:   "POST /api/keymaker/runs",
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewEndpoint() error = %v", err)
	}
	result, err := endpoint.CallTool(context.Background(), "create_spec_run", map[string]any{"run_name": "MCP run"})
	if err != nil || result.IsError {
		t.Fatalf("CallTool() result=%+v error=%v", result, err)
	}
	if gotBody["run_name"] != "MCP run" {
		t.Fatalf("body = %#v", gotBody)
	}
}

func TestEndpointBindsPathAndQueryArgumentsForHTTPAPI(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/keymaker/spec-definitions/spec%2Freview/topology" {
			t.Fatalf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		if got := r.URL.Query().Get("version"); got != "1.2.0" {
			t.Fatalf("version query = %q", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "50" {
			t.Fatalf("pageSize query = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"definition_id":"spec/review"}}`))
	}))
	defer target.Close()

	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "keymaker",
		HTTP:       types.McpHTTPConfiguration{BaseURL: target.URL},
		Tools: []types.McpToolDefinition{{
			Name:      "get_spec_topology",
			RiskLevel: "read",
			Target: types.McpTargetSpec{
				Kind:          TargetKindHTTPAPI,
				ID:            "GET /api/keymaker/spec-definitions/:definition_id/topology",
				PathArguments: map[string]string{"definition_id": "definition_id"},
				QueryArguments: map[string]string{
					"version":  "version",
					"pageSize": "page_size",
				},
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewEndpoint() error = %v", err)
	}
	result, err := endpoint.CallTool(context.Background(), "get_spec_topology", map[string]any{
		"definition_id": "spec/review",
		"version":       "1.2.0",
		"page_size":     float64(50),
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool() result=%+v error=%v", result, err)
	}
}

func TestEndpointCallHTTPAPIWithDevStaticContext(t *testing.T) {
	var gotUserID string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/identityx/auth/me/access" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotUserID = r.Header.Get("X-IdentityX-User-Id")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"userId":"user-local","companies":[]}`))
	}))
	defer target.Close()

	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		HTTP: types.McpHTTPConfiguration{
			BaseURL: target.URL,
		},
		AuthContexts: map[string]types.McpAuthContext{
			"dev_static_context": {
				Mode: AuthModeDevStaticContext,
				Headers: map[string]string{
					"X-IdentityX-User-Id": "${config:///IDENTITYX_MCP_DEV_USER_ID?scope=env&default=user-local}",
				},
			},
		},
		Tools: []types.McpToolDefinition{
			{
				Name:      "identityx_get_me_access",
				RiskLevel: "read",
				Target: types.McpTargetSpec{
					Kind: TargetKindHTTPAPI,
					ID:   "GET /api/identityx/auth/me/access",
				},
				AuthContext: "dev_static_context",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}

	result, err := endpoint.CallTool(context.Background(), "identityx_get_me_access", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success result, got error: %+v", result)
	}
	if gotUserID != "user-local" {
		t.Fatalf("expected dev static user header, got %q", gotUserID)
	}
	if len(result.Content) != 1 || !strings.Contains(result.Content[0].Text, `"userId":"user-local"`) {
		t.Fatalf("unexpected result content: %+v", result.Content)
	}
}

func TestEndpointRejectsForbiddenInputSchemaFields(t *testing.T) {
	_, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		Tools: []types.McpToolDefinition{
			{
				Name:      "identityx_get_me_access",
				RiskLevel: "read",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"user_id": map[string]any{"type": "string"},
					},
				},
				Target: types.McpTargetSpec{Kind: TargetKindHTTPAPI, URL: "http://127.0.0.1"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected forbidden input schema error")
	}
	if !strings.Contains(err.Error(), "forbidden security context fields") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEndpointRejectsForbiddenToolArguments(t *testing.T) {
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		Tools: []types.McpToolDefinition{
			{
				Name:      "identityx_get_me_access",
				RiskLevel: "read",
				Target:    types.McpTargetSpec{Kind: TargetKindHTTPAPI, URL: "http://127.0.0.1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}
	result, err := endpoint.CallTool(context.Background(), "identityx_get_me_access", map[string]any{
		"filters": map[string]any{
			"company_id": "company-forged",
		},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if !result.IsError || !strings.Contains(result.Content[0].Text, "company_id") {
		t.Fatalf("expected forbidden argument tool error, got %+v", result)
	}
}

func TestEndpointRejectsArgumentsThatDoNotMatchInputSchemaBeforeDispatch(t *testing.T) {
	dispatched := false
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "lingbao-kb",
		Tools: []types.McpToolDefinition{
			{
				Name:      "kb_read_document",
				RiskLevel: "read",
				InputSchema: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"doc_id"},
					"properties": map[string]any{
						"doc_id": map[string]any{"type": "string"},
					},
				},
				Target: types.McpTargetSpec{
					Kind: TargetKindHandler,
					ID:   "kb_read_document",
				},
			},
		},
	}, WithTargetDispatcher(TargetDispatcherFunc(func(context.Context, DispatchRequest) (types.McpToolResult, bool, error) {
		dispatched = true
		return NewTextToolResult(`{"ok":true}`), true, nil
	})))
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}

	result, err := endpoint.CallTool(context.Background(), "kb_read_document", map[string]any{
		"doc_id": float64(2),
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected schema validation tool error, got %+v", result)
	}
	if dispatched {
		t.Fatal("dispatcher must not run when arguments fail inputSchema validation")
	}
	if len(result.Content) != 1 ||
		!strings.Contains(result.Content[0].Text, "invalid arguments for tool") ||
		!strings.Contains(result.Content[0].Text, "doc_id") {
		t.Fatalf("expected actionable validation error, got %+v", result.Content)
	}

	result, err = endpoint.CallTool(context.Background(), "kb_read_document", map[string]any{
		"doc_id": "c39de16e54c159ce",
	})
	if err != nil {
		t.Fatalf("CallTool with valid arguments failed: %v", err)
	}
	if result.IsError || !dispatched {
		t.Fatalf("valid arguments should reach dispatcher, got result=%+v dispatched=%t", result, dispatched)
	}
}

func TestEndpointRedactsSecretsFromHTTPToolResult(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Authorization: Bearer secret-token","access_token":"secret-access"}`))
	}))
	defer target.Close()

	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		Tools: []types.McpToolDefinition{
			{
				Name:      "identityx_get_me_access",
				RiskLevel: "read",
				Target:    types.McpTargetSpec{Kind: TargetKindHTTPAPI, URL: target.URL},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}
	result, err := endpoint.CallTool(context.Background(), "identityx_get_me_access", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected HTTP error to map to tool error: %+v", result)
	}
	text := result.Content[0].Text
	if strings.Contains(text, "secret-token") || strings.Contains(text, "secret-access") {
		t.Fatalf("expected secret redaction, got %s", text)
	}
	if !strings.Contains(text, "[redacted]") || !strings.Contains(text, `"access_token":"[redacted]"`) {
		t.Fatalf("expected redacted placeholders, got %s", text)
	}
}

func TestEndpointDispatchesRulechainTargetWithGatewayAssertionContext(t *testing.T) {
	var got DispatchRequest
	endpoint, err := NewEndpoint(types.McpEndpointNodeConfiguration{
		ServerName: "identityx",
		AuthContexts: map[string]types.McpAuthContext{
			"gateway_assertion_context": {
				Mode: AuthModeGatewayAssertion,
				Headers: map[string]string{
					"X-IdentityX-User-Id": "X-IdentityX-User-Id",
					"X-IdentityX-Sub":     "X-IdentityX-Sub",
				},
			},
		},
		Tools: []types.McpToolDefinition{
			{
				Name:      "identityx_get_me_runtime_stats",
				RiskLevel: "read",
				Target: types.McpTargetSpec{
					Kind: TargetKindRuleChain,
					ID:   "identityx/rc-auth-get-me-runtime-stats",
				},
				AuthContext: "gateway_assertion_context",
			},
		},
	}, WithTargetDispatcher(TargetDispatcherFunc(func(ctx context.Context, req DispatchRequest) (types.McpToolResult, bool, error) {
		got = req
		return NewTextToolResult(`{"ok":true}`), true, nil
	})))
	if err != nil {
		t.Fatalf("NewEndpoint failed: %v", err)
	}

	ctx := WithIncomingHTTPHeaders(context.Background(), http.Header{
		"X-IdentityX-User-Id": []string{"user-from-gateway"},
		"X-IdentityX-Sub":     []string{"sub-from-gateway"},
	})
	result, err := endpoint.CallTool(ctx, "identityx_get_me_runtime_stats", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success result, got error: %+v", result)
	}
	if got.Tool.Name != "identityx_get_me_runtime_stats" {
		t.Fatalf("dispatcher received unexpected tool: %+v", got.Tool)
	}
	if got.AuthContext.Mode != AuthModeGatewayAssertion {
		t.Fatalf("expected gateway assertion mode, got %q", got.AuthContext.Mode)
	}
	if got.AuthContext.Headers["X-IdentityX-User-Id"] != "user-from-gateway" {
		t.Fatalf("unexpected resolved user header: %+v", got.AuthContext.Headers)
	}
	if got.AuthContext.Headers["X-IdentityX-Sub"] != "sub-from-gateway" {
		t.Fatalf("unexpected resolved sub header: %+v", got.AuthContext.Headers)
	}
}
