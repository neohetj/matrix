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
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/redis/go-redis/v9"
	"gitlab.com/neohet/matrix/pkg/connectors/redis"
	"gitlab.com/neohet/matrix/pkg/helper"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/trace"
	"gitlab.com/neohet/matrix/pkg/types"
	"gitlab.com/neohet/matrix/pkg/utils"
)

const (
	RedisCommandFuncID = "redisCommand"
)

func init() {
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: RedisCommandFunc,
		FuncObject: types.FuncObject{
			ID:        RedisCommandFuncID,
			Name:      "Redis Command",
			Desc:      "Executes one or more Redis commands.",
			Dimension: "External",
			Tags:      []string{"redis", "cache"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Name:     RedisCommandFuncID,
				FuncDesc: "Executes a list of Redis commands with placeholder substitution.",
				Business: []types.DynamicConfigField{
					{ID: "redisDsn", Name: "Redis DSN or Reference", Default: "ref://default_redis", Required: true, Type: "string"},
					{ID: "commands", Name: "Commands", Default: []string{}, Required: true, Type: "[]string"},
					{ID: "propagateMeta", Name: "Propagate Metadata", Desc: "If true, propagates specified metadata keys.", Default: false, Type: "bool"},
					{ID: "propagateKeys", Name: "Metadata Keys to Propagate", Desc: "List of metadata keys to propagate. If empty, only ExecutionID is propagated. Use ['*'] for all.", Default: []string{}, Type: "[]string"},
					{ID: "wrapPayload", Name: "Wrap Payload for PUBLISH", Desc: "For PUBLISH, wraps the original message and metadata into a standard JSON object.", Default: false, Type: "bool"},
				},
			},
		},
	})
}

// parseCommand splits a command string into parts, respecting quoted sections.
func parseCommand(cmdStr string) []string {
	if strings.TrimSpace(cmdStr) == "" {
		return []string{}
	}
	var parts []string
	var current strings.Builder
	inQuote := false
	for _, r := range cmdStr {
		switch r {
		case '"':
			if inQuote { // End of a quote
				parts = append(parts, current.String())
				current.Reset()
			}
			inQuote = !inQuote
		case ' ':
			if !inQuote {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// RedisCommandFunc executes a list of Redis commands.
func RedisCommandFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	bizConfig, ok := helper.GetBusinessConfig(ctx)
	if !ok {
		ctx.HandleError(msg, types.DefInvalidParams.Wrap(fmt.Errorf("business config not found")))
		return
	}

	redisDsn, _ := bizConfig["redisDsn"].(string)
	commandsTpl, _ := bizConfig["commands"].([]interface{})
	propagateMeta, _ := bizConfig["propagateMeta"].(bool)
	wrapPayload, _ := bizConfig["wrapPayload"].(bool)

	var propagateKeys []string
	if keys, ok := bizConfig["propagateKeys"].([]interface{}); ok {
		for _, k := range keys {
			if keyStr, ok := k.(string); ok {
				propagateKeys = append(propagateKeys, keyStr)
			}
		}
	}

	if len(commandsTpl) == 0 {
		ctx.TellSuccess(msg) // Nothing to do
		return
	}

	var nodePool types.NodePool
	if rt := ctx.GetRuntime(); rt != nil {
		nodePool = rt.GetNodePool()
	}

	client, isTemp, err := redis.GetRedisConnection(nodePool, redisDsn)
	if err != nil {
		ctx.HandleError(msg, err)
		return
	}
	if isTemp {
		defer client.Close()
	}

	dataSource := helper.BuildDataSource(msg)

	for _, cmdTpl := range commandsTpl {
		cmdStr := utils.ReplacePlaceholders(fmt.Sprintf("%v", cmdTpl), dataSource)

		parts := parseCommand(cmdStr)
		if len(parts) == 0 {
			continue
		}

		args := make([]any, 0, len(parts)+2)
		for _, p := range parts {
			args = append(args, p)
		}

		// Metadata propagation logic
		if propagateMeta {
			metaToPropagate := trace.GetMetadataToPropagate(msg.Metadata(), propagateKeys)

			if len(metaToPropagate) > 0 {
				command := strings.ToUpper(parts[0])
				switch command {
				case "XADD":
					metadataJson, err := json.Marshal(metaToPropagate)
					if err == nil {
						args = append(args, "__matrix_metadata", string(metadataJson))
					}
				case "PUBLISH":
					if wrapPayload && len(args) > 2 { // PUBLISH channel message
						originalPayload := args[2]
						wrappedPayload := map[string]interface{}{
							"__payload__":         originalPayload,
							"__matrix_metadata__": metaToPropagate,
						}
						wrappedJson, err := json.Marshal(wrappedPayload)
						if err == nil {
							args[2] = string(wrappedJson) // Replace original message with wrapped one
						}
					}
				}
			}
		}

		// This is a simplified execution model. It doesn't handle all command types.
		ctx.Debug("Executing Redis command", "args", args)
		err := client.Do(ctx.GetContext(), args...).Err()
		if err != nil {
			ctx.Error("Redis command execution failed", "error", err, "args", args)
			ctx.HandleError(msg, types.DefInternalError.Wrap(fmt.Errorf("redis command failed: %w", err)))
			return
		}
		ctx.Debug("Successfully executed Redis command", "command", parts[0])
	}

	ctx.TellSuccess(msg)
}
