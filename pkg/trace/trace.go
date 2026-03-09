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

package trace

import (
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

// Tracer is the new synchronous tracing service, responsible for recording node execution logs.
type Tracer struct {
	store types.Store
}

// NewTracer creates a new Tracer instance.
func NewTracer(store types.Store) *Tracer {
	return &Tracer{store: store}
}

// RecordNodeLog synchronously records a node's execution log.
func (t *Tracer) RecordNodeLog(executionID string, nodeLog types.RuleNodeRunLog) {
	status, ok := t.store.Get(executionID)
	if !ok {
		status = &types.ExecutionStatus{
			Snapshot: types.RuleChainRunSnapshot{
				Id:               executionID,
				StartRuleChainID: resolveStartRuleChainID(nodeLog),
				StartTs:          nodeLog.StartTs,
				Logs:             make([]types.RuleNodeRunLog, 0),
			},
		}
		t.store.Set(executionID, status)
	}

	status.Lock()
	defer status.Unlock()

	if status.Snapshot.StartRuleChainID == "" {
		status.Snapshot.StartRuleChainID = resolveStartRuleChainID(nodeLog)
	}
	status.Snapshot.Logs = append(status.Snapshot.Logs, nodeLog)
	status.LastUpdated = time.Now().UnixNano()
}

func resolveStartRuleChainID(nodeLog types.RuleNodeRunLog) string {
	if nodeLog.InMsg != nil {
		if v, ok := nodeLog.InMsg.Metadata()[types.ExecutionStartRuleChainIDKey]; ok && v != "" {
			return v
		}
	}
	return nodeLog.RuleChainID
}

// GetMetadataToPropagate filters the metadata based on the provided keys for tracing purposes.
func GetMetadataToPropagate(originalMeta types.Metadata, keysToPropagate []string) types.Metadata {
	if len(originalMeta) == 0 {
		return nil
	}

	metaToPropagate := make(types.Metadata)

	// Case 1: Propagate all keys if "*" is specified.
	if len(keysToPropagate) == 1 && keysToPropagate[0] == "*" {
		for k, v := range originalMeta {
			metaToPropagate[k] = v
		}
		return metaToPropagate
	}

	// Case 2: Propagate specific keys if a list is provided.
	if len(keysToPropagate) > 0 {
		for _, key := range keysToPropagate {
			if value, ok := originalMeta[key]; ok {
				metaToPropagate[key] = value
			}
		}
		return metaToPropagate
	}

	// Case 3 (Default): Propagate only the execution ID if it exists.
	if executionID, ok := originalMeta[types.ExecutionIDKey]; ok {
		metaToPropagate[types.ExecutionIDKey] = executionID
	}

	if len(metaToPropagate) == 0 {
		return nil
	}
	return metaToPropagate
}
