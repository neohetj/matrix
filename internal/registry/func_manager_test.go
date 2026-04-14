package registry

import (
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNodeFuncManager_Register(t *testing.T) {
	manager := NewNodeFuncManager()

	t.Run("Valid Configuration", func(t *testing.T) {
		validFunc := &types.NodeFuncObject{
			FuncObject: types.FuncObject{
				ID: "valid_func",
				Configuration: types.FuncObjConfiguration{
					Business: []types.DynamicConfigField{
						{
							ID:   "field1",
							Type: cnst.STRING,
						},
					},
				},
			},
		}
		assert.NotPanics(t, func() {
			manager.Register(validFunc)
		})

		retrieved, ok := manager.Get("valid_func")
		assert.True(t, ok)
		assert.Equal(t, validFunc, retrieved)
	})

	t.Run("Invalid Configuration Type", func(t *testing.T) {
		invalidFunc := &types.NodeFuncObject{
			FuncObject: types.FuncObject{
				ID: "invalid_func",
				Configuration: types.FuncObjConfiguration{
					Business: []types.DynamicConfigField{
						{
							ID:   "field1",
							Type: "INVALID_TYPE",
						},
					},
				},
			},
		}
		assert.Panics(t, func() {
			manager.Register(invalidFunc)
		}, "Registration should panic for invalid type")
	})
	t.Run("NotEditable Without Default", func(t *testing.T) {
		invalidFunc := &types.NodeFuncObject{
			FuncObject: types.FuncObject{
				ID: "invalid_not_editable",
				Configuration: types.FuncObjConfiguration{
					Business: []types.DynamicConfigField{
						{
							ID:          "field1",
							Type:        cnst.STRING,
							NotEditable: true,
						},
					},
				},
			},
		}
		assert.Panics(t, func() {
			manager.Register(invalidFunc)
		}, "Registration should panic when notEditable field has no default")
	})

	t.Run("Decision Routing Requires Declared Relations", func(t *testing.T) {
		invalidFunc := &types.NodeFuncObject{
			FuncObject: types.FuncObject{
				ID: "invalid_decision_missing_relations",
				Configuration: types.FuncObjConfiguration{
					RoutingMode: types.FunctionRoutingModeDecision,
				},
			},
		}
		assert.Panics(t, func() {
			manager.Register(invalidFunc)
		}, "Decision routing mode should require declaredRelations")
	})

	t.Run("Standard Routing Must Not Declare Custom Relations", func(t *testing.T) {
		invalidFunc := &types.NodeFuncObject{
			FuncObject: types.FuncObject{
				ID: "invalid_standard_declared_relations",
				Configuration: types.FuncObjConfiguration{
					DeclaredRelations: []string{"Create"},
				},
			},
		}
		assert.Panics(t, func() {
			manager.Register(invalidFunc)
		}, "Standard routing mode should reject declaredRelations")
	})

	t.Run("Decision Routing Accepts Valid Relations", func(t *testing.T) {
		validFunc := &types.NodeFuncObject{
			FuncObject: types.FuncObject{
				ID: "valid_decision_func",
				Configuration: types.FuncObjConfiguration{
					RoutingMode:       types.FunctionRoutingModeDecision,
					DeclaredRelations: []string{"Create", "Update"},
				},
			},
		}
		assert.NotPanics(t, func() {
			manager.Register(validFunc)
		})

		retrieved, ok := manager.Get("valid_decision_func")
		assert.True(t, ok)
		assert.Equal(t, validFunc, retrieved)
	})
}
