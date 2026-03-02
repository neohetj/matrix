package utils

import (
	"context"
	"errors"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/mock"
)

// ----------------------- MockNodeCtx -----------------------

type MockNodeCtx struct {
	types.NodeCtx
	Ctx                 context.Context
	SuccessMsg          types.RuleMsg
	FailureMsg          types.RuleMsg
	FailureErr          error
	NodeDef             types.NodeDef
	NodeValue           types.Node
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

func WithNodeValue(node types.Node) MockNodeCtxOption {
	return func(ctx *MockNodeCtx) {
		ctx.NodeValue = node
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
func (m *MockNodeCtx) GetNode() types.Node { return m.NodeValue }
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
//
// 兼容两种模式：
// 1) 传统 mock 模式：外部通过方法行为断言。
// 2) 直喂数据模式：直接往 Nodes/AllNodes/Endpoints 填值。
type MockNodePool struct {
	Nodes     map[string]types.NodeCtx
	AllNodes  []types.NodeCtx
	Endpoints []types.Endpoint
}

func (m *MockNodePool) Get(id string) (types.SharedNodeCtx, bool) {
	ctx, ok := m.Nodes[id]
	if !ok {
		return nil, false
	}
	return ctx.(types.SharedNodeCtx), true
}

func (m *MockNodePool) GetAll() []types.NodeCtx {
	if len(m.AllNodes) > 0 {
		return append([]types.NodeCtx(nil), m.AllNodes...)
	}
	if len(m.Nodes) == 0 {
		return nil
	}
	all := make([]types.NodeCtx, 0, len(m.Nodes))
	for _, n := range m.Nodes {
		all = append(all, n)
	}
	return all
}

func (m *MockNodePool) GetEndpoints() []types.Endpoint {
	if len(m.Endpoints) > 0 {
		return append([]types.Endpoint(nil), m.Endpoints...)
	}
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

// ----------------------- Feed Helpers -----------------------

// NewMockEndpointCtxWithNodeDef 构造一个可被 SharedNodePool.GetAll 直接消费的 endpoint NodeCtx。
func NewMockEndpointCtxWithNodeDef(id, name, nodeType string, config types.ConfigMap) *MockNodeCtx {
	ep := &LiteEndpointNode{id: id, name: name, nodeType: types.NodeType(nodeType)}
	return NewMockNodeCtx(
		WithNodeValue(ep),
		func(ctx *MockNodeCtx) {
			ctx.NodeIDValue = id
			ctx.NodeDef = types.NodeDef{ID: id, Name: name, Type: nodeType, Configuration: config}
		},
	)
}

// LiteEndpointNode 是一个无需 mock.On 的最小 endpoint 实现。
type LiteEndpointNode struct {
	id       string
	name     string
	nodeType types.NodeType
}

func (n *LiteEndpointNode) New() types.Node                            { return &LiteEndpointNode{nodeType: n.nodeType} }
func (n *LiteEndpointNode) Type() types.NodeType                       { return n.nodeType }
func (n *LiteEndpointNode) Init(config types.ConfigMap) error          { return nil }
func (n *LiteEndpointNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {}
func (n *LiteEndpointNode) Destroy()                                   {}
func (n *LiteEndpointNode) NodeMetadata() types.NodeMetadata {
	return types.NodeMetadata{Type: string(n.nodeType), Name: n.name}
}
func (n *LiteEndpointNode) DataContract() types.DataContract { return types.DataContract{} }
func (n *LiteEndpointNode) ID() string                       { return n.id }
func (n *LiteEndpointNode) SetID(id string)                  { n.id = id }
func (n *LiteEndpointNode) Name() string                     { return n.name }
func (n *LiteEndpointNode) SetName(name string)              { n.name = name }
func (n *LiteEndpointNode) Errors() []*types.Fault           { return nil }
func (n *LiteEndpointNode) ConfigSchema() *openapi3.Schema   { return nil }
func (n *LiteEndpointNode) GetInstance() (any, error)        { return n, nil }
func (n *LiteEndpointNode) SetRuntimePool(pool any) error    { return nil }
