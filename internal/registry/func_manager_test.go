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
}
