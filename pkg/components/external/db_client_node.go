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

package external

import (
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"gitlab.com/neohet/matrix/pkg/components/base"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	DBClientNodeType = "external/dbClient"
)

// dbClientNodePrototype is the shared prototype instance for registration.
var dbClientNodePrototype = &DBClientNode{
	BaseNode: *types.NewBaseNode(DBClientNodeType, types.NodeDefinition{
		Name:        "DB Client",
		Description: "Provides a shared database connection pool (*sqlx.DB).",
		Dimension:   "External",
		Tags:        []string{"external", "database", "sql"},
		Version:     "1.0.0",
	}),
}

func init() {
	registry.Default.NodeManager.Register(dbClientNodePrototype)
}

// DBClientNodeConfiguration holds the configuration for the DBClientNode.
type DBClientNodeConfiguration struct {
	DriverName string `json:"driverName"`
	DSN        string `json:"dsn"`
	PoolSize   int    `json:"poolSize"`
}

// DBClientNode is a component that provides a shared database connection pool (*sqlx.DB).
type DBClientNode struct {
	types.BaseNode
	types.Instance
	base.Shareable[*sqlx.DB]
	nodeConfig DBClientNodeConfiguration
	client     *sqlx.DB
	closeOnce  sync.Once
}

// New creates a new instance of the DBClientNode.
func (n *DBClientNode) New() types.Node {
	return &DBClientNode{
		BaseNode: n.BaseNode, // Reference the shared BaseNode template
	}
}

// Init initializes the node.
func (n *DBClientNode) Init(cfg types.Config) error {
	if err := utils.Decode(cfg, &n.nodeConfig); err != nil {
		return fmt.Errorf("failed to decode db client node config: %w", err)
	}

	initFunc := func() (*sqlx.DB, error) {
		if n.client != nil {
			return n.client, nil
		}
		db, err := sqlx.Connect(n.nodeConfig.DriverName, n.nodeConfig.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		if n.nodeConfig.PoolSize > 0 {
			db.SetMaxOpenConns(n.nodeConfig.PoolSize)
		}
		n.client = db
		return n.client, nil
	}

	// The nodePool is injected by the runtime. For now, we pass nil.
	// The Shareable helper will be properly initialized by the runtime.
	return n.Shareable.Init(nil, n.nodeConfig.DSN, initFunc)
}

// OnMsg for a resource node is typically a no-op.
func (n *DBClientNode) OnMsg(ctx types.NodeCtx, msg types.RuleMsg) {
	// No-op
}

// Destroy closes the database connection if it was created by this node.
func (n *DBClientNode) Destroy() {
	n.closeOnce.Do(func() {
		if n.client != nil {
			n.client.Close()
		}
	})
}
