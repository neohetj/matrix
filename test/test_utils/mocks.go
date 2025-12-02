package test_utils

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"gitlab.com/neohet/matrix/pkg/types"
)

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
	log.New(&m.buf, "", 0).Println(m.buildMessage(format, v...))
}

func (m *MockLogger) Debugf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.New(&m.buf, "DEBUG: ", 0).Println(m.buildMessage(format, v...))
}

func (m *MockLogger) Infof(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.New(&m.buf, "INFO: ", 0).Println(m.buildMessage(format, v...))
}

func (m *MockLogger) Warnf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.New(&m.buf, "WARN: ", 0).Println(m.buildMessage(format, v...))
}

func (m *MockLogger) Errorf(ctx context.Context, format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.New(&m.buf, "ERROR: ", 0).Println(m.buildMessage(format, v...))
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
func (m *MockNodeCtx) SelfDef() *types.NodeDef { return &m.NodeDef }
func (m *MockNodeCtx) Config() types.Config    { return m.NodeDef.Configuration }
func (m *MockNodeCtx) Logger() types.Logger {
	return &TestLogger{}
}
func (m *MockNodeCtx) Info(msg string, fields ...any) {
	m.Logger().Infof(m.GetContext(), msg, fields...)
}
