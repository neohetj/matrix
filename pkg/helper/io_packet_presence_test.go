package helper_test

import (
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/require"
)

type inboundOptionalScalars struct {
	Revision *int64  `json:"revision"`
	Enabled  *bool   `json:"enabled"`
	Label    *string `json:"label"`
}

func TestProcessInboundPreservesOptionalScalarPresence(t *testing.T) {
	const sid = "InboundOptionalScalars_V1"
	registerTestCoreObjDef(t, &inboundOptionalScalars{}, sid)
	for _, tc := range []struct {
		name     string
		body     map[string]any
		defaults bool
		wantNil  bool
	}{
		{name: "missing fields stay nil", body: map[string]any{}, wantNil: true},
		{name: "explicit null stays nil", body: map[string]any{"revision": nil, "enabled": nil, "label": nil}, wantNil: true},
		{name: "explicit zero values are present", body: map[string]any{"revision": 0, "enabled": false, "label": ""}},
		{name: "missing fields use explicit zero defaults", body: map[string]any{}, defaults: true},
		{name: "null does not select a default", body: map[string]any{"revision": nil, "enabled": nil, "label": nil}, defaults: true, wantNil: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := setupTestMsg(t)
			obj, err := msg.DataT().NewItem(sid, "optional")
			require.NoError(t, err)
			fields := []types.EndpointIOField{
				{Name: "revision", Type: cnst.INT64, BindPath: "rulemsg://dataT/optional.revision?sid=" + sid},
				{Name: "enabled", Type: cnst.BOOL, BindPath: "rulemsg://dataT/optional.enabled?sid=" + sid},
				{Name: "label", Type: cnst.STRING, BindPath: "rulemsg://dataT/optional.label?sid=" + sid},
			}
			if tc.defaults {
				fields[0].DefaultValue, fields[1].DefaultValue, fields[2].DefaultValue = int64(0), false, ""
			}
			err = helper.ProcessInbound(registry.NewMinimalNodeCtx("presence-test"), msg,
				types.EndpointIOPacket{Fields: fields}, helper.BodyProvider{Data: tc.body})
			require.NoError(t, err)
			got := obj.Body().(*inboundOptionalScalars)
			if tc.wantNil {
				require.Nil(t, got.Revision)
				require.Nil(t, got.Enabled)
				require.Nil(t, got.Label)
			} else {
				require.NotNil(t, got.Revision)
				require.NotNil(t, got.Enabled)
				require.NotNil(t, got.Label)
				require.Equal(t, int64(0), *got.Revision)
				require.False(t, *got.Enabled)
				require.Empty(t, *got.Label)
			}
		})
	}
}

func TestProcessInboundMissingFieldPreservesMappedValue(t *testing.T) {
	const sid = "InboundMappedOptionalScalars_V1"
	registerTestCoreObjDef(t, &inboundOptionalScalars{}, sid)
	msg := setupTestMsg(t)
	path := "rulemsg://dataT/optional?sid=" + sid
	err := helper.ProcessInbound(registry.NewMinimalNodeCtx("presence-test"), msg, types.EndpointIOPacket{
		MapAll: &path,
		Fields: []types.EndpointIOField{{Name: "missing", Type: cnst.INT64, BindPath: "rulemsg://dataT/optional.revision?sid=" + sid}},
	}, helper.BodyProvider{Data: map[string]any{"revision": 9}})
	require.NoError(t, err)
	obj, ok := msg.DataT().Get("optional")
	require.True(t, ok)
	require.Equal(t, int64(9), *obj.Body().(*inboundOptionalScalars).Revision)
}

func TestProcessInboundRequiredMissingFieldStillFails(t *testing.T) {
	msg := setupTestMsg(t)
	err := helper.ProcessInbound(registry.NewMinimalNodeCtx("presence-test"), msg, types.EndpointIOPacket{
		Fields: []types.EndpointIOField{{Name: "revision", Type: cnst.INT64, Required: true, BindPath: "rulemsg://metadata/revision"}},
	}, helper.BodyProvider{Data: map[string]any{}})
	require.Error(t, err)
	assertFaultCode(t, err, cnst.CodeRequiredFieldMissing)
}

func TestProcessInboundNullClearsNullableField(t *testing.T) {
	const sid = "InboundNullableScalars_V1"
	registerTestCoreObjDef(t, &inboundOptionalScalars{}, sid)
	msg := setupTestMsg(t)
	obj, err := msg.DataT().NewItem(sid, "optional")
	require.NoError(t, err)
	revision := int64(9)
	obj.Body().(*inboundOptionalScalars).Revision = &revision
	err = helper.ProcessInbound(registry.NewMinimalNodeCtx("presence-test"), msg, types.EndpointIOPacket{
		Fields: []types.EndpointIOField{{Name: "reset", Type: cnst.INT64, BindPath: "rulemsg://dataT/optional.revision?sid=" + sid}},
	}, helper.BodyProvider{Data: map[string]any{"reset": nil}})
	require.NoError(t, err)
	require.Nil(t, obj.Body().(*inboundOptionalScalars).Revision)
}
