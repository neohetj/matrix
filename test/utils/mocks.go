package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/mock"
)

// ----------------------- MockRuleMsg -----------------------
// MockRuleMsg
type MockRuleMsg struct {
	mock.Mock
}

func (m *MockRuleMsg) ID() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockRuleMsg) Ts() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}
func (m *MockRuleMsg) Type() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockRuleMsg) DataFormat() cnst.MFormat {
	args := m.Called()
	return args.Get(0).(cnst.MFormat)
}
func (m *MockRuleMsg) WithDataFormat(dataFormat cnst.MFormat) types.RuleMsg {
	args := m.Called(dataFormat)
	return args.Get(0).(types.RuleMsg)
}
func (m *MockRuleMsg) Data() types.Data {
	args := m.Called()
	return args.Get(0).(types.Data)
}
func (m *MockRuleMsg) DataT() types.DataT {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(types.DataT)
}
func (m *MockRuleMsg) Metadata() types.Metadata {
	args := m.Called()
	return args.Get(0).(types.Metadata)
}
func (m *MockRuleMsg) SetData(data string, format cnst.MFormat) {
	m.Called(data, format)
}
func (m *MockRuleMsg) SetMetadata(metadata types.Metadata) {
	m.Called(metadata)
}
func (m *MockRuleMsg) Copy() types.RuleMsg {
	args := m.Called()
	return args.Get(0).(types.RuleMsg)
}
func (m *MockRuleMsg) DeepCopy() (types.RuleMsg, error) {
	args := m.Called()
	return args.Get(0).(types.RuleMsg), args.Error(1)
}

// ----------------------- MockDataT -----------------------
// MockDataT
type MockDataT struct {
	mock.Mock
}

func (m *MockDataT) Get(objId string) (types.CoreObj, bool) {
	args := m.Called(objId)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(types.CoreObj), args.Bool(1)
}
func (m *MockDataT) Set(objId string, value types.CoreObj) {
	m.Called(objId, value)
}
func (m *MockDataT) NewItem(sid, objId string) (types.CoreObj, error) {
	args := m.Called(sid, objId)
	return args.Get(0).(types.CoreObj), args.Error(1)
}
func (m *MockDataT) GetAll() map[string]types.CoreObj {
	args := m.Called()
	return args.Get(0).(map[string]types.CoreObj)
}
func (m *MockDataT) Copy() types.DataT {
	args := m.Called()
	return args.Get(0).(types.DataT)
}
func (m *MockDataT) DeepCopy() (types.DataT, error) {
	args := m.Called()
	return args.Get(0).(types.DataT), args.Error(1)
}
func (m *MockDataT) GetByParam(ctx types.NodeCtx, pname string) (types.CoreObj, error) {
	args := m.Called(ctx, pname)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(types.CoreObj), args.Error(1)
}
func (m *MockDataT) NewItemByParam(ctx types.NodeCtx, pname string) (types.CoreObj, error) {
	args := m.Called(ctx, pname)
	return args.Get(0).(types.CoreObj), args.Error(1)
}

// ----------------------- MockCoreObj -----------------------
// MockCoreObj
type MockCoreObj struct {
	mock.Mock
}

func (m *MockCoreObj) Key() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockCoreObj) Definition() types.CoreObjDef {
	args := m.Called()
	return args.Get(0).(types.CoreObjDef)
}
func (m *MockCoreObj) Body() any {
	args := m.Called()
	return args.Get(0)
}
func (m *MockCoreObj) SetBody(body any) error {
	args := m.Called(body)
	return args.Error(0)
}
func (m *MockCoreObj) DeepCopy() (types.CoreObj, error) {
	args := m.Called()
	return args.Get(0).(types.CoreObj), args.Error(1)
}

// ----------------------- TestLogger -----------------------

// TestLogger is a simple logger that writes to stdout for testing purposes.
type TestLogger struct{}

func (l *TestLogger) Printf(ctx context.Context, format string, v ...any) {
	log.Printf(format, v...)
}
func (l *TestLogger) Debugf(ctx context.Context, format string, v ...any) {
	log.Printf("[DEBUG] "+format, v...)
}
func (l *TestLogger) Infof(ctx context.Context, format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}
func (l *TestLogger) Warnf(ctx context.Context, format string, v ...any) {
	log.Printf("[WARN] "+format, v...)
}
func (l *TestLogger) Errorf(ctx context.Context, format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
}
func (l *TestLogger) With(fields ...any) types.Logger { return l }

// ----------------------- MockLogger -----------------------
// MockLogger is a logger that captures output for assertion in tests.
type MockLogger struct {
	buf    bytes.Buffer
	mu     sync.Mutex
	fields []any
}

func (m *MockLogger) buildMessage(format string, v ...any) string {
	msg := fmt.Sprintf(format, v...)
	if len(m.fields) > 0 {
		msg = fmt.Sprintf("%s fields=%+v", msg, m.fields)
	}
	return msg
}

func (m *MockLogger) Printf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.buildMessage(format, v...)
	log.Println(msg) // Also print to console
	log.New(&m.buf, "", 0).Println(msg)
}

func (m *MockLogger) Debugf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.buildMessage(format, v...)
	log.Println("DEBUG: " + msg) // Also print to console
	log.New(&m.buf, "DEBUG: ", 0).Println(msg)
}

func (m *MockLogger) Infof(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.buildMessage(format, v...)
	log.Println("INFO: " + msg) // Also print to console
	log.New(&m.buf, "INFO: ", 0).Println(msg)
}

func (m *MockLogger) Warnf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.buildMessage(format, v...)
	log.Println("WARN: " + msg) // Also print to console
	log.New(&m.buf, "WARN: ", 0).Println(msg)
}

func (m *MockLogger) Errorf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.buildMessage(format, v...)
	log.Println("ERROR: " + msg) // Also print to console
	log.New(&m.buf, "ERROR: ", 0).Println(msg)
}

func (m *MockLogger) With(fields ...any) types.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fields = fields
	return m
}

func (m *MockLogger) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}
func (m *MockLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf.Reset()
}

// ----------------------- MockNodeCtx -----------------------

type MockNodeCtx struct {
	types.NodeCtx
	Ctx                 context.Context
	SuccessMsg          types.RuleMsg
	FailureMsg          types.RuleMsg
	FailureErr          error
	NodeDef             types.NodeDef
	NodeIDValue         string
	PreviousNodeIDValue string
	ChainIDValue        string
	ChainConfigValue    types.ConfigMap
	chainInstance       types.ChainInstance
	runtime             types.Runtime
}

type MockNodeCtxOption func(*MockNodeCtx)

func NewMockNodeCtx(opts ...MockNodeCtxOption) *MockNodeCtx {
	ctx := &MockNodeCtx{
		Ctx:         context.Background(),
		NodeIDValue: "test-node",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(ctx)
		}
	}
	return ctx
}

func WithTestNodeConfig(config map[string]any) MockNodeCtxOption {
	return func(ctx *MockNodeCtx) {
		ctx.NodeDef.Configuration = types.ConfigMap(config)
	}
}
func (m *MockNodeCtx) GetContext() context.Context { return m.Ctx }
func (m *MockNodeCtx) SetContext(ctx context.Context) {
	m.Ctx = ctx
}
func (m *MockNodeCtx) ChainConfig() types.ConfigMap {
	return m.ChainConfigValue
}
func (m *MockNodeCtx) ChainID() string { return m.ChainIDValue }
func (m *MockNodeCtx) NodeID() string {
	if m.NodeIDValue == "" {
		return "test-node"
	}
	return m.NodeIDValue
}
func (m *MockNodeCtx) PreviousNodeID() string {
	return m.PreviousNodeIDValue
}
func (m *MockNodeCtx) GetNode() types.Node { return nil }
func (m *MockNodeCtx) GetRuntime() types.Runtime {
	return m.runtime
}
func (m *MockNodeCtx) SetRuntime(r types.Runtime) {
	m.runtime = r
}
func (m *MockNodeCtx) TellSuccess(msg types.RuleMsg) { m.SuccessMsg = msg }
func (m *MockNodeCtx) TellFailure(msg types.RuleMsg, err error) {
	m.FailureMsg = msg
	m.FailureErr = err
}
func (m *MockNodeCtx) HandleError(msg types.RuleMsg, err error) {
	m.TellFailure(msg, err)
}
func (m *MockNodeCtx) TellNext(msg types.RuleMsg, relationTypes ...string) {}
func (m *MockNodeCtx) NewMsg(msgType string, metaData types.Metadata, data string) types.RuleMsg {
	return contract.NewDefaultRuleMsg(msgType, data, metaData, contract.NewDataT())
}
func (m *MockNodeCtx) Config() types.ConfigMap         { return m.NodeDef.Configuration }
func (m *MockNodeCtx) SelfDef() *types.NodeDef         { return &m.NodeDef }
func (m *MockNodeCtx) SetOnAllNodesCompleted(f func()) {}
func (m *MockNodeCtx) GetInstance() (any, error) {
	return m, nil
}
func (m *MockNodeCtx) Logger() types.Logger {
	return &TestLogger{}
}
func (m *MockNodeCtx) Info(msg string, fields ...any) {
	m.Logger().Infof(m.GetContext(), msg, fields...)
}
func (m *MockNodeCtx) Debug(msg string, fields ...any) {
	m.Logger().Debugf(m.GetContext(), msg, fields...)
}
func (m *MockNodeCtx) Warn(msg string, fields ...any) {
	m.Logger().Warnf(m.GetContext(), msg, fields...)
}
func (m *MockNodeCtx) Error(msg string, fields ...any) {
	m.Logger().Errorf(m.GetContext(), msg, fields...)
}

// SetChainInstance sets the chain instance for the mock context.
func (m *MockNodeCtx) SetChainInstance(instance types.ChainInstance) {
	m.chainInstance = instance
}

// ChainInstance returns the configured chain instance.
func (m *MockNodeCtx) ChainInstance() types.ChainInstance {
	return m.chainInstance
}

// ----------------------- MockEndpoint -----------------------
// MockEndpoint is a mock implementation of types.Endpoint.
type MockEndpoint struct {
	MockNode
	RuntimePool types.RuntimePool
}

func (m *MockEndpoint) SetRuntimePool(pool any) error {
	if p, ok := pool.(types.RuntimePool); ok {
		m.RuntimePool = p
	}
	return nil
}

func (m *MockEndpoint) GetInstance() (any, error) {
	return m, nil
}

// ----------------------- MockScheduler -----------------------
// MockScheduler implements types.Scheduler
type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Submit(task func()) error {
	args := m.Called(task)
	// Execute the task synchronously for testing purposes if configured to do so
	if args.Bool(1) {
		task()
	}
	return args.Error(0)
}

func (m *MockScheduler) Stop() {
	m.Called()
}

// ----------------------- MockChainInstance -----------------------
// MockChainInstance implements types.ChainInstance
type MockChainInstance struct {
	mock.Mock
}

func (m *MockChainInstance) GetNode(id string) (types.Node, bool) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(types.Node), args.Bool(1)
}

func (m *MockChainInstance) GetNodeDef(id string) (*types.NodeDef, bool) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*types.NodeDef), args.Bool(1)
}

func (m *MockChainInstance) GetConnections(fromNodeID string) []types.Connection {
	args := m.Called(fromNodeID)
	return args.Get(0).([]types.Connection)
}

func (m *MockChainInstance) Definition() *types.RuleChainDef {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*types.RuleChainDef)
}

func (m *MockChainInstance) GetRootNodeIDs() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockChainInstance) GetAllNodes() map[string]types.Node {
	args := m.Called()
	return args.Get(0).(map[string]types.Node)
}

func (m *MockChainInstance) Destroy() {
	m.Called()
}

// ----------------------- MockNode -----------------------
// MockNode is a mock implementation of types.Node.
type MockNode struct {
	mock.Mock
}

func (m *MockNode) Init(config types.ConfigMap) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	m.Called(ctx, msg)
}

func (m *MockNode) Destroy() {
	m.Called()
}

func (m *MockNode) Type() types.NodeType {
	args := m.Called()
	return args.Get(0).(types.NodeType)
}

func (m *MockNode) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockNode) SetID(id string) {
	m.Called(id)
}

func (m *MockNode) SetName(name string) {
	m.Called(name)
}

func (m *MockNode) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockNode) DataContract() types.DataContract {
	args := m.Called()
	return args.Get(0).(types.DataContract)
}

func (m *MockNode) New() types.Node {
	args := m.Called()
	return args.Get(0).(types.Node)
}

func (m *MockNode) NodeMetadata() types.NodeMetadata {
	args := m.Called()
	return args.Get(0).(types.NodeMetadata)
}

func (m *MockNode) Errors() []*types.Fault {
	args := m.Called()
	return args.Get(0).([]*types.Fault)
}

func (m *MockNode) ConfigSchema() *openapi3.Schema {
	args := m.Called()
	return args.Get(0).(*openapi3.Schema)
}

// ----------------------- MockAspect -----------------------
// MockAspect implements types.Aspect
type MockAspect struct {
	mock.Mock
}

func (m *MockAspect) Before(ctx types.NodeCtx, msg types.RuleMsg) (types.RuleMsg, error) {
	args := m.Called(ctx, msg)
	return args.Get(0).(types.RuleMsg), args.Error(1)
}

func (m *MockAspect) After(ctx types.NodeCtx, msg types.RuleMsg, err error) {
	m.Called(ctx, msg, err)
}

// ----------------------- MockNodeManager -----------------------
// MockNodeManager is a mock implementation of types.NodeManager.
type MockNodeManager struct {
	NodePrototypes map[string]types.Node
}

func (m *MockNodeManager) Register(node types.Node) error           { return nil }
func (m *MockNodeManager) Unregister(nodeType types.NodeType) error { return nil }
func (m *MockNodeManager) Get(nodeType types.NodeType) (types.Node, bool) {
	if node, ok := m.NodePrototypes[string(nodeType)]; ok {
		return node, true
	}
	return nil, false
}
func (m *MockNodeManager) GetComponents() map[types.NodeType]types.Node {
	return nil
}
func (m *MockNodeManager) NewNode(nodeType types.NodeType) (types.Node, error) {
	if node, ok := m.NodePrototypes[string(nodeType)]; ok {
		return node, nil
	}
	return nil, errors.New("node not found")
}

// ----------------------- MockNodePool -----------------------
// MockNodePool is a mock implementation of types.NodePool.
type MockNodePool struct {
	Nodes map[string]types.NodeCtx
}

func (m *MockNodePool) Get(id string) (types.SharedNodeCtx, bool) {
	ctx, ok := m.Nodes[id]
	if !ok {
		return nil, false
	}
	return ctx.(types.SharedNodeCtx), true
}

func (m *MockNodePool) GetAll() []types.NodeCtx {
	return nil
}

func (m *MockNodePool) GetEndpoints() []types.Endpoint {
	return nil
}

func (m *MockNodePool) GetInstance(id string) (any, error) {
	ctx, ok := m.Get(id)
	if !ok {
		return nil, errors.New("not found")
	}
	return ctx, nil
}

func (m *MockNodePool) Load(dsl []byte, mgr types.NodeManager) (types.NodePool, error) {
	return m, nil
}

func (m *MockNodePool) NewFromNodeDef(def types.NodeDef, mgr types.NodeManager) (types.SharedNodeCtx, error) {
	if _, ok := m.Nodes[def.ID]; ok {
		return nil, errors.New("node already exists")
	}
	_, ok := mgr.Get(types.NodeType(def.Type))
	if !ok {
		return nil, errors.New("node not found")
	}
	ctx := NewMockNodeCtx()
	m.Nodes[def.ID] = ctx
	return ctx, nil
}

func (m *MockNodePool) GetNodeContext(id string) (types.NodeCtx, bool) {
	ctx, ok := m.Nodes[id]
	return ctx, ok
}

func (m *MockNodePool) LoadFromRuleChainDef(def *types.RuleChainDef, mgr types.NodeManager) (types.NodePool, error) {
	for _, nodeDef := range def.Metadata.Nodes {
		_, err := m.NewFromNodeDef(nodeDef, mgr)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}
func (m *MockNodePool) AddEndpoint(endpoint types.Endpoint) {}
func (m *MockNodePool) Del(id string)                       {}
func (m *MockNodePool) Stop()                               {}

// ----------------------- MockRuntime -----------------------
// MockRuntime implements types.Runtime
type MockRuntime struct {
	mock.Mock
	ChainInstance types.ChainInstance
}

func (m *MockRuntime) Execute(ctx context.Context, fromNodeID string, msg types.RuleMsg, onEnd func(msg types.RuleMsg, err error)) error {
	args := m.Called(ctx, fromNodeID, msg, onEnd)
	return args.Error(0)
}

func (m *MockRuntime) ExecuteAndWait(ctx context.Context, fromNodeID string, msg types.RuleMsg, onEnd func(msg types.RuleMsg, err error)) (types.RuleMsg, error) {
	args := m.Called(ctx, fromNodeID, msg, onEnd)
	return args.Get(0).(types.RuleMsg), args.Error(1)
}

func (m *MockRuntime) Reload(newChainDef *types.RuleChainDef) error {
	args := m.Called(newChainDef)
	return args.Error(0)
}

func (m *MockRuntime) Destroy() {
	m.Called()
}

func (m *MockRuntime) Definition() *types.RuleChainDef {
	args := m.Called()
	return args.Get(0).(*types.RuleChainDef)
}

func (m *MockRuntime) GetNodePool() types.NodePool {
	args := m.Called()
	return args.Get(0).(types.NodePool)
}

func (m *MockRuntime) GetEngine() types.MatrixEngine {
	args := m.Called()
	return args.Get(0).(types.MatrixEngine)
}

func (m *MockRuntime) GetChainInstance() types.ChainInstance {
	if m.ChainInstance != nil {
		return m.ChainInstance
	}
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(types.ChainInstance)
}

// ----------------------- MockRuntimePool -----------------------
// MockRuntimePool is a mock implementation of types.RuntimePool.
type MockRuntimePool struct{}

func (m *MockRuntimePool) Get(id string) (types.Runtime, bool)  { return nil, false }
func (m *MockRuntimePool) Put(id string, runtime types.Runtime) {}
func (m *MockRuntimePool) Remove(id string)                     {}
func (m *MockRuntimePool) ListByViewType(viewType string) []types.Runtime {
	return nil
}
func (m *MockRuntimePool) ListIDs() []string {
	return nil
}
func (m *MockRuntimePool) Register(id string, runtime types.Runtime) error { return nil }
func (m *MockRuntimePool) Unregister(id string)                            {}

// ----------------------- MockResourceProvider -----------------------
// MockResourceProvider is a mock implementation of types.ResourceProvider.
type MockResourceProvider struct {
	Files map[string]struct {
		Content string
		IsDir   bool
	}
}

func (m *MockResourceProvider) WalkDir(root string, fn fs.WalkDirFunc) error {
	for path, file := range m.Files {
		if strings.HasPrefix(path, root) {
			parts := strings.Split(path, "/")
			filename := parts[len(parts)-1]
			d := &MockDirEntry{name: filename, isDir: file.IsDir}
			if err := fn(path, d, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MockResourceProvider) Priority() int {
	return 0
}

func (m *MockResourceProvider) Name() string {
	return "mock"
}

func (m *MockResourceProvider) Open(name string) (fs.File, error) {
	if file, ok := m.Files[name]; ok && !file.IsDir {
		return &MockFSFile{name: name, content: file.Content}, nil
	}
	return nil, fs.ErrNotExist
}

func (m *MockResourceProvider) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	for path, file := range m.Files {
		if strings.HasPrefix(path, name) {
			parts := strings.Split(path, "/")
			filename := parts[len(parts)-1]
			entries = append(entries, &MockDirEntry{name: filename, isDir: file.IsDir})
		}
	}
	return entries, nil
}

func (m *MockResourceProvider) ReadFile(name string) (*types.Resource, error) {
	if file, ok := m.Files[name]; ok {
		return &types.Resource{Content: []byte(file.Content), Source: types.FromExternal}, nil
	}
	return nil, fs.ErrNotExist
}

func (m *MockResourceProvider) Stat(name string) (fs.FileInfo, error) {
	if _, ok := m.Files[name]; ok {
		return &MockFileInfo{}, nil
	}
	return nil, fs.ErrNotExist
}

// ----------------------- MockFSFile -----------------------
// MockFSFile is a mock implementation of fs.File.
type MockFSFile struct {
	name    string
	content string
	offset  int64
}

func (f *MockFSFile) Stat() (fs.FileInfo, error) { return &MockFileInfo{name: f.name}, nil }
func (f *MockFSFile) Read(b []byte) (int, error) {
	n := copy(b, f.content[f.offset:])
	f.offset += int64(n)
	return n, nil
}
func (f *MockFSFile) Close() error { return nil }

// ----------------------- MockFileInfo -----------------------
// MockFileInfo is a mock implementation of fs.FileInfo.
type MockFileInfo struct {
	name  string
	isDir bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return 0 }
func (m *MockFileInfo) Mode() fs.FileMode  { return 0 }
func (m *MockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

// ----------------------- MockDirEntry -----------------------
// MockDirEntry is a mock implementation of fs.DirEntry.
type MockDirEntry struct {
	name  string
	isDir bool
}

func (m *MockDirEntry) Name() string      { return m.name }
func (m *MockDirEntry) IsDir() bool       { return m.isDir }
func (m *MockDirEntry) Type() fs.FileMode { return 0 }
func (m *MockDirEntry) Info() (fs.FileInfo, error) {
	return &MockFileInfo{name: m.name, isDir: m.isDir}, nil
}
