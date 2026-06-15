package validation

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

func validateKnownNodeTypes(report *Report, defs []*types.RuleChainDef, knownNodeTypes []string) {
	known := stringSet(knownNodeTypes)
	if len(known) == 0 {
		return
	}
	for _, def := range defs {
		if def == nil {
			continue
		}
		for i, node := range def.Metadata.Nodes {
			nodeType := strings.TrimSpace(node.Type)
			if nodeType == "" {
				continue
			}
			if _, ok := known[nodeType]; ok {
				continue
			}
			report.AddIssue(Issue{
				Code:     CodeUnknownNodeType,
				Severity: SeverityError,
				Message:  "node type is not present in the supplied node catalog",
				Target: Target{
					Kind:       TargetNode,
					ID:         node.ID,
					Path:       fmt.Sprintf("metadata.nodes[%d].type", i),
					SourcePath: node.SourcePath,
				},
				Details: map[string]any{
					"ruleChainId": def.RuleChain.ID,
					"nodeType":    nodeType,
				},
			})
		}
	}
}

func validateFunctionCatalog(report *Report, defs []*types.RuleChainDef, functions []FunctionDescriptor) {
	known := functionDescriptorMap(functions)
	if len(known) == 0 {
		return
	}
	for _, def := range defs {
		if def == nil {
			continue
		}
		for nodeIndex, node := range def.Metadata.Nodes {
			if node.Type != "functions" {
				continue
			}
			functionName := stringConfigValue(node.Configuration, "functionName")
			if functionName == "" {
				continue
			}
			function, ok := known[functionName]
			if !ok {
				report.AddIssue(Issue{
					Code:     CodeUnknownFunction,
					Severity: SeverityError,
					Message:  "function is not present in the supplied function catalog",
					Target: Target{
						Kind:       TargetFunction,
						ID:         functionName,
						Path:       fmt.Sprintf("metadata.nodes[%d].configuration.functionName", nodeIndex),
						SourcePath: node.SourcePath,
					},
					Details: map[string]any{
						"ruleChainId":  def.RuleChain.ID,
						"functionName": functionName,
						"nodeId":       node.ID,
					},
				})
				continue
			}
			validateFunctionRelations(report, def, node, functionName, function)
		}
	}
}

func validateFunctionRelations(report *Report, def *types.RuleChainDef, node types.NodeDef, functionName string, function FunctionDescriptor) {
	mode := function.RoutingMode.Normalize()
	allowed := map[string]struct{}{
		"Success": {},
		"Failure": {},
	}
	if mode == types.FunctionRoutingModeDecision {
		for _, relation := range function.DeclaredRelations {
			relation = strings.TrimSpace(relation)
			if relation != "" {
				allowed[relation] = struct{}{}
			}
		}
	}
	for i, conn := range def.Metadata.Connections {
		if conn.FromID != node.ID {
			continue
		}
		if _, ok := allowed[conn.Type]; ok {
			continue
		}
		report.AddIssue(Issue{
			Code:     CodeInvalidFunctionRelation,
			Severity: SeverityError,
			Message:  "function node connection uses a relation not allowed by the function routing contract",
			Target: Target{
				Kind:       TargetConnection,
				ID:         fmt.Sprintf("%s:%s:%s", conn.FromID, conn.Type, conn.ToID),
				Path:       fmt.Sprintf("metadata.connections[%d]", i),
				SourcePath: node.SourcePath,
			},
			Details: map[string]any{
				"ruleChainId":       def.RuleChain.ID,
				"nodeId":            node.ID,
				"functionName":      functionName,
				"relation":          conn.Type,
				"routingMode":       string(mode),
				"declaredRelations": function.DeclaredRelations,
			},
		})
	}
}

func validateEndpointTargets(report *Report, endpointDefs []*types.NodeDef, ruleChainIDs map[string]struct{}) {
	for _, endpointDef := range endpointDefs {
		if endpointDef == nil {
			continue
		}
		targetID := stringConfigValue(endpointDef.Configuration, "ruleChainId")
		if targetID == "" {
			continue
		}
		if _, ok := ruleChainIDs[targetID]; ok {
			continue
		}
		report.AddIssue(Issue{
			Code:     CodeMissingEndpointTarget,
			Severity: SeverityError,
			Message:  "endpoint target rulechain is missing",
			Target: Target{
				Kind:       TargetEndpoint,
				ID:         endpointDef.ID,
				Path:       "configuration.ruleChainId",
				SourcePath: endpointDef.SourcePath,
			},
			Details: map[string]any{
				"ruleChainId":  targetID,
				"endpointType": endpointDef.Type,
			},
		})
	}
}

func validateEndpointIOContracts(report *Report, endpointDefs []*types.NodeDef) {
	for _, endpointDef := range endpointDefs {
		if endpointDef == nil || endpointDef.Type != "endpoint/http" {
			continue
		}
		endpointDefinition, ok := mapValue(endpointDef.Configuration, "endpointDefinition")
		if !ok {
			continue
		}
		if request, ok := mapValue(endpointDefinition, "request"); ok {
			validateEndpointFields(report, endpointDef, sliceValue(request, "pathParams"), "configuration.endpointDefinition.request.pathParams")
			validateEndpointPacket(report, endpointDef, mapValueOrNil(request, "queryParams"), "configuration.endpointDefinition.request.queryParams")
			validateEndpointPacket(report, endpointDef, mapValueOrNil(request, "headers"), "configuration.endpointDefinition.request.headers")
			validateEndpointPacket(report, endpointDef, mapValueOrNil(request, "body"), "configuration.endpointDefinition.request.body")
		}
		if response, ok := mapValue(endpointDefinition, "response"); ok {
			validateEndpointPacket(report, endpointDef, mapValueOrNil(response, "headers"), "configuration.endpointDefinition.response.headers")
			validateEndpointPacket(report, endpointDef, mapValueOrNil(response, "body"), "configuration.endpointDefinition.response.body")
		}
	}
}

func validateEndpointPacket(report *Report, endpointDef *types.NodeDef, packet map[string]any, basePath string) {
	if len(packet) == 0 {
		return
	}
	if rawMapAll, ok := packet["mapAll"]; ok {
		validateRuleMsgURI(report, endpointDef, rawMapAll, basePath+".mapAll")
	}
	validateEndpointFields(report, endpointDef, sliceValue(packet, "fields"), basePath+".fields")
}

func validateEndpointFields(report *Report, endpointDef *types.NodeDef, fields []any, basePath string) {
	for i, rawField := range fields {
		field, ok := rawField.(map[string]any)
		if !ok {
			continue
		}
		fieldPath := fmt.Sprintf("%s[%d]", basePath, i)
		if rawBindPath, ok := field["bindPath"]; ok {
			validateRuleMsgURI(report, endpointDef, rawBindPath, fieldPath+".bindPath")
		}
		if rawType, ok := field["type"]; ok {
			validateEndpointFieldType(report, endpointDef, rawType, fieldPath+".type")
		}
	}
}

func validateRuleMsgURI(report *Report, endpointDef *types.NodeDef, raw any, targetPath string) {
	uri, ok := raw.(string)
	if !ok || strings.TrimSpace(uri) == "" {
		return
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		addEndpointIOIssue(report, endpointDef, targetPath, fmt.Sprintf("invalid rulemsg URI: %v", err), map[string]any{"value": uri})
		return
	}
	if err := (asset.RuleMsgAsset{}).Validate(parsed); err != nil {
		addEndpointIOIssue(report, endpointDef, targetPath, err.Error(), map[string]any{"value": uri})
	}
}

func validateEndpointFieldType(report *Report, endpointDef *types.NodeDef, raw any, targetPath string) {
	fieldType, ok := raw.(string)
	if !ok || strings.TrimSpace(fieldType) == "" {
		return
	}
	if cnst.MType(fieldType).IsSupported() {
		return
	}
	addEndpointIOIssue(report, endpointDef, targetPath, "endpoint IO field type is not supported", map[string]any{
		"type": fieldType,
	})
}

func addEndpointIOIssue(report *Report, endpointDef *types.NodeDef, targetPath string, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	details["endpointType"] = endpointDef.Type
	report.AddIssue(Issue{
		Code:     CodeInvalidEndpointIO,
		Severity: SeverityError,
		Message:  message,
		Target: Target{
			Kind:       TargetEndpoint,
			ID:         endpointDef.ID,
			Path:       targetPath,
			SourcePath: endpointDef.SourcePath,
		},
		Details: details,
	})
}
