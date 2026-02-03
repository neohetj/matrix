package utils

import (
	"context"

	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/mock"
)

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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
type MockRuntimePool struct {
	mock.Mock
}

func (m *MockRuntimePool) Get(id string) (types.Runtime, bool) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(types.Runtime), args.Bool(1)
}

func (m *MockRuntimePool) Register(id string, runtime types.Runtime) error {
	args := m.Called(id, runtime)
	return args.Error(0)
}

func (m *MockRuntimePool) Unregister(id string) {
	m.Called(id)
}

func (m *MockRuntimePool) ListIDs() []string {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *MockRuntimePool) ListByViewType(viewType string) []types.Runtime {
	args := m.Called(viewType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]types.Runtime)
}

func (m *MockRuntimePool) GetTriggers(chainID string) []types.TriggerSource {
	args := m.Called(chainID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]types.TriggerSource)
}

func (m *MockRuntimePool) RegisterTrigger(targetChainID string, source types.TriggerSource) {
	m.Called(targetChainID, source)
}

func (m *MockRuntimePool) UnregisterTrigger(targetChainID string, source types.TriggerSource) {
	m.Called(targetChainID, source)
}

// ----------------------- MockEngine -----------------------
// MockEngine mocks the MatrixEngine interface.
type MockEngine struct {
	mock.Mock
}

func (m *MockEngine) GetEngineConfig(key string) (any, bool) {
	args := m.Called(key)
	return args.Get(0), args.Bool(1)
}

func (m *MockEngine) RuntimePool() types.RuntimePool {
	args := m.Called()
	return args.Get(0).(types.RuntimePool)
}

func (m *MockEngine) SharedNodePool() types.NodePool {
	args := m.Called()
	return args.Get(0).(types.NodePool)
}

func (m *MockEngine) NodeManager() types.NodeManager {
	args := m.Called()
	return args.Get(0).(types.NodeManager)
}

func (m *MockEngine) NodeFuncManager() types.NodeFuncManager {
	args := m.Called()
	return args.Get(0).(types.NodeFuncManager)
}

func (m *MockEngine) BizConfig() types.ConfigMap {
	args := m.Called()
	return args.Get(0).(types.ConfigMap)
}

func (m *MockEngine) Loader() types.ResourceProvider {
	args := m.Called()
	return args.Get(0).(types.ResourceProvider)
}

func (m *MockEngine) Logger() types.Logger {
	args := m.Called()
	return args.Get(0).(types.Logger)
}
