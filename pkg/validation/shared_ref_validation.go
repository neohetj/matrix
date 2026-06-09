package validation

import (
	"fmt"
	"strings"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

func validateSharedRefs(report *Report, defs []*types.RuleChainDef, endpointDefs []*types.NodeDef, sharedIDs map[string]struct{}) {
	for _, def := range defs {
		for _, node := range def.Metadata.Nodes {
			scanConfigRefs(report, node.Configuration, refScanContext{
				targetID:   node.ID,
				sourcePath: node.SourcePath,
				basePath:   "configuration",
				sharedIDs:  sharedIDs,
			})
		}
	}
	for _, endpointDef := range endpointDefs {
		if endpointDef == nil {
			continue
		}
		scanConfigRefs(report, endpointDef.Configuration, refScanContext{
			targetID:   endpointDef.ID,
			sourcePath: endpointDef.SourcePath,
			basePath:   "configuration",
			sharedIDs:  sharedIDs,
		})
	}
}

type refScanContext struct {
	targetID   string
	sourcePath string
	basePath   string
	sharedIDs  map[string]struct{}
}

func scanConfigRefs(report *Report, value any, ctx refScanContext) {
	switch typed := value.(type) {
	case types.ConfigMap:
		scanConfigMap(report, map[string]any(typed), ctx)
	case map[string]any:
		scanConfigMap(report, typed, ctx)
	case []any:
		for i, child := range typed {
			next := ctx
			next.basePath = fmt.Sprintf("%s[%d]", ctx.basePath, i)
			if refID, ok := refIDFromValue(child); ok {
				if _, exists := ctx.sharedIDs[refID]; !exists {
					report.AddIssue(sharedRefIssue(refID, child, map[string]any{}, next))
				}
			}
			scanConfigRefs(report, child, next)
		}
	}
}

func scanConfigMap(report *Report, typed map[string]any, ctx refScanContext) {
	for key, child := range typed {
		next := ctx
		next.basePath = appendPath(ctx.basePath, key)
		if refID, ok := refIDFromValue(child); ok {
			if _, exists := ctx.sharedIDs[refID]; !exists {
				report.AddIssue(sharedRefIssue(refID, child, typed, next))
			}
		}
		scanConfigRefs(report, child, next)
	}
}

func sharedRefIssue(refID string, raw any, parent map[string]any, ctx refScanContext) Issue {
	severity := SeverityError
	code := CodeMissingSharedRef
	message := "shared resource reference is missing"
	if isOptionalFallback(parent) {
		severity = SeverityWarning
		code = CodeOptionalFallback
		message = "shared resource reference is missing and optional fallback is declared"
	}
	return Issue{
		Code:     code,
		Severity: severity,
		Message:  message,
		Target: Target{
			Kind:       TargetSharedRef,
			ID:         "ref://" + refID,
			Path:       ctx.basePath,
			SourcePath: ctx.sourcePath,
		},
		Details: map[string]any{
			"nodeId": ctx.targetID,
			"refId":  refID,
			"value":  raw,
		},
	}
}

func refIDFromValue(value any) (string, bool) {
	raw, ok := value.(string)
	if !ok || !strings.HasPrefix(raw, "ref://") {
		return "", false
	}
	parsed, err := asset.ParseRef(raw)
	if err != nil || parsed.RefID == "" {
		return "", false
	}
	return parsed.RefID, true
}

func isOptionalFallback(parent map[string]any) bool {
	if optional, ok := parent["optional"].(bool); ok && optional {
		return true
	}
	for _, key := range []string{"fallback", "fallbackUri", "fallbackURI", "default", "defaultValue"} {
		if value, ok := parent[key]; ok && value != nil {
			return true
		}
	}
	return false
}
