package helper_test

import (
	"fmt"
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleMockCoreObj is a simple implementation of types.CoreObj for testing values
type SimpleMockCoreObj struct {
	body any
}

func (m *SimpleMockCoreObj) Key() string                  { return "" }
func (m *SimpleMockCoreObj) Definition() types.CoreObjDef { return nil }
func (m *SimpleMockCoreObj) Body() any                    { return m.body }
func (m *SimpleMockCoreObj) SetBody(body any) error       { m.body = body; return nil }
func (m *SimpleMockCoreObj) DeepCopy() (types.CoreObj, error) {
	return &SimpleMockCoreObj{body: m.body}, nil
}

func NewSimpleMockCoreObj(val any) types.CoreObj {
	return &SimpleMockCoreObj{body: val}
}

func TestProcessOutbound_BoolConversion(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")

	t.Run("Map *bool from DataT to bool output", func(t *testing.T) {
		msg := setupTestMsg(t)
		success := true
		// Store as pointer wrapped in CoreObj
		msg.DataT().Set("success", NewSimpleMockCoreObj(&success))

		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "success",
					BindPath: fmt.Sprintf("rulemsg://dataT/success?sid=%s", cnst.SID_BOOL),
					Type:     "bool",
				},
			},
		}

		// This simulates the scenario where node output returns &success (pointer)
		// and Endpoint expects "bool" (value).
		result, err := helper.ProcessOutbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		require.NoError(t, err)

		resMap, ok := result.(map[string]any)
		require.True(t, ok)
		val, exists := resMap["success"]
		require.True(t, exists)

		// Assert that the result is a boolean value (true), not a pointer
		// If this passes, it means ProcessOutbound correctly dereferences *bool to bool
		assert.Equal(t, true, val)
	})

	t.Run("Map bool from DataT to bool output", func(t *testing.T) {
		msg := setupTestMsg(t)
		success := true
		// Store as value wrapped in CoreObj
		msg.DataT().Set("success", NewSimpleMockCoreObj(success))

		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "success",
					BindPath: fmt.Sprintf("rulemsg://dataT/success?sid=%s", cnst.SID_BOOL),
					Type:     "bool",
				},
			},
		}

		result, err := helper.ProcessOutbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		require.NoError(t, err)

		resMap, ok := result.(map[string]any)
		require.True(t, ok)
		val, exists := resMap["success"]
		require.True(t, exists)
		assert.Equal(t, true, val)
	})

	t.Run("Map *bool to bool - Failure Reproduction?", func(t *testing.T) {
		msg := setupTestMsg(t)
		success := true
		msg.DataT().Set("success", NewSimpleMockCoreObj(&success))

		// Try without explicit SID in BindPath, relying on CoreObj body type
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "success",
					BindPath: "rulemsg://dataT/success", // No SID
					Type:     "bool",
				},
			},
		}

		_, err := helper.ProcessOutbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		// If it fails with "can't convert *bool to bool", then we know the issue is
		// possibly related to missing SID or how reflection handles it.
		if err != nil {
			t.Logf("Error without SID: %v", err)
			// assert.ErrorContains(t, err, "can't convert")
		} else {
			t.Log("No error without SID")
		}
	})
}
