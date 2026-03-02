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
//
// 兼容两种用法：
// 1) 传统 testify/mock：通过 On(...).Return(...) 配置期望。
// 2) 直喂数据模式：不配置 On 时，直接从 Runtimes 读取。
type MockRuntimePool struct {
	mock.Mock
	Runtimes map[string]types.Runtime
}

func (m *MockRuntimePool) Get(id string) (types.Runtime, bool) {
	if len(m.ExpectedCalls) == 0 {
		rt, ok := m.Runtimes[id]
		return rt, ok
	}
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(types.Runtime), args.Bool(1)
}

func (m *MockRuntimePool) Register(id string, runtime types.Runtime) error {
	if len(m.ExpectedCalls) == 0 {
		if m.Runtimes == nil {
			m.Runtimes = map[string]types.Runtime{}
		}
		m.Runtimes[id] = runtime
		return nil
	}
	args := m.Called(id, runtime)
	return args.Error(0)
}

func (m *MockRuntimePool) Unregister(id string) {
	if len(m.ExpectedCalls) == 0 {
		delete(m.Runtimes, id)
		return
	}
	m.Called(id)
}

func (m *MockRuntimePool) ListIDs() []string {
	if len(m.ExpectedCalls) == 0 {
		ids := make([]string, 0, len(m.Runtimes))
		for id := range m.Runtimes {
			ids = append(ids, id)
		}
		return ids
	}
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *MockRuntimePool) ListByViewType(viewType string) []types.Runtime {
	if len(m.ExpectedCalls) == 0 {
		return nil
	}
	args := m.Called(viewType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]types.Runtime)
}

func (m *MockRuntimePool) GetTriggers(chainID string) []types.TriggerSource {
	if len(m.ExpectedCalls) == 0 {
		return nil
	}
	args := m.Called(chainID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]types.TriggerSource)
}

func (m *MockRuntimePool) RegisterTrigger(targetChainID string, source types.TriggerSource) {
	if len(m.ExpectedCalls) == 0 {
		return
	}
	m.Called(targetChainID, source)
}

func (m *MockRuntimePool) UnregisterTrigger(targetChainID string, source types.TriggerSource) {
	if len(m.ExpectedCalls) == 0 {
		return
	}
	m.Called(targetChainID, source)
}

// ----------------------- MockEngine -----------------------
// MockEngine mocks the MatrixEngine interface.
//
// 兼容两种用法：
// 1) 传统 testify/mock：通过 On(...).Return(...) 配置。
// 2) 直喂数据模式：不配置 On 时，直接返回字段值。
type MockEngine struct {
	mock.Mock
	EngineConfig         map[string]any
	RuntimePoolValue     types.RuntimePool
	SharedNodePoolValue  types.NodePool
	NodeManagerValue     types.NodeManager
	NodeFuncManagerValue types.NodeFuncManager
	BizConfigValue       types.ConfigMap
	LoaderValue          types.ResourceProvider
	LoggerValue          types.Logger
}

func (m *MockEngine) GetEngineConfig(key string) (any, bool) {
	if len(m.ExpectedCalls) == 0 {
		v, ok := m.EngineConfig[key]
		return v, ok
	}
	args := m.Called(key)
	return args.Get(0), args.Bool(1)
}

func (m *MockEngine) RuntimePool() types.RuntimePool {
	if len(m.ExpectedCalls) == 0 {
		return m.RuntimePoolValue
	}
	args := m.Called()
	return args.Get(0).(types.RuntimePool)
}

func (m *MockEngine) SharedNodePool() types.NodePool {
	if len(m.ExpectedCalls) == 0 {
		return m.SharedNodePoolValue
	}
	args := m.Called()
	return args.Get(0).(types.NodePool)
}

func (m *MockEngine) NodeManager() types.NodeManager {
	if len(m.ExpectedCalls) == 0 {
		return m.NodeManagerValue
	}
	args := m.Called()
	return args.Get(0).(types.NodeManager)
}

func (m *MockEngine) NodeFuncManager() types.NodeFuncManager {
	if len(m.ExpectedCalls) == 0 {
		return m.NodeFuncManagerValue
	}
	args := m.Called()
	return args.Get(0).(types.NodeFuncManager)
}

func (m *MockEngine) BizConfig() types.ConfigMap {
	if len(m.ExpectedCalls) == 0 {
		return m.BizConfigValue
	}
	args := m.Called()
	return args.Get(0).(types.ConfigMap)
}

func (m *MockEngine) Loader() types.ResourceProvider {
	if len(m.ExpectedCalls) == 0 {
		return m.LoaderValue
	}
	args := m.Called()
	return args.Get(0).(types.ResourceProvider)
}

func (m *MockEngine) Logger() types.Logger {
	if len(m.ExpectedCalls) == 0 {
		return m.LoggerValue
	}
	args := m.Called()
	return args.Get(0).(types.Logger)
}
