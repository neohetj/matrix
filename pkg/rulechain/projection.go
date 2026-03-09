package rulechain

import (
	"sort"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// AnalyzeCoreObjProjection derives required inputs, produced objects, and edge live sets
// for a single rule chain instance.
func AnalyzeCoreObjProjection(def *types.RuleChainDef, instance types.ChainInstance) types.RuleChainCoreObjAnalysis {
	analysis := types.RuleChainCoreObjAnalysis{
		RequiredInputs:    types.CoreObjSet{},
		ProducedObjects:   types.CoreObjSet{},
		LiveObjectsByEdge: map[string]types.CoreObjSet{},
	}
	if def == nil || instance == nil {
		return analysis
	}

	contracts := CollectChainContracts(instance)
	successors := map[string][]string{}
	allNodes := map[string]struct{}{}
	for _, conn := range def.Metadata.Connections {
		if strings.TrimSpace(conn.FromID) == "" || strings.TrimSpace(conn.ToID) == "" {
			continue
		}
		successors[conn.FromID] = append(successors[conn.FromID], conn.ToID)
		allNodes[conn.FromID] = struct{}{}
		allNodes[conn.ToID] = struct{}{}
	}
	for nodeID := range instance.GetAllNodes() {
		allNodes[nodeID] = struct{}{}
	}

	needBefore := map[string]types.CoreObjSet{}
	for nodeID, access := range contracts {
		if access.ConsumesAll || access.IsPassThrough {
			needBefore[nodeID] = types.CoreObjSet{RetainAll: true}
			continue
		}
		needBefore[nodeID] = coreObjSetFromMap(access.Reads)
	}

	changed := true
	for changed {
		changed = false
		for nodeID := range allNodes {
			access := contracts[nodeID]
			nextNeed := types.CoreObjSet{}
			if access.ConsumesAll || access.IsPassThrough {
				nextNeed = types.CoreObjSet{RetainAll: true}
			} else {
				propagated := types.CoreObjSet{}
				for _, succID := range successors[nodeID] {
					propagated = unionCoreObjSets(propagated, needBefore[succID])
				}
				propagated = subtractCoreObjSet(propagated, access.Writes)
				nextNeed = unionCoreObjSets(coreObjSetFromMap(access.Reads), propagated)
			}
			if !coreObjSetsEqual(needBefore[nodeID], nextNeed) {
				needBefore[nodeID] = nextNeed
				changed = true
			}
		}
	}

	for nodeID, access := range contracts {
		analysis.ProducedObjects = unionCoreObjSets(analysis.ProducedObjects, coreObjSetFromMap(access.Writes))
		for _, succID := range successors[nodeID] {
			analysis.LiveObjectsByEdge[LiveObjectsEdgeKey(nodeID, succID)] = normalizeCoreObjSet(needBefore[succID])
		}
	}

	rootIDs := instance.GetRootNodeIDs()
	requiredInputs := types.CoreObjSet{}
	for _, rootID := range rootIDs {
		requiredInputs = unionCoreObjSets(requiredInputs, needBefore[rootID])
	}
	analysis.RequiredInputs = normalizeCoreObjSet(requiredInputs)
	analysis.ProducedObjects = normalizeCoreObjSet(analysis.ProducedObjects)
	return analysis
}

// ResolveProjection returns the cached runtime projection when available.
func ResolveProjection(rt types.Runtime) types.RuleChainCoreObjAnalysis {
	if rt == nil {
		return types.RuleChainCoreObjAnalysis{RequiredInputs: types.CoreObjSet{RetainAll: true}}
	}
	if provider, ok := rt.(types.CoreObjProjectionProvider); ok {
		return provider.CoreObjProjection()
	}
	def := rt.Definition()
	instance := rt.GetChainInstance()
	if def == nil || instance == nil {
		return types.RuleChainCoreObjAnalysis{RequiredInputs: types.CoreObjSet{RetainAll: true}}
	}
	return AnalyzeCoreObjProjection(def, instance)
}

// ResolveRequiredInputs returns the derived external input requirements for a runtime.
func ResolveRequiredInputs(rt types.Runtime) types.CoreObjSet {
	return ResolveProjection(rt).RequiredInputs
}

// LiveObjectsEdgeKey creates the stable edge key used for live-object lookups.
func LiveObjectsEdgeKey(fromNodeID string, toNodeID string) string {
	return strings.TrimSpace(fromNodeID) + "->" + strings.TrimSpace(toNodeID)
}

func coreObjSetFromMap(ids map[string]struct{}) types.CoreObjSet {
	result := types.CoreObjSet{}
	for objID := range ids {
		objID = strings.TrimSpace(objID)
		if objID == "" || isInternalPlaceholderObjID(objID) {
			continue
		}
		result.ObjIDs = append(result.ObjIDs, objID)
	}
	sort.Strings(result.ObjIDs)
	return result
}

func unionCoreObjSets(base types.CoreObjSet, others ...types.CoreObjSet) types.CoreObjSet {
	if base.RetainAll {
		return types.CoreObjSet{RetainAll: true}
	}
	acc := map[string]struct{}{}
	for _, objID := range base.ObjIDs {
		objID = strings.TrimSpace(objID)
		if objID != "" {
			acc[objID] = struct{}{}
		}
	}
	for _, other := range others {
		if other.RetainAll {
			return types.CoreObjSet{RetainAll: true}
		}
		for _, objID := range other.ObjIDs {
			objID = strings.TrimSpace(objID)
			if objID != "" {
				acc[objID] = struct{}{}
			}
		}
	}
	return coreObjSetFromMap(acc)
}

func subtractCoreObjSet(base types.CoreObjSet, kills map[string]struct{}) types.CoreObjSet {
	if base.RetainAll {
		return types.CoreObjSet{RetainAll: true}
	}
	if len(base.ObjIDs) == 0 || len(kills) == 0 {
		return normalizeCoreObjSet(base)
	}
	acc := map[string]struct{}{}
	for _, objID := range base.ObjIDs {
		objID = strings.TrimSpace(objID)
		if objID == "" {
			continue
		}
		if _, killed := kills[objID]; killed {
			continue
		}
		acc[objID] = struct{}{}
	}
	return coreObjSetFromMap(acc)
}

func normalizeCoreObjSet(set types.CoreObjSet) types.CoreObjSet {
	if set.RetainAll {
		return types.CoreObjSet{RetainAll: true}
	}
	if len(set.ObjIDs) == 0 {
		return types.CoreObjSet{}
	}
	seen := map[string]struct{}{}
	objIDs := make([]string, 0, len(set.ObjIDs))
	for _, objID := range set.ObjIDs {
		objID = strings.TrimSpace(objID)
		if objID == "" || isInternalPlaceholderObjID(objID) {
			continue
		}
		if _, ok := seen[objID]; ok {
			continue
		}
		seen[objID] = struct{}{}
		objIDs = append(objIDs, objID)
	}
	sort.Strings(objIDs)
	return types.CoreObjSet{ObjIDs: objIDs}
}

func coreObjSetsEqual(left types.CoreObjSet, right types.CoreObjSet) bool {
	left = normalizeCoreObjSet(left)
	right = normalizeCoreObjSet(right)
	if left.RetainAll || right.RetainAll {
		return left.RetainAll == right.RetainAll
	}
	if len(left.ObjIDs) != len(right.ObjIDs) {
		return false
	}
	for i := range left.ObjIDs {
		if left.ObjIDs[i] != right.ObjIDs[i] {
			return false
		}
	}
	return true
}
