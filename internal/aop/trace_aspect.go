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
	"fmt"
	"time"

	"context"

	"github.com/google/uuid"
	"gitlab.com/neohet/matrix/pkg/trace"
	"gitlab.com/neohet/matrix/pkg/types"
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
	nodeLog := &types.RuleNodeRunLog{
		Id:      uuid.New().String(),
		NodeID:  ctx.NodeID(),
		Name:    ctx.SelfDef().Name,
		StartTs: time.Now().UnixNano(),
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
	if errMsg, ok := msg.Metadata()["error"]; ok {
		logInProgress.Err = appendError(logInProgress.Err, fmt.Sprintf("Metadata error: %v", errMsg))
	}

	// 4. Record the complete log synchronously.
	a.tracer.RecordNodeLog(executionID, *logInProgress)
}
