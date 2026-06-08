package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

type LoaderPaths struct {
	RuleChains []string
	Endpoints  []string
	Shared     []string
}

func ValidateLoaderResources(provider types.ResourceProvider, paths LoaderPaths) *Report {
	report := NewReport(DefaultReportSchemaVersion, ModeReportOnly, Scope{
		Kind: "loader",
		ID:   providerName(provider),
	})
	if provider == nil {
		report.AddIssue(Issue{
			Code:     CodeLoaderFailure,
			Severity: SeverityError,
			Message:  "resource provider is nil",
			Target: Target{
				Kind: TargetLoader,
			},
		})
		return report
	}

	ruleChainIDs := map[string]struct{}{}
	sharedIDs := map[string]struct{}{}
	var defs []*types.RuleChainDef
	var endpointDefs []*types.NodeDef

	scanRuleChainPaths(provider, paths.RuleChains, report, ruleChainIDs, &defs)
	scanSharedPaths(provider, paths.Shared, report, sharedIDs)
	scanEndpointPaths(provider, paths.Endpoints, report, &endpointDefs)
	validateDuplicateNodeIDs(report, defs)
	validateRuleChainConnections(report, defs)
	validateRuleChainCycles(report, defs)
	validateEndpointTargets(report, endpointDefs, ruleChainIDs)
	validateSharedRefs(report, defs, endpointDefs, sharedIDs)

	return report
}

func scanRuleChainPaths(provider types.ResourceProvider, paths []string, report *Report, ruleChainIDs map[string]struct{}, defs *[]*types.RuleChainDef) {
	for _, basePath := range paths {
		walkJSON(provider, basePath, report, TargetRuleChain, func(filePath string, content []byte) {
			var def types.RuleChainDef
			if err := json.Unmarshal(content, &def); err != nil {
				addLoaderFailure(report, filePath, fmt.Errorf("decode rulechain: %w", err))
				return
			}
			if def.RuleChain.ID == "" {
				def.RuleChain.ID = strings.TrimSuffix(filepath.Base(filePath), ".json")
			}
			setSourcePath(def.Metadata.Nodes, filePath)
			ruleChainIDs[def.RuleChain.ID] = struct{}{}
			*defs = append(*defs, &def)
		})
	}
}

func scanSharedPaths(provider types.ResourceProvider, paths []string, report *Report, sharedIDs map[string]struct{}) {
	for _, basePath := range paths {
		walkJSON(provider, basePath, report, TargetSharedRef, func(filePath string, content []byte) {
			var def types.RuleChainDef
			if err := json.Unmarshal(content, &def); err != nil {
				addLoaderFailure(report, filePath, fmt.Errorf("decode shared nodes: %w", err))
				return
			}
			for _, node := range def.Metadata.Nodes {
				if strings.TrimSpace(node.ID) != "" {
					sharedIDs[node.ID] = struct{}{}
				}
			}
		})
	}
}

func scanEndpointPaths(provider types.ResourceProvider, paths []string, report *Report, endpoints *[]*types.NodeDef) {
	for _, basePath := range paths {
		walkJSON(provider, basePath, report, TargetEndpoint, func(filePath string, content []byte) {
			var def types.NodeDef
			if err := json.Unmarshal(content, &def); err != nil {
				addLoaderFailure(report, filePath, fmt.Errorf("decode endpoint: %w", err))
				return
			}
			def.SourcePath = filePath
			*endpoints = append(*endpoints, &def)
		})
	}
}

func walkJSON(provider types.ResourceProvider, basePath string, report *Report, targetKind TargetKind, handle func(filePath string, content []byte)) {
	basePath = filepath.ToSlash(strings.TrimSpace(basePath))
	if basePath == "" {
		return
	}
	err := provider.WalkDir(basePath, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			report.AddIssue(Issue{
				Code:     CodeLoaderFailure,
				Severity: SeverityError,
				Message:  err.Error(),
				Target: Target{
					Kind: targetKind,
					Path: filepath.ToSlash(filePath),
				},
			})
			return nil
		}
		if d == nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		filePath = filepath.ToSlash(filePath)
		res, readErr := provider.ReadFile(filePath)
		if readErr != nil {
			addLoaderFailure(report, filePath, fmt.Errorf("read file: %w", readErr))
			return nil
		}
		handle(filePath, res.Content)
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		report.AddIssue(Issue{
			Code:     CodeLoaderFailure,
			Severity: SeverityError,
			Message:  err.Error(),
			Target: Target{
				Kind: targetKind,
				Path: basePath,
			},
		})
	}
}

func validateDuplicateNodeIDs(report *Report, defs []*types.RuleChainDef) {
	for _, def := range defs {
		if def == nil {
			continue
		}
		firstIndexByID := map[string]int{}
		for i, node := range def.Metadata.Nodes {
			nodeID := strings.TrimSpace(node.ID)
			if nodeID == "" {
				continue
			}
			if firstIndex, exists := firstIndexByID[nodeID]; exists {
				report.AddIssue(Issue{
					Code:     CodeDuplicateNodeID,
					Severity: SeverityError,
					Message:  "rulechain contains duplicate node id",
					Target: Target{
						Kind:       TargetNode,
						ID:         nodeID,
						Path:       fmt.Sprintf("metadata.nodes[%d].id", i),
						SourcePath: node.SourcePath,
					},
					Details: map[string]any{
						"ruleChainId":    def.RuleChain.ID,
						"nodeId":         nodeID,
						"firstIndex":     firstIndex,
						"duplicateIndex": i,
					},
				})
				continue
			}
			firstIndexByID[nodeID] = i
		}
	}
}

func validateRuleChainConnections(report *Report, defs []*types.RuleChainDef) {
	for _, def := range defs {
		if def == nil {
			continue
		}
		nodeIDs := map[string]struct{}{}
		sourcePathByNode := map[string]string{}
		for _, node := range def.Metadata.Nodes {
			nodeIDs[node.ID] = struct{}{}
			sourcePathByNode[node.ID] = node.SourcePath
		}
		for i, conn := range def.Metadata.Connections {
			if _, ok := nodeIDs[conn.FromID]; !ok {
				report.AddIssue(danglingConnectionIssue(conn, i, sourcePathByNode[conn.FromID], "fromId"))
			}
			if _, ok := nodeIDs[conn.ToID]; !ok {
				report.AddIssue(danglingConnectionIssue(conn, i, sourcePathByNode[conn.FromID], "toId"))
			}
		}
	}
}

func validateRuleChainCycles(report *Report, defs []*types.RuleChainDef) {
	for _, def := range defs {
		if def == nil {
			continue
		}
		nodeOrder := make([]string, 0, len(def.Metadata.Nodes))
		nodeIDs := map[string]struct{}{}
		for _, node := range def.Metadata.Nodes {
			nodeID := strings.TrimSpace(node.ID)
			if nodeID == "" {
				continue
			}
			if _, exists := nodeIDs[nodeID]; !exists {
				nodeOrder = append(nodeOrder, nodeID)
			}
			nodeIDs[nodeID] = struct{}{}
		}

		adjacency := map[string][]string{}
		for _, conn := range def.Metadata.Connections {
			if _, ok := nodeIDs[conn.FromID]; !ok {
				continue
			}
			if _, ok := nodeIDs[conn.ToID]; !ok {
				continue
			}
			adjacency[conn.FromID] = append(adjacency[conn.FromID], conn.ToID)
		}

		if cycle := findCycle(nodeOrder, adjacency); len(cycle) > 0 {
			report.AddIssue(Issue{
				Code:     CodeCycleDetected,
				Severity: SeverityError,
				Message:  "rulechain graph contains a cycle",
				Target: Target{
					Kind:       TargetRuleChain,
					ID:         def.RuleChain.ID,
					Path:       "metadata.connections",
					SourcePath: sourcePathFromDef(def),
				},
				Details: map[string]any{
					"ruleChainId": def.RuleChain.ID,
					"cycle":       cycle,
				},
			})
		}
	}
}

func findCycle(nodeOrder []string, adjacency map[string][]string) []string {
	color := map[string]int{}
	stack := []string{}
	stackIndex := map[string]int{}

	var visit func(string) []string
	visit = func(nodeID string) []string {
		color[nodeID] = 1
		stackIndex[nodeID] = len(stack)
		stack = append(stack, nodeID)

		for _, nextID := range adjacency[nodeID] {
			switch color[nextID] {
			case 1:
				cycle := append([]string{}, stack[stackIndex[nextID]:]...)
				cycle = append(cycle, nextID)
				return cycle
			case 0:
				if cycle := visit(nextID); len(cycle) > 0 {
					return cycle
				}
			}
		}

		stack = stack[:len(stack)-1]
		delete(stackIndex, nodeID)
		color[nodeID] = 2
		return nil
	}

	for _, nodeID := range nodeOrder {
		if color[nodeID] != 0 {
			continue
		}
		if cycle := visit(nodeID); len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}

func danglingConnectionIssue(conn types.Connection, index int, sourcePath string, field string) Issue {
	return Issue{
		Code:     CodeDanglingConnection,
		Severity: SeverityError,
		Message:  "rulechain connection references a missing node",
		Target: Target{
			Kind:       TargetConnection,
			ID:         fmt.Sprintf("%s:%s:%s", conn.FromID, conn.Type, conn.ToID),
			Path:       fmt.Sprintf("metadata.connections[%d]", index),
			SourcePath: sourcePath,
		},
		Details: map[string]any{
			"field":  field,
			"fromId": conn.FromID,
			"toId":   conn.ToID,
			"type":   conn.Type,
		},
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

func sourcePathFromDef(def *types.RuleChainDef) string {
	if def == nil {
		return ""
	}
	for _, node := range def.Metadata.Nodes {
		if node.SourcePath != "" {
			return node.SourcePath
		}
	}
	return ""
}

func setSourcePath(nodes []types.NodeDef, sourcePath string) {
	for i := range nodes {
		nodes[i].SourcePath = sourcePath
	}
}

func addLoaderFailure(report *Report, filePath string, err error) {
	report.AddIssue(Issue{
		Code:     CodeLoaderFailure,
		Severity: SeverityError,
		Message:  err.Error(),
		Target: Target{
			Kind:       TargetLoader,
			Path:       filepath.ToSlash(filePath),
			SourcePath: filepath.ToSlash(filePath),
		},
	})
}

func stringConfigValue(config types.ConfigMap, key string) string {
	if config == nil {
		return ""
	}
	value, ok := config[key]
	if !ok {
		return ""
	}
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func providerName(provider types.ResourceProvider) string {
	if provider == nil {
		return ""
	}
	return provider.Name()
}

func appendPath(base string, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
