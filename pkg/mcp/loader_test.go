package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEndpointsFromDir(t *testing.T) {
	dir := t.TempDir()
	endpointJSON := `{
  "id": "ep-test-mcp",
  "type": "endpoint/mcp",
  "name": "Test MCP Endpoint",
  "configuration": {
    "serverName": "test",
    "argumentPolicy": {},
    "http": {"baseURL": "http://127.0.0.1:1"},
    "tools": [
      {
        "name": "test_read",
        "inputSchema": {"type": "object", "properties": {}},
        "target": {"kind": "http_api", "id": "GET /test"},
        "riskLevel": "read"
      }
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "test_endpoint.json"), []byte(endpointJSON), 0o644); err != nil {
		t.Fatalf("write endpoint fixture: %v", err)
	}

	endpoints, err := LoadEndpointsFromDir(dir)
	if err != nil {
		t.Fatalf("LoadEndpointsFromDir failed: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	tools, err := endpoints[0].ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "test_read" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}
