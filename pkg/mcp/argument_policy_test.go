package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

func policyTestConfig(t *testing.T, policy string) types.McpEndpointNodeConfiguration {
	t.Helper()
	var cfg types.McpEndpointNodeConfiguration
	if err := json.Unmarshal([]byte(`{"serverName":"example","argumentPolicy":`+policy+`,"tools":[{"name":"read","riskLevel":"read","target":{"kind":"handler","id":"read"}}]}`), &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestArgumentPolicyRejectsNestedAndNormalizedFields(t *testing.T) {
	cfg := policyTestConfig(t, `{"denyKeys":["example_roles"],"denyPrefixes":["x_example_"]}`)
	calls := 0
	ep, err := NewEndpoint(cfg, WithTargetDispatcher(TargetDispatcherFunc(func(_ context.Context, _ DispatchRequest) (types.McpToolResult, bool, error) {
		calls++
		return textToolResult("ok"), true, nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"example_roles", " EXAMPLE-Roles ", "X.Example.Session", "x_example_future", "authorization", "company_id"} {
		t.Run(key, func(t *testing.T) {
			result, err := ep.CallTool(context.Background(), "read", map[string]any{"payload": []any{map[string]any{key: "private-value"}}, "argumentPolicy": map[string]any{}})
			if err != nil || !result.IsError || !strings.Contains(result.Content[0].Text, "security context") {
				t.Fatalf("expected policy rejection: result=%+v err=%v", result, err)
			}
			if strings.Contains(result.Content[0].Text, "private-value") {
				t.Fatal("argument value leaked")
			}
		})
	}
	if calls != 0 {
		t.Fatalf("rejected arguments dispatched %d times", calls)
	}
	result, err := ep.CallTool(context.Background(), "read", map[string]any{"query": "ordinary", "example_roles_description": "allowed"})
	if err != nil || result.IsError || calls != 1 {
		t.Fatalf("ordinary call failed: %+v %v calls=%d", result, err, calls)
	}
}

func TestArgumentPolicyRejectsSchemaAndAuthHeaderFields(t *testing.T) {
	for _, key := range []string{"example_roles", "X-Example-Future", "X-Trusted-Actor"} {
		cfg := policyTestConfig(t, `{"denyKeys":["example_roles"],"denyPrefixes":["x_example_"]}`)
		cfg.AuthContexts = map[string]types.McpAuthContext{"trusted": {Mode: AuthModeDevStaticContext, Headers: map[string]string{"X-Trusted-Actor": "server-owned"}}}
		cfg.Tools[0].InputSchema = map[string]any{"type": "object", "properties": map[string]any{"payload": map[string]any{"type": "object", "properties": map[string]any{key: map[string]any{"type": "string"}}}}}
		if _, err := NewEndpoint(cfg); err == nil || !strings.Contains(err.Error(), "forbidden security context") {
			t.Errorf("schema key %q accepted: %v", key, err)
		}
	}
}

func TestArgumentPolicyRequiresExplicitValidDeclaration(t *testing.T) {
	for _, policy := range []string{`null`, `{"denyKey":["example_roles"]}`, `{"denyKeys":[""]}`, `{"denyPrefixes":["  "]}`, `{"denyKeys":[null]}`, `{"denyPrefixes":"x_example_"}`, `[]`} {
		t.Run(policy, func(t *testing.T) {
			var cfg types.McpEndpointNodeConfiguration
			err := json.Unmarshal([]byte(`{"serverName":"example","argumentPolicy":`+policy+`}`), &cfg)
			if err == nil {
				_, err = NewEndpoint(cfg)
			}
			if err == nil {
				t.Fatal("invalid policy accepted")
			}
		})
	}
	if _, err := NewEndpoint(types.McpEndpointNodeConfiguration{ServerName: "example"}); err == nil || !strings.Contains(err.Error(), "argumentPolicy") {
		t.Fatalf("missing policy accepted: %v", err)
	}
	if _, err := NewEndpoint(policyTestConfig(t, `{}`)); err != nil {
		t.Fatalf("explicit generic-only policy rejected: %v", err)
	}
}

func TestArgumentPolicyInfersTrustedHeadersWithoutBlockingTheirValues(t *testing.T) {
	cfg := policyTestConfig(t, `{}`)
	cfg.AuthContexts = map[string]types.McpAuthContext{"trusted": {Mode: AuthModeDevStaticContext, Headers: map[string]string{"X-Trusted-Actor": "server-owned"}}}
	cfg.Tools[0].AuthContext = "trusted"
	calls := 0
	ep, err := NewEndpoint(cfg, WithTargetDispatcher(TargetDispatcherFunc(func(_ context.Context, req DispatchRequest) (types.McpToolResult, bool, error) {
		calls++
		if req.AuthContext.Headers["X-Trusted-Actor"] != "server-owned" {
			t.Fatal("trusted header changed")
		}
		return textToolResult("ok"), true, nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ep.CallTool(context.Background(), "read", map[string]any{"x.trusted.actor": "caller-owned"})
	if err != nil || !result.IsError || calls != 0 {
		t.Fatalf("trusted header spoof accepted: %+v %v", result, err)
	}
	result, err = ep.CallTool(context.Background(), "read", nil)
	if err != nil || result.IsError || calls != 1 {
		t.Fatalf("trusted context rejected: %+v %v", result, err)
	}
}

func TestArgumentPolicySchemaKeywordsAreNotArgumentNames(t *testing.T) {
	cfg := policyTestConfig(t, `{"denyKeys":["type","description"]}`)
	cfg.Tools[0].InputSchema = map[string]any{"type": "object", "description": "ordinary schema metadata", "properties": map[string]any{"query": map[string]any{"type": "string"}}}
	if _, err := NewEndpoint(cfg); err != nil {
		t.Fatalf("schema metadata mistaken for arguments: %v", err)
	}
	cfg.Tools[0].InputSchema["properties"] = map[string]any{"type": map[string]any{"type": "string"}}
	if _, err := NewEndpoint(cfg); err == nil {
		t.Fatal("forbidden property accepted")
	}
}

func TestArgumentPolicyIsEndpointLocalAndCompiled(t *testing.T) {
	cfg := policyTestConfig(t, `{"denyKeys":["example_roles"],"denyPrefixes":["x_example_"]}`)
	dispatcher := WithTargetDispatcher(TargetDispatcherFunc(func(context.Context, DispatchRequest) (types.McpToolResult, bool, error) {
		return textToolResult("ok"), true, nil
	}))
	protected, err := NewEndpoint(cfg, dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ArgumentPolicy.DenyKeys[0] = "other_key"
	cfg.ArgumentPolicy.DenyPrefixes[0] = "x_other_"
	ordinary, err := NewEndpoint(policyTestConfig(t, `{}`), dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"example_roles", "x_example_session"} {
		result, _ := protected.CallTool(context.Background(), "read", map[string]any{key: "value"})
		if !result.IsError {
			t.Fatalf("compiled policy changed for %s", key)
		}
		result, _ = ordinary.CallTool(context.Background(), "read", map[string]any{key: "value"})
		if result.IsError {
			t.Fatalf("policy leaked into another endpoint for %s", key)
		}
	}
}
