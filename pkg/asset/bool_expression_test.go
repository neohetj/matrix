package asset

import "testing"

func TestValidateBoolExpressionAcceptsEmptyAndLiterals(t *testing.T) {
	for _, value := range []string{"", "true", "false"} {
		if err := ValidateBoolExpression(value); err != nil {
			t.Fatalf("value %q should be accepted, got %v", value, err)
		}
	}
}

func TestValidateBoolExpressionAcceptsTemplates(t *testing.T) {
	values := []string{
		"${config:///feature.enabled}",
		"${config:///feature.enabled?scope=engine,env&default=false}",
	}
	for _, value := range values {
		if err := ValidateBoolExpression(value); err != nil {
			t.Fatalf("value %q should be accepted, got %v", value, err)
		}
	}
}

func TestValidateBoolExpressionRejectsLooseBooleans(t *testing.T) {
	// Deliberately stricter than strconv.ParseBool: a DSL author writing "yes"
	// or "TRUE" should find out at load time, not at the next restart.
	for _, value := range []string{"yes", "no", "1", "0", "TRUE", "False"} {
		if err := ValidateBoolExpression(value); err == nil {
			t.Fatalf("value %q should be rejected", value)
		}
	}
}

func TestValidateBoolExpressionRejectsSurroundingWhitespace(t *testing.T) {
	for _, value := range []string{" true", "false ", " ${config:///a}"} {
		if err := ValidateBoolExpression(value); err == nil {
			t.Fatalf("value %q should be rejected", value)
		}
	}
}

func TestValidateBoolExpressionRejectsMalformedTemplate(t *testing.T) {
	if err := ValidateBoolExpression("${}"); err == nil {
		t.Fatal("an empty placeholder should be rejected")
	}
}
