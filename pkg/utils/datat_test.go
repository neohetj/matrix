package utils_test

import (
	"reflect"
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/utils"
	testutils "github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/mock"
)

type MyStruct struct {
	Name string
}

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

func TestSetCoreObjBody_PointerToMapToStruct(t *testing.T) {
	mockObj := new(testutils.MockCoreObj)
	var myStruct MyStruct
	mockObj.On("Body").Return(&myStruct)

	input := map[string]any{"Name": "testMap"}
	inputPtr := &input

	// SetBody shouldn't be called directly for Decode, but Body() is.
	// The Decode happens in place.

	ok, err := utils.SetCoreObjBody(mockObj, inputPtr, "SomeSID")
	if err != nil {
		t.Fatalf("SetCoreObjBody failed: %v", err)
	}
	if !ok {
		t.Fatalf("SetCoreObjBody returned false")
	}

	if myStruct.Name != "testMap" {
		t.Errorf("Expected Name testMap, got %s", myStruct.Name)
	}
}

func TestSetCoreObjBody_PointerToStructToMap(t *testing.T) {
	mockObj := new(testutils.MockCoreObj)
	// Initialize map because mapstructure might need it to be non-nil or it allocates?
	// reflect.New(map type) gives a pointer to a nil map.
	// mapstructure handles it?
	// Let's use a pointer to a map variable.
	var mapBody map[string]any
	mockObj.On("Body").Return(&mapBody)

	input := MyStruct{Name: "testStruct"}
	inputPtr := &input

	ok, err := utils.SetCoreObjBody(mockObj, inputPtr, "SomeSID")
	if err != nil {
		t.Fatalf("SetCoreObjBody failed: %v", err)
	}
	if !ok {
		t.Fatalf("SetCoreObjBody returned false")
	}

	if mapBody["Name"] != "testStruct" {
		t.Errorf("Expected Name testStruct, got %v", mapBody["Name"])
	}
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

func TestSetCoreObjBody_GenericSliceToStructSlice(t *testing.T) {
	mockObj := new(testutils.MockCoreObj)
	var structSlice []MyStruct
	mockObj.On("Body").Return(&structSlice)

	input := []any{
		map[string]any{"Name": "test1"},
		map[string]any{"Name": "test2"},
	}

	// SetBody should NOT be called because Decode modifies the body directly via pointer
	// But we need to ensure mockObj.Body() is called.
	// It is called at the beginning of SetCoreObjBody.

	ok, err := utils.SetCoreObjBody(mockObj, input, "SomeSID")
	if err != nil {
		t.Fatalf("SetCoreObjBody failed: %v", err)
	}
	if !ok {
		t.Fatalf("SetCoreObjBody returned false")
	}

	// Verify structSlice content
	if len(structSlice) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(structSlice))
	}
	if structSlice[0].Name != "test1" {
		t.Errorf("Expected Name test1, got %s", structSlice[0].Name)
	}
	if structSlice[1].Name != "test2" {
		t.Errorf("Expected Name test2, got %s", structSlice[1].Name)
	}
}
