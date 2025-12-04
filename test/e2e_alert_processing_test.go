package test

import (
	"context"
	"encoding/json"
	"testing"

	"net/http/httptest"
	"strings"

	"github.com/stretchr/testify/assert"
	"gitlab.com/neohet/matrix"
	"gitlab.com/neohet/matrix/pkg/config"
	"gitlab.com/neohet/matrix/pkg/registry"

	"gitlab.com/neohet/matrix/pkg/components/endpoint"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/test/test_utils"
)

func cleanup() {
	registry.Default.RuntimePool.Unregister("rc-simple-e2e-test")
	registry.Default.SharedNodePool.Stop()
}

func TestE2EAlertProcessing(t *testing.T) {
	cleanup()
	defer cleanup()

	// 1. Define and register the CoreObj for the alert
	type Alert struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	}
	registry.Default.CoreObjRegistry.Register(types.NewCoreObjDef(&Alert{}, "parsedAlert", "Parsed Alert"))

	// 2. Initialize the Matrix engine with a mock logger
	mockLogger := &test_utils.MockLogger{}
	cfg := config.Config{
		Loader: config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{"."}, // 从当前目录开始查找
				},
			},
			ComponentsRoot: ".", // 设置ComponentsRoot为当前目录
		},
		EnabledComponents: []string{"e2e_alert"}, // 启用"test"组件，让Discover函数找到test/rulechains
	}
	eng, err := matrix.New(cfg, matrix.WithLogger(mockLogger))
	assert.NoError(t, err)

	// 2. Get the runtime for the rule chain
	rt, ok := eng.RuntimePool().Get("rc-simple-e2e-test")
	if !ok {
		t.Fatal("runtime not found")
	}
	assert.NotNil(t, rt)

	// 3. Create a test message with a critical alert
	alertData := map[string]any{
		"labels": map[string]string{
			"severity": "critical",
		},
		"annotations": map[string]string{
			"summary": "High CPU usage detected",
		},
	}
	alertDataBytes, err := json.Marshal(alertData)
	assert.NoError(t, err)

	msg := test_utils.NewTestRuleMsg()
	msg.SetData(string(alertDataBytes))

	// 4. Start the rule chain
	_, err = rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	assert.NoError(t, err)

	// 5. Assert that the correct log message was captured
	logOutput := mockLogger.String()
	assert.Contains(t, logOutput, "Critical alert received: High CPU usage detected")
	assert.NotContains(t, logOutput, "Warning alert received")
	assert.NotContains(t, logOutput, "Info alert received")

}

func TestHttpEndpointTrigger(t *testing.T) {
	cleanup()
	defer cleanup()

	// 1. Register CoreObj
	type Alert struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	}
	// Ensure it's registered (idempotent usually, or check if already registered)
	// In a real test suite, we might need to clean up or check registry.
	// For simplicity here, we assume it might be registered by previous test or we register again.
	// registry.Default.CoreObjRegistry.Register(...) might panic if duplicate?
	// Let's assume it's safe or use a different name if needed.
	// Actually, `Register` usually overwrites or we can check.
	registry.Default.CoreObjRegistry.Register(types.NewCoreObjDef(&Alert{}, "parsedAlert", "Parsed Alert"))

	// 2. Setup Engine and Runtime
	mockLogger := &test_utils.MockLogger{}
	cfg := config.Config{
		Loader: config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{"."},
				},
			},
			ComponentsRoot: ".",
		},
		EnabledComponents: []string{"e2e_alert"},
	}
	eng, err := matrix.New(cfg, matrix.WithLogger(mockLogger))
	assert.NoError(t, err)

	// 3. Retrieve the loaded endpoint from the SharedNodePool
	epCtx, ok := eng.SharedNodePool().Get("ep_http_alert_trigger")
	assert.True(t, ok, "endpoint not found in shared node pool")

	epNode, ok := epCtx.GetNode().(endpoint.HttpEndpoint)
	assert.True(t, ok, "node is not an HttpEndpoint")

	// 5. Create HTTP Request
	reqBody := `{"severity": "critical", "summary": "Http Triggered Alert"}`
	req := httptest.NewRequest("POST", "/alert", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 6. Handle Request
	err = epNode.HandleHttpRequest(w, req)
	assert.NoError(t, err)

	// 7. Verify Response
	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)

	var respBody map[string]any
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	assert.NoError(t, err)
	assert.Equal(t, "ok", respBody["status"])
	assert.Equal(t, "Http Triggered Alert", respBody["summary"])

	// 8. Verify Logs
	logOutput := mockLogger.String()
	assert.Contains(t, logOutput, "Critical alert received: Http Triggered Alert")
}

func TestHttpEndpointTriggerError(t *testing.T) {
	cleanup()
	defer cleanup()

	// 1. Register CoreObj
	type Alert struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	}
	registry.Default.CoreObjRegistry.Register(types.NewCoreObjDef(&Alert{}, "parsedAlert", "Parsed Alert"))

	// 2. Setup Engine
	mockLogger := &test_utils.MockLogger{}
	cfg := config.Config{
		Loader: config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{"."},
				},
			},
			ComponentsRoot: ".",
		},
		EnabledComponents: []string{"e2e_alert"},
	}
	eng, err := matrix.New(cfg, matrix.WithLogger(mockLogger))
	assert.NoError(t, err)

	// 3. Retrieve Endpoint
	epCtx, ok := eng.SharedNodePool().Get("ep_http_alert_trigger")
	assert.True(t, ok)
	epNode, ok := epCtx.GetNode().(endpoint.HttpEndpoint)
	assert.True(t, ok)

	// 4. Create Invalid Request (Missing required 'severity')
	reqBody := `{"summary": "Invalid Alert"}`
	req := httptest.NewRequest("POST", "/alert", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 5. Handle Request
	err = epNode.HandleHttpRequest(w, req)

	// 6. Verify Error
	// HandleHttpRequest returns error for parameter validation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field not found")
	assert.Contains(t, err.Error(), "severity")
}
