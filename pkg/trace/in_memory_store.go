package trace

import (
	"sort"
	"sync"
	"time"

	"github.com/neohetj/matrix/pkg/types"
)

// InMemoryStore implements the Store interface, storing snapshots in memory.
type InMemoryStore struct {
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	store     sync.Map // key: string, value: *types.ExecutionStatus
	ttl       time.Duration
}

// NewInMemoryStore creates a new in-memory store instance.
func NewInMemoryStore(ttl time.Duration) types.Store {
	store := &InMemoryStore{
		stop: make(chan struct{}),
		done: make(chan struct{}),
		ttl:  ttl,
	}
	go store.cleanupLoop(1 * time.Minute)
	return store
}

// Set adds or updates a snapshot status.
func (s *InMemoryStore) Set(executionID string, status *types.ExecutionStatus) {
	s.store.Store(executionID, status)
}

// Get retrieves a snapshot status by its ID.
func (s *InMemoryStore) Get(executionID string) (*types.ExecutionStatus, bool) {
	if val, ok := s.store.Load(executionID); ok {
		return val.(*types.ExecutionStatus), true
	}
	return nil, false
}

// loadOrStore keeps all node writers and the finalizer on the same status.
func (s *InMemoryStore) loadOrStore(executionID string, status *types.ExecutionStatus) *types.ExecutionStatus {
	actual, _ := s.store.LoadOrStore(executionID, status)
	return actual.(*types.ExecutionStatus)
}

type atomicStatusStore interface {
	loadOrStore(string, *types.ExecutionStatus) *types.ExecutionStatus
}

func loadOrStoreStatus(store types.Store, executionID string, status *types.ExecutionStatus) *types.ExecutionStatus {
	if atomicStore, ok := store.(atomicStatusStore); ok {
		return atomicStore.loadOrStore(executionID, status)
	}
	// Custom stores retain the existing Store contract.
	store.Set(executionID, status)
	return status
}

// List returns a list of execution statuses, ordered by StartTs descending.
// Implements types.StoreListable interface.
func (s *InMemoryStore) List(limit int) []*types.ExecutionStatus {
	var statuses []*types.ExecutionStatus
	s.store.Range(func(key, value interface{}) bool {
		statuses = append(statuses, value.(*types.ExecutionStatus))
		return true
	})

	// Sort by StartTs descending
	sort.Slice(statuses, func(i, j int) bool {
		// Note: We might need to lock if StartTs changes, but for listing it's usually fine
		// to read potentially stale StartTs. Or lock each?
		// For performance in List, we avoid heavy locking. StartTs is set early.
		return statuses[i].Snapshot.StartTs > statuses[j].Snapshot.StartTs
	})

	if limit > 0 && len(statuses) > limit {
		return statuses[:limit]
	}
	return statuses
}

// Delete removes a snapshot status by its ID.
func (s *InMemoryStore) Delete(executionID string) {
	s.store.Delete(executionID)
}

// cleanupLoop is a background routine that periodically cleans up expired completed snapshots.
func (s *InMemoryStore) cleanupLoop(interval time.Duration) {
	defer close(s.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
		}
		now := time.Now().UnixNano()
		s.store.Range(func(key, value interface{}) bool {
			status := value.(*types.ExecutionStatus)

			status.Lock()
			isCompleted := status.Snapshot.EndTs > 0
			lastUpdated := status.LastUpdated
			status.Unlock()

			if isCompleted && (now-lastUpdated) > int64(s.ttl) {
				s.store.Delete(key)
			}
			return true
		})
	}
}

// Close stops background maintenance without deleting stored trace data.
func (s *InMemoryStore) Close() error {
	if s.stop == nil {
		return nil
	}
	s.closeOnce.Do(func() { close(s.stop) })
	<-s.done
	return nil
}
