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

package functions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"gitlab.com/neohet/matrix/pkg/connectors/db"
	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

// txCtxKey is a private context key type to prevent collisions.
type txCtxKey string

const (
	StartTransactionFuncID    = "startTransaction"
	CommitTransactionFuncID   = "commitTransaction"
	RollbackTransactionFuncID = "rollbackTransaction"
)

func init() {
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: StartTransactionFunc,
		FuncObject: types.FuncObject{
			ID:        StartTransactionFuncID,
			Name:      "Start Transaction",
			Desc:      "Starts a database transaction and stores it in the context.",
			Dimension: "External",
			Tags:      []string{"database", "sql", "transaction"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Business: []types.DynamicConfigField{
					{ID: "dsn", Name: "Data Source Name", Desc: "DSN for the database connection (e.g., ref://mysql_default)", Type: "string", Required: true},
					{ID: "txContextKey", Name: "Transaction Context Key", Desc: "The key to store the transaction object in the context", Type: "string", Required: true, Default: "tx"},
				},
			},
		},
	})
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: CommitTransactionFunc,
		FuncObject: types.FuncObject{
			ID:        CommitTransactionFuncID,
			Name:      "Commit Transaction",
			Desc:      "Commits a transaction stored in the context.",
			Dimension: "External",
			Tags:      []string{"database", "sql", "transaction"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Business: []types.DynamicConfigField{
					{ID: "txContextKey", Name: "Transaction Context Key", Desc: "The key of the transaction object in the context", Type: "string", Required: true, Default: "tx"},
				},
			},
		},
	})
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: RollbackTransactionFunc,
		FuncObject: types.FuncObject{
			ID:        RollbackTransactionFuncID,
			Name:      "Rollback Transaction",
			Desc:      "Rolls back a transaction stored in the context.",
			Dimension: "External",
			Tags:      []string{"database", "sql", "transaction"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Business: []types.DynamicConfigField{
					{ID: "txContextKey", Name: "Transaction Context Key", Desc: "The key of the transaction object in the context", Type: "string", Required: true, Default: "tx"},
				},
			},
		},
	})
}

// StartTransactionFunc starts a new database transaction.
func StartTransactionFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	bizConfig, ok := helper.GetBusinessConfig(ctx)
	if !ok {
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("business config not found")))
		return
	}

	dsn, _ := bizConfig["dsn"].(string)
	txContextKey, _ := bizConfig["txContextKey"].(string)

	if dsn == "" || txContextKey == "" {
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("dsn and txContextKey are required")))
		return
	}

	var nodePool types.NodePool
	if rt := ctx.GetRuntime(); rt != nil {
		nodePool = rt.GetNodePool()
	}

	db, isTemp, err := db.GetDBConnection(nodePool, dsn)
	if err != nil {
		ctx.HandleError(msg, err)
		return
	}
	// If the connection is temporary, we can't use it for a transaction
	// that spans multiple nodes, as we can't guarantee it will be closed correctly.
	if isTemp {
		// It's important to close the temporary connection if we're not going to use it.
		db.Close()
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("transactions can only be used with shared connections (ref://)")))
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("failed to begin transaction: %w", err)))
		return
	}

	// Store the transaction in a new context and set it back.
	newGoCtx := context.WithValue(ctx.GetContext(), txCtxKey(txContextKey), tx)
	ctx.SetContext(newGoCtx)

	ctx.TellSuccess(msg)
}

// CommitTransactionFunc commits a transaction.
func CommitTransactionFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	bizConfig, ok := helper.GetBusinessConfig(ctx)
	if !ok {
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("business config not found")))
		return
	}
	txContextKey, _ := bizConfig["txContextKey"].(string)
	if txContextKey == "" {
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("txContextKey is required")))
		return
	}

	tx, ok := ctx.GetContext().Value(txCtxKey(txContextKey)).(*sqlx.Tx)
	if !ok {
		ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("transaction not found in context with key: %s", txContextKey)))
		return
	}

	if err := tx.Commit(); err != nil {
		ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("failed to commit transaction: %w", err)))
		return
	}
	ctx.TellSuccess(msg)
}

// RollbackTransactionFunc rolls back a transaction.
func RollbackTransactionFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	bizConfig, ok := helper.GetBusinessConfig(ctx)
	if !ok {
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("business config not found")))
		return
	}
	txContextKey, _ := bizConfig["txContextKey"].(string)
	if txContextKey == "" {
		ctx.HandleError(msg, types.DefInvalidConfiguration.Wrap(fmt.Errorf("txContextKey is required")))
		return
	}

	tx, ok := ctx.GetContext().Value(txCtxKey(txContextKey)).(*sqlx.Tx)
	if !ok {
		// If rollback is called but no tx is found, it might not be a hard error.
		// For example, it could be on a failure path where the transaction was never started.
		// We can log a warning and proceed.
		ctx.Warn("Transaction not found in context for rollback, this may be expected.", "key", txContextKey)
		ctx.TellSuccess(msg)
		return
	}

	if err := tx.Rollback(); err != nil {
		// A failed rollback is a more serious issue, unless the transaction is already done.
		if err != sql.ErrTxDone {
			ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("failed to rollback transaction: %w", err)))
			return
		}
		// If the transaction is already done, we can consider the rollback successful for idempotency.
		ctx.Warn("Rollback attempted on an already completed transaction.", "key", txContextKey)
	}
	ctx.TellSuccess(msg)
}
