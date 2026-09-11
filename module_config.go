package matrix

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/types"
)

// WithModuleConfig 为一个明确模块绑定实例 Reader；不会推断默认模块。
func WithModuleConfig(moduleID string, reader types.ConfigReader) Option {
	return func(e *MatrixEngine) {
		if strings.TrimSpace(moduleID) == "" || nilConfigReader(reader) {
			e.moduleConfigErr = types.ErrConfigReaderUnavailable
			return
		}
		if e.moduleConfigs == nil {
			e.moduleConfigs = make(map[string]types.ConfigReader)
		}
		if _, exists := e.moduleConfigs[moduleID]; exists {
			e.moduleConfigErr = fmt.Errorf("duplicate module configuration binding")
			return
		}
		e.moduleConfigs[moduleID] = reader
	}
}

// WithNodeConfigOwners 复制显式节点归属，后续修改调用方 map 不改变 Engine。
func WithNodeConfigOwners(owners map[string]string) Option {
	return func(e *MatrixEngine) {
		if e.nodeConfigOwners == nil {
			e.nodeConfigOwners = make(map[string]string)
		}
		for nodeID, moduleID := range owners {
			if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(moduleID) == "" {
				e.moduleConfigErr = types.ErrConfigReaderUnavailable
				continue
			}
			if previous, exists := e.nodeConfigOwners[nodeID]; exists && previous != moduleID {
				e.moduleConfigErr = fmt.Errorf("conflicting node configuration ownership")
				continue
			}
			e.nodeConfigOwners[nodeID] = moduleID
		}
	}
}

// WithRuleChainConfigOwners 复制规则链的模块归属，不使用链内局部 node ID 推断模块。
func WithRuleChainConfigOwners(owners map[string]string) Option {
	return func(e *MatrixEngine) {
		if e.ruleChainConfigOwners == nil {
			e.ruleChainConfigOwners = make(map[string]string)
		}
		for chainID, moduleID := range owners {
			if strings.TrimSpace(chainID) == "" || strings.TrimSpace(moduleID) == "" {
				e.moduleConfigErr = types.ErrConfigReaderUnavailable
				continue
			}
			if previous, exists := e.ruleChainConfigOwners[chainID]; exists && previous != moduleID {
				e.moduleConfigErr = fmt.Errorf("conflicting rule chain configuration ownership")
				continue
			}
			e.ruleChainConfigOwners[chainID] = moduleID
		}
	}
}

// ConfigReaderForRuleChain 只按规则链归属或显式 standalone 默认模块选择配置。
func (e *MatrixEngine) ConfigReaderForRuleChain(chainID string) (types.ConfigReader, bool) {
	return e.ConfigReader(e.configModuleForRuleChain(chainID))
}

// configModuleForRuleChain 统一规则链显式归属与 standalone 默认模块选择。
func (e *MatrixEngine) configModuleForRuleChain(chainID string) string {
	owner, exists := e.ruleChainConfigOwners[chainID]
	if !exists {
		owner = e.defaultConfigModule
	}
	return owner
}

// ValidateRuleChainConfigImports 拒绝 flatten 后无法安全区分的跨模块或未归属导入。
func (e *MatrixEngine) ValidateRuleChainConfigImports(chainID string, imports []string) error {
	owner := e.configModuleForRuleChain(chainID)
	if _, ok := e.ConfigReader(owner); !ok {
		return types.ErrConfigReaderUnavailable
	}
	for _, imported := range imports {
		if importedOwner := e.configModuleForRuleChain(imported); importedOwner == "" || importedOwner != owner {
			return types.ErrConfigReaderUnavailable
		}
	}
	return nil
}

// WithDefaultConfigModule 显式指定 standalone 节点缺少单独归属时的模块。
func WithDefaultConfigModule(moduleID string) Option {
	return func(e *MatrixEngine) {
		if strings.TrimSpace(moduleID) == "" {
			e.moduleConfigErr = types.ErrConfigReaderUnavailable
			return
		}
		if e.defaultConfigModule != "" && e.defaultConfigModule != moduleID {
			e.moduleConfigErr = fmt.Errorf("conflicting default configuration module")
			return
		}
		e.defaultConfigModule = moduleID
	}
}

// ConfigReader 返回当前 Engine 的模块配置，不回退到其他 Engine 或全局变量。
func (e *MatrixEngine) ConfigReader(moduleID string) (types.ConfigReader, bool) {
	reader, ok := e.moduleConfigs[moduleID]
	return reader, ok && !nilConfigReader(reader)
}

// ConfigReaderForNode 只读取显式映射或显式默认归属。
func (e *MatrixEngine) ConfigReaderForNode(nodeID string) (types.ConfigReader, bool) {
	owner, exists := e.nodeConfigOwners[nodeID]
	if !exists {
		owner = e.defaultConfigModule
	}
	return e.ConfigReader(owner)
}

// nilConfigReader 同时拒绝 nil 接口和包装了 nil 指针的接口。
func nilConfigReader(reader types.ConfigReader) bool {
	if reader == nil {
		return true
	}
	v := reflect.ValueOf(reader)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// instanceRegistry 复用注册定义，但私有持有运行态资源与运行时池。
type instanceRegistry struct {
	types.RegistryProvider
	nodes    types.NodePool
	runtimes types.RuntimePool
}

// GetSharedNodePool 返回 Engine 私有资源池。
func (r *instanceRegistry) GetSharedNodePool() types.NodePool { return r.nodes }

// GetRuntimePool 返回 Engine 私有规则链运行时池。
func (r *instanceRegistry) GetRuntimePool() types.RuntimePool { return r.runtimes }

// prepareModuleConfig 在资源加载前验证绑定并隔离 Reader 消费者的运行态。
func (e *MatrixEngine) prepareModuleConfig() error {
	if e.moduleConfigErr != nil {
		return e.moduleConfigErr
	}
	for _, owner := range e.nodeConfigOwners {
		if _, ok := e.ConfigReader(owner); !ok {
			return types.ErrConfigReaderUnavailable
		}
	}
	for _, owner := range e.ruleChainConfigOwners {
		if _, ok := e.ConfigReader(owner); !ok {
			return types.ErrConfigReaderUnavailable
		}
	}
	if e.defaultConfigModule != "" {
		if _, ok := e.ConfigReader(e.defaultConfigModule); !ok {
			return types.ErrConfigReaderUnavailable
		}
	}
	if len(e.moduleConfigs) == 0 {
		return nil
	}
	pool := registry.NewNodePool(nil)
	pool.(interface {
		SetConfigReaderProvider(types.NodeConfigReaderProvider)
	}).SetConfigReaderProvider(e)
	e.registry = &instanceRegistry{RegistryProvider: e.registry, nodes: pool, runtimes: registry.NewRuntimePool()}
	return nil
}

// abortModuleConfigStartup 只撤销本轮配置实例拥有的资源，不触碰调用方或旧全局池。
func (e *MatrixEngine) abortModuleConfigStartup() {
	owned, ok := e.registry.(*instanceRegistry)
	if !ok {
		return
	}
	if e.activeCancel != nil {
		e.activeCancel()
		e.activeCancel = nil
	}
	for _, endpoint := range e.configActiveEndpoints {
		// 保留主启动错误，不输出可能携带连接信息的 Stop 错误。
		_ = endpoint.Stop()
	}
	e.configActiveEndpoints = nil
	e.activeWG.Wait()
	for _, id := range owned.runtimes.ListIDs() {
		rt, exists := owned.runtimes.Get(id)
		if !exists {
			continue
		}
		if chain := rt.GetChainInstance(); chain != nil {
			for _, node := range chain.GetAllNodes() {
				// 共享节点由池统一释放，避免跨链复用导致重复 Destroy。
				if _, shared := owned.nodes.Get(node.ID()); !shared {
					node.Destroy()
				}
			}
		}
		owned.runtimes.Unregister(id)
	}
	owned.nodes.Stop()
}
