package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type PlannerStep struct {
	StepID      string `json:"step_id"`
	StepType    string `json:"step_type"`
	Description string `json:"description"`
}

type PlannerOutput struct {
	StepByStepPlan []PlannerStep `json:"step_by_step_plan"`
}

func TestDecodePlannerOutput(t *testing.T) {
	// Construct data similar to the user's feedback
	data := map[string]interface{}{
		"step_by_step_plan": []interface{}{
			map[string]interface{}{
				"step_id":     "Step 1",
				"step_type":   "normal",
				"description": "Open the Microsoft Edge browser.",
			},
		},
	}

	targetStructPtr := &PlannerOutput{}

	// Attempt to decode
	err := Decode(data, targetStructPtr)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify the result
	if len(targetStructPtr.StepByStepPlan) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(targetStructPtr.StepByStepPlan))
	}

	step := targetStructPtr.StepByStepPlan[0]
	if step.StepID != "Step 1" {
		t.Errorf("Expected StepID 'Step 1', got '%s'", step.StepID)
	}
	if step.StepType != "normal" {
		t.Errorf("Expected StepType 'normal', got '%s'", step.StepType)
	}
	if step.Description != "Open the Microsoft Edge browser." {
		t.Errorf("Expected Description 'Open the Microsoft Edge browser.', got '%s'", step.Description)
	}

	t.Logf("Decode successful: %+v", targetStructPtr)
}

func TestDecodePlannerOutput_JSONTagMismatch(t *testing.T) {
	// Test case where map keys match JSON tags
	data := map[string]interface{}{
		"step_by_step_plan": []interface{}{
			map[string]interface{}{
				"step_id":     "Step 1",
				"step_type":   "normal",
				"description": "Open the Microsoft Edge browser.",
			},
		},
	}

	targetStructPtr := &PlannerOutput{}
	err := Decode(data, targetStructPtr)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
}

func TestReflectAnalysis(t *testing.T) {
	p := &PlannerOutput{}
	v := reflect.ValueOf(p).Elem()
	f := v.FieldByName("StepByStepPlan")
	t.Logf("Field StepByStepPlan type: %s", f.Type())
	t.Logf("Field StepByStepPlan kind: %s", f.Kind())
}

type TestStructWithValidation struct {
	Name string `json:"name"`
}

func (v *TestStructWithValidation) UnmarshalJSON(data []byte) error {
	type Alias TestStructWithValidation
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(v),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if v.Name == "invalid" {
		return fmt.Errorf("validation error: name cannot be 'invalid'")
	}
	return nil
}

func TestDecode_JsonUnmarshaler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		dataSuccess := map[string]interface{}{
			"name": "valid",
		}
		targetSuccess := &TestStructWithValidation{}
		if err := Decode(dataSuccess, targetSuccess); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		if targetSuccess.Name != "valid" {
			t.Errorf("Expected 'valid', got '%s'", targetSuccess.Name)
		}
	})

	t.Run("Validation Failure", func(t *testing.T) {
		dataFail := map[string]interface{}{
			"name": "invalid",
		}
		targetFail := &TestStructWithValidation{}
		err := Decode(dataFail, targetFail)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "validation error: name cannot be 'invalid'") {
			t.Errorf("Expected error to contain validation message, got: %v", err)
		}
	})
}
