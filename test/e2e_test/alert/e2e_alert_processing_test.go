package test

import (
	"context"
	"encoding/json"
	"testing"

	"net/http"
	"net/http/httptest"
	"strings"

	matrix "github.com/neohetj/matrix"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/stretchr/testify/assert"

	"github.com/neohetj/matrix/pkg/components/endpoint"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
)

func assertErrorCode(t *testing.T, err error, expectedCode cnst.ErrCode) {
	var fault *types.Fault
	if assert.ErrorAs(t, err, &fault) {
		assert.Equal(t, expectedCode, fault.Code)
	}
}

// Helper struct for setup
type TestEnv struct {
	Engine     *matrix.MatrixEngine
	MockLogger *utils.MockLogger
}

// setup initializes the Matrix engine and mock logger for testing
func setup(t *testing.T) *TestEnv {
	cleanup()

	// Register common CoreObj if not present (idempotent registry handles this usually,
	// but good practice to ensure it's there)
	type Alert struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	}
	registry.Default.CoreObjRegistry.Register(message.NewCoreObjDef(&Alert{}, "parsedAlert", "Parsed Alert"))

	mockLogger := &utils.MockLogger{}
	cfg := config.MatrixConfig{
		Loader: config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{".."}, // 从上级目录开始查找
				},
			},
			ComponentsRoot: ".", // 设置ComponentsRoot为"../."，即上级目录
		},
		EnabledComponents: []string{"alert"},
	}
	eng, err := matrix.New(cfg, matrix.WithLogger(mockLogger))
	assert.NoError(t, err)

	return &TestEnv{
		Engine:     eng,
		MockLogger: mockLogger,
	}
}

func cleanup() {
	registry.Default.RuntimePool.Unregister("rc-simple-e2e-test")
	registry.Default.RuntimePool.Unregister("rc-e2e-alert-webhook")
	registry.Default.RuntimePool.Unregister("rc-e2e-alert-override")
	registry.Default.SharedNodePool.Stop()
}

func TestE2EAlertProcessing(t *testing.T) {
	defer cleanup()
	env := setup(t)

	// 2. Get the runtime for the rule chain
	rt, ok := env.Engine.RuntimePool().Get("rc-simple-e2e-test")
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

	msg := utils.NewTestRuleMsg()
	msg.SetData(string(alertDataBytes), cnst.JSON)

	// 4. Start the rule chain
	_, err = rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	assert.NoError(t, err)

	// 5. Assert that the correct log message was captured
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "Critical alert received: High CPU usage detected")
	assert.NotContains(t, logOutput, "Warning alert received")
	assert.NotContains(t, logOutput, "Info alert received")
}

func TestHttpEndpointTrigger(t *testing.T) {
	defer cleanup()
	env := setup(t)

	// 3. Retrieve the loaded endpoint from the SharedNodePool
	epCtx, ok := env.Engine.SharedNodePool().Get("ep_http_alert_trigger")
	assert.True(t, ok, "endpoint not found in shared node pool")

	epNode, ok := epCtx.GetNode().(endpoint.HttpEndpoint)
	assert.True(t, ok, "node is not an HttpEndpoint")

	// 5. Create HTTP Request
	reqBody := `{"severity": "critical", "summary": "Http Triggered Alert"}`
	req := httptest.NewRequest("POST", "/alert", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 6. Handle Request
	err := epNode.HandleHttpRequest(w, req)
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
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "Critical alert received: Http Triggered Alert")
}

func TestHttpEndpointTriggerError(t *testing.T) {
	defer cleanup()
	env := setup(t)

	// 3. Retrieve Endpoint
	epCtx, ok := env.Engine.SharedNodePool().Get("ep_http_alert_trigger")
	assert.True(t, ok)
	epNode, ok := epCtx.GetNode().(endpoint.HttpEndpoint)
	assert.True(t, ok)

	// 4. Create Invalid Request (Missing required 'severity')
	reqBody := `{"summary": "Invalid Alert"}`
	req := httptest.NewRequest("POST", "/alert", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 5. Handle Request
	err := epNode.HandleHttpRequest(w, req)

	// 6. Verify Error
	// HandleHttpRequest returns error for parameter validation failure
	assertErrorCode(t, err, cnst.CodeRequiredFieldMissing)
}

func TestE2EAlertWebhook(t *testing.T) {
	defer cleanup()
	// Note: setup is called inside to ensure clean state, but we need mock server URL first?
	// Actually, we can start server first, then setup engine.

	// 1. Start a mock webhook server
	webhookReceived := false
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/webhook" && r.Method == "POST" {
			webhookReceived = true
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	// 2. Setup Engine
	env := setup(t)

	// 4. Get the runtime for the webhook rule chain
	rt, ok := env.Engine.RuntimePool().Get("rc-e2e-alert-webhook")
	if !ok {
		t.Fatal("runtime not found: rc-e2e-alert-webhook")
	}
	assert.NotNil(t, rt)

	// 5. Create a test message with a critical alert
	alertData := map[string]any{
		"labels": map[string]string{
			"severity": "critical",
		},
		"annotations": map[string]string{
			"summary": "Database connection lost",
		},
	}
	alertDataBytes, err := json.Marshal(alertData)
	assert.NoError(t, err)

	msg := utils.NewTestRuleMsg()
	msg.SetData(string(alertDataBytes), cnst.JSON)
	// Inject the dynamic mock server URL into metadata
	msg.Metadata()["webhook_url"] = ts.URL

	// 6. Start the rule chain
	_, err = rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	assert.NoError(t, err)

	// 7. Verify that the webhook was received
	assert.True(t, webhookReceived, "Webhook should have been received")

	// 8. Verify the content of the webhook payload
	// The rule chain maps dataT.asdflasjgie (parsedAlert) to the body.
	// parsedAlert struct has "labels" and "annotations".
	assert.NotNil(t, receivedBody)
	labels, ok := receivedBody["labels"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "critical", labels["severity"])

	annotations, ok := receivedBody["annotations"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Database connection lost", annotations["summary"])

	// 9. Verify Logs (optional, ensuring critical alert logic was hit)
	// Note: The new rule chain doesn't have a specific log node for critical alerts,
	// it routes directly to webhook. So we check if *start* log is present.
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "E2E test started")
}

func TestE2EAlertOverride(t *testing.T) {
	defer cleanup()

	// 1. Start a mock webhook server
	webhookReceived := false
	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/webhook" && r.Method == "POST" {
			webhookReceived = true
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	// 2. Setup Engine
	env := setup(t)

	// 3. Get the runtime for the override rule chain
	rt, ok := env.Engine.RuntimePool().Get("rc-e2e-alert-override")
	if !ok {
		t.Fatal("runtime not found: rc-e2e-alert-override")
	}
	assert.NotNil(t, rt)

	// 4. Create a test message with a critical alert
	alertData := map[string]any{
		"labels": map[string]string{
			"severity": "critical",
		},
		"annotations": map[string]string{
			"summary": "Database connection lost",
		},
	}
	alertDataBytes, err := json.Marshal(alertData)
	assert.NoError(t, err)

	msg := utils.NewTestRuleMsg()
	msg.SetData(string(alertDataBytes), cnst.JSON)
	// Inject metadata
	msg.Metadata()["webhook_url"] = ts.URL
	msg.Metadata()["ts"] = "1700000000"

	// 5. Start the rule chain
	_, err = rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	assert.NoError(t, err)

	// 6. Verify that the webhook was received
	assert.True(t, webhookReceived, "Webhook should have been received")

	// 7. Verify the content of the webhook payload
	// The rule chain maps parsedAlert to body, but overrides severity and adds forwarded_at.
	assert.NotNil(t, receivedBody)

	// Check overrides: labels.severity should be CRITICAL_URGENT, not critical
	labels, ok := receivedBody["labels"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "CRITICAL_URGENT", labels["severity"])

	// Check supplements: forwarded_at should exist and match metadata.ts
	// Note: JSON unmarshals numbers as float64
	forwardedAt, ok := receivedBody["forwarded_at"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(1700000000), forwardedAt)

	// Check base content: annotations.summary should be preserved
	annotations, ok := receivedBody["annotations"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Database connection lost", annotations["summary"])
}
