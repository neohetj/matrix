package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

// TestArgumentPolicyJSONRPCHTTP 确认经 HTTP JSON-RPC 解码的嵌套字段在派发前被拒绝。
func TestArgumentPolicyJSONRPCHTTP(t *testing.T) {
	calls := 0
	ep, err := NewEndpoint(policyTestConfig(t, `{"denyPrefixes":["x_example_"]}`), WithTargetDispatcher(TargetDispatcherFunc(func(context.Context, DispatchRequest) (types.McpToolResult, bool, error) {
		calls++
		return textToolResult("ok"), true, nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer([]ToolProvider{ep})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"data":[{"X.Example.Actor":"private-value"}],"argumentPolicy":{}}}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	var envelope struct {
		Result types.McpToolResult `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || !envelope.Result.IsError || calls != 0 {
		t.Fatalf("unexpected response: %d %s calls=%d", response.Code, response.Body, calls)
	}
	if strings.Contains(response.Body.String(), "private-value") {
		t.Fatal("rejected argument value leaked")
	}
}

// TestArgumentPolicyFileLoader 确认磁盘目录入口使用同一严格策略解析与 schema 校验。
func TestArgumentPolicyFileLoader(t *testing.T) {
	for _, tc := range []struct {
		name, config string
		wantError    string
	}{
		{"missing", `{"serverName":"example"}`, "argumentPolicy is required"},
		{"typo", `{"serverName":"example","argumentPolicy":{"denyPrefix":["x_example_"]}}`, "unknown field"},
		{"schema", `{"serverName":"example","argumentPolicy":{"denyKeys":["example_actor"]},"tools":[{"name":"read","riskLevel":"read","target":{"kind":"handler"},"inputSchema":{"type":"object","properties":{"example_actor":{"type":"string"}}}}]}`, "forbidden security context"},
		{"explicit", `{"serverName":"example","argumentPolicy":{}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "endpoint.json"), []byte(`{"type":"endpoint/mcp","configuration":`+tc.config+`}`), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadEndpointsFromDir(dir)
			if tc.wantError == "" && err != nil || tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("loader error=%v wantError=%q", err, tc.wantError)
			}
		})
	}
}
