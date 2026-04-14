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

package registry

import (
	"fmt"
	"sync"

	"github.com/neohetj/matrix/pkg/types"
)

// DefaultNodeFuncManager is the default thread-safe implementation of the NodeFuncManager interface.
type DefaultNodeFuncManager struct {
	functions sync.Map
}

// NewNodeFuncManager creates a new instance of DefaultNodeFuncManager.
func NewNodeFuncManager() *DefaultNodeFuncManager {
	return &DefaultNodeFuncManager{}
}

// Register adds a new function node definition to the manager.
func (m *DefaultNodeFuncManager) Register(funcs ...*types.NodeFuncObject) {
	for _, f := range funcs {
		if f != nil {
			// Validate business config definitions
			for _, field := range f.FuncObject.Configuration.Business {
				if !field.Type.IsSupported() {
					panic(fmt.Sprintf("Function %s registration failed: invalid business config type '%s' for field '%s'", f.FuncObject.ID, field.Type, field.Name))
				}
				if field.NotEditable && field.Default == nil {
					panic(fmt.Sprintf("Function %s registration failed: field '%s' is notEditable but missing defaultValue", f.FuncObject.ID, field.ID))
				}
			}
			validateFunctionRouting(f)
			m.functions.Store(f.FuncObject.ID, f)
		}
	}
}

func validateFunctionRouting(f *types.NodeFuncObject) {
	cfg := f.FuncObject.Configuration
	mode := cfg.EffectiveRoutingMode()
	if !mode.IsValid() {
		panic(fmt.Sprintf("Function %s registration failed: invalid routingMode '%s'", f.FuncObject.ID, cfg.RoutingMode))
	}

	switch mode {
	case types.FunctionRoutingModeStandard:
		if len(cfg.DeclaredRelations) > 0 {
			panic(fmt.Sprintf("Function %s registration failed: standard routingMode must not declare custom relations", f.FuncObject.ID))
		}
	case types.FunctionRoutingModeDecision:
		if len(cfg.DeclaredRelations) == 0 {
			panic(fmt.Sprintf("Function %s registration failed: decision routingMode requires declaredRelations", f.FuncObject.ID))
		}
		seen := make(map[string]struct{}, len(cfg.DeclaredRelations))
		for _, relation := range cfg.DeclaredRelations {
			if relation == "" {
				panic(fmt.Sprintf("Function %s registration failed: declaredRelations must not contain empty values", f.FuncObject.ID))
			}
			if relation == "Success" || relation == "Failure" {
				panic(fmt.Sprintf("Function %s registration failed: declaredRelations must not include reserved relation '%s'", f.FuncObject.ID, relation))
			}
			if _, ok := seen[relation]; ok {
				panic(fmt.Sprintf("Function %s registration failed: declaredRelations contains duplicate relation '%s'", f.FuncObject.ID, relation))
			}
			seen[relation] = struct{}{}
		}
	}
}

// Get retrieves a function node definition by its ID.
func (m *DefaultNodeFuncManager) Get(id string) (*types.NodeFuncObject, bool) {
	value, ok := m.functions.Load(id)
	if !ok {
		return nil, false
	}
	f, ok := value.(*types.NodeFuncObject)
	return f, ok
}

// List returns all registered function node objects.
func (m *DefaultNodeFuncManager) List() []*types.NodeFuncObject {
	var funcs []*types.NodeFuncObject
	m.functions.Range(func(key, value interface{}) bool {
		if f, ok := value.(*types.NodeFuncObject); ok {
			funcs = append(funcs, f)
		}
		return true
	})
	return funcs
}
