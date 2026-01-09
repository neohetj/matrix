package message

import (
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestExtractFromMsg(t *testing.T) {
	// Test Case 1: Extract from Metadata using new ExtractFromMsg
	t.Run("Extract from Metadata", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		metadata := types.Metadata{"key1": "value1"}
		mockMsg.On("Metadata").Return(metadata).Once()

		val, err := ExtractFromMsg[string](mockMsg, "rulemsg://metadata/key1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)
		mockMsg.AssertExpectations(t)
	})

	// Test Case 2: Extract from Data (JSON) using new ExtractFromMsg
	t.Run("Extract from Data JSON", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		jsonStr := `{"field1": "value1", "nested": {"field2": 123}}`
		mockMsg.On("Data").Return(types.Data(jsonStr)).Once()
		mockMsg.On("DataFormat").Return(cnst.JSON).Once()

		val, err := ExtractFromMsg[float64](mockMsg, "rulemsg://data/nested.field2")
		assert.NoError(t, err)
		assert.Equal(t, float64(123), val)
		mockMsg.AssertExpectations(t)
	})

	// Test Case 3: Extract from DataT (Whole Object Body) using new ExtractFromMsg
	t.Run("Extract from DataT Body", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		mockDataT := new(utils.MockDataT)
		mockCoreObj := new(utils.MockCoreObj)

		mockMsg.On("DataT").Return(mockDataT).Once()
		mockDataT.On("Get", "obj1").Return(mockCoreObj, true).Once()
		body := map[string]any{"foo": "bar"}
		mockCoreObj.On("Body").Return(body).Once()

		val, err := ExtractFromMsg[map[string]any](mockMsg, "rulemsg://dataT/obj1")
		assert.NoError(t, err)
		assert.Equal(t, body, val)
		mockMsg.AssertExpectations(t)
		mockDataT.AssertExpectations(t)
		mockCoreObj.AssertExpectations(t)
	})

	// Test Case 4: Extract from DataT Object Field using new ExtractFromMsg
	t.Run("Extract from DataT Object Field", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		mockDataT := new(utils.MockDataT)
		mockCoreObj := new(utils.MockCoreObj)

		mockMsg.On("DataT").Return(mockDataT).Once()
		mockDataT.On("Get", "obj1").Return(mockCoreObj, true).Once()
		body := map[string]any{"foo": "bar", "nested": map[string]any{"val": 42}}
		mockCoreObj.On("Body").Return(body).Once()

		val, err := ExtractFromMsg[float64](mockMsg, "rulemsg://dataT/obj1.nested.val")
		assert.NoError(t, err)
		assert.Equal(t, float64(42), val)
		mockMsg.AssertExpectations(t)
		mockDataT.AssertExpectations(t)
		mockCoreObj.AssertExpectations(t)
	})

	// Test Case 5: Literal String (non-URI) using new ExtractFromMsg
	t.Run("Literal String", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		val, err := ExtractFromMsg[string](mockMsg, "literal string")
		assert.NoError(t, err)
		assert.Equal(t, "literal string", val)
	})
}

func TestSetInMsg(t *testing.T) {
	t.Run("Set Data JSON", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		val := map[string]any{"foo": "bar"}
		mockMsg.On("SetData", `{"foo":"bar"}`, cnst.JSON).Return().Once()

		err := SetInMsg(mockMsg, "rulemsg://data?format=JSON", val)
		assert.NoError(t, err)
		mockMsg.AssertExpectations(t)
	})

	t.Run("Set Metadata", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		metadata := types.Metadata{}
		mockMsg.On("Metadata").Return(metadata).Once()

		err := SetInMsg(mockMsg, "rulemsg://metadata/key1", "value1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", metadata["key1"])
		mockMsg.AssertExpectations(t)
	})

	t.Run("Set DataT Field", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		mockDataT := new(utils.MockDataT)
		mockCoreObj := new(utils.MockCoreObj)

		mockMsg.On("DataT").Return(mockDataT).Once()
		mockDataT.On("Get", "obj1").Return(mockCoreObj, true).Once()
		body := map[string]any{"foo": "old"}
		mockCoreObj.On("Body").Return(body).Twice() // One for ToMap, one for Decode back

		err := SetInMsg(mockMsg, "rulemsg://dataT/obj1.foo", "new")
		assert.NoError(t, err)
		assert.Equal(t, "new", body["foo"])
		mockMsg.AssertExpectations(t)
		mockDataT.AssertExpectations(t)
		mockCoreObj.AssertExpectations(t)
	})

	t.Run("Set DataT New Item", func(t *testing.T) {
		mockMsg := new(utils.MockRuleMsg)
		mockDataT := new(utils.MockDataT)
		mockCoreObj := new(utils.MockCoreObj)

		mockMsg.On("DataT").Return(mockDataT).Once()
		mockDataT.On("Get", "newObj").Return(nil, false).Once()
		mockDataT.On("NewItem", "someSid", "newObj").Return(mockCoreObj, nil).Once()

		body := map[string]any{}
		mockCoreObj.On("Body").Return(body).Twice()

		err := SetInMsg(mockMsg, "rulemsg://dataT/newObj?sid=someSid", map[string]any{"key": "val"})
		assert.NoError(t, err)
		assert.Equal(t, "val", body["key"])

		mockMsg.AssertExpectations(t)
		mockDataT.AssertExpectations(t)
		mockCoreObj.AssertExpectations(t)
	})
}
