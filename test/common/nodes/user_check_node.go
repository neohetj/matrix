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

// Package testcomponents provides custom node components for testing purposes.
package testcomponents

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"gitlab.com/neohet/matrix/pkg/components/base"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	UserCheckNodeType = "test/userCheck"
)

// userCheckNodePrototype is the shared prototype instance for registration.
var userCheckNodePrototype = &UserCheckNode{
	BaseNode: *types.NewBaseNode(UserCheckNodeType, types.NodeDefinition{
		Name:        "User Check (Test)",
		Description: "A test node that uses a shared database connection to perform a check.",
		Dimension:   "Test",
		Tags:        []string{"test", "database"},
		Version:     "1.0.0",
	}),
}

// This init function will only be executed when the test package is compiled,
// ensuring that the test node is only registered during tests.
func init() {
	registry.Default.NodeManager.Register(userCheckNodePrototype)
}

// UserCheckNodeConfiguration holds the configuration for the UserCheckNode.
type UserCheckNodeConfiguration struct {
	// DBRef is the reference to the shared dbClient node, e.g., "ref://my_db".
	DBRef string `json:"dbRef"`
}

// UserCheckNode is a test node that uses a shared database connection.
type UserCheckNode struct {
	types.BaseNode
	types.Instance
	nodeConfig  UserCheckNodeConfiguration
	dbShareable base.Shareable[*sqlx.DB]
	nodePool    types.NodePool
}

// New creates a new instance of the UserCheckNode.
func (n *UserCheckNode) New() types.Node {
	return &UserCheckNode{
		BaseNode: n.BaseNode, // Reference the shared BaseNode template
	}
}

// Init initializes the node.
func (n *UserCheckNode) Init(cfg types.Config) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode user check node config: %w", err)
	}

	// For testing purposes, we directly use the global DefaultNodePool.
	// A production-grade component might need a more sophisticated way to access the pool.
	n.nodePool = registry.Default.SharedNodePool
	return n.dbShareable.Init(n.nodePool, n.nodeConfig.DBRef, nil)
}

// OnMsg is where the node's logic is executed.
func (n *UserCheckNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	db, err := n.dbShareable.Get()
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("test node failed to get shared db client: %w", err))
		return
	}

	// Ping the database to verify the connection.
	err = db.Ping()
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("test node failed to ping shared db: %w", err))
		return
	}

	fmt.Printf("UserCheckNode successfully pinged shared database via reference: %s\n", n.nodeConfig.DBRef)
	ctx.TellSuccess(msg)
}

// Destroy is a no-op.
func (n *UserCheckNode) Destroy() {}
