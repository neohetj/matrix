package trace

import (
	"sync"
	"time"

	"gitlab.com/neohet/matrix/pkg/types"
)

// InMemoryStore implements the Store interface, storing snapshots in memory.
type InMemoryStore struct {
	store sync.Map // key: string, value: *types.ExecutionStatus
	ttl   time.Duration
}

// NewInMemoryStore creates a new in-memory store instance.
func NewInMemoryStore(ttl time.Duration) types.Store {
	store := &InMemoryStore{
		ttl: ttl,
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

// Delete removes a snapshot status by its ID.
func (s *InMemoryStore) Delete(executionID string) {
	s.store.Delete(executionID)
}

// cleanupLoop is a background routine that periodically cleans up expired completed snapshots.
func (s *InMemoryStore) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
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
