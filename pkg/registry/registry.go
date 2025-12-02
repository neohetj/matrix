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

// Package registry provides a centralized access point for all component and object registries.
// Author: Neohet
package registry

import (
	"gitlab.com/neohet/matrix/pkg/types"
)

// Registry holds references to all specialized registries and managers.
type Registry struct {
	NodeManager     types.NodeManager
	ErrorRegistry   types.ErrorRegistry
	CoreObjRegistry types.CoreObjRegistry
	NodeFuncManager types.NodeFuncManager
	RuntimePool     types.RuntimePool
	SharedNodePool  types.NodePool
}

// Default is the default, global instance of the registry.
var Default = NewRegistry()

// NewRegistry creates a new instance of the central registry, initializing all sub-registries.
func NewRegistry() *Registry {
	return &Registry{
		NodeManager:     NewNodeManager(),
		ErrorRegistry:   NewErrorRegistry(),
		CoreObjRegistry: NewCoreObjRegistry(),
		NodeFuncManager: NewNodeFuncManager(),
		RuntimePool:     NewRuntimePool(),
		SharedNodePool:  NewNodePool(nil),
	}
}

// GetRuntimePool implements the types.RegistryProvider interface.
func (r *Registry) GetRuntimePool() types.RuntimePool {
	return r.RuntimePool
}

// GetSharedNodePool implements the types.RegistryProvider interface.
func (r *Registry) GetSharedNodePool() types.NodePool {
	return r.SharedNodePool
}

// GetNodeManager implements the types.RegistryProvider interface.
func (r *Registry) GetNodeManager() types.NodeManager {
	return r.NodeManager
}

// GetNodeFuncManager implements the types.RegistryProvider interface.
func (r *Registry) GetNodeFuncManager() types.NodeFuncManager {
	return r.NodeFuncManager
}

// GetCoreObjRegistry implements the types.RegistryProvider interface.
func (r *Registry) GetCoreObjRegistry() types.CoreObjRegistry {
	return r.CoreObjRegistry
}

// GetErrorRegistry implements the types.RegistryProvider interface.
func (r *Registry) GetErrorRegistry() types.ErrorRegistry {
	return r.ErrorRegistry
}

func init() {
	// Set the global provider so that the types package can access the registry
	// without creating an import cycle.
	types.DefaultRegistry = Default

	// Register all predefined errors from the core package.
	Default.ErrorRegistry.Register(
		types.ErrInternal,
		types.ErrInvalidParams,
		types.ErrInvalidConfiguration,
		types.ErrNodeNotFound,
		types.ErrFuncNotFound,
	)

	Default.CoreObjRegistry.Register(
		types.NewCoreObjDef(
			"", // Default value for string
			types.SID_STRING,
			"基本类型：字符串",
		),
		types.NewCoreObjDef(
			int64(0), // Default value for int64
			types.SID_INT64,
			"基本类型：64位整数",
		),
		types.NewCoreObjDef(
			float64(0), // Default value for float64
			types.SID_FLOAT64,
			"基本类型：64位浮点数",
		),
		types.NewCoreObjDef(
			false, // Default value for bool
			types.SID_BOOL,
			"基本类型：布尔值",
		),
		types.NewCoreObjDef(
			map[string]string{},
			types.SID_MAP_STRING_STRING,
			"基本类型：字符串到字符串的映射",
		),
		types.NewCoreObjDef(
			map[string]interface{}{},
			types.SID_MAP_STRING_INTERFACE,
			"基本类型：字符串到任意类型的映射",
		),
	)
}
