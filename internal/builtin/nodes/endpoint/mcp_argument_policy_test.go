package endpoint

import (
	"context"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

// TestMcpNodeArgumentPolicy 校验真正的 DSL Init 路径读取策略，而非仅 Go Option 路径。
func TestMcpNodeArgumentPolicy(t *testing.T) {
	for _, schemaField := range []string{"query", "x_example_actor"} {
		node := (&McpEndpointNode{BaseNode: *types.NewBaseNode(McpEndpointNodeType, types.NodeMetadata{})}).New().(types.McpEndpoint)
		cfg := types.ConfigMap{
			"serverName":     "example",
			"argumentPolicy": map[string]any{"denyPrefixes": []any{"x_example_"}},
			"tools": []any{map[string]any{
				"name": "read", "riskLevel": "read", "target": map[string]any{"kind": "handler", "id": "read"},
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{schemaField: map[string]any{"type": "string"}}},
			}},
		}
		err := node.Init(cfg)
		if schemaField == "x_example_actor" {
			if err == nil || !strings.Contains(err.Error(), "forbidden security context") {
				t.Fatalf("schema accepted: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		result, err := node.CallTool(context.Background(), "read", map[string]any{"X.Example.Actor": "untrusted"})
		if err != nil || !result.IsError || !strings.Contains(result.Content[0].Text, "security context") {
			t.Fatalf("runtime policy ignored: %+v %v", result, err)
		}
		delete(cfg, "argumentPolicy")
		if err := node.Init(cfg); err == nil || !strings.Contains(err.Error(), "argumentPolicy") {
			t.Fatalf("missing policy accepted: %v", err)
		}
		cfg["argumentPolicy"] = map[string]any{"denyPrefix": []any{"x_example_"}}
		if err := node.Init(cfg); err == nil {
			t.Fatal("misspelled policy accepted")
		}
	}
}
