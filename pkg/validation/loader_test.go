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

func TestValidateLoaderResourcesReportsDuplicateNodeIDs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "code/dsl/rulechains/duplicate_nodes.json", `{
		"ruleChain": {"id": "duplicate_nodes"},
		"metadata": {
			"nodes": [
				{"id": "lookup", "type": "functions", "configuration": {}},
				{"id": "lookup", "type": "functions", "configuration": {}}
			],
			"connections": []
		}
	}`)

	report := ValidateLoaderResources(loader.NewFileProvider(root, 50), LoaderPaths{
		RuleChains: []string{"code/dsl/rulechains"},
	})

	assertIssue(t, report, CodeDuplicateNodeID, SeverityError, TargetNode, "metadata.nodes[1].id")
	if got, want := report.ErrorCount(), 1; got != want {
		t.Fatalf("ErrorCount() = %d, want %d; issues = %#v", got, want, report.Issues)
	}
}

func TestValidateLoaderResourcesReportsRuleChainCycles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "code/dsl/rulechains/cycle.json", `{
		"ruleChain": {"id": "cycle"},
		"metadata": {
			"nodes": [
				{"id": "start", "type": "functions", "configuration": {}},
				{"id": "middle", "type": "functions", "configuration": {}},
				{"id": "end", "type": "functions", "configuration": {}}
			],
			"connections": [
				{"fromId": "start", "toId": "middle", "type": "Success"},
				{"fromId": "middle", "toId": "end", "type": "Success"},
				{"fromId": "end", "toId": "start", "type": "Success"}
			]
		}
	}`)

	report := ValidateLoaderResources(loader.NewFileProvider(root, 50), LoaderPaths{
		RuleChains: []string{"code/dsl/rulechains"},
	})

	assertIssue(t, report, CodeCycleDetected, SeverityError, TargetRuleChain, "metadata.connections")
	if got, want := report.ErrorCount(), 1; got != want {
		t.Fatalf("ErrorCount() = %d, want %d; issues = %#v", got, want, report.Issues)
	}
}

func TestValidateLoaderResourcesBuildsEndpointCatalog(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "code/dsl/rulechains/order_submit.json", `{
		"ruleChain": {"id": "order_submit"},
		"metadata": {
			"nodes": [
				{"id": "start", "type": "action/log", "configuration": {}}
			],
			"connections": []
		}
	}`)
	writeFile(t, root, "code/dsl/rulechains/order_pipeline_stage.json", `{
		"ruleChain": {"id": "order_pipeline_stage"},
		"metadata": {
			"nodes": [
				{"id": "start", "type": "action/log", "configuration": {}}
			],
			"connections": []
		}
	}`)
	writeFile(t, root, "code/dsl/endpoints/http_order_submit.json", `{
		"id": "ep_order_submit",
		"type": "endpoint/http",
		"name": "Submit Order",
		"configuration": {
			"ruleChainId": "order_submit",
			"startNodeId": "start",
			"httpMethod": "POST",
			"httpPath": "/orders",
			"domain": "Commerce",
			"async": false,
			"endpointDefinition": {
				"request": {
					"body": {
						"fields": [
							{"name": "orderId", "bindPath": "rulemsg://metadata/orderId", "type": "string", "required": true}
						]
					}
				},
				"response": {
					"body": {
						"fields": [
							{"name": "status", "bindPath": "rulemsg://metadata/status", "type": "string"}
						]
					}
				}
			}
		}
	}`)
	writeFile(t, root, "code/dsl/endpoints/mcp_order_tools.json", `{
		"id": "ep_order_tools",
		"type": "endpoint/mcp",
		"name": "Order Tools",
		"configuration": {
			"serverName": "order-tools",
			"tools": [
				{
					"name": "get_order",
					"target": {"kind": "rulechain", "id": "order_submit"}
				}
			]
		}
	}`)
	writeFile(t, root, "code/dsl/endpoints/pipeline_order.json", `{
		"id": "ep_order_pipeline",
		"type": "endpoint/pipeline",
		"name": "Order Pipeline",
		"configuration": {
			"channelManager": "ref://channel_manager",
			"exposedChannels": {"input": "orders_in"},
			"stages": [
				{
					"id": "stage_submit",
					"name": "Submit Stage",
					"processor": {"id": "order_pipeline_stage", "type": "rulechain"},
					"inputChannel": "orders_in",
					"outputChannel": "orders_out"
				}
			]
		}
	}`)
	writeFile(t, root, "code/dsl/shared/channel_manager.json", `{
		"ruleChain": {"id": "shared_resources"},
		"metadata": {
			"nodes": [
				{"id": "channel_manager", "type": "resource/channel_manager", "configuration": {}}
			],
			"connections": []
		}
	}`)

	report := ValidateLoaderResources(loader.NewFileProvider(root, 50), LoaderPaths{
		RuleChains: []string{"code/dsl/rulechains"},
		Endpoints:  []string{"code/dsl/endpoints"},
		Shared:     []string{"code/dsl/shared"},
	})

	if report.EndpointCatalog == nil {
		t.Fatalf("expected endpoint catalog")
	}
	if got, want := len(report.EndpointCatalog.Endpoints), 3; got != want {
		t.Fatalf("endpoint catalog length = %d, want %d: %#v", got, want, report.EndpointCatalog.Endpoints)
	}

	httpEndpoint := findEndpointDescriptor(t, report, "ep_order_submit")
	if httpEndpoint.Protocol != "http" {
		t.Fatalf("http endpoint protocol = %s", httpEndpoint.Protocol)
	}
	if httpEndpoint.HTTP == nil || httpEndpoint.HTTP.Method != "POST" || httpEndpoint.HTTP.Path != "/orders" {
		t.Fatalf("http descriptor = %#v", httpEndpoint.HTTP)
	}
	if got, want := httpEndpoint.Targets[0].ID, "order_submit"; got != want {
		t.Fatalf("http target id = %s, want %s", got, want)
	}
	if httpEndpoint.InputMapping == nil || len(httpEndpoint.InputMapping.Fields) != 1 {
		t.Fatalf("http input mapping = %#v", httpEndpoint.InputMapping)
	}

	mcpEndpoint := findEndpointDescriptor(t, report, "ep_order_tools")
	if mcpEndpoint.Protocol != "mcp" {
		t.Fatalf("mcp endpoint protocol = %s", mcpEndpoint.Protocol)
	}
	if mcpEndpoint.MCP == nil || mcpEndpoint.MCP.ServerName != "order-tools" || len(mcpEndpoint.MCP.ToolNames) != 1 {
		t.Fatalf("mcp descriptor = %#v", mcpEndpoint.MCP)
	}
	if got, want := mcpEndpoint.Targets[0].ID, "order_submit"; got != want {
		t.Fatalf("mcp target id = %s, want %s", got, want)
	}

	pipelineEndpoint := findEndpointDescriptor(t, report, "ep_order_pipeline")
	if pipelineEndpoint.Protocol != "pipeline" {
		t.Fatalf("pipeline endpoint protocol = %s", pipelineEndpoint.Protocol)
	}
	if pipelineEndpoint.Pipeline == nil || len(pipelineEndpoint.Pipeline.Stages) != 1 {
		t.Fatalf("pipeline descriptor = %#v", pipelineEndpoint.Pipeline)
	}
	if got, want := pipelineEndpoint.Targets[0].ID, "order_pipeline_stage"; got != want {
		t.Fatalf("pipeline target id = %s, want %s", got, want)
	}
}

func TestValidateLoaderResourcesWithOptionsReportsCatalogAndContractIssues(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "code/dsl/rulechains/catalog_checks.json", `{
		"ruleChain": {"id": "catalog_checks"},
		"metadata": {
			"nodes": [
				{"id": "unknown_node", "type": "custom/missing", "configuration": {}},
				{"id": "unknown_func", "type": "functions", "configuration": {"functionName": "missing_function"}},
				{"id": "standard_func", "type": "functions", "configuration": {"functionName": "standard_lookup"}},
				{"id": "done", "type": "action/log", "configuration": {}}
			],
			"connections": [
				{"fromId": "standard_func", "toId": "done", "type": "Approved"}
			]
		}
	}`)
	writeFile(t, root, "code/dsl/endpoints/invalid_io.json", `{
		"id": "ep_invalid_io",
		"type": "endpoint/http",
		"configuration": {
			"ruleChainId": "catalog_checks",
			"httpMethod": "GET",
			"httpPath": "/catalog-checks",
			"endpointDefinition": {
				"request": {
					"queryParams": {
						"fields": [
							{"name": "id", "bindPath": "not-rulemsg://metadata/id", "type": "definitely_bad"}
						]
					}
				},
				"response": {}
			}
		}
	}`)

	report := ValidateLoaderResourcesWithOptions(loader.NewFileProvider(root, 50), LoaderPaths{
		RuleChains: []string{"code/dsl/rulechains"},
		Endpoints:  []string{"code/dsl/endpoints"},
	}, ValidationOptions{
		KnownNodeTypes: []string{"action/log", "functions"},
		Functions: []FunctionDescriptor{
			{
				ID:          "standard_lookup",
				RoutingMode: "standard",
			},
		},
	})

	assertIssue(t, report, CodeUnknownNodeType, SeverityError, TargetNode, "metadata.nodes[0].type")
	assertIssue(t, report, CodeUnknownFunction, SeverityError, TargetFunction, "metadata.nodes[1].configuration.functionName")
	assertIssue(t, report, CodeInvalidFunctionRelation, SeverityError, TargetConnection, "metadata.connections[0]")
	assertIssue(t, report, CodeInvalidEndpointIO, SeverityError, TargetEndpoint, "configuration.endpointDefinition.request.queryParams.fields[0].bindPath")
	assertIssue(t, report, CodeInvalidEndpointIO, SeverityError, TargetEndpoint, "configuration.endpointDefinition.request.queryParams.fields[0].type")

	if got, want := report.ErrorCount(), 5; got != want {
		t.Fatalf("ErrorCount() = %d, want %d; issues = %#v", got, want, report.Issues)
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

func findEndpointDescriptor(t *testing.T, report *Report, id string) EndpointDescriptor {
	t.Helper()
	if report.EndpointCatalog == nil {
		t.Fatalf("endpoint catalog is nil")
	}
	for _, endpoint := range report.EndpointCatalog.Endpoints {
		if endpoint.ID == id {
			return endpoint
		}
	}
	t.Fatalf("endpoint descriptor %s not found in %#v", id, report.EndpointCatalog.Endpoints)
	return EndpointDescriptor{}
}
