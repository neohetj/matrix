package asset

import (
	"os"
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock objects
type MockRuleMsg struct {
	types.RuleMsg
	data     types.Data
	metadata types.Metadata
}

func (m *MockRuleMsg) Data() types.Data                 { return m.data }
func (m *MockRuleMsg) Metadata() types.Metadata         { return m.metadata }
func (m *MockRuleMsg) DataFormat() cnst.MFormat         { return cnst.JSON } // Mock as needed
func (m *MockRuleMsg) SetData(d string, f cnst.MFormat) { m.data = types.Data(d) }

type MockNodeCtx struct {
	types.NodeCtx
	config  types.ConfigMap
	runtime types.Runtime
}

func (m *MockNodeCtx) GetRuntime() types.Runtime { return m.runtime }
func (m *MockNodeCtx) Config() types.ConfigMap   { return m.config }

type MockRuntime struct {
	types.Runtime
	engine types.MatrixEngine
	pool   types.NodePool
}

func (m *MockRuntime) GetEngine() types.MatrixEngine { return m.engine }
func (m *MockRuntime) GetNodePool() types.NodePool   { return m.pool } // Assuming this exists or using context injection

type MockEngine struct {
	types.MatrixEngine
	bizConfig types.ConfigMap
}

func (m *MockEngine) BizConfig() types.ConfigMap { return m.bizConfig }

func (m *MockEngine) GetEngineConfig(path string) (any, bool) {
	if m.bizConfig == nil {
		return nil, false
	}
	val, ok := m.bizConfig[path]
	return val, ok
}

type MockNodePool struct {
	types.NodePool
	mock.Mock
}

func (m *MockNodePool) GetInstance(id string) (any, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func TestAssetResolve_RuleMsg(t *testing.T) {
	InitRegistry()

	msg := &MockRuleMsg{
		data:     `{"key": "value", "nested": {"foo": "bar"}}`,
		metadata: types.Metadata{"trace_id": "12345"},
	}

	ctx := NewAssetContext(WithRuleMsg(msg))

	// Test Data Extraction
	a1 := Asset[string]{URI: "rulemsg://data/key?format=JSON"}
	v1, err := a1.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "value", v1)

	a1BadFormat := Asset[string]{URI: "rulemsg://data/key?format=TEXT"}
	_, err = a1BadFormat.Resolve(ctx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "rulemsg data format mismatch")

	// Test Data Writing
	// NOTE: partial updates to rulemsg data are not allowed: rulemsg://data/key
	// We need to test full data write instead.
	a1Full := Asset[string]{URI: "rulemsg://data?format=JSON"}
	err = a1Full.Set(ctx, "new_value")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "data is not valid JSON")

	a1MissingFormat := Asset[string]{URI: "rulemsg://data"}
	err = a1MissingFormat.Set(ctx, `{"key":"new_value"}`)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "valid data format is required")

	err = a1Full.Set(ctx, `{"key":"new_value"}`)
	assert.NoError(t, err)
	assert.Equal(t, types.Data(`{"key":"new_value"}`), msg.data)
	assert.Equal(t, cnst.JSON, msg.DataFormat())

	// Test Metadata Extraction
	a2 := Asset[string]{URI: "rulemsg://metadata/trace_id"}
	v2, err := a2.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "12345", v2)

	a2Missing := Asset[string]{URI: "rulemsg://metadata/missing"}
	_, err = a2Missing.Resolve(ctx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "metadata key not found: missing")

	// Test Metadata Writing
	err = a2.Set(ctx, "67890")
	assert.NoError(t, err)
	assert.Equal(t, "67890", msg.metadata["trace_id"])
}

func TestAssetResolve_Config(t *testing.T) {
	InitRegistry()

	nodeConfig := types.ConfigMap{"myKey": "nodeVal"}
	engineConfig := types.ConfigMap{"engineKey": "engineVal"}

	mockEngine := &MockEngine{bizConfig: engineConfig}
	mockRuntime := &MockRuntime{engine: mockEngine}
	mockNodeCtx := &MockNodeCtx{runtime: mockRuntime, config: nodeConfig}

	ctx := NewAssetContext(
		WithConfig(nodeConfig),
		WithNodeCtx(mockNodeCtx),
	)

	// Test Node Config
	a1 := Asset[string]{URI: "config:///myKey?scope=node"}
	v1, err := a1.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "nodeVal", v1)

	// Test Engine Config
	a2 := Asset[string]{URI: "config:///engineKey?scope=engine"}
	v2, err := a2.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "engineVal", v2)

	// Test Default
	a3 := Asset[string]{URI: "config:///missing?default=defVal"}
	v3, err := a3.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "defVal", v3)
}

func TestAssetResolve_Rel(t *testing.T) {
	InitRegistry()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "test_rel_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("hello world")
	assert.NoError(t, err)
	tmpFile.Close()

	a1 := Asset[string]{URI: "rel://" + tmpFile.Name()}
	v1, err := a1.Resolve(NewAssetContext())
	assert.NoError(t, err)
	assert.Equal(t, "hello world", v1)
}

func TestAssetResolve_Ref(t *testing.T) {
	InitRegistry()

	mockPool := new(MockNodePool)
	mockPool.On("GetInstance", "myDB").Return("db_connection_mock", nil)

	ctx := NewAssetContext(WithNodePool(mockPool))

	// Assuming WithNodeCtx is also needed if we use runtime lookup,
	// but our implementation prioritized getting from context options if available?
	// Let's check handlers_other.go implementation again...
	// It calls GetNodePool(ctx).

	a1 := Asset[string]{URI: "ref:///myDB"}
	v1, err := a1.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "db_connection_mock", v1)
}

func AssertErrorCode(t *testing.T, err error, expectedFault *types.Fault) {
	t.Helper()
	if customErr, ok := err.(*types.Fault); ok {
		assert.Equal(t, expectedFault.Code, customErr.Code)
	} else {
		assert.Fail(t, "error is not of type *types.Fault")
	}
}

func TestAssetResolve_NonURILiteral(t *testing.T) {
	InitRegistry()

	// 1. String with control characters (invalid for url.Parse)
	invalidURI := "not a uri\x01"
	a2 := Asset[string]{URI: invalidURI}
	_, err := a2.Resolve(NewAssetContext())
	assert.Error(t, err)
	AssertErrorCode(t, err, AssetInvalidURI)

	// 2. Regular string without scheme
	regular := "just a string"
	a3 := Asset[string]{URI: regular}
	_, err = a3.Resolve(NewAssetContext())
	assert.Error(t, err)
	AssertErrorCode(t, err, AssetSchemeNotRegistered)
}

func TestAssetResolve_EmptyURI(t *testing.T) {
	InitRegistry()

	// Test with an empty URI
	a := Asset[string]{URI: ""}
	_, err := a.Resolve(NewAssetContext())

	// Expect an error
	assert.Error(t, err)

	// Check if the error is the specific AssetInvalidURI fault
	AssertErrorCode(t, err, AssetInvalidURI)
}

func TestDecodeAssetString(t *testing.T) {
	type PromptConfig struct {
		System Asset[string] `json:"system"`
		User   Asset[string] `json:"user"`
	}

	data := map[string]any{
		"system": "rel://./rulechains/auto_run_system.txt",
		"user":   "rel://./rulechains/auto_run_user.txt",
	}

	cfg := &PromptConfig{}
	if err := utils.Decode(data, cfg); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if cfg.System.URI != data["system"] {
		t.Fatalf("unexpected system uri: %s", cfg.System.URI)
	}
	if cfg.User.URI != data["user"] {
		t.Fatalf("unexpected user uri: %s", cfg.User.URI)
	}
}
