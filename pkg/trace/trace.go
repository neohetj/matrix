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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

const defaultExternalizedTracePayloadThreshold = 256 * 1024
const defaultTraceInlineFastPathThreshold = 16 * 1024

// Tracer is the new synchronous tracing service, responsible for recording node execution logs.
type Tracer struct {
	store        types.Store
	payloadStore PayloadStore
}

// NewTracer creates a new Tracer instance.
func NewTracer(store types.Store, payloadStores ...PayloadStore) *Tracer {
	var payloadStore PayloadStore
	if len(payloadStores) > 0 {
		payloadStore = payloadStores[0]
	}
	return &Tracer{
		store:        store,
		payloadStore: payloadStore,
	}
}

// RecordNodeLog synchronously records a node's execution log.
func (t *Tracer) RecordNodeLog(executionID string, nodeLog types.RuleNodeRunLog) {
	nodeLog = t.prepareNodeLog(executionID, nodeLog)

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

func (t *Tracer) prepareNodeLog(executionID string, nodeLog types.RuleNodeRunLog) types.RuleNodeRunLog {
	nodeLog.InMsg = t.externalizeMessage(executionID, nodeLog.Id, "inMsg", nodeLog.InMsg)
	nodeLog.OutMsg = t.externalizeMessage(executionID, nodeLog.Id, "outMsg", nodeLog.OutMsg)
	return nodeLog
}

func (t *Tracer) externalizeMessage(executionID, logID, source string, msg types.RuleMsg) types.RuleMsg {
	if t.payloadStore == nil || msg == nil {
		return msg
	}

	hasImagePreview, _, imagePath := detectTraceImagePayload(msg)
	if !hasImagePreview && canSkipTracePayloadMarshal(msg) {
		return msg
	}

	msgToStore := msg
	if imagePath != "" {
		if snapshotter, ok := t.payloadStore.(ImageSnapshotter); ok {
			snapshotPath, err := snapshotter.SnapshotImage(executionID, logID, source, imagePath)
			if err == nil {
				msgToStore = msg.Copy()
				msgToStore.SetData(snapshotPath, msg.DataFormat())
			}
		}
	}

	raw, err := json.Marshal(msgToStore)
	if err != nil {
		return msg
	}

	shouldExternalize, payloadRef := buildTracePayloadRef(executionID, logID, source, msgToStore, raw)
	if !shouldExternalize {
		return msg
	}

	if err := t.payloadStore.SaveMessage(executionID, logID, source, raw); err != nil {
		return msg
	}

	return NewSummaryRuleMsg(msgToStore, payloadRef)
}

func buildTracePayloadRef(executionID, logID, source string, msg types.RuleMsg, raw []byte) (bool, *types.TracePayloadRef) {
	if msg == nil {
		return false, nil
	}

	hasImagePreview, mimeType, _ := detectTraceImagePayload(msg)
	isLargePayload := len(raw) > defaultExternalizedTracePayloadThreshold
	if !hasImagePreview && !isLargePayload {
		return false, nil
	}

	reason := "size"
	if hasImagePreview {
		reason = "image"
	}

	return true, &types.TracePayloadRef{
		ExecutionID:     executionID,
		LogID:           logID,
		Source:          source,
		SizeBytes:       len(raw),
		Reason:          reason,
		MimeType:        mimeType,
		HasImagePreview: hasImagePreview,
		Externalized:    true,
	}
}

func detectTraceImagePayload(msg types.RuleMsg) (bool, string, string) {
	data := strings.TrimSpace(string(msg.Data()))
	lowerData := strings.ToLower(data)

	if strings.HasPrefix(lowerData, "data:image/") {
		semicolonIndex := strings.Index(lowerData, ";")
		if semicolonIndex > len("data:") {
			return true, lowerData[len("data:"):semicolonIndex], ""
		}
		return true, "image/*", ""
	}

	if isTraceImageFilePath(data) {
		return true, detectTraceImageMimeType(lowerData), data
	}

	if msg.DataFormat() == cnst.IMAGE {
		return true, detectTraceImageMimeType(lowerData), ""
	}

	return false, "", ""
}

func canSkipTracePayloadMarshal(msg types.RuleMsg) bool {
	if msg == nil {
		return true
	}

	dataT := msg.DataT()
	if dataT != nil && len(dataT.GetAll()) > 0 {
		return false
	}

	if len(msg.Data()) > defaultTraceInlineFastPathThreshold {
		return false
	}

	totalMetaSize := 0
	for key, value := range msg.Metadata() {
		totalMetaSize += len(key) + len(value)
	}
	return totalMetaSize <= 8*1024
}

func isTraceImageFilePath(path string) bool {
	if path == "" {
		return false
	}

	lowerPath := strings.ToLower(path)
	if !strings.HasSuffix(lowerPath, ".png") &&
		!strings.HasSuffix(lowerPath, ".jpg") &&
		!strings.HasSuffix(lowerPath, ".jpeg") &&
		!strings.HasSuffix(lowerPath, ".gif") &&
		!strings.HasSuffix(lowerPath, ".webp") {
		return false
	}

	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return true
	}

	return filepath.IsAbs(path)
}

func detectTraceImageMimeType(path string) string {
	switch {
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".gif"):
		return "image/gif"
	case strings.HasSuffix(path, ".webp"):
		return "image/webp"
	default:
		return "image/png"
	}
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
