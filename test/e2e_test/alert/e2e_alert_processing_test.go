package test

import (
	"context"
	"encoding/json"
	"io/fs"
	"testing"

	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	matrix "github.com/neohetj/matrix"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
)

// Helper struct for setup
type TestEnv struct {
	Engine     *matrix.MatrixEngine
	MockLogger *utils.MockLogger
}

func copyJSONFixtures(t *testing.T, srcDir, dstDir string) {
	t.Helper()
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read fixture dir %s: %v", srcDir, err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("create fixture dir %s: %v", dstDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read fixture file %s: %v", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, fs.FileMode(0o644)); err != nil {
			t.Fatalf("write fixture file %s: %v", dstPath, err)
		}
	}
}

// setup initializes the Matrix engine and mock logger for testing
func setup(t *testing.T) *TestEnv {
	t.Helper()

	// DataT resolves definitions through the shared catalog. Scope this synthetic
	// definition to the test; runtime and shared-node instances stay engine-local.
	type Alert struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	}
	const sid = "AlertFixture_V1"
	coreRegistry := registry.Default.CoreObjRegistry.(*registry.DefaultCoreObjRegistry)
	_, exists := coreRegistry.Get(sid)
	require.False(t, exists, "fixture definition leaked from another test")
	coreRegistry.Register(types.NewCoreObjDef(&Alert{}, sid, "Alert test fixture"))
	t.Cleanup(func() { coreRegistry.Unregister(sid) })

	mockLogger := &utils.MockLogger{}
	workspace := t.TempDir()
	componentRoot := filepath.Join(workspace, "alert", "dsl")
	copyJSONFixtures(t, filepath.Join(".", "rulechains"), filepath.Join(componentRoot, "rulechains"))
	copyJSONFixtures(t, filepath.Join(".", "endpoints"), filepath.Join(componentRoot, "endpoints"))

	cfg := config.MatrixConfig{
		Loader: config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{workspace},
				},
			},
		},
		EnabledComponents: []string{"alert"},
	}
	eng, err := matrix.New(cfg, matrix.WithLogger(mockLogger), matrix.WithDeferredActiveEndpoints())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, eng.Close()) })

	return &TestEnv{
		Engine:     eng,
		MockLogger: mockLogger,
	}
}

func TestE2EAlertProcessing(t *testing.T) {
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
	require.NoError(t, err)

	msg := types.NewMsg("ALERT", "", nil, types.NewDataT())
	msg.SetData(string(alertDataBytes), cnst.JSON)

	// 4. Start the rule chain
	result, err := rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	require.NoError(t, err)

	// 5. Assert the mapper produced the typed alert, then verify the routed log.
	alert, ok := result.DataT().Get("alert")
	require.True(t, ok)
	assert.Equal(t, "AlertFixture_V1", alert.Definition().SID())

	// Assert that the correct log message was captured
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "Critical alert received: High CPU usage detected")
	assert.NotContains(t, logOutput, "Warning alert received")
	assert.NotContains(t, logOutput, "Info alert received")
}

func TestHttpEndpointTrigger(t *testing.T) {
	env := setup(t)

	// 3. Retrieve the loaded endpoint from the SharedNodePool
	epCtx, ok := env.Engine.SharedNodePool().Get("ep_http_alert_trigger")
	require.True(t, ok, "endpoint not found in shared node pool")

	epNode, ok := epCtx.GetNode().(types.HttpEndpoint)
	require.True(t, ok, "node is not an HttpEndpoint")

	// 5. Create HTTP Request
	reqBody := `{"severity": "critical", "summary": "Http Triggered Alert"}`
	req := httptest.NewRequest("POST", "/alert", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 6. Handle Request
	err := epNode.HandleHttpRequest(w, req)
	require.NoError(t, err)

	// 7. Verify Response
	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)

	var respBody map[string]any
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)
	assert.Equal(t, "ok", respBody["status"])
	assert.Equal(t, "Http Triggered Alert", respBody["summary"])

	// 8. Verify Logs
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "Critical alert received: Http Triggered Alert")
}

func TestHttpEndpointTriggerError(t *testing.T) {
	env := setup(t)

	// 3. Retrieve Endpoint
	epCtx, ok := env.Engine.SharedNodePool().Get("ep_http_alert_trigger")
	require.True(t, ok)
	epNode, ok := epCtx.GetNode().(types.HttpEndpoint)
	require.True(t, ok)

	// 4. Create Invalid Request (Missing required 'severity')
	reqBody := `{"summary": "Invalid Alert"}`
	req := httptest.NewRequest("POST", "/alert", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 5. Handle Request
	err := epNode.HandleHttpRequest(w, req)

	// 6. Verify the endpoint handled the internal error and returned a safe public response.
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	decodeErr := json.NewDecoder(w.Body).Decode(&errorResponse)
	assert.NoError(t, decodeErr)
	assert.Equal(t, http.StatusBadRequest, errorResponse.Code)
	assert.Equal(t, "invalid request", errorResponse.Message)
	assert.NotContains(t, w.Body.String(), "severity")
	assert.NotContains(t, w.Body.String(), "required field")
}

func TestE2EAlertWebhook(t *testing.T) {
	// 1. Start a mock webhook server
	received := make(chan map[string]any, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/webhook" && r.Method == "POST" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid webhook body", http.StatusBadRequest)
				return
			}
			received <- body
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(ts.Close)

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
	require.NoError(t, err)

	msg := types.NewMsg("ALERT", "", nil, types.NewDataT())
	msg.SetData(string(alertDataBytes), cnst.JSON)
	// Inject the dynamic mock server URL into metadata
	msg.Metadata()["webhook_url"] = ts.URL

	// 6. Start the rule chain
	result, err := rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	require.NoError(t, err)

	// 7. Verify that the webhook was received
	var receivedBody map[string]any
	select {
	case receivedBody = <-received:
	default:
		t.Fatal("webhook should have been received before the HTTP client completed")
	}
	assert.Equal(t, "200", result.Metadata()["webhook_status"])

	// 8. Verify the content of the webhook payload
	// The rule chain maps dataT.alert (AlertFixture_V1) to the body.
	// Alert fixture struct has "labels" and "annotations".
	assert.NotNil(t, receivedBody)
	labels, ok := receivedBody["labels"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "critical", labels["severity"])

	annotations, ok := receivedBody["annotations"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Database connection lost", annotations["summary"])

	// 9. Verify Logs (optional, ensuring critical alert logic was hit)
	// Note: The new rule chain doesn't have a specific log node for critical alerts,
	// it routes directly to webhook. So we check if *start* log is present.
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "E2E test started")
}

func TestE2EAlertOverride(t *testing.T) {
	// 1. Start a mock webhook server
	received := make(chan map[string]any, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/webhook" && r.Method == "POST" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid webhook body", http.StatusBadRequest)
				return
			}
			received <- body
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(ts.Close)

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
	require.NoError(t, err)

	msg := types.NewMsg("ALERT", "", nil, types.NewDataT())
	msg.SetData(string(alertDataBytes), cnst.JSON)
	// Inject metadata
	msg.Metadata()["webhook_url"] = ts.URL
	msg.Metadata()["ts"] = "1700000000"

	// 5. Start the rule chain
	result, err := rt.ExecuteAndWait(context.Background(), "node-start", msg, nil)
	require.NoError(t, err)

	// 6. Verify that the webhook was received
	var receivedBody map[string]any
	select {
	case receivedBody = <-received:
	default:
		t.Fatal("webhook should have been received before the HTTP client completed")
	}
	assert.Equal(t, "200", result.Metadata()["webhook_status"])

	// 7. Verify the content of the webhook payload
	// The rule chain maps the alert to body, but overrides severity and adds forwarded_at.
	assert.NotNil(t, receivedBody)

	// Check overrides: labels.severity should be CRITICAL_URGENT, not critical
	labels, ok := receivedBody["labels"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "CRITICAL_URGENT", labels["severity"])

	// Check supplements: forwarded_at should exist and match metadata.ts
	// Note: JSON unmarshals numbers as float64
	forwardedAt, ok := receivedBody["forwarded_at"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(1700000000), forwardedAt)

	// Check base content: annotations.summary should be preserved
	annotations, ok := receivedBody["annotations"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Database connection lost", annotations["summary"])
}
