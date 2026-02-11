package asset_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/neohetj/matrix/pkg/asset"
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
	dataT    types.DataT
}

func (m *MockRuleMsg) Data() types.Data                 { return m.data }
func (m *MockRuleMsg) Metadata() types.Metadata         { return m.metadata }
func (m *MockRuleMsg) DataT() types.DataT               { return m.dataT }
func (m *MockRuleMsg) DataFormat() cnst.MFormat         { return cnst.JSON } // Mock as needed
func (m *MockRuleMsg) SetData(d string, f cnst.MFormat) { m.data = types.Data(d) }

type MockNodeCtx struct {
	types.NodeCtx
	mock.Mock
	config  types.ConfigMap
	runtime types.Runtime
}

func (m *MockNodeCtx) GetRuntime() types.Runtime { return m.runtime }
func (m *MockNodeCtx) Config() types.ConfigMap   { return m.config }
func (m *MockNodeCtx) Warn(msg string, fields ...any) {
	m.Called(msg, fields)
}
func (m *MockNodeCtx) Debug(msg string, fields ...any) {}
func (m *MockNodeCtx) Info(msg string, fields ...any)  {}
func (m *MockNodeCtx) Error(msg string, fields ...any) {}

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

type MockDataT struct {
	types.DataT
	objects map[string]types.CoreObj
}

func (m *MockDataT) Get(id string) (types.CoreObj, bool) {
	o, ok := m.objects[id]
	return o, ok
}

type MockCoreObj struct {
	types.CoreObj
	body any
}

func (m *MockCoreObj) Body() any { return m.body }

func TestAssetResolve_RuleMsg_DataT_StructToMap(t *testing.T) {
	myObj := &MockCoreObj{body: &TestStruct{Name: "Foo"}}
	dataT := &MockDataT{objects: map[string]types.CoreObj{"obj1": myObj}}
	msg := &MockRuleMsg{dataT: dataT}

	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	a := asset.Asset[map[string]any]{URI: "rulemsg://dataT/obj1?sid=MapStringInterface"}
	val, err := a.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Foo", val["name"])
}

func TestAssetResolve_RuleMsg(t *testing.T) {
	asset.InitRegistry()

	msg := &MockRuleMsg{
		data:     `{"key": "value", "nested": {"foo": "bar"}}`,
		metadata: types.Metadata{"trace_id": "12345"},
	}

	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	// Test Data Extraction
	a1 := asset.Asset[string]{URI: "rulemsg://data/key?format=JSON"}
	v1, err := a1.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "value", v1)

	a1BadFormat := asset.Asset[string]{URI: "rulemsg://data/key?format=TEXT"}
	_, err = a1BadFormat.Resolve(ctx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "rulemsg data format mismatch")

	// Test Data Writing
	// NOTE: partial updates to rulemsg data are not allowed: rulemsg://data/key
	// We need to test full data write instead.
	a1Full := asset.Asset[string]{URI: "rulemsg://data?format=JSON"}
	err = a1Full.Set(ctx, "new_value")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "data is not valid JSON")

	a1MissingFormat := asset.Asset[string]{URI: "rulemsg://data"}
	err = a1MissingFormat.Set(ctx, `{"key":"new_value"}`)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "valid data format is required")

	err = a1Full.Set(ctx, `{"key":"new_value"}`)
	assert.NoError(t, err)
	assert.Equal(t, types.Data(`{"key":"new_value"}`), msg.data)
	assert.Equal(t, cnst.JSON, msg.DataFormat())

	// Test Metadata Extraction
	a2 := asset.Asset[string]{URI: "rulemsg://metadata/trace_id"}
	v2, err := a2.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "12345", v2)

	a2Missing := asset.Asset[string]{URI: "rulemsg://metadata/missing"}
	_, err = a2Missing.Resolve(ctx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "metadata key not found: missing")

	// Test Metadata Writing
	err = a2.Set(ctx, "67890")
	assert.NoError(t, err)
	assert.Equal(t, "67890", msg.metadata["trace_id"])
}

func TestAssetResolve_Config(t *testing.T) {
	asset.InitRegistry()

	nodeConfig := types.ConfigMap{"myKey": "nodeVal"}
	engineConfig := types.ConfigMap{"engineKey": "engineVal"}

	mockEngine := &MockEngine{bizConfig: engineConfig}
	mockRuntime := &MockRuntime{engine: mockEngine}
	mockNodeCtx := &MockNodeCtx{runtime: mockRuntime, config: nodeConfig}

	ctx := asset.NewAssetContext(
		asset.WithConfig(nodeConfig),
		asset.WithNodeCtx(mockNodeCtx),
	)

	// Test Node Config
	a1 := asset.Asset[string]{URI: "config:///myKey?scope=node"}
	v1, err := a1.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "nodeVal", v1)

	// Test Engine Config
	a2 := asset.Asset[string]{URI: "config:///engineKey?scope=engine"}
	v2, err := a2.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "engineVal", v2)

	// Test Default
	a3 := asset.Asset[string]{URI: "config:///missing?default=defVal"}
	v3, err := a3.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "defVal", v3)
}

func TestAssetResolve_Rel(t *testing.T) {
	asset.InitRegistry()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "test_rel_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("hello world")
	assert.NoError(t, err)
	tmpFile.Close()

	a1 := asset.Asset[string]{URI: "rel://" + tmpFile.Name()}
	v1, err := a1.Resolve(asset.NewAssetContext())
	assert.NoError(t, err)
	assert.Equal(t, "hello world", v1)
}

func TestAssetResolve_Ref(t *testing.T) {
	asset.InitRegistry()

	mockPool := new(MockNodePool)
	mockPool.On("GetInstance", "myDB").Return("db_connection_mock", nil)

	ctx := asset.NewAssetContext(asset.WithNodePool(mockPool))

	// Assuming WithNodeCtx is also needed if we use runtime lookup,
	// but our implementation prioritized getting from context options if available?
	// Let's check handlers_other.go implementation again...
	// It calls GetNodePool(ctx).

	a1 := asset.Asset[string]{URI: "ref:///myDB"}
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
	asset.InitRegistry()

	// 1. String with control characters (invalid for url.Parse)
	invalidURI := "not a uri\x01"
	a2 := asset.Asset[string]{URI: invalidURI}
	_, err := a2.Resolve(asset.NewAssetContext())
	assert.Error(t, err)
	AssertErrorCode(t, err, asset.AssetInvalidURI)

	// 2. Regular string without scheme
	regular := "just a string"
	a3 := asset.Asset[string]{URI: regular}
	_, err = a3.Resolve(asset.NewAssetContext())
	assert.Error(t, err)
	AssertErrorCode(t, err, asset.AssetSchemeNotRegistered)
}

func TestAssetResolve_EmptyURI(t *testing.T) {
	asset.InitRegistry()

	// Test with an empty URI
	a := asset.Asset[string]{URI: ""}
	_, err := a.Resolve(asset.NewAssetContext())

	// Expect an error
	assert.Error(t, err)

	// Check if the error is the specific asset.AssetInvalidURI fault
	AssertErrorCode(t, err, asset.AssetInvalidURI)
}

func TestDecodeAssetString(t *testing.T) {
	type PromptConfig struct {
		System asset.Asset[string] `json:"system"`
		User   asset.Asset[string] `json:"user"`
	}

	data := map[string]any{
		"system": "rel://./prompts/auto_run_system.txt",
		"user":   "rel://./prompts/auto_run_user.txt",
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

type MockSchemeHandler struct {
	Val any
}

func (m *MockSchemeHandler) Handle(uri *url.URL, ctx *asset.AssetContext) (any, error) {
	return m.Val, nil
}
func (m *MockSchemeHandler) Set(uri *url.URL, ctx *asset.AssetContext, value any) error {
	return nil
}
func (m *MockSchemeHandler) NormalizeAssetURI(uri string) string { return uri }

type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestAssetResolve_StructToMapConversion(t *testing.T) {
	mockHandler := &MockSchemeHandler{
		Val: &TestStruct{Name: "Test", Age: 30},
	}
	asset.RegisterScheme("mockstruct", mockHandler)

	a := asset.Asset[map[string]any]{URI: "mockstruct:///data"}
	ctx := asset.NewAssetContext()

	val, err := a.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Test", val["name"])
	// JSON unmarshal usually converts numbers to float64 for map[string]any
	assert.Equal(t, float64(30), val["age"])
}

type TestStruct2 struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestAssetResolve_StructToStructConversion(t *testing.T) {
	mockHandler := &MockSchemeHandler{
		Val: &TestStruct{Name: "Test", Age: 30}, // Returns TestStruct
	}
	asset.RegisterScheme("mockstruct2", mockHandler)

	// Resolve as TestStruct2
	a := asset.Asset[*TestStruct2]{URI: "mockstruct2:///data"}
	ctx := asset.NewAssetContext()

	val, err := a.Resolve(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, "Test", val.Name)
	assert.Equal(t, 30, val.Age)
}

func TestAssetResolve_PointerStringToString(t *testing.T) {
	asset.InitRegistry()

	ptrVal := "mock_value"
	mockHandler := &MockSchemeHandler{Val: &ptrVal}
	asset.RegisterScheme("mockstrptr", mockHandler)

	a := asset.Asset[string]{URI: "mockstrptr:///data"}
	ctx := asset.NewAssetContext()

	val, err := a.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ptrVal, val)
}

// Local definition with same name as types.NodeMetadata
// Must match all fields of types.NodeMetadata because utils.Decode uses ErrorUnused: true
type NodeMetadata struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Dimension   string   `json:"dimension"`
	Tags        []string `json:"tags"`
	Version     string   `json:"version"`
	Icon        string   `json:"icon,omitempty"`
	NodeReads   []any    `json:"nodeReads,omitempty"`
	NodeWrites  []any    `json:"nodeWrites,omitempty"`
}

func TestAssetResolve_TypeCoercionLog(t *testing.T) {
	// Source is types.NodeMetadata
	srcVal := &types.NodeMetadata{
		Type: "testType",
		Name: "testName",
	}

	mockHandler := &MockSchemeHandler{
		Val: srcVal,
	}
	asset.RegisterScheme("mocklog", mockHandler)

	// Target is asset_test.NodeMetadata (same name, different pkg)
	a := asset.Asset[*NodeMetadata]{URI: "mocklog:///data"}

	// Setup Mock Context with Logger
	mockNodeCtx := new(MockNodeCtx)
	// Expect Warn to be called
	mockNodeCtx.On("Warn",
		"Type coercion triggered for identical struct names (likely package mismatch)",
		mock.Anything, // fields
	).Return()

	ctx := asset.NewAssetContext(asset.WithNodeCtx(mockNodeCtx))

	val, err := a.Resolve(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, "testType", val.Type)

	mockNodeCtx.AssertExpectations(t)
}
