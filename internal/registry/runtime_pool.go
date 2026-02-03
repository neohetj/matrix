package registry

import (
	"fmt"
	"sync"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

// runtimePool is the default thread-safe implementation of RuntimePool.
type runtimePool struct {
	mu       sync.RWMutex
	runtimes map[string]types.Runtime
	triggers map[string][]types.TriggerSource
}

// NewRuntimePool creates a new instance of the default runtime pool.
func NewRuntimePool() types.RuntimePool {
	return &runtimePool{
		runtimes: make(map[string]types.Runtime),
		triggers: make(map[string][]types.TriggerSource),
	}
}

func (p *runtimePool) Get(id string) (types.Runtime, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	runtime, ok := p.runtimes[id]
	return runtime, ok
}

func (p *runtimePool) Register(id string, runtime types.Runtime) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.runtimes[id]; exists {
		return fmt.Errorf("runtime with id '%s' already registered", id)
	}
	p.runtimes[id] = runtime
	p.registerTriggersLocked(id, runtime)
	return nil
}

func (p *runtimePool) Unregister(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if runtime, exists := p.runtimes[id]; exists {
		p.unregisterTriggersLocked(id, runtime)
		delete(p.runtimes, id)
	}
}

func (p *runtimePool) ListIDs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, 0, len(p.runtimes))
	for id := range p.runtimes {
		ids = append(ids, id)
	}
	return ids
}

// TODO: The current implementation of ListByViewType is O(n). For scenarios with a very
// large number of runtimes, this could be optimized by adding an index (e.g., a map
// from viewType to a list of runtime IDs) to the runtimePool struct. This would
// trade a small amount of write-time complexity and memory for O(1) read-time complexity.
func (p *runtimePool) ListByViewType(viewType string) []types.Runtime {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var result []types.Runtime
	for _, r := range p.runtimes {
		if r.Definition().RuleChain.Attrs.ViewType == cnst.ViewType(viewType) {
			result = append(result, r)
		}
	}
	return result
}

func (p *runtimePool) GetTriggers(chainID string) []types.TriggerSource {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// Return a copy to avoid race conditions if the caller modifies the slice
	if sources, ok := p.triggers[chainID]; ok {
		result := make([]types.TriggerSource, len(sources))
		copy(result, sources)
		return result
	}
	return nil
}

func (p *runtimePool) RegisterTrigger(targetChainID string, source types.TriggerSource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.triggers[targetChainID] = append(p.triggers[targetChainID], source)
}

func (p *runtimePool) UnregisterTrigger(targetChainID string, source types.TriggerSource) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.removeTriggerSourceLocked(targetChainID, source)
}

// Helper methods (must be called with lock held)

func (p *runtimePool) registerTriggersLocked(runtimeID string, runtime types.Runtime) {
	instance := runtime.GetChainInstance()
	if instance == nil {
		return
	}
	nodes := instance.GetAllNodes()
	for _, node := range nodes {
		// 1. Check for SubChainTrigger (e.g., flow node)
		if trigger, ok := node.(types.SubChainTrigger); ok {
			targetID := trigger.GetTargetChainID()
			if targetID != "" {
				source := types.TriggerSource{
					SourceChainID: runtimeID,
					NodeID:        node.ID(),
					NodeType:      string(node.Type()),
					IsEndpoint:    false,
				}
				p.triggers[targetID] = append(p.triggers[targetID], source)
			}
		}
		// 1.5 Check for MultiChainTrigger (e.g., pipeline endpoint)
		if trigger, ok := node.(types.MultiChainTrigger); ok {
			for _, targetID := range trigger.GetTargetChainIDs() {
				if targetID != "" {
					source := types.TriggerSource{
						SourceChainID: runtimeID,
						NodeID:        node.ID(),
						NodeType:      string(node.Type()),
						IsEndpoint:    false,
					}
					p.triggers[targetID] = append(p.triggers[targetID], source)
				}
			}
		}
		// 2. Check for Endpoint (self-trigger)
		if _, ok := node.(types.Endpoint); ok {
			source := types.TriggerSource{
				SourceChainID: runtimeID,
				NodeID:        node.ID(),
				NodeType:      string(node.Type()),
				IsEndpoint:    true,
			}
			p.triggers[runtimeID] = append(p.triggers[runtimeID], source)
		}
	}
}

func (p *runtimePool) unregisterTriggersLocked(runtimeID string, runtime types.Runtime) {
	instance := runtime.GetChainInstance()
	if instance == nil {
		return
	}
	nodes := instance.GetAllNodes()
	for _, node := range nodes {
		if trigger, ok := node.(types.SubChainTrigger); ok {
			targetID := trigger.GetTargetChainID()
			if targetID != "" {
				source := types.TriggerSource{
					SourceChainID: runtimeID,
					NodeID:        node.ID(),
				}
				p.removeTriggerSourceLocked(targetID, source)
			}
		}
		if trigger, ok := node.(types.MultiChainTrigger); ok {
			for _, targetID := range trigger.GetTargetChainIDs() {
				if targetID != "" {
					source := types.TriggerSource{
						SourceChainID: runtimeID,
						NodeID:        node.ID(),
					}
					p.removeTriggerSourceLocked(targetID, source)
				}
			}
		}
		if _, ok := node.(types.Endpoint); ok {
			source := types.TriggerSource{
				SourceChainID: runtimeID,
				NodeID:        node.ID(),
			}
			p.removeTriggerSourceLocked(runtimeID, source)
		}
	}
}

func (p *runtimePool) removeTriggerSourceLocked(targetChainID string, sourceToRemove types.TriggerSource) {
	sources := p.triggers[targetChainID]
	for i, s := range sources {
		// Match by SourceChainID and NodeID
		if s.SourceChainID == sourceToRemove.SourceChainID && s.NodeID == sourceToRemove.NodeID {
			// Remove the element
			p.triggers[targetChainID] = append(sources[:i], sources[i+1:]...)
			return
		}
	}
}
