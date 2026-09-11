package matrix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neohetj/matrix/internal/builder"
	"github.com/neohetj/matrix/internal/loader"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/internal/runtime"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
)

type fixtureReader string

// LookupConfig 返回测试实例自己的值，验证注入不依赖进程状态。
func (r fixtureReader) LookupConfig(context.Context, string, types.ConfigOverride) (any, bool, error) {
	return string(r), true, nil
}

// TestModuleConfigNewEngineIsolation 验证公开构造路径加载同一 DSL 时保持配置和运行态隔离。
func TestModuleConfigNewEngineIsolation(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"code/dsl/shared/resources.json": `{"ruleChain":{"id":"resources"},"metadata":{"nodes":[{"id":"resource","type":"test/config-aware"}]}}`,
		"code/dsl/rulechains/flow.json":  `{"ruleChain":{"id":"flow"},"metadata":{"nodes":[{"id":"local-node","type":"test/config-aware"}]}}`,
	}
	for name, content := range files {
		file := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	prototype := &configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{})}
	if err := registry.Default.NodeManager.Register(prototype); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = registry.Default.NodeManager.Unregister(prototype.Type()) })
	cfg := config.MatrixConfig{EnabledComponents: []string{"code"}, Loader: config.LoaderConfig{Providers: []config.LoaderProviderConfig{{Type: "file", Args: []string{dir}}}}}
	var engines []*MatrixEngine
	for _, value := range []fixtureReader{"first", "second"} {
		e, err := New(cfg, WithModuleConfig("sample", value), WithDefaultConfigModule("sample"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			for _, id := range e.RuntimePool().ListIDs() {
				rt, _ := e.RuntimePool().Get(id)
				rt.Destroy()
			}
			e.SharedNodePool().Stop()
		})
		rt, ok := e.RuntimePool().Get("flow")
		if !ok {
			t.Fatal("runtime was not loaded")
		}
		valueRead, err := e.SharedNodePool().GetInstance("resource")
		if err != nil || valueRead != string(value) {
			t.Fatalf("shared got %v, %v", valueRead, err)
		}
		node, _ := rt.GetChainInstance().GetNode("local-node")
		if node.(*configFixtureNode).value != string(value) {
			t.Fatal("runtime config crossed engine boundary")
		}
		engines = append(engines, e)
	}
	if engines[0].SharedNodePool() == engines[1].SharedNodePool() {
		t.Fatal("engines shared resource pool")
	}
	if _, found := registry.Default.SharedNodePool.Get("resource"); found {
		t.Fatal("module resource leaked into default global pool")
	}
}

// TestModuleConfigActiveEndpointUsesPrivateSharedPool 覆盖配置实例启用后 active endpoint 的资源查找。
func TestModuleConfigActiveEndpointUsesPrivateSharedPool(t *testing.T) {
	cfg := lifecycleConfig(t, map[string]string{
		"shared/event_redis.json": `{"ruleChain":{"id":"resources"},"metadata":{"nodes":[{"id":"event-redis","type":"external/redisClient","configuration":{"uri":"redis://127.0.0.1:1/0","dialTimeout":"10ms"}}]}}`,
		"rulechains/flow.json":    `{"ruleChain":{"id":"event-flow"},"metadata":{"nodes":[]}}`,
		"endpoints/events.json":   `{"id":"event-endpoint","type":"endpoint/redis_stream","configuration":{"redisClient":"ref://event-redis","stream":"events","group":"consumers","ruleChainId":"event-flow","blockMs":10}}`,
	})
	e, err := New(cfg, WithModuleConfig("sample", fixtureReader("value")), WithDefaultConfigModule("sample"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = e.StopActiveEndpoints()
		for _, id := range e.RuntimePool().ListIDs() {
			rt, ok := e.RuntimePool().Get(id)
			if ok {
				rt.Destroy()
			}
		}
		e.SharedNodePool().Stop()
	})
	if _, ok := e.SharedNodePool().Get("event-redis"); !ok {
		t.Fatal("event redis missing from engine pool")
	}
	if _, ok := registry.Default.SharedNodePool.Get("event-redis"); ok {
		t.Fatal("event redis leaked into global pool")
	}
}

// TestModuleConfigRuntimeInjection 验证非共享规则链节点也在 Init 之前获得 Reader。
func TestModuleConfigRuntimeInjection(t *testing.T) {
	reg := registry.NewRegistry()
	_ = reg.NodeManager.Register(&configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{})})
	e := &MatrixEngine{registry: reg}
	WithModuleConfig("sample", fixtureReader("runtime-value"))(e)
	WithDefaultConfigModule("sample")(e)
	if err := e.initRegistryAndLoadComponents(nil, nil); err != nil {
		t.Fatal(err)
	}
	def := &types.RuleChainDef{}
	def.Metadata.Nodes = []types.NodeDef{{ID: "local-node", Type: "test/config-aware"}}
	rt, err := runtime.NewDefaultRuntime(nil, def, runtime.WithEngine(e))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Destroy)
	node, ok := rt.GetChainInstance().GetNode("local-node")
	if !ok || node.(*configFixtureNode).value != "runtime-value" {
		t.Fatal("runtime node did not use its engine reader")
	}
}

// TestModuleConfigBuilderFailsClosed 验证 builder 不能把配置装配失败转换为 warning 后继续启动。
func TestModuleConfigBuilderFailsClosed(t *testing.T) {
	reg := registry.NewRegistry()
	_ = reg.NodeManager.Register(&configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{})})
	for _, endpoint := range []bool{false, true} {
		dir := t.TempDir()
		content := `{"ruleChain":{"id":"resources"},"metadata":{"nodes":[{"id":"resource","type":"test/config-aware"}]}}`
		if endpoint {
			content = `{"id":"entry","type":"test/config-aware"}`
		}
		if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		provider := loader.NewFileProvider(dir, 50)
		var err error
		if endpoint {
			err = builder.LoadEndpoints(provider, []string{"."}, reg.NodeManager, reg.SharedNodePool, reg.RuntimePool)
		} else {
			err = builder.LoadSharedNodes(provider, []string{"."}, reg.NodeManager, reg.SharedNodePool)
		}
		if !errors.Is(err, types.ErrConfigReaderUnavailable) {
			t.Fatalf("endpoint=%v: expected missing-reader error, got %v", endpoint, err)
		}
	}
}

// TestModuleConfigBindingValidation 验证绑定错误在加载节点前显式失败。
func TestModuleConfigBindingValidation(t *testing.T) {
	for _, options := range [][]Option{
		{WithModuleConfig("", fixtureReader("x"))},
		{WithModuleConfig("sample", nil)},
		{WithModuleConfig("sample", (*fixtureReader)(nil))},
		{WithModuleConfig("sample", fixtureReader("x")), WithModuleConfig("sample", fixtureReader("y"))},
		{WithNodeConfigOwners(map[string]string{"node": "absent"})},
		{WithDefaultConfigModule("absent")},
		{WithNodeConfigOwners(map[string]string{"node": "one"}), WithNodeConfigOwners(map[string]string{"node": "two"})},
	} {
		e := &MatrixEngine{}
		for _, option := range options {
			option(e)
		}
		if err := e.initRegistryAndLoadComponents(nil, nil); err == nil {
			t.Fatal("invalid binding was accepted")
		}
	}
}

// TestModuleConfigBuilderPreservesLegacyFailure 验证旧的未知节点仍维持 warning 容错，不受新依赖约束影响。
func TestModuleConfigBuilderPreservesLegacyFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "legacy.json"), []byte(`{"ruleChain":{"id":"legacy"},"metadata":{"nodes":[{"id":"node","type":"test/not-registered"}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	reg := registry.NewRegistry()
	if err := builder.LoadSharedNodes(loader.NewFileProvider(dir, 50), []string{"."}, reg.NodeManager, reg.SharedNodePool); err != nil {
		t.Fatalf("legacy loading policy changed: %v", err)
	}
}

// TestModuleConfigOwnershipCopied 验证宿主 map 变动不会在启动后重定向读取。
func TestModuleConfigOwnershipCopied(t *testing.T) {
	e := &MatrixEngine{}
	WithModuleConfig("one", fixtureReader("one"))(e)
	WithModuleConfig("two", fixtureReader("two"))(e)
	owners := map[string]string{"node": "one"}
	WithNodeConfigOwners(owners)(e)
	owners["node"] = "two"
	reader, ok := e.ConfigReaderForNode("node")
	if !ok || reader != fixtureReader("one") {
		t.Fatal("caller changed retained owner map")
	}
}

// TestModuleConfigRuleChainOwners 隔离不同规则链中同名节点的配置归属。
func TestModuleConfigRuleChainOwners(t *testing.T) {
	reg := registry.NewRegistry()
	_ = reg.NodeManager.Register(&configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{})})
	e := &MatrixEngine{registry: reg}
	WithModuleConfig("one", fixtureReader("one"))(e)
	WithModuleConfig("two", fixtureReader("two"))(e)
	WithNodeConfigOwners(map[string]string{"same-node": "one"})(e)
	WithRuleChainConfigOwners(map[string]string{"chain-one": "one", "chain-two": "two"})(e)
	if err := e.initRegistryAndLoadComponents(nil, nil); err != nil {
		t.Fatal(err)
	}
	for chain, value := range map[string]string{"chain-one": "one", "chain-two": "two"} {
		def := &types.RuleChainDef{}
		def.RuleChain.ID = chain
		def.Metadata.Nodes = []types.NodeDef{{ID: "same-node", Type: "test/config-aware"}}
		rt, err := runtime.NewDefaultRuntime(nil, def, runtime.WithEngine(e))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(rt.Destroy)
		node, _ := rt.GetChainInstance().GetNode("same-node")
		if node.(*configFixtureNode).value != value {
			t.Fatalf("%s used another chain's reader", chain)
		}
	}
	def := &types.RuleChainDef{}
	def.RuleChain.ID = "unknown"
	def.Metadata.Nodes = []types.NodeDef{{ID: "same-node", Type: "test/config-aware"}}
	if _, err := runtime.NewDefaultRuntime(nil, def, runtime.WithEngine(e)); !errors.Is(err, types.ErrConfigReaderUnavailable) {
		t.Fatalf("runtime must not fall back to flat shared-node ownership: %v", err)
	}
}

// TestModuleConfigRejectsAmbiguousImports 拒绝把被导入模块的配置误绑定为根链模块配置。
func TestModuleConfigRejectsAmbiguousImports(t *testing.T) {
	reg := registry.NewRegistry()
	_ = reg.NodeManager.Register(&configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{})})
	e := &MatrixEngine{registry: reg}
	WithModuleConfig("one", fixtureReader("one"))(e)
	WithModuleConfig("two", fixtureReader("two"))(e)
	WithRuleChainConfigOwners(map[string]string{"root": "one", "same": "one", "different": "two"})(e)
	if err := e.initRegistryAndLoadComponents(nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, imported := range []string{"same", "different", "unknown"} {
		def := &types.RuleChainDef{}
		def.RuleChain.ID = "root"
		def.RuleChain.Attrs.Imports = []string{imported}
		def.Metadata.Nodes = []types.NodeDef{{ID: "node", Type: "test/config-aware"}}
		rt, err := runtime.NewDefaultRuntime(nil, def, runtime.WithEngine(e))
		if imported == "same" {
			if err != nil {
				t.Fatal(err)
			}
			rt.Destroy()
		} else if !errors.Is(err, types.ErrConfigReaderUnavailable) {
			t.Fatalf("%s import must reject uncertain ownership: %v", imported, err)
		}
	}
}

type configFixtureNode struct {
	*types.BaseNode
	id, name  string
	reader    types.ConfigReader
	value     any
	initError error
}

// New 构造不携带配置 Reader 的新节点。
func (n *configFixtureNode) New() types.Node {
	return &configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{}), initError: n.initError}
}

// ID 返回实例标识。
func (n *configFixtureNode) ID() string { return n.id }

// Name 返回实例名称。
func (n *configFixtureNode) Name() string { return n.name }

// SetID 保存 DSL 标识。
func (n *configFixtureNode) SetID(id string) { n.id = id }

// SetName 保存 DSL 名称。
func (n *configFixtureNode) SetName(name string) { n.name = name }

// SetConfigReader 接收 Engine 注入的配置实例。
func (n *configFixtureNode) SetConfigReader(reader types.ConfigReader) { n.reader = reader }

// Init 必须在 Reader 可用后读取配置。
func (n *configFixtureNode) Init(types.ConfigMap) error {
	if n.reader == nil {
		return errors.New("reader was not injected before Init")
	}
	if n.initError != nil {
		return n.initError
	}
	var err error
	n.value, _, err = n.reader.LookupConfig(context.Background(), "VALUE", types.ConfigOverride{})
	return err
}

// GetInstance 暴露初始化结果，供测试确认实例隔离。
func (n *configFixtureNode) GetInstance() (any, error) { return n.value, nil }

// TestModuleConfigSharedInstancesAreIsolated 验证相同节点定义在两个 Engine 中绑定不同 Reader。
func TestModuleConfigSharedInstancesAreIsolated(t *testing.T) {
	prototype := &configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{})}
	if err := registry.Default.NodeManager.Register(prototype); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = registry.Default.NodeManager.Unregister(prototype.Type()) })
	engines := make([]*MatrixEngine, 0, 2)
	for _, value := range []fixtureReader{"first", "second"} {
		e := &MatrixEngine{}
		WithModuleConfig("sample", value)(e)
		WithNodeConfigOwners(map[string]string{"resource": "sample"})(e)
		if err := e.initRegistryAndLoadComponents(nil, nil); err != nil {
			t.Fatal(err)
		}
		ctx, err := e.SharedNodePool().NewFromNodeDef(types.NodeDef{ID: "resource", Type: "test/config-aware"}, e.NodeManager())
		if err != nil {
			t.Fatal(err)
		}
		got, err := ctx.GetInstance()
		if err != nil || got != string(value) {
			t.Fatalf("got %v, %v; want %q", got, err, value)
		}
		engines = append(engines, e)
	}
	if engines[0].RuntimePool() == engines[1].RuntimePool() {
		t.Fatal("runtime pools must be instance-local")
	}
	engines[1].SharedNodePool().Stop()
	if _, ok := engines[0].SharedNodePool().Get("resource"); !ok {
		t.Fatal("stopping second engine removed first engine resource")
	}
}

// TestModuleConfigRequiresExplicitOwner 验证单个 Reader 也不会隐式取得无归属节点。
func TestModuleConfigRequiresExplicitOwner(t *testing.T) {
	e := &MatrixEngine{}
	WithModuleConfig("sample", fixtureReader("value"))(e)
	if _, ok := e.ConfigReaderForNode("unknown"); ok {
		t.Fatal("node owner must be explicit")
	}
	WithDefaultConfigModule("sample")(e)
	if _, ok := e.ConfigReaderForNode("unknown"); !ok {
		t.Fatal("explicit default owner was ignored")
	}
}

// TestModuleConfigMissingReaderIsSafe 验证未装配或带敏感错误的消费者均使用安全错误。
func TestModuleConfigMissingReaderIsSafe(t *testing.T) {
	for _, tc := range []struct {
		name      string
		reader    bool
		initError error
	}{
		{name: "missing"},
		{name: "init error", reader: true, initError: errors.New("mongodb://user:private-password@host")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := registry.NewRegistry()
			_ = reg.NodeManager.Register(&configFixtureNode{BaseNode: types.NewBaseNode("test/config-aware", types.NodeMetadata{}), initError: tc.initError})
			e := &MatrixEngine{registry: reg}
			if tc.reader {
				WithModuleConfig("sample", fixtureReader("value"))(e)
				WithDefaultConfigModule("sample")(e)
			}
			if err := e.initRegistryAndLoadComponents(nil, nil); err != nil {
				t.Fatal(err)
			}
			_, err := e.SharedNodePool().NewFromNodeDef(types.NodeDef{ID: "resource", Type: "test/config-aware"}, e.NodeManager())
			if err == nil {
				t.Fatal("missing/invalid config must fail closed")
			}
			if strings.Contains(err.Error(), "private-password") {
				t.Fatal("configuration error leaked its raw cause")
			}
			if !tc.reader && !errors.Is(err, types.ErrConfigReaderUnavailable) {
				t.Fatalf("wrong missing-reader error: %v", err)
			}
		})
	}
}
