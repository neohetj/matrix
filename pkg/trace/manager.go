package trace

import (
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

// Manager manages the lifecycle of trace snapshots.
// It provides a tracer for the runtime and methods to interact with the snapshot store.
type Manager struct {
	store types.Store
}

// NewManager creates a new Manager instance.
func NewManager(store types.Store) *Manager {
	return &Manager{
		store: store,
	}
}

// GetTracer creates a new Tracer instance for injection into AOP aspects.
func (m *Manager) GetTracer() *Tracer {
	return NewTracer(m.store)
}

// GetSnapshot retrieves an execution snapshot from the store.
func (m *Manager) GetSnapshot(executionID string) (*types.ExecutionStatus, bool) {
	return m.store.Get(executionID)
}

// GetRecentSnapshots returns the most recent snapshots when the store supports listing.
// When the underlying store does not implement types.StoreListable, it returns an empty slice.
func (m *Manager) GetRecentSnapshots(limit int) []*types.ExecutionStatus {
	listable, ok := m.store.(types.StoreListable)
	if !ok {
		return nil
	}
	return listable.List(limit)
}

// FinalizeSnapshot marks a trace snapshot as complete by setting its EndTs.
// This method implements the types.SnapshotFinalizer interface.
func (m *Manager) FinalizeSnapshot(executionID string) {
	status, found := m.store.Get(executionID)
	if !found {
		now := time.Now().UnixNano()
		status = &types.ExecutionStatus{
			Snapshot: types.RuleChainRunSnapshot{
				Id:      executionID,
				StartTs: now, // Best effort start time
				EndTs:   now,
			},
			LastUpdated: now,
		}
	} else {
		status.Lock()
		if status.Snapshot.EndTs == 0 {
			now := time.Now().UnixNano()
			status.Snapshot.EndTs = now
			status.LastUpdated = now
		}
		status.Unlock()
	}
	m.store.Set(executionID, status)
}
