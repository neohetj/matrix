package helper

import (
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

// GetBusinessConfig extracts the business-specific configuration from a node's context.
// It intelligently merges the default values defined in the function's registration
// with the values provided in the rule chain DSL.
func GetBusinessConfig(ctx types.NodeCtx) (map[string]any, bool) {
	nodeDef := ctx.SelfDef()
	if nodeDef == nil {
		return nil, false
	}

	// 1. Get the configuration provided in the DSL.
	dslConfig, _ := nodeDef.Configuration["business"].(map[string]any)
	if dslConfig == nil {
		dslConfig = make(map[string]any)
	}

	// 2. Get the function's static definition from the registry.
	// This requires getting the functionName from the node's main configuration.
	var funcName string
	if fn, ok := nodeDef.Configuration["functionName"].(string); ok {
		funcName = fn
	} else {
		// If functionName is not present (e.g., not a 'functions' node), we can't get defaults.
		return dslConfig, true
	}

	funcDef, ok := registry.Default.NodeFuncManager.Get(funcName)
	if !ok || funcDef.FuncObject.Configuration.Business == nil {
		// No definition or no business configs defined, return what we have from DSL.
		return dslConfig, true
	}

	// 3. Merge defaults.
	for _, fieldDef := range funcDef.FuncObject.Configuration.Business {
		// If the key is NOT present in the DSL config, and a default value IS defined...
		if _, exists := dslConfig[fieldDef.ID]; !exists && fieldDef.Default != nil {
			dslConfig[fieldDef.ID] = fieldDef.Default
		}
	}

	return dslConfig, true
}
