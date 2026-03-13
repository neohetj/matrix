package asset_test

import (
	"reflect"
	"testing"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestRuleMsgAssetSet_DataTBasicTypes(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	tests := []struct {
		name  string
		objID string
		sid   string
		value any
		want  any
	}{
		{
			name:  "string",
			objID: "tool_name",
			sid:   cnst.SID_STRING,
			value: "open_app",
			want:  "open_app",
		},
		{
			name:  "int64_from_string",
			objID: "step_count",
			sid:   cnst.SID_INT64,
			value: "42",
			want:  int64(42),
		},
		{
			name:  "float64_from_string",
			objID: "score",
			sid:   cnst.SID_FLOAT64,
			value: "12.5",
			want:  float64(12.5),
		},
		{
			name:  "bool_from_string",
			objID: "flag",
			sid:   cnst.SID_BOOL,
			value: "true",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := asset.Asset[any]{URI: "rulemsg://dataT/" + tt.objID + "?sid=" + tt.sid}
			err := a.Set(ctx, tt.value)
			assert.NoError(t, err)

			obj, ok := msg.DataT().Get(tt.objID)
			assert.True(t, ok)
			assertBasicValue(t, tt.want, obj.Body())
		})
	}
}

func TestRuleMsgAssetSet_DataTMapTypes(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	stringMap := map[string]string{"k": "v"}
	mapAsset := asset.Asset[any]{URI: "rulemsg://dataT/headers?sid=" + cnst.SID_MAP_STRING_STRING}
	err := mapAsset.Set(ctx, stringMap)
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("headers")
	assert.True(t, ok)
	storedHeaders := derefMap[string](t, obj.Body())
	assert.Equal(t, stringMap, storedHeaders)

	interfaceMap := map[string]any{"app": "Chrome", "ok": true}
	interfaceAsset := asset.Asset[any]{URI: "rulemsg://dataT/params?sid=" + cnst.SID_MAP_STRING_INTERFACE}
	err = interfaceAsset.Set(ctx, interfaceMap)
	assert.NoError(t, err)

	obj, ok = msg.DataT().Get("params")
	assert.True(t, ok)
	storedParams := derefMap[any](t, obj.Body())
	assert.Equal(t, interfaceMap, storedParams)
}

func TestRuleMsgAssetSet_DataTMapNestedFieldOnNewItem(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	nestedAsset := asset.Asset[any]{URI: "rulemsg://dataT/stats.originalLeadCount?sid=" + cnst.SID_MAP_STRING_INTERFACE}
	err := nestedAsset.Set(ctx, int64(10))
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("stats")
	assert.True(t, ok)
	stored := derefMap[any](t, obj.Body())
	assert.Equal(t, int64(10), stored["originalLeadCount"])
}

func TestRuleMsgAssetSet_DataTMapNestedFieldOnNilMapBody(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	obj, err := msg.DataT().NewItem(cnst.SID_MAP_STRING_INTERFACE, "patch")
	assert.NoError(t, err)
	assert.NotNil(t, obj)

	nestedAsset := asset.Asset[any]{URI: "rulemsg://dataT/patch.scrapedProfileCount?sid=" + cnst.SID_MAP_STRING_INTERFACE}
	err = nestedAsset.Set(ctx, int64(5))
	assert.NoError(t, err)

	stored := derefMap[any](t, obj.Body())
	assert.Equal(t, int64(5), stored["scrapedProfileCount"])
}

func TestRuleMsgAssetSet_DataTSliceTypes(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	stringSlice := []string{"a", "b", "c"}
	sliceAsset := asset.Asset[any]{URI: "rulemsg://dataT/urls?sid=" + cnst.SID_SLICE_STRING}
	err := sliceAsset.Set(ctx, stringSlice)
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("urls")
	assert.True(t, ok)
	storedSlice, ok := obj.Body().(*[]string)
	assert.True(t, ok)
	assert.Equal(t, stringSlice, *storedSlice)
}

func TestRuleMsgAssetSet_DataTSliceAnyTypes(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	anySlice := []any{"a", 123, true}
	sliceAsset := asset.Asset[any]{URI: "rulemsg://dataT/mixed?sid=" + cnst.SID_SLICE_ANY}
	err := sliceAsset.Set(ctx, anySlice)
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("mixed")
	assert.True(t, ok)
	storedSlice, ok := obj.Body().(*[]any)
	assert.True(t, ok)
	assert.Equal(t, anySlice, *storedSlice)
}

func TestRuleMsgAssetSet_DataTBasicTypes_FromNumeric(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	tests := []struct {
		name  string
		objID string
		sid   string
		value any
		want  any
	}{
		{
			name:  "int64_from_float64",
			objID: "step_count",
			sid:   cnst.SID_INT64,
			value: float64(7),
			want:  int64(7),
		},
		{
			name:  "bool_from_int",
			objID: "flag",
			sid:   cnst.SID_BOOL,
			value: 1,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := asset.Asset[any]{URI: "rulemsg://dataT/" + tt.objID + "?sid=" + tt.sid}
			err := a.Set(ctx, tt.value)
			assert.NoError(t, err)

			obj, ok := msg.DataT().Get(tt.objID)
			assert.True(t, ok)
			assertBasicValue(t, tt.want, obj.Body())
		})
	}
}

func TestRuleMsgAssetResolve_DataTPointerBasic(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	assetKey := "string_test"
	setAsset := asset.Asset[any]{URI: "rulemsg://dataT/" + assetKey + "?sid=" + cnst.SID_STRING}
	assert.NoError(t, setAsset.Set(ctx, "open_app"))

	getAsset := asset.Asset[string]{URI: "rulemsg://dataT/" + assetKey + "?sid=" + cnst.SID_STRING}
	val, err := getAsset.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "open_app", val)
}

func TestRuleMsgAssetResolve_DataTPointerMap(t *testing.T) {
	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	assetKey := "map_test"
	setAsset := asset.Asset[any]{URI: "rulemsg://dataT/" + assetKey + "?sid=" + cnst.SID_MAP_STRING_INTERFACE}
	assert.NoError(t, setAsset.Set(ctx, map[string]any{"k": "v"}))

	getAsset := asset.Asset[map[string]any]{URI: "rulemsg://dataT/" + assetKey + "?sid=" + cnst.SID_MAP_STRING_INTERFACE}
	val, err := getAsset.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "v", val["k"])
	val["k"] = "v2"

	obj, ok := msg.DataT().Get(assetKey)
	assert.True(t, ok)
	stored := derefMap[any](t, obj.Body())
	assert.Equal(t, "v2", stored["k"])
}

func TestRuleMsgAssetSet_DataTStructPointer(t *testing.T) {
	type TestMemoryContext struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}

	registry.Default.CoreObjRegistry.Register(
		types.NewCoreObjDef(&TestMemoryContext{}, "TestMemoryContextV1_0", "test memory context"),
	)

	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	mc := &TestMemoryContext{Status: "success", Summary: "ok"}
	a := asset.Asset[any]{URI: "rulemsg://dataT/memory?sid=TestMemoryContextV1_0"}
	err := a.Set(ctx, mc)
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("memory")
	assert.True(t, ok)
	stored, ok := obj.Body().(*TestMemoryContext)
	assert.True(t, ok)
	assert.Equal(t, mc.Status, stored.Status)
	assert.Equal(t, mc.Summary, stored.Summary)
}

func TestRuleMsgAssetSet_DataTSliceOfStruct(t *testing.T) {
	type TestMemoryContext struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}

	registry.Default.CoreObjRegistry.Register(
		types.NewCoreObjDef(&[]TestMemoryContext{}, "TestMemoryContextSlice", "test memory context slice"),
	)

	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	mcSlice := []TestMemoryContext{
		{Status: "success", Summary: "ok"},
		{Status: "failed", Summary: "error"},
	}
	a := asset.Asset[any]{URI: "rulemsg://dataT/memory_slice?sid=TestMemoryContextSlice"}
	err := a.Set(ctx, mcSlice)
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("memory_slice")
	assert.True(t, ok)
	storedSlice, ok := obj.Body().(*[]TestMemoryContext)
	assert.True(t, ok)
	assert.Equal(t, mcSlice, *storedSlice)
}

func TestRuleMsgAssetSet_DataTStructValue(t *testing.T) {
	type TestMemoryContext struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}

	// Use a unique SID to avoid conflicts with other tests if run in parallel
	const sid = "TestMemoryContextV1_1"
	registry.Default.CoreObjRegistry.Register(
		types.NewCoreObjDef(&TestMemoryContext{}, sid, "test memory context value"),
	)

	msg := types.NewMsg("test", "", nil, types.NewDataT())
	ctx := asset.NewAssetContext(asset.WithRuleMsg(msg))

	// mc is a struct VALUE, not a pointer
	mc := TestMemoryContext{Status: "failure", Summary: "bad value"}
	a := asset.Asset[any]{URI: "rulemsg://dataT/memory_val?sid=" + sid}
	err := a.Set(ctx, mc)

	// Assert that NO error is returned for this assignment type
	assert.NoError(t, err)

	obj, ok := msg.DataT().Get("memory_val")
	assert.True(t, ok)
	stored, ok := obj.Body().(*TestMemoryContext)
	assert.True(t, ok)
	assert.Equal(t, mc.Status, stored.Status)
	assert.Equal(t, mc.Summary, stored.Summary)
}

func derefMap[T any](t *testing.T, actual any) map[string]T {
	t.Helper()

	if actual == nil {
		assert.Fail(t, "unexpected nil map")
		return nil
	}

	if m, ok := actual.(map[string]T); ok {
		return m
	}
	if m, ok := actual.(*map[string]T); ok {
		return *m
	}

	assert.Fail(t, "unexpected map type")
	return nil
}

func assertBasicValue(t *testing.T, expected, actual any) {
	t.Helper()

	v := reflect.ValueOf(actual)
	if v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			assert.Fail(t, "unexpected nil pointer value")
			return
		}
		assert.Equal(t, expected, v.Elem().Interface())
		return
	}

	assert.Equal(t, expected, actual)
}
