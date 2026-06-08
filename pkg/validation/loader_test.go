package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neohetj/matrix/internal/loader"
)

func TestValidateLoaderResourcesReportsParseFailureAndMissingTargets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "code/dsl/rulechains/valid.json", `{
		"ruleChain": {"id": "valid"},
		"metadata": {
			"nodes": [
				{"id": "start", "type": "action/log", "configuration": {}}
			],
			"connections": [
				{"fromId": "start", "toId": "missing", "type": "Success"}
			]
		}
	}`)
	writeFile(t, root, "code/dsl/rulechains/broken.json", `{not json`)
	writeFile(t, root, "code/dsl/endpoints/http_missing.json", `{
		"id": "ep_missing",
		"type": "endpoint/http",
		"configuration": {
			"ruleChainId": "missing_chain",
			"httpMethod": "POST",
			"httpPath": "/orders"
		}
	}`)

	report := ValidateLoaderResources(loader.NewFileProvider(root, 50), LoaderPaths{
		RuleChains: []string{"code/dsl/rulechains"},
		Endpoints:  []string{"code/dsl/endpoints"},
	})

	assertIssue(t, report, CodeLoaderFailure, SeverityError, TargetLoader, "code/dsl/rulechains/broken.json")
	assertIssue(t, report, CodeDanglingConnection, SeverityError, TargetConnection, "metadata.connections[0]")
	assertIssue(t, report, CodeMissingEndpointTarget, SeverityError, TargetEndpoint, "configuration.ruleChainId")

	if got, want := report.ErrorCount(), 3; got != want {
		t.Fatalf("ErrorCount() = %d, want %d; issues = %#v", got, want, report.Issues)
	}
}

func TestValidateLoaderResourcesReportsOptionalSharedFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "code/dsl/rulechains/ref_fallback.json", `{
		"ruleChain": {"id": "ref_fallback"},
		"metadata": {
			"nodes": [
				{
					"id": "uses_store",
					"type": "functions",
					"configuration": {
						"store": {
							"resource": "ref://missing_store",
							"optional": true,
							"fallback": "local"
						}
					}
				}
			],
			"connections": []
		}
	}`)

	report := ValidateLoaderResources(loader.NewFileProvider(root, 50), LoaderPaths{
		RuleChains: []string{"code/dsl/rulechains"},
		Shared:     []string{"code/dsl/shared"},
	})

	assertIssue(t, report, CodeOptionalFallback, SeverityWarning, TargetSharedRef, "configuration.store.resource")
	if report.HasErrors() {
		t.Fatalf("expected optional fallback to be warning-only, got %#v", report.Issues)
	}
	if got, want := report.WarningCount(), 1; got != want {
		t.Fatalf("WarningCount() = %d, want %d", got, want)
	}
}

func writeFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertIssue(t *testing.T, report *Report, code IssueCode, severity Severity, targetKind TargetKind, targetPath string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Code == code && issue.Severity == severity && issue.Target.Kind == targetKind && issue.Target.Path == targetPath {
			return
		}
	}
	t.Fatalf("missing issue code=%s severity=%s targetKind=%s targetPath=%s in %#v", code, severity, targetKind, targetPath, report.Issues)
}
