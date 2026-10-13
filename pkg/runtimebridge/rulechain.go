package runtimebridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// FillFunc mutates the initial RuleMsg before the target rulechain executes.
type FillFunc func(types.RuleMsg) error

// EndFunc is passed through to Runtime.ExecuteAndWait.
type EndFunc func(types.RuleMsg, error)

type executeOptions struct {
	startNodeID string
	metadata    types.Metadata
	onEnd       EndFunc
}

// Option configures either rulechain execution entry.
type Option func(*executeOptions)

// WithStartNodeID starts execution from a specific rule node.
func WithStartNodeID(startNodeID string) Option {
	return func(opts *executeOptions) {
		opts.startNodeID = strings.TrimSpace(startNodeID)
	}
}

// WithMetadata copies metadata into the initial RuleMsg.
func WithMetadata(metadata types.Metadata) Option {
	return func(opts *executeOptions) {
		if len(metadata) == 0 {
			return
		}
		if opts.metadata == nil {
			opts.metadata = types.Metadata{}
		}
		for key, value := range metadata {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			opts.metadata[key] = value
		}
	}
}

// WithExecutionID sets the execution trace id on the initial RuleMsg metadata.
func WithExecutionID(executionID string) Option {
	return WithMetadata(types.Metadata{
		types.ExecutionIDKey: strings.TrimSpace(executionID),
	})
}

// WithStartRuleChainID sets the root rulechain id on the initial RuleMsg metadata.
func WithStartRuleChainID(ruleChainID string) Option {
	return WithMetadata(types.Metadata{
		types.ExecutionStartRuleChainIDKey: strings.TrimSpace(ruleChainID),
	})
}

// WithEndFunc passes a completion callback to Runtime.ExecuteAndWait.
func WithEndFunc(onEnd EndFunc) Option {
	return func(opts *executeOptions) {
		opts.onEnd = onEnd
	}
}

// ExecuteRuleChain bridges trusted Go/orchestrator code into a Matrix rulechain.
// It centralizes runtime lookup, initial RuleMsg construction, metadata propagation,
// optional message filling, and synchronous execution.
func ExecuteRuleChain(
	ctx context.Context,
	engine types.MatrixEngine,
	ruleChainID string,
	fill FillFunc,
	options ...Option,
) (types.RuleMsg, error) {
	return executeRuleChain(ctx, engine, ruleChainID, fill, false, options...)
}

// ExecuteRootRuleChain explicitly owns a new root trace. The caller must supply
// an independent, non-empty execution ID. Preparing the message never creates a
// snapshot; once execution starts, every return completes the root trace.
// A timeout ends the wait, not necessarily the underlying asynchronous work.
func ExecuteRootRuleChain(ctx context.Context, engine types.MatrixEngine, ruleChainID string, fill FillFunc, options ...Option) (types.RuleMsg, error) {
	return executeRuleChain(ctx, engine, ruleChainID, fill, true, options...)
}

func executeRuleChain(ctx context.Context, engine types.MatrixEngine, ruleChainID string, fill FillFunc, ownsRoot bool, options ...Option) (types.RuleMsg, error) {
	ruleChainID = strings.TrimSpace(ruleChainID)
	if ruleChainID == "" {
		return nil, fmt.Errorf("rulechain id is required")
	}
	if engine == nil {
		return nil, fmt.Errorf("matrix engine is nil")
	}
	pool := engine.RuntimePool()
	if pool == nil {
		return nil, fmt.Errorf("matrix runtime pool is nil")
	}
	runtime, ok := pool.Get(ruleChainID)
	if !ok || runtime == nil {
		return nil, fmt.Errorf("runtime not found for rulechain %s", ruleChainID)
	}

	opts := executeOptions{metadata: types.Metadata{}}
	for _, option := range options {
		if option != nil {
			previousID, hadID := opts.metadata[types.ExecutionIDKey]
			previousRoot, hadRoot := opts.metadata[types.ExecutionStartRuleChainIDKey]
			option(&opts)
			if ownsRoot && ((hadID && previousID != opts.metadata[types.ExecutionIDKey]) || (hadRoot && previousRoot != opts.metadata[types.ExecutionStartRuleChainIDKey])) {
				return nil, fmt.Errorf("option changed previously declared root execution identity")
			}
		}
	}
	executionID := opts.metadata[types.ExecutionIDKey]
	if ownsRoot {
		if executionID == "" || executionID != strings.TrimSpace(executionID) {
			return nil, fmt.Errorf("root execution requires a non-empty execution id without surrounding whitespace")
		}
		if root, present := opts.metadata[types.ExecutionStartRuleChainIDKey]; present && root != ruleChainID {
			return nil, fmt.Errorf("root rulechain identity conflicts with target rulechain")
		}
	}
	if _, ok := opts.metadata[types.ExecutionStartRuleChainIDKey]; !ok {
		if _, hasExecutionID := opts.metadata[types.ExecutionIDKey]; hasExecutionID {
			opts.metadata[types.ExecutionStartRuleChainIDKey] = ruleChainID
		}
	}

	msg := types.NewMsg(ruleChainID, "", opts.metadata, types.NewDataT())
	if fill != nil {
		if err := fill(msg); err != nil {
			return nil, err
		}
	}
	if ownsRoot {
		metadata := msg.Metadata()
		if metadata[types.ExecutionIDKey] != executionID || metadata[types.ExecutionStartRuleChainIDKey] != ruleChainID {
			return nil, fmt.Errorf("fill changed root execution identity")
		}
		if owner, ok := engine.(interface {
			TraceManager() types.SnapshotFinalizer
		}); ok {
			if finalizer := owner.TraceManager(); finalizer != nil {
				defer finalizer.FinalizeSnapshot(executionID)
			}
		}
	}
	return runtime.ExecuteAndWait(ctx, opts.startNodeID, msg, opts.onEnd)
}
