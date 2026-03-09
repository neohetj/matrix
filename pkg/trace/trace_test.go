package trace

import (
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

type mockRuleMsg struct {
	metadata types.Metadata
}

func (m *mockRuleMsg) ID() string                               { return "" }
func (m *mockRuleMsg) Ts() int64                                { return 0 }
func (m *mockRuleMsg) Type() string                             { return "" }
func (m *mockRuleMsg) DataFormat() cnst.MFormat                 { return "" }
func (m *mockRuleMsg) Data() types.Data                         { return "" }
func (m *mockRuleMsg) DataT() types.DataT                       { return nil }
func (m *mockRuleMsg) Metadata() types.Metadata                 { return m.metadata }
func (m *mockRuleMsg) SetData(data string, format cnst.MFormat) {}
func (m *mockRuleMsg) SetMetadata(metadata types.Metadata)      { m.metadata = metadata }
func (m *mockRuleMsg) Copy() types.RuleMsg                      { return &mockRuleMsg{metadata: m.metadata.Copy()} }
func (m *mockRuleMsg) DeepCopy() (types.RuleMsg, error) {
	return &mockRuleMsg{metadata: m.metadata.Copy()}, nil
}
func (m *mockRuleMsg) WithDataFormat(format cnst.MFormat) types.RuleMsg { return m }

func TestRecordNodeLogUsesMetadataStartRuleChainID(t *testing.T) {
	store := NewInMemoryStore(time.Minute)
	tracer := NewTracer(store)

	log := types.RuleNodeRunLog{
		RuleChainID: "child-chain",
		StartTs:     100,
		InMsg: &mockRuleMsg{
			metadata: types.Metadata{
				types.ExecutionStartRuleChainIDKey: "root-chain",
			},
		},
	}

	tracer.RecordNodeLog("exec-1", log)
	status, ok := store.Get("exec-1")
	if !ok {
		t.Fatalf("expected snapshot to be created")
	}
	if got, want := status.Snapshot.StartRuleChainID, "root-chain"; got != want {
		t.Fatalf("StartRuleChainID = %q, want %q", got, want)
	}
}

func TestRecordNodeLogFallsBackToNodeRuleChainID(t *testing.T) {
	store := NewInMemoryStore(time.Minute)
	tracer := NewTracer(store)

	log := types.RuleNodeRunLog{
		RuleChainID: "child-chain",
		StartTs:     100,
	}

	tracer.RecordNodeLog("exec-2", log)
	status, ok := store.Get("exec-2")
	if !ok {
		t.Fatalf("expected snapshot to be created")
	}
	if got, want := status.Snapshot.StartRuleChainID, "child-chain"; got != want {
		t.Fatalf("StartRuleChainID = %q, want %q", got, want)
	}
}

func TestRecordNodeLogBackfillsStartRuleChainIDWhenEmpty(t *testing.T) {
	store := NewInMemoryStore(time.Minute)
	tracer := NewTracer(store)

	store.Set("exec-3", &types.ExecutionStatus{
		Snapshot: types.RuleChainRunSnapshot{
			Id:   "exec-3",
			Logs: []types.RuleNodeRunLog{},
		},
	})

	tracer.RecordNodeLog("exec-3", types.RuleNodeRunLog{
		RuleChainID: "child-chain",
		StartTs:     10,
		InMsg: &mockRuleMsg{
			metadata: types.Metadata{
				types.ExecutionStartRuleChainIDKey: "root-chain",
			},
		},
	})

	status, ok := store.Get("exec-3")
	if !ok {
		t.Fatalf("expected snapshot to exist")
	}
	if got, want := status.Snapshot.StartRuleChainID, "root-chain"; got != want {
		t.Fatalf("StartRuleChainID = %q, want %q", got, want)
	}
}
