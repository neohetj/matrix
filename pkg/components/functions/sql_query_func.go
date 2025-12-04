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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"gitlab.com/neohet/matrix/pkg/connectors/db"
	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	SQLQueryFuncID = "sqlQuery"
)

func init() {
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: SQLQueryFunc,
		FuncObject: types.FuncObject{
			ID:        SQLQueryFuncID,
			Name:      "SQL Query",
			Desc:      "Executes a SQL query using a shared DB client.",
			Dimension: "External",
			Tags:      []string{"database", "sql"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Name:     SQLQueryFuncID,
				FuncDesc: "Executes a configured SQL query. Can participate in transactions.",
				Business: []types.DynamicConfigField{
					{ID: "dsn", Name: "DSN or Reference", Default: "ref://default_db", Required: true, Type: "string", Desc: "数据库连接字符串(DSN)或共享客户端引用(ref://...)"},
					{ID: "txContextKey", Name: "Transaction Context Key", Default: "", Required: false, Type: "string", Desc: "参与事务的上下文键"},
					{ID: "query", Name: "SQL Query", Default: "", Required: true, Type: "string", Desc: "SQL查询语句模板"},
					{ID: "params", Name: "SQL Parameters", Default: []interface{}{}, Required: false, Type: "[]interface{}", Desc: "查询参数，支持 ${...} 占位符"},
					{ID: "isDynamicSql", Name: "Is Dynamic SQL", Default: false, Required: false, Type: "bool", Desc: "是否为动态SQL（直接模板替换），有SQL注入风险"},
				},
			},
		},
	})
}

// SQLQueryFunc is a function that executes a SQL query.
func SQLQueryFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	bizConfig, ok := helper.GetBusinessConfig(ctx)
	if !ok {
		ctx.HandleError(msg, types.DefInvalidParams.Wrap(fmt.Errorf("business config not found")))
		return
	}

	dsn, _ := bizConfig["dsn"].(string)
	txContextKey, _ := bizConfig["txContextKey"].(string)
	query, _ := bizConfig["query"].(string)
	paramsTpl, _ := bizConfig["params"].([]interface{})
	isDynamicSql, _ := bizConfig["isDynamicSql"].(bool)

	if query == "" {
		ctx.HandleError(msg, types.DefInvalidParams.Wrap(fmt.Errorf("query is not configured")))
		return
	}

	// 1. Get executor (transaction or standalone connection)
	type sqlExecutor interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
		SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
		QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	}
	var executor sqlExecutor

	goCtx := ctx.GetContext()
	if txContextKey != "" {
		if txUntyped := goCtx.Value(txContextKey); txUntyped != nil {
			if tx, ok := txUntyped.(*sqlx.Tx); ok {
				executor = tx
			}
		}
	}

	if executor == nil {
		var nodePool types.NodePool
		if rt := ctx.GetRuntime(); rt != nil {
			nodePool = rt.GetNodePool()
		}
		db, isTemp, err := db.GetDBConnection(nodePool, dsn)
		if err != nil {
			ctx.HandleError(msg, err)
			return
		}
		if isTemp {
			defer db.Close()
		}
		executor = db
	}

	// 2. Prepare query and args based on mode
	var finalQuery string
	var args []interface{}

	if isDynamicSql {
		// In dynamic SQL mode, we build the query string directly by replacing placeholders.
		// This is useful for dynamic table/column names but carries a significant SQL injection risk.
		// In this mode, the 'params' configuration is ignored.
		dataSource := helper.BuildDataSource(msg)
		finalQuery = utils.ReplacePlaceholders(query, dataSource)
	} else {
		// In secure mode (default), we use prepared statements with '?' placeholders.
		// The 'query' string is used as is, and parameters are processed safely.
		finalQuery = query
		if len(paramsTpl) > 0 {
			args = make([]interface{}, len(paramsTpl))
			for i, param := range paramsTpl {
				paramStr, ok := param.(string)
				if ok && strings.HasPrefix(paramStr, "${") && strings.HasSuffix(paramStr, "}") {
					// It's a placeholder, extract the path.
					path := strings.TrimSuffix(strings.TrimPrefix(paramStr, "${"), "}")

					// Use the robust ExtractFromMsgByPath helper to preserve type information.
					val, found, err := helper.ExtractFromMsgByPath(msg, path)
					if err != nil {
						ctx.HandleError(msg, types.DefInvalidParams.Wrap(fmt.Errorf("error extracting param for path '%s': %w", path, err)))
						return
					}
					if !found {
						ctx.HandleError(msg, types.DefInvalidParams.Wrap(fmt.Errorf("param for path '%s' not found in message", path)))
						return
					}

					// Check if the value is a complex type that needs to be marshaled to JSON.
					// This is useful for storing structured data in text-based columns.
					switch v := val.(type) {
					case []interface{}, map[string]interface{}, []string, []int, []float64, []bool:
						jsonVal, err := json.Marshal(v)
						if err != nil {
							ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("failed to marshal param for path '%s': %w", path, err)))
							return
						}
						args[i] = string(jsonVal)
					default:
						args[i] = val
					}
				} else {
					// It's a literal value.
					args[i] = param
				}
			}
		}
	}

	// 3. Execute query
	ctx.Debug("Executing SQL query.", "query", finalQuery, "args", args)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(finalQuery)), "select") {
		rows, err := executor.QueryxContext(goCtx, finalQuery, args...)
		if err != nil {
			ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("sql query execution failed: %w", err)))
			return
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			rowMap := make(map[string]interface{})
			if err := rows.MapScan(rowMap); err != nil {
				ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("sql row scan failed: %w", err)))
				return
			}
			// sqlx may scan TEXT/BLOB as []byte, convert to string for JSON serialization.
			for key, val := range rowMap {
				if b, ok := val.([]byte); ok {
					rowMap[key] = string(b)
				}
			}
			results = append(results, rowMap)
		}

		if err := rows.Err(); err != nil {
			ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("sql rows iteration error: %w", err)))
			return
		}

		newData, _ := json.Marshal(results)
		msg.SetData(string(newData))
	} else {
		ctx.Debug("Executing SQL exec.", "query", finalQuery, "args", args)
		_, err := executor.ExecContext(goCtx, finalQuery, args...)
		if err != nil {
			ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("sql exec execution failed: %w", err)))
			return
		}
	}

	ctx.TellSuccess(msg)
}
