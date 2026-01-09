/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package base

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

const (
	FunctionsNodeType = "functions"
	FunctionNameKey   = "functionName"
)

// FunctionsNodePrototype is the shared prototype instance used for registration.
// Exported for centralized registration in builtin/init.go.
var FunctionsNodePrototype = &FunctionsNode{
	BaseNode: *types.NewBaseNode(FunctionsNodeType, types.NodeMetadata{
		Name:        "Function",
		Description: "Executes a pre-registered function by its name.",
		Dimension:   "Action",
		Tags:        []string{"action", "function"},
		Version:     "1.0.0",
	}),
}

// FunctionsNodeConfiguration holds the instance-specific configuration.
type FunctionsNodeConfiguration struct {
	FunctionName string          `json:"functionName"`
	Business     types.ConfigMap `json:"business"`
}

// FunctionsNode is a generic node that executes a registered function.
type FunctionsNode struct {
	types.BaseNode
	types.Instance
	FuncConfig FunctionsNodeConfiguration
}

// New creates a new instance of FunctionsNode, referencing the prototype's BaseNode.
func (n *FunctionsNode) New() types.Node {
	return &FunctionsNode{
		BaseNode: n.BaseNode, // Reference the shared BaseNode template
	}
}

// Type returns the node type.
func (n *FunctionsNode) Type() types.NodeType {
	return FunctionsNodeType
}

// Init initializes the node instance with its specific configuration from the DSL.
func (n *FunctionsNode) Init(configuration types.ConfigMap) error {
	if err := utils.Decode(configuration, &n.FuncConfig); err != nil {
		return types.InvalidConfiguration.Wrap(fmt.Errorf("failed to decode functions node config: %w", err))
	}
	if n.FuncConfig.FunctionName == "" {
		return types.InvalidConfiguration.Wrap(fmt.Errorf("'%s' is not specified in configuration for node %s", FunctionNameKey, n.ID()))
	}
	return nil
}

// OnMsg finds the configured function from the registry and executes it.
func (n *FunctionsNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	var mgr types.NodeFuncManager
	if r := ctx.GetRuntime(); r != nil && r.GetEngine() != nil {
		mgr = r.GetEngine().NodeFuncManager()
	} else {
		mgr = types.DefaultRegistry.GetNodeFuncManager()
	}

	f, ok := mgr.Get(n.FuncConfig.FunctionName)
	if !ok {
		err := types.FuncNotFound.Wrap(fmt.Errorf("function name: '%s' not found in registry", n.FuncConfig.FunctionName))
		ctx.HandleError(msg, err)
		return
	}
	// Execute the found function.
	f.Func(ctx, msg)
}

// Errors returns the list of possible faults that this node can produce.
// It dynamically retrieves the faults from the specific function's definition.
func (n *FunctionsNode) Errors() []*types.Fault {
	if n.FuncConfig.FunctionName == "" {
		return nil
	}
	f, ok := types.DefaultRegistry.GetNodeFuncManager().Get(n.FuncConfig.FunctionName)
	if !ok {
		return nil
	}
	return f.FuncObject.Configuration.Errors
}

// DataContract provides a specialized implementation for function nodes.
// It dynamically retrieves the contract from the specific function's definition.
func (n *FunctionsNode) DataContract() types.DataContract {
	if n.FuncConfig.FunctionName == "" {
		return types.DataContract{}
	}
	f, ok := types.DefaultRegistry.GetNodeFuncManager().Get(n.FuncConfig.FunctionName)
	if !ok {
		// Return an empty contract if the function isn't found,
		// as this might happen during validation of a not-yet-fully-configured node.
		return types.DataContract{}
	}
	config := f.FuncObject.Configuration
	contract := types.DataContract{
		Reads:  make([]string, 0, len(config.FuncReads)+len(config.Inputs)),
		Writes: make([]string, 0, len(config.FuncWrites)+len(config.Outputs)),
	}

	resolveObjID := func(paramName string, io string) (string, bool) {
		boundDef := n.GetNodeDef()
		if boundDef == nil {
			return paramName, false
		}

		var bindings map[string]any
		switch io {
		case "output":
			bindings = boundDef.Outputs
		default:
			bindings = boundDef.Inputs
		}
		raw, ok := bindings[paramName]
		if !ok {
			return paramName, false
		}
		bindingMap, ok := raw.(map[string]any)
		if !ok {
			return paramName, false
		}
		objID, _ := bindingMap["objId"].(string)
		if objID == "" {
			return paramName, false
		}
		return objID, true
	}

	for _, r := range config.FuncReads {
		contract.Reads = append(contract.Reads, r.URI)
	}
	for _, input := range config.Inputs {
		objID, found := resolveObjID(input.ParamName, "input")
		if !found {
			if input.Required {
				objID = "<RequiredNotBound>"
			} else {
				objID = "<OptionalNotBound>"
			}
		}
		contract.Reads = append(contract.Reads, asset.DataTURI(objID, "", input.DefineSID))
	}

	for _, w := range config.FuncWrites {
		contract.Writes = append(contract.Writes, w.URI)
	}
	for _, output := range config.Outputs {
		objID, found := resolveObjID(output.ParamName, "output")
		if !found {
			// Outputs are generally expected to be bound if they are to be used,
			// but we apply similar logic.
			objID = "<OutputNotBound>"
		}
		contract.Writes = append(contract.Writes, asset.DataTURI(objID, "", output.DefineSID))
	}

	return contract
}

// ConfigSchema generates the OpenAPI schema for the function node's configuration.
// It dynamically converts the registered function's configuration (Business fields) into an OpenAPI schema.
func (n *FunctionsNode) ConfigSchema() *openapi3.Schema {
	if n.FuncConfig.FunctionName == "" {
		return nil
	}
	f, ok := types.DefaultRegistry.GetNodeFuncManager().Get(n.FuncConfig.FunctionName)
	if !ok {
		return nil
	}

	properties := make(map[string]*openapi3.SchemaRef)
	required := []string{}

	// Add 'functionName' field which is part of FunctionsNodeConfiguration
	properties[FunctionNameKey] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:        &openapi3.Types{"string"},
			Description: "The name of the function to execute",
		},
	}
	required = append(required, FunctionNameKey)

	// Add Business fields from FuncObjConfiguration
	// These are dynamic fields defined in the function registration
	for _, field := range f.FuncObject.Configuration.Business {
		// Use generic helper to convert MType to OpenAPI schema
		schema := utils.MTypeToOpenAPISchema(field.Type)
		schema.Description = field.Desc
		schema.Title = field.Name

		if field.Default != nil {
			schema.Default = field.Default
		}

		properties[field.ID] = &openapi3.SchemaRef{
			Value: schema,
		}

		if field.Required {
			required = append(required, field.ID)
		}
	}

	return &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: properties,
		Required:   required,
	}
}
