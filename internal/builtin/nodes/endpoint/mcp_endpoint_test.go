package endpoint

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

func TestMcpEndpointNodeImplementsEndpointContract(t *testing.T) {
	node := (&McpEndpointNode{BaseNode: *types.NewBaseNode(McpEndpointNodeType, types.NodeMetadata{})}).New()
	ep, ok := node.(types.McpEndpoint)
	if !ok {
		t.Fatalf("expected McpEndpoint implementation, got %T", node)
	}
	err := ep.Init(types.ConfigMap{
		"serverName": "identityx",
		"http": map[string]any{
			"baseURL": "http://127.0.0.1:1",
		},
		"tools": []any{
			map[string]any{
				"name":      "identityx_get_me_access",
				"riskLevel": "read",
				"target": map[string]any{
					"kind": "http_api",
					"id":   "GET /api/identityx/auth/me/access",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	tools, err := ep.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "identityx_get_me_access" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}
