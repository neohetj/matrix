package utils_test

import (
	"testing"

	"github.com/neohetj/matrix/pkg/utils"
	testutils "github.com/neohetj/matrix/test/utils"
)

func TestSetCoreObjBody_NilPanic(t *testing.T) {
	// Setup: Use MockCoreObj from test/utils
	mockObj := new(testutils.MockCoreObj)

	// Expectation: Body() should NOT be called because we return early when value is nil

	// Action: Pass nil value
	// We expect this to fail gracefully or return error, but definitely not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetCoreObjBody panicked with nil value: %v", r)
		}
	}()

	_, err := utils.SetCoreObjBody(mockObj, nil, "SomeSID")

	if err != nil {
		t.Logf("Error: %v", err)
	} else {
		t.Log("Success: No panic and no error")
	}
}
