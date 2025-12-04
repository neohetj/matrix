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

package db

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

// GetDBConnection retrieves a database connection, either from a shared node pool
// via a reference string (e.g., "ref://my_db") or by creating a new temporary
// connection from a DSN.
// It returns the connection and a boolean indicating if the connection is temporary
// (and thus should be closed by the caller).
func GetDBConnection(nodePool types.NodePool, dsn string) (*sqlx.DB, bool, error) {
	if nodePool == nil {
		nodePool = registry.Default.SharedNodePool
	}

	if strings.HasPrefix(dsn, "ref://") {
		// Get from shared node pool
		nodeId := strings.TrimPrefix(dsn, "ref://")
		sharedCtx, ok := nodePool.Get(nodeId)
		if !ok {
			return nil, false, types.DefNodeNotFound.Wrap(fmt.Errorf("db client node not found by ref: %s", dsn))
		}

		// The GetNode() method on SharedNodeCtx returns an interface{}, which is the node instance.
		node := sharedCtx.GetNode()
		dbClient, ok := node.(types.SharedNode)
		if !ok {
			return nil, false, types.DefInternalError.Wrap(fmt.Errorf("node %s is not a SharedNode", dsn))
		}

		dbInstance, err := dbClient.GetInstance()
		if err != nil {
			return nil, false, types.DefInternalError.Wrap(fmt.Errorf("failed to get db instance from %s: %w", dsn, err))
		}
		db, ok := dbInstance.(*sqlx.DB)
		if !ok {
			return nil, false, types.DefInternalError.Wrap(fmt.Errorf("shared instance from %s is not a *sqlx.DB", dsn))
		}
		return db, false, nil // false indicates this is a shared connection and should not be closed.
	} else {
		// Create temporary connection
		// A real implementation should allow specifying the driverName.
		db, err := sqlx.Connect("mysql", dsn)
		if err != nil {
			return nil, false, types.DefInternalError.Wrap(fmt.Errorf("failed to create temporary db connection: %w", err))
		}
		return db, true, nil // true indicates this is a temporary connection that the caller must close.
	}
}
