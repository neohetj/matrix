package helper_test

import (
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessOutbound(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")
	msg := setupTestMsg(t)
	provider := helper.RuleMsgProvider{Msg: msg}

	t.Run("BindPath with found value", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "reqId", BindPath: "rulemsg://metadata/requestId"},
			},
		}
		res, err := helper.ProcessOutbound(ctx, msg, packet, provider)
		require.NoError(t, err)
		result := res.(map[string]any)
		assert.Equal(t, "req-123", result["reqId"])
	})

	t.Run("BindPath not found, fallback to DefaultValue", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "nonexistent", BindPath: "rulemsg://metadata/nonexistent", DefaultValue: "default"},
			},
		}
		res, err := helper.ProcessOutbound(ctx, msg, packet, provider)
		require.NoError(t, err)
		result := res.(map[string]any)
		assert.Equal(t, "default", result["nonexistent"])
	})

	t.Run("Empty BindPath, use DefaultValue", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "static", DefaultValue: "static-value"},
			},
		}
		res, err := helper.ProcessOutbound(ctx, msg, packet, provider)
		require.NoError(t, err)
		result := res.(map[string]any)
		assert.Equal(t, "static-value", result["static"])
	})

	t.Run("Not found, no DefaultValue, not required", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "optional", BindPath: "rulemsg://metadata/optional"},
			},
		}
		res, err := helper.ProcessOutbound(ctx, msg, packet, provider)
		require.NoError(t, err)
		result := res.(map[string]any)
		_, found := result["optional"]
		assert.False(t, found)
	})

	t.Run("Not found, no DefaultValue, required", func(t *testing.T) {
		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{Name: "required_field", BindPath: "rulemsg://metadata/required", Required: true},
			},
		}
		_, err := helper.ProcessOutbound(ctx, msg, packet, provider)
		assert.Error(t, err)
		assertFaultCode(t, err, cnst.CodeRequiredFieldMissing)
	})

	t.Run("MapAll with Slice", func(t *testing.T) {
		// Setup message with a slice in DataT
		sliceSid := cnst.SID_SLICE_ANY
		dataT := msg.DataT()
		item, _ := dataT.NewItem(sliceSid, "myList")
		expectedSlice := []any{"item1", "item2"}
		item.SetBody(&expectedSlice)

		mapAllPath := "rulemsg://dataT/myList?sid=" + sliceSid
		packet := types.EndpointIOPacket{
			MapAll: &mapAllPath,
		}

		res, err := helper.ProcessOutbound(ctx, msg, packet, provider)
		require.NoError(t, err)

		// Expect result to be the slice pointer (since SetBody stores pointer)
		resultSlice, ok := res.(*[]any)
		assert.True(t, ok)
		assert.Equal(t, expectedSlice, *resultSlice)
	})
}
