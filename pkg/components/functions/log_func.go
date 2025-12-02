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
	"log"

	"github.com/dop251/goja"
	"gitlab.com/neohet/matrix/pkg/registry"
	"gitlab.com/neohet/matrix/pkg/types"
)

const (
	LogFuncID = "log"
)

func init() {
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: LogFunc,
		FuncObject: types.FuncObject{
			ID:        LogFuncID,
			Name:      "Log Message",
			Desc:      "Logs the message payload and metadata to the console.",
			Dimension: "System",
			Tags:      []string{"debug", "io"},
			Version:   "1.0.0",
			Configuration: types.FuncObjConfiguration{
				Name:     LogFuncID,
				FuncDesc: "A simple function to print message details.",
			},
		},
	})
}

// LogFunc is a function that logs message details.
// It can be customized with a "script" in its configuration.
// The script should return a JSON string with optional "log" and "metadata" fields.
func LogFunc(ctx types.NodeCtx, msg types.RuleMsg) {
	config := ctx.Config()
	if script, ok := config["script"].(string); ok && script != "" {
		vm := goja.New()
		// Inject original msg data and metadata into the script context.
		vm.Set("msg", msg.Data())
		vm.Set("metadata", msg.Metadata())

		res, err := vm.RunString(script)
		if err != nil {
			ctx.TellFailure(msg, types.ErrInternal.Wrap(fmt.Errorf("script execution failed: %w", err)))
			return
		}

		// The script is expected to return a JSON string.
		// We use a struct to unmarshal the result.
		var scriptResult struct {
			Log      string         `json:"log"`
			Metadata types.Metadata `json:"metadata"`
		}

		// Default the log content to the script's string result.
		scriptResult.Log = res.String()

		// Try to unmarshal if the result looks like a JSON object.
		if json.Unmarshal([]byte(res.String()), &scriptResult) == nil {
			// If unmarshal is successful, update metadata.
			if len(scriptResult.Metadata) > 0 {
				// Get a mutable copy of the message's metadata.
				newMeta := msg.Metadata()
				if newMeta == nil {
					newMeta = make(types.Metadata)
				}
				for k, v := range scriptResult.Metadata {
					newMeta[k] = v
				}
				msg.SetMetadata(newMeta)
			}
		}

		log.Printf("LOG_FUNC: %s", scriptResult.Log)

	} else {
		// Default behavior: log the standard message details.
		log.Printf("LOG_FUNC: MsgType=%s, Data=%s, Metadata=%v", msg.Type(), msg.Data(), msg.Metadata())
	}

	// Pass the potentially modified message to the next node.
	ctx.TellSuccess(msg)
}
