package trace

import (
	"testing"
	"time"
)

func TestManagerCloseStopsMaintenanceWithoutDeletingPayload(t *testing.T) {
	store := NewInMemoryStore(time.Hour).(*InMemoryStore)
	payload := NewFilePayloadStore(t.TempDir(), time.Hour)
	manager := &Manager{store: store, payloadStore: payload}
	if err := payload.SaveMessage("execution", "log", "input", []byte(`{"value":1}`)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := manager.Close(); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-store.done:
	default:
		t.Fatal("memory maintenance still running")
	}
	select {
	case <-payload.done:
	default:
		t.Fatal("payload maintenance still running")
	}
	if raw, err := payload.LoadMessage("execution", "log", "input"); err != nil || string(raw) != `{"value":1}` {
		t.Fatal("close removed payload")
	}
}
