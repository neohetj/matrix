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

type rulechainValidatorLead struct {
	Username  string `json:"username"`
	Followers int    `json:"followers"`
}

type rulechainValidatorProfile struct {
	Username string   `json:"username"`
	Tags     []string `json:"tags"`
}

func registerTestCoreObjDef(t *testing.T, sample any, sid string) {
	t.Helper()
	registry.Default.CoreObjRegistry.Register(types.NewCoreObjDef(sample, sid, "test coreobj def"))
	t.Cleanup(func() {
		if r, ok := registry.Default.CoreObjRegistry.(*registry.DefaultCoreObjRegistry); ok {
			r.Unregister(sid)
		}
	})
}

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

	t.Run("Inbound field passthrough preserves slice when type omitted", func(t *testing.T) {
		sliceSid := cnst.SID_SLICE_ANY
		dataT := msg.DataT()
		source, err := dataT.NewItem(sliceSid, "sourceList")
		require.NoError(t, err)

		expectedSlice := []any{"item1", map[string]any{"username": "alice"}}
		require.NoError(t, source.SetBody(&expectedSlice))

		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "rulemsg://dataT/sourceList?sid=" + sliceSid,
					BindPath: "rulemsg://dataT/copiedList?sid=" + sliceSid,
				},
			},
		}

		err = helper.ProcessInbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		require.NoError(t, err)

		copied, ok := msg.DataT().Get("copiedList")
		require.True(t, ok)

		resultSlice, ok := copied.Body().(*[]any)
		require.True(t, ok)
		assert.Equal(t, expectedSlice, *resultSlice)
	})
}

func TestProcessInbound_TypedCollectionPassthroughAndObjectConversion(t *testing.T) {
	ctx := registry.NewMinimalNodeCtx("test-node")

	t.Run("Typed business slice passthrough preserves typed slice when type omitted", func(t *testing.T) {
		const leadSliceSID = "[]RulechainValidatorLead_V1"
		registerTestCoreObjDef(t, &[]rulechainValidatorLead{}, leadSliceSID)

		msg := setupTestMsg(t)
		source, err := msg.DataT().NewItem(leadSliceSID, "finalleadslist")
		require.NoError(t, err)

		expected := []rulechainValidatorLead{
			{Username: "alice", Followers: 1200},
			{Username: "bob", Followers: 980},
		}
		require.NoError(t, source.SetBody(&expected))

		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "rulemsg://dataT/finalleadslist?sid=" + leadSliceSID,
					BindPath: "rulemsg://dataT/filteredLeadBatch?sid=" + leadSliceSID,
				},
			},
		}

		err = helper.ProcessInbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		require.NoError(t, err)

		copied, ok := msg.DataT().Get("filteredLeadBatch")
		require.True(t, ok)

		resultSlice, ok := copied.Body().(*[]rulechainValidatorLead)
		require.True(t, ok)
		assert.Equal(t, expected, *resultSlice)
	})

	t.Run("Typed business slice fails when object conversion is forced", func(t *testing.T) {
		const leadSliceSID = "[]RulechainValidatorLead_V1"
		registerTestCoreObjDef(t, &[]rulechainValidatorLead{}, leadSliceSID)

		msg := setupTestMsg(t)
		source, err := msg.DataT().NewItem(leadSliceSID, "finalleadslist")
		require.NoError(t, err)

		expected := []rulechainValidatorLead{
			{Username: "alice", Followers: 1200},
		}
		require.NoError(t, source.SetBody(&expected))

		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "rulemsg://dataT/finalleadslist?sid=" + leadSliceSID,
					BindPath: "rulemsg://dataT/filteredLeadBatch?sid=" + leadSliceSID,
					Type:     "object",
				},
			},
		}

		err = helper.ProcessInbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		require.Error(t, err)
		assertFaultCode(t, err, cnst.CodeFieldConversionFailed)
		assert.Contains(t, err.Error(), "can't convert")
	})

	t.Run("String JSON still converts to object when type is object", func(t *testing.T) {
		const profileSID = "RulechainValidatorProfile_V1"
		registerTestCoreObjDef(t, &rulechainValidatorProfile{}, profileSID)

		msg := setupTestMsg(t)
		rawJSON := `{"username":"alice","tags":["fashion","sale"]}`
		source, err := msg.DataT().NewItem(cnst.SID_STRING, "llmResponse")
		require.NoError(t, err)
		require.NoError(t, source.SetBody(&rawJSON))

		packet := types.EndpointIOPacket{
			Fields: []types.EndpointIOField{
				{
					Name:     "rulemsg://dataT/llmResponse?sid=" + cnst.SID_STRING,
					BindPath: "rulemsg://dataT/profile?sid=" + profileSID,
					Type:     "object",
				},
			},
		}

		err = helper.ProcessInbound(ctx, msg, packet, helper.RuleMsgProvider{Msg: msg})
		require.NoError(t, err)

		profileObj, ok := msg.DataT().Get("profile")
		require.True(t, ok)

		profile, ok := profileObj.Body().(*rulechainValidatorProfile)
		require.True(t, ok)
		assert.Equal(t, rulechainValidatorProfile{
			Username: "alice",
			Tags:     []string{"fashion", "sale"},
		}, *profile)
	})
}
