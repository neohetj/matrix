package trace

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

type mockRuleMsg struct {
	id         string
	ts         int64
	msgType    string
	dataFormat cnst.MFormat
	data       types.Data
	metadata   types.Metadata
}

func (m *mockRuleMsg) ID() string               { return m.id }
func (m *mockRuleMsg) Ts() int64                { return m.ts }
func (m *mockRuleMsg) Type() string             { return m.msgType }
func (m *mockRuleMsg) DataFormat() cnst.MFormat { return m.dataFormat }
func (m *mockRuleMsg) Data() types.Data         { return m.data }
func (m *mockRuleMsg) DataT() types.DataT       { return nil }
func (m *mockRuleMsg) Metadata() types.Metadata { return m.metadata }
func (m *mockRuleMsg) SetData(data string, format cnst.MFormat) {
	m.data, m.dataFormat = types.Data(data), format
}
func (m *mockRuleMsg) SetMetadata(metadata types.Metadata) { m.metadata = metadata }
func (m *mockRuleMsg) Copy() types.RuleMsg {
	return &mockRuleMsg{
		id:         m.id,
		ts:         m.ts,
		msgType:    m.msgType,
		dataFormat: m.dataFormat,
		data:       m.data,
		metadata:   m.metadata.Copy(),
	}
}
func (m *mockRuleMsg) DeepCopy() (types.RuleMsg, error) {
	return m.Copy(), nil
}
func (m *mockRuleMsg) WithDataFormat(format cnst.MFormat) types.RuleMsg { return m }
func (m *mockRuleMsg) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":         m.id,
		"ts":         m.ts,
		"msgType":    m.msgType,
		"dataFormat": m.dataFormat,
		"data":       m.data,
		"dataT":      nil,
		"metadata":   m.metadata,
	})
}

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

func TestRecordNodeLogExternalizesImagePayloads(t *testing.T) {
	store := NewInMemoryStore(time.Minute)
	payloadStore := NewFilePayloadStore(t.TempDir(), time.Minute)
	tracer := NewTracer(store, payloadStore)

	tracer.RecordNodeLog("exec-image", types.RuleNodeRunLog{
		Id:          "log-image",
		RuleChainID: "chain-image",
		StartTs:     100,
		InMsg: &mockRuleMsg{
			id:         "msg-1",
			ts:         1,
			msgType:    "chain-image",
			dataFormat: cnst.IMAGE,
			data:       types.Data("data:image/png;base64," + strings.Repeat("a", 128)),
			metadata:   types.Metadata{},
		},
	})

	status, ok := store.Get("exec-image")
	if !ok {
		t.Fatalf("expected snapshot to be created")
	}
	if len(status.Snapshot.Logs) != 1 {
		t.Fatalf("expected exactly one stored log, got %d", len(status.Snapshot.Logs))
	}

	payloadCarrier, ok := status.Snapshot.Logs[0].InMsg.(types.TracePayloadCarrier)
	if !ok {
		t.Fatalf("expected input message to be externalized")
	}

	payloadRef := payloadCarrier.GetTracePayload()
	if payloadRef == nil || !payloadRef.Externalized {
		t.Fatalf("expected externalized payload ref, got %#v", payloadRef)
	}

	raw, err := payloadStore.LoadMessage("exec-image", "log-image", "inMsg")
	if err != nil {
		t.Fatalf("expected payload file to exist: %v", err)
	}
	if !strings.Contains(string(raw), "data:image/png;base64") {
		t.Fatalf("expected stored payload to contain original image data, got %s", string(raw))
	}
}
