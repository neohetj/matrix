package validation

import (
	"fmt"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

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
