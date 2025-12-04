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

package action

import (
	"fmt"

	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	FunctionsNodeType = "functions"
	FunctionNameKey   = "functionName"
)

// functionsNodePrototype is the shared prototype instance used for registration.
var functionsNodePrototype = &FunctionsNode{
	BaseNode: *types.NewBaseNode(FunctionsNodeType, types.NodeDefinition{
		Name:        "Function",
		Description: "Executes a pre-registered function by its name.",
		Dimension:   "Action",
		Tags:        []string{"action", "function"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(functionsNodePrototype)
}

// FunctionsNodeConfiguration holds the instance-specific configuration.
type FunctionsNodeConfiguration struct {
	FunctionName string `json:"functionName"`
}

// FunctionsNode is a generic node that executes a registered function.
type FunctionsNode struct {
	types.BaseNode
	types.Instance
	nodeConfig FunctionsNodeConfiguration
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
func (n *FunctionsNode) Init(configuration types.Config) error {
	if err := utils.Decode(configuration, &n.nodeConfig); err != nil {
		return types.DefInvalidConfiguration.Wrap(fmt.Errorf("failed to decode functions node config: %w", err))
	}
	if n.nodeConfig.FunctionName == "" {
		return types.DefInvalidConfiguration.Wrap(fmt.Errorf("'%s' is not specified in configuration for node %s", FunctionNameKey, n.ID()))
	}
	return nil
}

// OnMsg finds the configured function from the registry and executes it.
func (n *FunctionsNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	f, ok := registry.Default.NodeFuncManager.Get(n.nodeConfig.FunctionName)
	if !ok {
		err := types.DefFuncNotFound.Wrap(fmt.Errorf("function name: '%s' not found in registry", n.nodeConfig.FunctionName))
		ctx.HandleError(msg, err)
		return
	}
	// Execute the found function.
	f.Func(ctx, msg)
}

// GetDataContract provides a specialized implementation for function nodes.
// It dynamically retrieves the contract from the specific function's definition.
func (n *FunctionsNode) GetDataContract() types.DataContract {
	if n.nodeConfig.FunctionName == "" {
		return types.DataContract{}
	}
	f, ok := registry.Default.NodeFuncManager.Get(n.nodeConfig.FunctionName)
	if !ok {
		// Return an empty contract if the function isn't found,
		// as this might happen during validation of a not-yet-fully-configured node.
		return types.DataContract{}
	}
	config := f.FuncObject.Configuration
	return types.DataContract{
		Inputs:         config.Inputs,
		Outputs:        config.Outputs,
		ReadsData:      config.ReadsData,
		ReadsMetadata:  config.ReadsMetadata,
		WritesMetadata: config.WritesMetadata,
	}
}
