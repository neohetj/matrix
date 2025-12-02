package types

import "sync"

const (
	// ExecutionIDKey is the unified key for tracing a single execution instance
	// across the entire call chain.
	ExecutionIDKey = "X-Execution-ID"
)

// RuleNodeRunLog 存储单个节点的执行日志
type RuleNodeRunLog struct {
	Id      string  `json:"id"`
	NodeID  string  `json:"nodeId"`
	Name    string  `json:"name"`
	StartTs int64   `json:"startTs"`
	EndTs   int64   `json:"endTs"`
	InMsg   RuleMsg `json:"inMsg"`
	OutMsg  RuleMsg `json:"outMsg"`
	Err     string  `json:"err,omitempty"`
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

// SnapshotFinalizer defines an interface for marking a trace instance as complete.
// This is used to allow the core framework (like an endpoint) to call back to
// an upper-level application's (like trinity's) trace manager without creating
// a circular dependency.
type SnapshotFinalizer interface {
	FinalizeSnapshot(executionID string)
}
