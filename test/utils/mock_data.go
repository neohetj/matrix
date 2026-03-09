package utils

import (
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
	for _, c := range m.ExpectedCalls {
		if c.Method == "Type" {
			args := m.Called()
			return args.String(0)
		}
	}
	return ""
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
func (m *MockDataT) Project(keepObjIDs []string) (types.DataT, error) {
	args := m.Called(keepObjIDs)
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
