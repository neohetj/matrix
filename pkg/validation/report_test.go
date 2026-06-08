package validation

import (
	"encoding/json"
	"testing"
)

func TestReportJSONSchemaAndErrorState(t *testing.T) {
	report := NewReport("matrix.validation.report/v1", ModeReportOnly, Scope{
		Kind: "rulechain",
		ID:   "rc_order_submit",
	})
	report.AddIssue(Issue{
		Code:     CodeDanglingConnection,
		Severity: SeverityError,
		Message:  "connection target node is missing",
		Target: Target{
			Kind:       TargetConnection,
			ID:         "edge-start-missing",
			Path:       "metadata.connections[0]",
			SourcePath: "code/dsl/rulechains/order_submit.json",
		},
		Details: map[string]any{
			"fromId": "start",
			"toId":   "missing",
		},
	})
	report.AddIssue(Issue{
		Code:     CodeOptionalFallback,
		Severity: SeverityWarning,
		Message:  "optional shared resource fallback will be used",
		Target: Target{
			Kind: TargetSharedRef,
			ID:   "ref://storage/default",
		},
	})

	if !report.HasErrors() {
		t.Fatalf("expected report to have errors")
	}
	if report.ErrorCount() != 1 {
		t.Fatalf("expected one error, got %d", report.ErrorCount())
	}
	if report.WarningCount() != 1 {
		t.Fatalf("expected one warning, got %d", report.WarningCount())
	}

	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal report payload: %v", err)
	}

	if decoded["schemaVersion"] != "matrix.validation.report/v1" {
		t.Fatalf("schemaVersion = %v", decoded["schemaVersion"])
	}
	if decoded["mode"] != string(ModeReportOnly) {
		t.Fatalf("mode = %v", decoded["mode"])
	}
	if decoded["hasErrors"] != true {
		t.Fatalf("hasErrors = %v", decoded["hasErrors"])
	}
	if decoded["errorCount"] != float64(1) {
		t.Fatalf("errorCount = %v", decoded["errorCount"])
	}
	if decoded["warningCount"] != float64(1) {
		t.Fatalf("warningCount = %v", decoded["warningCount"])
	}

	issues, ok := decoded["issues"].([]any)
	if !ok || len(issues) != 2 {
		t.Fatalf("issues = %#v", decoded["issues"])
	}
	firstIssue := issues[0].(map[string]any)
	if firstIssue["code"] != string(CodeDanglingConnection) {
		t.Fatalf("first issue code = %v", firstIssue["code"])
	}
	if firstIssue["severity"] != string(SeverityError) {
		t.Fatalf("first issue severity = %v", firstIssue["severity"])
	}

	target := firstIssue["target"].(map[string]any)
	if target["kind"] != string(TargetConnection) {
		t.Fatalf("target kind = %v", target["kind"])
	}
	if target["sourcePath"] != "code/dsl/rulechains/order_submit.json" {
		t.Fatalf("target sourcePath = %v", target["sourcePath"])
	}
}
