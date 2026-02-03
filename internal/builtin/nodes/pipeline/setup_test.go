package pipeline

import (
	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/mock"
)

func init() {
	types.NewNodeCtx = facotry.NewNodeCtx
	types.NewMsg = facotry.NewMsg
	types.NewDataT = facotry.NewDataT
	types.NewSubMsg = facotry.NewSubMsg
	types.NewCoreObj = facotry.NewCoreObj
	types.NewCoreObjDef = facotry.NewCoreObjDef
}

// ------------------- Mock Runtime Pool -------------------
type MockRuntimePoolForPipeline struct {
	mock.Mock
}

func (m *MockRuntimePoolForPipeline) Get(id string) (types.Runtime, bool) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(types.Runtime), args.Bool(1)
}

func (m *MockRuntimePoolForPipeline) Put(id string, runtime types.Runtime) {}
func (m *MockRuntimePoolForPipeline) Remove(id string)                     {}
func (m *MockRuntimePoolForPipeline) ListByViewType(viewType string) []types.Runtime {
	return nil
}
func (m *MockRuntimePoolForPipeline) ListIDs() []string {
	return nil
}
func (m *MockRuntimePoolForPipeline) Register(id string, runtime types.Runtime) error { return nil }
func (m *MockRuntimePoolForPipeline) Unregister(id string)                            {}
