package utils_test

import (
	"reflect"
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/utils"
	testutils "github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/mock"
)

func TestSetCoreObjBody_SliceAny(t *testing.T) {
	// Setup: Mock CoreObj
	mockObj := new(testutils.MockCoreObj)

	// Mock Body() to return pointer to Items slice
	var sliceAny []any
	mockObj.On("Body").Return(&sliceAny)

	// We need to mock SetBody to verify it's called with the correct value
	// datat.go calls SetBody with a pointer to the slice
	expected := []any{"a", "b", 123}
	mockObj.On("SetBody", mock.MatchedBy(func(arg *[]any) bool {
		if arg == nil {
			return false
		}
		return reflect.DeepEqual(*arg, expected)
	})).Return(nil)

	// Action: Call SetCoreObjBody with []any
	value := []any{"a", "b", 123}
	ok, err := utils.SetCoreObjBody(mockObj, value, cnst.SID_SLICE_ANY)

	// Verify
	if err != nil {
		t.Errorf("SetCoreObjBody returned error: %v", err)
	}
	if !ok {
		t.Errorf("SetCoreObjBody returned false")
	}

	mockObj.AssertExpectations(t)
}

func TestSetCoreObjBody_SliceAny_Pointer(t *testing.T) {
	mockObj := new(testutils.MockCoreObj)
	var sliceAny []any
	mockObj.On("Body").Return(&sliceAny)

	expected := []any{"x", "y"}
	// For pointer input, datat.go passes the pointer directly
	// So we expect *[]any with value {"x", "y"}
	mockObj.On("SetBody", mock.MatchedBy(func(arg *[]any) bool {
		if arg == nil {
			return false
		}
		return reflect.DeepEqual(*arg, expected)
	})).Return(nil)

	value := []any{"x", "y"}
	// Pass pointer to slice
	ok, err := utils.SetCoreObjBody(mockObj, &value, cnst.SID_SLICE_ANY)

	if err != nil {
		t.Errorf("SetCoreObjBody returned error: %v", err)
	}
	if !ok {
		t.Errorf("SetCoreObjBody returned false")
	}

	mockObj.AssertExpectations(t)
}
