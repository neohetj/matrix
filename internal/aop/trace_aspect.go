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

package aop

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"context"

	"github.com/google/uuid"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/trace"
	"github.com/neohetj/matrix/pkg/types"
)

// contextKey is a private type to avoid collisions in context keys.
type contextKey string

const traceLogContextKey = contextKey("trace_log_inprogress")

// TraceAspect is an AOP aspect that intercepts node execution to record trace logs.
type TraceAspect struct {
	tracer *trace.Tracer
}

// NewTraceAspect creates a new trace aspect.
func NewTraceAspect(tracer *trace.Tracer) types.Aspect {
	return &TraceAspect{tracer: tracer}
}

// Before is called before a node's OnMsg method is executed.
// It captures the input message state.
func (a *TraceAspect) Before(ctx types.NodeCtx, msg types.RuleMsg) (types.RuleMsg, error) {
	if _, ok := msg.Metadata()[types.ExecutionIDKey]; !ok {
		return msg, nil // No trace ID, do not process.
	}

	// 1. Deep copy the input message to freeze its state.
	inMsgCopy, deepCopyErr := msg.DeepCopy()
	// Process Image Data in Input
	inMsgCopy = a.processImages(inMsgCopy)

	nodeLog := &types.RuleNodeRunLog{
		Id:          uuid.New().String(),
		NodeID:      ctx.NodeID(),
		RuleChainID: ctx.ChainID(),
		Name:        ctx.SelfDef().Name,
		StartTs:     time.Now().UnixNano(),
	}

	if deepCopyErr != nil {
		// Log the error but don't block the chain.
		nodeLog.Err = fmt.Sprintf("Trace AOP Before Func: failed to deep copy in-message: %v", deepCopyErr)
		// Even if deep copy fails, we still want to record the log with a nil InMsg.
		nodeLog.InMsg = nil
	} else {
		nodeLog.InMsg = inMsgCopy
	}

	// 2. Store the partial log in a new context and update the NodeCtx.
	// This is safe because each node execution gets a unique NodeCtx instance.
	newCtx := context.WithValue(ctx.GetContext(), traceLogContextKey, nodeLog)
	ctx.SetContext(newCtx)

	return msg, nil
}

// After is called after a node's OnMsg method has been executed.
// It captures the output message state and completes the log entry.
func (a *TraceAspect) After(ctx types.NodeCtx, msg types.RuleMsg, err error) {
	executionID, ok := msg.Metadata()[types.ExecutionIDKey]
	if !ok {
		return
	}

	// 1. Retrieve the partial log from the context.
	logInProgress, ok := ctx.GetContext().Value(traceLogContextKey).(*types.RuleNodeRunLog)
	if !ok {
		// Before hook may have failed or been skipped.
		return
	}

	// Helper to append errors without overwriting.
	appendError := func(originalErr, newErr string) string {
		if originalErr == "" {
			return newErr
		}
		return fmt.Sprintf("%s; %s", originalErr, newErr)
	}

	// 2. Deep copy the output message to freeze its state.
	outMsgCopy, deepCopyErr := msg.DeepCopy()
	// Process Image Data in Output
	outMsgCopy = a.processImages(outMsgCopy)

	if deepCopyErr != nil {
		// Log the error but don't block the chain.
		// We can still record the log with a nil OutMsg.
		logInProgress.Err = appendError(logInProgress.Err, fmt.Sprintf("Trace AOP After Func: failed to deep copy out-message: %v", deepCopyErr))
	}

	// 3. Complete the log entry.
	logInProgress.EndTs = time.Now().UnixNano()
	logInProgress.OutMsg = outMsgCopy

	// If there was an error from the node execution.
	if err != nil {
		logInProgress.Err = appendError(logInProgress.Err, fmt.Sprintf("Node execution error: %v", err.Error()))
	}

	// Additionally, check if a handled error was stored in the metadata.
	// We only record it if the error originated from the current node or has no attribution.
	if errMsg, ok := msg.Metadata()["error"]; ok {
		// Check if the error belongs to this node by comparing NodeID
		errNodeId, idOk := msg.Metadata()[types.MetaErrorNodeID]

		// If errNodeId is missing, we assume it's a new error (or legacy).
		// If errNodeId is present, it MUST match the current node ID.
		if !idOk || errNodeId == logInProgress.NodeID {
			logInProgress.Err = appendError(logInProgress.Err, fmt.Sprintf("Metadata error: %v", errMsg))
		}
	}

	// 4. Record the complete log synchronously.
	a.tracer.RecordNodeLog(executionID, *logInProgress)
}

// processImages checks if the message data is an image and converts it to a base64 string for display.
func (a *TraceAspect) processImages(msg types.RuleMsg) types.RuleMsg {
	if msg == nil {
		return nil
	}

	// Since we cannot modify the msg in-place if it doesn't support it,
	// and we want to ensure the log contains the image content,
	// we might need to construct a new message if changes are needed.
	// But `RuleMsg` is an interface. We rely on `types.NewMsg` if we need to replace it.

	needsUpdate := false
	newData := msg.Data()
	newFormat := msg.DataFormat()

	// 1. Check Main Data
	var dataStr string
	// Handle types.Data which might be string or []byte alias depending on definition.
	// Assuming it's compatible with string conversion or casting.
	// If types.Data is `type Data string`, we need to cast.
	// If it's interface, we type assert.
	// Based on error `cannot use newData (variable of string type types.Data) as string value`,
	// it seems `types.Data` is a defined type, likely `type Data string`.
	dataStr = string(newData)

	if newFormat == "Image" || strings.HasSuffix(dataStr, ".png") || strings.HasSuffix(dataStr, ".jpg") {
		if !strings.HasPrefix(dataStr, "data:image") && len(dataStr) < 1024 && (strings.Contains(dataStr, "/") || strings.Contains(dataStr, "\\")) {
			// Try to read file
			bytes, err := os.ReadFile(dataStr)
			if err == nil {
				// Detect mime type simple heuristic
				mimeType := "image/png"
				if strings.HasSuffix(dataStr, ".jpg") || strings.HasSuffix(dataStr, ".jpeg") {
					mimeType = "image/jpeg"
				}
				b64 := base64.StdEncoding.EncodeToString(bytes)
				// Convert back to types.Data
				newData = types.Data(fmt.Sprintf("data:%s;base64,%s", mimeType, b64))
				needsUpdate = true
			}
		}
	}

	// 2. Check DataT items
	newDataT := msg.DataT()
	if newDataT != nil {
		allObjs := newDataT.GetAll()
		for _, obj := range allObjs {
			// Check if body is a string path to image
			if strVal, ok := obj.Body().(string); ok {
				if strings.HasSuffix(strVal, ".png") || strings.HasSuffix(strVal, ".jpg") {
					if !strings.HasPrefix(strVal, "data:image") && len(strVal) < 1024 && (strings.Contains(strVal, "/") || strings.Contains(strVal, "\\")) {
						bytes, err := os.ReadFile(strVal)
						if err == nil {
							// We found an image path and read it.
							// However, we can't easily update the CoreObject in DataT in this generic Aspect
							// without access to CoreObject setters or DataT mutators which might not be exposed.
							// For the strict requirement "rulemsg://data format is Image", we prioritized Main Data.
							// We will skip DataT modification for now to avoid complexity with immutable/interface types.
							_ = bytes
						}
					}
				}
			}
		}
	}

	if needsUpdate {
		// message with updated data
		msg.SetData(string(newData), cnst.IMAGE)
	}

	return msg
}
