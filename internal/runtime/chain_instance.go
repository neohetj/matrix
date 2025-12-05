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

package runtime

import (
	"github.com/NeohetJ/Matrix/pkg/types"
)

// DefaultChainInstance is the default implementation of the ChainInstance interface.
type DefaultChainInstance struct {
	def         *types.RuleChainDef
	nodes       map[string]types.Node
	nodeDefs    map[string]*types.NodeDef
	connections map[string][]types.Connection
}

// NewDefaultChainInstance creates a new DefaultChainInstance.
func NewDefaultChainInstance(def *types.RuleChainDef) *DefaultChainInstance {
	return &DefaultChainInstance{
		def:         def,
		nodes:       make(map[string]types.Node),
		nodeDefs:    make(map[string]*types.NodeDef),
		connections: make(map[string][]types.Connection),
	}
}

// GetNode returns the node instance for the given ID.
func (c *DefaultChainInstance) GetNode(id string) (types.Node, bool) {
	node, ok := c.nodes[id]
	return node, ok
}

// GetNodeDef returns the node definition for the given ID.
func (c *DefaultChainInstance) GetNodeDef(id string) (*types.NodeDef, bool) {
	def, ok := c.nodeDefs[id]
	return def, ok
}

// GetConnections returns the outgoing connections for the given node ID.
func (c *DefaultChainInstance) GetConnections(fromNodeID string) []types.Connection {
	return c.connections[fromNodeID]
}

// Definition returns the rule chain definition.
func (c *DefaultChainInstance) Definition() *types.RuleChainDef {
	return c.def
}

// GetRootNodeIDs returns the IDs of the root nodes (nodes with no incoming connections).
func (c *DefaultChainInstance) GetRootNodeIDs() []string {
	inDegrees := make(map[string]int)
	for id := range c.nodes {
		inDegrees[id] = 0
	}

	for _, conns := range c.connections {
		for _, conn := range conns {
			inDegrees[conn.ToID]++
		}
	}

	var rootNodes []string
	for id, degree := range inDegrees {
		if degree == 0 {
			rootNodes = append(rootNodes, id)
		}
	}
	return rootNodes
}

// GetAllNodes returns all nodes in the chain.
func (c *DefaultChainInstance) GetAllNodes() map[string]types.Node {
	return c.nodes
}

// Destroy releases resources held by the chain instance.
func (c *DefaultChainInstance) Destroy() {
	for _, node := range c.nodes {
		node.Destroy()
	}
}

// AddNode adds a node to the chain instance.
func (c *DefaultChainInstance) AddNode(node types.Node, def *types.NodeDef) {
	c.nodes[def.ID] = node
	c.nodeDefs[def.ID] = def
}

// AddConnection adds a connection to the chain instance.
func (c *DefaultChainInstance) AddConnection(conn types.Connection) {
	c.connections[conn.FromID] = append(c.connections[conn.FromID], conn)
}
