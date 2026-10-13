package runtimebridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/trace"
	"github.com/neohetj/matrix/pkg/types"
	tu "github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/mock"
)

const rootTestChain = "test/root"

type rootTestEngine struct {
	*tu.MockEngine
	finalizer types.SnapshotFinalizer
}

func (engine *rootTestEngine) TraceManager() types.SnapshotFinalizer { return engine.finalizer }

func newRootFixture(t *testing.T) (*rootTestEngine, *tu.MockRuntime, *trace.Manager) {
	t.Helper()
	store := trace.NewInMemoryStore(time.Hour)
	manager := trace.NewManager(store)
	t.Cleanup(func() { _ = manager.Close() })
	runtime := &tu.MockRuntime{}
	engine := &rootTestEngine{
		MockEngine: &tu.MockEngine{RuntimePoolValue: &tu.MockRuntimePool{
			Runtimes: map[string]types.Runtime{rootTestChain: runtime},
		}},
		finalizer: manager,
	}
	return engine, runtime, manager
}

func TestRootExecutionCompletesTraceAndPreservesCallback(t *testing.T) {
	for _, outcome := range []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "failure", err: errors.New("failure")},
		{name: "deadline", err: context.DeadlineExceeded},
		{name: "cancelled", err: context.Canceled},
	} {
		t.Run(outcome.name, func(t *testing.T) {
			engine, runtime, manager := newRootFixture(t)
			result := types.NewMsg("result", "", nil, types.NewDataT())
			callbacks := 0
			runtime.On("ExecuteAndWait", mock.Anything, "start", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					msg := args.Get(2).(types.RuleMsg)
					if msg.Metadata()[types.ExecutionStartRuleChainIDKey] != rootTestChain || msg.Metadata()["filled"] != "yes" {
						t.Fatalf("root identity or fill lost: %+v", msg.Metadata())
					}
					manager.GetTracer().RecordNodeLog("root-1", types.RuleNodeRunLog{StartTs: time.Now().UnixNano(), InMsg: msg, OutMsg: msg})
					args.Get(3).(func(types.RuleMsg, error))(result, outcome.err)
				}).Return(result, outcome.err)
			got, err := ExecuteRootRuleChain(context.Background(), engine, rootTestChain,
				func(msg types.RuleMsg) error { msg.Metadata()["filled"] = "yes"; return nil },
				WithExecutionID("root-1"), WithStartNodeID("start"), WithEndFunc(func(msg types.RuleMsg, failure error) {
					callbacks++
					if msg != result || failure != outcome.err {
						t.Fatal("business callback changed")
					}
				}))
			if got != result || err != outcome.err || callbacks != 1 {
				t.Fatal("runtime result or callback changed")
			}
			status, found := manager.GetSnapshot("root-1")
			if !found || status.Snapshot.EndTs == 0 || len(status.Snapshot.Logs) != 1 {
				t.Fatalf("root trace not completed: %+v", status)
			}
		})
	}
}

func TestRootExecutionRejectsPreparationErrorsWithoutSnapshots(t *testing.T) {
	for _, test := range []struct {
		name           string
		options        []Option
		fill           FillFunc
		missingRuntime bool
	}{
		{name: "missing execution ID"},
		{name: "blank execution ID", options: []Option{WithExecutionID(" ")}},
		{name: "conflicting root", options: []Option{WithExecutionID("root-1"), WithStartRuleChainID("test/outer")}},
		{name: "blank root", options: []Option{WithExecutionID("root-1"), WithStartRuleChainID("")}},
		{name: "option changes ID", options: []Option{WithExecutionID("root-1"), WithMetadata(types.Metadata{types.ExecutionIDKey: "inherited"})}},
		{name: "option replaces root", options: []Option{WithExecutionID("root-1"), WithStartRuleChainID("test/outer"), WithStartRuleChainID(rootTestChain)}},
		{name: "fill failure", options: []Option{WithExecutionID("root-1")}, fill: func(types.RuleMsg) error { return errors.New("fill failed") }},
		{name: "fill changes ID", options: []Option{WithExecutionID("root-1")}, fill: func(msg types.RuleMsg) error { msg.Metadata()[types.ExecutionIDKey] = "other"; return nil }},
		{name: "fill changes root", options: []Option{WithExecutionID("root-1")}, fill: func(msg types.RuleMsg) error {
			msg.Metadata()[types.ExecutionStartRuleChainIDKey] = "test/outer"
			return nil
		}},
		{name: "fill removes metadata", options: []Option{WithExecutionID("root-1")}, fill: func(msg types.RuleMsg) error { msg.SetMetadata(nil); return nil }},
		{name: "missing runtime", options: []Option{WithExecutionID("root-1")}, missingRuntime: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, runtime, manager := newRootFixture(t)
			if test.missingRuntime {
				engine.RuntimePoolValue = &tu.MockRuntimePool{}
			}
			runtime.On("ExecuteAndWait", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			if _, err := ExecuteRootRuleChain(context.Background(), engine, rootTestChain, test.fill, test.options...); err == nil {
				t.Fatal("invalid root preparation accepted")
			}
			runtime.AssertNotCalled(t, "ExecuteAndWait", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			if len(manager.GetRecentSnapshots(0)) != 0 {
				t.Fatal("preparation error created a fake snapshot")
			}
		})
	}
}

func TestOrdinaryExecutionDoesNotFinalizeInheritedTrace(t *testing.T) {
	for _, root := range []string{rootTestChain, "test/outer"} {
		t.Run(root, func(t *testing.T) {
			engine, runtime, manager := newRootFixture(t)
			runtime.On("ExecuteAndWait", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					manager.GetTracer().RecordNodeLog("outer-1", types.RuleNodeRunLog{StartTs: time.Now().UnixNano(), InMsg: args.Get(2).(types.RuleMsg)})
				}).Return(nil, nil)
			if _, err := ExecuteRuleChain(context.Background(), engine, rootTestChain, nil, WithExecutionID("outer-1"), WithStartRuleChainID(root)); err != nil {
				t.Fatal(err)
			}
			status, found := manager.GetSnapshot("outer-1")
			if !found || status.Snapshot.EndTs != 0 {
				t.Fatal("ordinary execution ended an inherited trace")
			}
		})
	}
}

func TestRootExecutionWithoutTrace(t *testing.T) {
	for _, api := range []bool{false, true} {
		engine, runtime, _ := newRootFixture(t)
		engine.finalizer = nil
		var target types.MatrixEngine = engine.MockEngine
		if api {
			target = engine
		}
		runtime.On("ExecuteAndWait", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
		if _, err := ExecuteRootRuleChain(context.Background(), target, rootTestChain, nil, WithExecutionID("root-1")); err != nil {
			t.Fatal(err)
		}
		runtime.AssertExpectations(t)
	}
}

func TestRootTimeoutAllowsLateLogsWithoutReopeningSnapshot(t *testing.T) {
	engine, runtime, manager := newRootFixture(t)
	var message types.RuleMsg
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime.On("ExecuteAndWait", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			message = args.Get(2).(types.RuleMsg)
			cancel()
			<-args.Get(0).(context.Context).Done()
		}).Return(nil, context.Canceled)
	if _, err := ExecuteRootRuleChain(ctx, engine, rootTestChain, nil, WithExecutionID("root-late"), WithStartRuleChainID(rootTestChain)); err != context.Canceled {
		t.Fatal(err)
	}
	status, found := manager.GetSnapshot("root-late")
	if !found || status.Snapshot.EndTs == 0 {
		t.Fatal("cancelled wait did not complete root")
	}
	end := status.Snapshot.EndTs
	manager.GetTracer().RecordNodeLog("root-late", types.RuleNodeRunLog{StartTs: time.Now().UnixNano(), InMsg: message})
	late, found := manager.GetSnapshot("root-late")
	if !found || late != status || late.Snapshot.EndTs != end || len(late.Snapshot.Logs) != 1 || late.LastUpdated < end {
		t.Fatal("late After log lost completion or failed to refresh retention")
	}
}

func TestIdentityOptionOverridesRemainCompatibleForOrdinaryExecution(t *testing.T) {
	engine, runtime, _ := newRootFixture(t)
	runtime.On("ExecuteAndWait", mock.Anything, mock.Anything, mock.MatchedBy(func(msg types.RuleMsg) bool {
		return msg.Metadata()[types.ExecutionIDKey] == "inherited" && msg.Metadata()[types.ExecutionStartRuleChainIDKey] == "test/outer"
	}), mock.Anything).Return(nil, nil).Once()
	if _, err := ExecuteRuleChain(context.Background(), engine, rootTestChain, nil, WithExecutionID("first"), WithMetadata(types.Metadata{types.ExecutionIDKey: "inherited"}), WithStartRuleChainID("test/outer")); err != nil {
		t.Fatal(err)
	}
	runtime.AssertExpectations(t)
}

func TestRootExecutionAcceptsRepeatedIdenticalIdentityOptions(t *testing.T) {
	engine, runtime, manager := newRootFixture(t)
	runtime.On("ExecuteAndWait", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
	if _, err := ExecuteRootRuleChain(context.Background(), engine, rootTestChain, nil, WithExecutionID("root-1"), WithMetadata(types.Metadata{types.ExecutionIDKey: "root-1"}), WithStartRuleChainID(rootTestChain), WithStartRuleChainID(rootTestChain)); err != nil {
		t.Fatal(err)
	}
	status, found := manager.GetSnapshot("root-1")
	if !found || status.Snapshot.EndTs == 0 {
		t.Fatal("valid root identity was not completed")
	}
}
