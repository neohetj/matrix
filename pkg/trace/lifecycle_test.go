package trace

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

func TestConcurrentRecordAndFinalizePreserveCompletedSnapshot(t *testing.T) {
	store := NewInMemoryStore(time.Hour)
	manager := &Manager{store: store}
	t.Cleanup(func() { _ = manager.Close() })
	tracer := NewTracer(store)
	for index := 0; index < 3000; index++ {
		executionID := fmt.Sprintf("execution-%d", index)
		start := make(chan struct{})
		var workers sync.WaitGroup
		workers.Add(3)
		for logIndex := 0; logIndex < 2; logIndex++ {
			go func() {
				defer workers.Done()
				<-start
				tracer.RecordNodeLog(executionID, types.RuleNodeRunLog{StartTs: time.Now().UnixNano()})
			}()
		}
		go func() {
			defer workers.Done()
			<-start
			manager.FinalizeSnapshot(executionID)
		}()
		close(start)
		workers.Wait()
		status, found := store.Get(executionID)
		if !found || status.Snapshot.EndTs == 0 || len(status.Snapshot.Logs) != 2 {
			t.Fatalf("concurrent trace creation lost completion or logs: %+v", status)
		}
	}
}

func TestCompletedSnapshotsExpireWhileActiveSnapshotsRemain(t *testing.T) {
	store := &InMemoryStore{stop: make(chan struct{}), done: make(chan struct{}), ttl: time.Hour}
	go store.cleanupLoop(time.Millisecond)
	manager := &Manager{store: store}
	t.Cleanup(func() { _ = manager.Close() })
	tracer := NewTracer(store)
	for _, executionID := range []string{"completed", "active"} {
		tracer.RecordNodeLog(executionID, types.RuleNodeRunLog{StartTs: time.Now().UnixNano()})
	}
	manager.FinalizeSnapshot("completed")
	for _, executionID := range []string{"completed", "active"} {
		status, _ := store.Get(executionID)
		status.Lock()
		status.LastUpdated = time.Now().Add(-2 * time.Hour).UnixNano()
		status.Unlock()
	}
	deadline := time.Now().Add(time.Second)
	for {
		if _, found := store.Get("completed"); !found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expired completed snapshot was not collected")
		}
		time.Sleep(time.Millisecond)
	}
	if _, found := store.Get("active"); !found {
		t.Fatal("cleanup removed an unfinished execution")
	}
}

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
