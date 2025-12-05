package utils

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/stretchr/testify/mock"
)

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

// --- Mocks for Logger ---

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

// --- Mock for NodeCtx ---

type MockNodeCtx struct {
	types.NodeCtx
	Ctx        context.Context
	SuccessMsg types.RuleMsg
	FailureMsg types.RuleMsg
	FailureErr error
	NodeDef    types.NodeDef
}

func NewMockNodeCtx() *MockNodeCtx {
	return &MockNodeCtx{Ctx: context.Background()}
}
func (m *MockNodeCtx) GetContext() context.Context   { return m.Ctx }
func (m *MockNodeCtx) TellSuccess(msg types.RuleMsg) { m.SuccessMsg = msg }
func (m *MockNodeCtx) TellFailure(msg types.RuleMsg, err error) {
	m.FailureMsg = msg
	m.FailureErr = err
}
func (m *MockNodeCtx) HandleError(msg types.RuleMsg, err error) {
	m.TellFailure(msg, err)
}
func (m *MockNodeCtx) SelfDef() *types.NodeDef { return &m.NodeDef }
func (m *MockNodeCtx) Config() types.ConfigMap { return m.NodeDef.Configuration }
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
