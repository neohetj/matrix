package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

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
