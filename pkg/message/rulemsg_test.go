package message

import (
	"encoding/json"
	"testing"

	"github.com/neohetj/matrix/internal/contract"
	_ "github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMessageForTest() types.RuleMsg {
	return contract.NewDefaultRuleMsg("test", "", nil, contract.NewDataT())
}

func addMapForTest(t *testing.T, msg types.RuleMsg, id, body string) types.CoreObj {
	t.Helper()
	obj, err := msg.DataT().NewItem(cnst.SID_MAP_STRING_INTERFACE, id)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(body), obj.Body()))
	return obj
}

func TestExtractFromMsg(t *testing.T) {
	t.Run("Metadata", func(t *testing.T) {
		msg := newMessageForTest()
		msg.Metadata()["key1"] = "value1"
		val, err := ExtractFromMsg[string](msg, "rulemsg://metadata/key1")
		require.NoError(t, err)
		assert.Equal(t, "value1", val)
	})

	t.Run("Data JSON requires explicit format", func(t *testing.T) {
		msg := newMessageForTest()
		msg.SetData(`{"field1":"value1","nested":{"field2":123}}`, cnst.JSON)
		val, err := ExtractFromMsg[float64](msg, "rulemsg://data/nested.field2?format=JSON")
		require.NoError(t, err)
		assert.Equal(t, float64(123), val)
	})

	t.Run("DataT body", func(t *testing.T) {
		msg := newMessageForTest()
		addMapForTest(t, msg, "obj1", `{"foo":"bar"}`)
		val, err := ExtractFromMsg[map[string]any](msg, "rulemsg://dataT/obj1?sid="+cnst.SID_MAP_STRING_INTERFACE)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"foo": "bar"}, val)
	})

	t.Run("DataT nested field", func(t *testing.T) {
		msg := newMessageForTest()
		addMapForTest(t, msg, "obj1", `{"foo":"bar","nested":{"val":42}}`)
		val, err := ExtractFromMsg[float64](msg, "rulemsg://dataT/obj1.nested.val?sid="+cnst.SID_MAP_STRING_INTERFACE)
		require.NoError(t, err)
		assert.Equal(t, float64(42), val)
	})

	for _, tt := range []struct{ name, uri, errorText string }{
		{"missing data format", "rulemsg://data/nested.val", "data format is required"},
		{"mismatched data format", "rulemsg://data?format=TEXT", "mismatch"},
		{"missing DataT SID", "rulemsg://dataT/obj1", "missing sid"},
		{"plain text is not an asset URI", "literal string", "no handler registered for scheme"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			msg := newMessageForTest()
			msg.SetData(`{"nested":{"val":42}}`, cnst.JSON)
			addMapForTest(t, msg, "obj1", `{"foo":"bar"}`)
			_, err := ExtractFromMsg[any](msg, tt.uri)
			require.ErrorContains(t, err, tt.errorText)
		})
	}
}

func TestSetInMsg(t *testing.T) {
	t.Run("Data JSON", func(t *testing.T) {
		msg := newMessageForTest()
		require.NoError(t, SetInMsg(msg, "rulemsg://data?format=JSON", map[string]any{"foo": "bar"}))
		assert.JSONEq(t, `{"foo":"bar"}`, string(msg.Data()))
		assert.Equal(t, cnst.JSON, msg.DataFormat())
	})

	t.Run("Metadata", func(t *testing.T) {
		msg := newMessageForTest()
		require.NoError(t, SetInMsg(msg, "rulemsg://metadata/key1", "value1"))
		assert.Equal(t, "value1", msg.Metadata()["key1"])
	})

	t.Run("DataT field preserves other fields", func(t *testing.T) {
		msg := newMessageForTest()
		obj := addMapForTest(t, msg, "obj1", `{"foo":"old","other":"keep"}`)
		require.NoError(t, SetInMsg(msg, "rulemsg://dataT/obj1.foo?sid="+cnst.SID_MAP_STRING_INTERFACE, "new"))
		assert.Equal(t, map[string]any{"foo": "new", "other": "keep"}, *obj.Body().(*map[string]any))
	})

	t.Run("DataT new item uses registered definition", func(t *testing.T) {
		msg := newMessageForTest()
		body := map[string]any{"foo": "bar"}
		require.NoError(t, SetInMsg(msg, "rulemsg://dataT/newObj?sid="+cnst.SID_MAP_STRING_INTERFACE, body))
		obj, ok := msg.DataT().Get("newObj")
		require.True(t, ok)
		assert.Equal(t, cnst.SID_MAP_STRING_INTERFACE, obj.Definition().SID())
		assert.Equal(t, body, *obj.Body().(*map[string]any))
	})

	t.Run("missing SID rejects write without mutation", func(t *testing.T) {
		msg := newMessageForTest()
		obj := addMapForTest(t, msg, "obj1", `{"foo":"old"}`)
		require.ErrorContains(t, SetInMsg(msg, "rulemsg://dataT/obj1.foo", "new"), "missing sid")
		assert.Equal(t, "old", (*obj.Body().(*map[string]any))["foo"])
	})
}

func TestReplaceRuleMsg(t *testing.T) {
	msg := newMessageForTest()
	msg.Metadata()["id"] = "123"
	for _, tt := range []struct{ template, want string }{
		{"literal string", "literal string"},
		{"http://example.com/items/${rulemsg://metadata/id}", "http://example.com/items/123"},
	} {
		value, err := ReplaceRuleMsg(tt.template, msg)
		require.NoError(t, err)
		assert.Equal(t, tt.want, value)
	}
}

func TestNilMessage(t *testing.T) {
	_, err := ExtractFromMsg[string](nil, "rulemsg://metadata/key")
	require.ErrorContains(t, err, "rule message is nil")
	require.ErrorContains(t, SetInMsg(nil, "rulemsg://metadata/key", "value"), "rule message is nil")
	value, err := ReplaceRuleMsg("unchanged", nil)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", value)
}
