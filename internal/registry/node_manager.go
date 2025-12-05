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

	"github.com/NeohetJ/Matrix/pkg/types"
)

// DefaultNodeManager is the default thread-safe implementation of the types.NodeManager interface.
// It uses a sync.Map to store registered components.
type DefaultNodeManager struct {
	components sync.Map
}

// NewNodeManager creates a new instance of DefaultNodeManager.
func NewNodeManager() *DefaultNodeManager {
	return &DefaultNodeManager{}
}

// Register adds a new component to the manager.
func (m *DefaultNodeManager) Register(node types.Node) error {
	nodeType := node.Type()
	if _, loaded := m.components.LoadOrStore(nodeType, node); loaded {
		return fmt.Errorf("component with type '%s' already registered", nodeType)
	}
	return nil
}

// Get retrieves a registered node prototype by its type.
func (m *DefaultNodeManager) Get(nodeType types.NodeType) (types.Node, bool) {
	value, ok := m.components.Load(nodeType)
	if !ok {
		return nil, false
	}
	node, ok := value.(types.Node)
	return node, ok
}

// Unregister removes a component by its type.
func (m *DefaultNodeManager) Unregister(nodeType types.NodeType) error {
	if _, loaded := m.components.LoadAndDelete(nodeType); !loaded {
		return fmt.Errorf("component with type '%s' not found", nodeType)
	}
	return nil
}

// NewNode creates a new instance of a node by its type.
func (m *DefaultNodeManager) NewNode(nodeType types.NodeType) (types.Node, error) {
	value, ok := m.components.Load(nodeType)
	if !ok {
		return nil, fmt.Errorf("component with type '%s' not found", nodeType)
	}

	if node, ok := value.(types.Node); ok {
		return node.New(), nil
	}

	return nil, fmt.Errorf("registered value for type '%s' is not a valid types.Node", nodeType)
}

// GetComponents returns a map of all registered components.
func (m *DefaultNodeManager) GetComponents() map[types.NodeType]types.Node {
	components := make(map[types.NodeType]types.Node)
	m.components.Range(func(key, value any) bool {
		if nodeType, ok := key.(types.NodeType); ok {
			if node, ok := value.(types.Node); ok {
				components[nodeType] = node
			}
		}
		return true
	})
	return components
}
