package inspection

import (
	"encoding/json"
	"testing"

	"github.com/neohetj/matrix/pkg/validation"
)

func TestSnapshotJSONSchemaIncludesRuntimeFactsAndValidation(t *testing.T) {
	report := validation.NewReport("matrix.validation.report/v1", validation.ModeReportOnly, validation.Scope{
		Kind: "engine",
		ID:   "test-engine",
	})
	report.AddIssue(validation.Issue{
		Code:     validation.CodeUnknownFunction,
		Severity: validation.SeverityWarning,
		Message:  "function metadata is missing from catalog",
		Target: validation.Target{
			Kind: validation.TargetFunction,
			ID:   "sendEmail",
		},
	})

	snapshot := NewSnapshot("matrix.inspection.snapshot/v1", "test-engine")
	snapshot.Validation = report
	snapshot.RuleChains = append(snapshot.RuleChains, RuntimeFactDescriptor{
		Kind:       FactRuleChain,
		ID:         "rc_order_submit",
		Name:       "Order Submit",
		Type:       "rulechain",
		SourcePath: "code/dsl/rulechains/order_submit.json",
		Refs:       []string{"ref://storage/default"},
		Metadata: map[string]any{
			"viewType": "runtime",
		},
	})
	snapshot.Endpoints = append(snapshot.Endpoints, RuntimeFactDescriptor{
		Kind: FactEndpoint,
		ID:   "ep_order_submit",
		Type: "endpoint/http",
		Refs: []string{"rc_order_submit"},
	})
	snapshot.Functions = append(snapshot.Functions, RuntimeFactDescriptor{
		Kind: FactFunction,
		ID:   "sendEmail",
		Type: "functions",
	})

	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal snapshot payload: %v", err)
	}

	if decoded["schemaVersion"] != "matrix.inspection.snapshot/v1" {
		t.Fatalf("schemaVersion = %v", decoded["schemaVersion"])
	}
	if decoded["engineId"] != "test-engine" {
		t.Fatalf("engineId = %v", decoded["engineId"])
	}

	ruleChains := decoded["ruleChains"].([]any)
	if len(ruleChains) != 1 {
		t.Fatalf("ruleChains length = %d", len(ruleChains))
	}
	ruleChain := ruleChains[0].(map[string]any)
	if ruleChain["kind"] != string(FactRuleChain) {
		t.Fatalf("ruleChain kind = %v", ruleChain["kind"])
	}
	if ruleChain["sourcePath"] != "code/dsl/rulechains/order_submit.json" {
		t.Fatalf("ruleChain sourcePath = %v", ruleChain["sourcePath"])
	}

	validationPayload := decoded["validation"].(map[string]any)
	if validationPayload["warningCount"] != float64(1) {
		t.Fatalf("validation warningCount = %v", validationPayload["warningCount"])
	}
}
