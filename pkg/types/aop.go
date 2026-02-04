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

package types

import "sync"

// Aspect is an interface for components that can intercept the execution of a node.
// Aspects are designed to be stateless and can be executed in parallel.
// They are suitable for implementing cross-cutting concerns like logging, metrics,
// security checks, or request/response manipulation.
type Aspect interface {
	// Before is executed before a node's OnMsg method is called.
	// It can modify the message before the node processes it.
	// If it returns an error, the node's OnMsg will be skipped, and the error
	// will be passed to the After method.
	Before(ctx NodeCtx, msg RuleMsg) (RuleMsg, error)

	// After is executed after a node's OnMsg method completes.
	// It receives the original message and any error that occurred during
	// the node's execution (or from the Before method).
	After(ctx NodeCtx, msg RuleMsg, err error)
}

// CallbackFunc is an interface for components that process aggregated results
// from a rule chain execution. Callbacks are stateful and are executed serially.
// They are ideal for scenarios that require a complete picture of the execution,
// such as generating a final report, snapshotting the entire run, or
// broadcasting detailed execution logs.
type CallbackFunc interface {
	// OnNodeCompleted is called serially every time a node finishes its execution.
	OnNodeCompleted(ctx NodeCtx, msg RuleMsg, err error)

	// OnChainCompleted is called once at the very end of the entire rule chain execution.
	// It receives the final message and error of the chain.
	OnChainCompleted(msg RuleMsg, err error)
}

const (
	// ExecutionIDKey is the unified key for tracing a single execution instance
	// across the entire call chain.
	ExecutionIDKey = "X-Execution-ID"
)

// RuleNodeRunLog 存储单个节点的执行日志
type RuleNodeRunLog struct {
	Id          string  `json:"id"`
	NodeID      string  `json:"nodeId"`
	RuleChainID string  `json:"ruleChainId"`
	Name        string  `json:"name"`
	StartTs     int64   `json:"startTs"`
	EndTs       int64   `json:"endTs"`
	InMsg       RuleMsg `json:"inMsg"`
	OutMsg      RuleMsg `json:"outMsg"`
	Err         string  `json:"err,omitempty"`
}

// RuleChainRunSnapshot 存储整个规则链的执行快照
type RuleChainRunSnapshot struct {
	Id      string           `json:"id"`
	StartTs int64            `json:"startTs"`
	EndTs   int64            `json:"endTs"`
	Err     string           `json:"err,omitempty"`
	Logs    []RuleNodeRunLog `json:"logs"`
}

// ExecutionStatus stores the complete snapshot and metadata for a single
// rule chain execution.
type ExecutionStatus struct {
	sync.Mutex
	Snapshot    RuleChainRunSnapshot
	LastUpdated int64
}

// Store defines the generic interface for storing snapshot data.
type Store interface {
	Set(executionID string, status *ExecutionStatus)
	Get(executionID string) (*ExecutionStatus, bool)
	Delete(executionID string)
}

// StoreListable extends Store with the ability to list snapshots.
// This is an optional interface that a Store implementation can provide.
type StoreListable interface {
	Store
	// List returns a list of execution statuses, optionally filtered or limited.
	List(limit int) []*ExecutionStatus
}

// SnapshotFinalizer defines an interface for marking a trace instance as complete.
// This is used to allow the core framework (like an endpoint) to call back to
// an upper-level application's (like trinity's) trace manager without creating
// a circular dependency.
type SnapshotFinalizer interface {
	FinalizeSnapshot(executionID string)
}
