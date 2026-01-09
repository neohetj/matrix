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
}

// NewRuntimePool creates a new instance of the default runtime pool.
func NewRuntimePool() types.RuntimePool {
	return &runtimePool{
		runtimes: make(map[string]types.Runtime),
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
	return nil
}

func (p *runtimePool) Unregister(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.runtimes, id)
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
