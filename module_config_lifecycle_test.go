package matrix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
)

type lifetimeResource struct {
	*types.BaseNode
	types.Instance
	reader    types.ConfigReader
	destroyed map[string]int
}

// New 为生命周期测试创建独立资源，并保留销毁计数器。
func (n *lifetimeResource) New() types.Node {
	return &lifetimeResource{BaseNode: n.BaseNode, destroyed: n.destroyed}
}

// SetConfigReader 记录初始化依赖。
func (n *lifetimeResource) SetConfigReader(reader types.ConfigReader) { n.reader = reader }

// Init 模拟可能在配置解析后失败的资源初始化。
func (n *lifetimeResource) Init(cfg types.ConfigMap) error {
	if n.reader == nil {
		return types.ErrConfigReaderUnavailable
	}
	if cfg["fail"] == true {
		return errors.New("private initialization cause")
	}
	return nil
}

// GetInstance 暴露资源本身，不访问外部系统。
func (n *lifetimeResource) GetInstance() (any, error) { return n, nil }

// Destroy 记录资源是否恰好释放一次。
func (n *lifetimeResource) Destroy() { n.destroyed[n.ID()]++ }

type lifetimeEndpoint struct {
	lifetimeResource
	done    chan struct{}
	stopped *bool
}

type activationLifetimeResource struct {
	*types.BaseNode
	types.Instance
	destroyed map[string]int
}

// New 为无模块配置的激活计划测试创建独立资源。
func (n *activationLifetimeResource) New() types.Node {
	return &activationLifetimeResource{BaseNode: n.BaseNode, destroyed: n.destroyed}
}

// Init 不依赖模块配置，用于覆盖纯激活计划场景。
func (n *activationLifetimeResource) Init(types.ConfigMap) error { return nil }

// GetInstance 暴露资源本身。
func (n *activationLifetimeResource) GetInstance() (any, error) { return n, nil }

// Destroy 记录启动失败时的回滚次数。
func (n *activationLifetimeResource) Destroy() { n.destroyed[n.ID()]++ }

type activationLifetimeEndpoint struct {
	activationLifetimeResource
	fail    bool
	done    chan struct{}
	stopped map[string]int
}

// New 创建纯激活计划场景的主动端点。
func (n *activationLifetimeEndpoint) New() types.Node {
	return &activationLifetimeEndpoint{
		activationLifetimeResource: activationLifetimeResource{BaseNode: n.BaseNode, destroyed: n.destroyed},
		stopped:                    n.stopped,
	}
}

// Init 记录当前端点是否在 Start 阶段失败。
func (n *activationLifetimeEndpoint) Init(cfg types.ConfigMap) error {
	n.fail = cfg["fail"] == true
	return nil
}

// SetRuntimePool 接收端点装配所需的运行时池。
func (n *activationLifetimeEndpoint) SetRuntimePool(any) error { return nil }

// Start 启动可观测的监听生命周期，并可按配置模拟失败。
func (n *activationLifetimeEndpoint) Start(ctx context.Context) error {
	n.done = make(chan struct{})
	go func() { <-ctx.Done(); close(n.done) }()
	if n.fail {
		return errors.New("activation endpoint failed")
	}
	return nil
}

// Stop 等待 Engine 先取消上下文，再记录端点回滚。
func (n *activationLifetimeEndpoint) Stop() error {
	select {
	case <-n.done:
		n.stopped[n.ID()]++
		return nil
	case <-time.After(time.Second):
		return errors.New("activation endpoint context was not cancelled")
	}
}

// New 创建有独立监听生命周期的端点。
func (n *lifetimeEndpoint) New() types.Node {
	return &lifetimeEndpoint{lifetimeResource: lifetimeResource{BaseNode: n.BaseNode, destroyed: n.destroyed}, stopped: n.stopped}
}

// SetRuntimePool 接收正常端点装配，不参与测试行为。
func (n *lifetimeEndpoint) SetRuntimePool(any) error { return nil }

// Start 模拟部分启动后失败，要求 Engine 撤销上下文并调用 Stop。
func (n *lifetimeEndpoint) Start(ctx context.Context) error {
	n.done = make(chan struct{})
	go func() { <-ctx.Done(); close(n.done) }()
	return errors.New("private-start-cause")
}

// Stop 等待已启动监听完成，超时表示 Engine 未先撤销上下文。
func (n *lifetimeEndpoint) Stop() error {
	select {
	case <-n.done:
		*n.stopped = true
		return nil
	case <-time.After(time.Second):
		return errors.New("endpoint context was not cancelled")
	}
}

// lifecycleConfig 写入自包含 DSL fixture，不读取工作区或真实环境配置。
func lifecycleConfig(t *testing.T, files map[string]string) config.MatrixConfig {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		file := filepath.Join(dir, "code/dsl", name)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return config.MatrixConfig{EnabledComponents: []string{"code"}, Loader: config.LoaderConfig{Providers: []config.LoaderProviderConfig{{Type: "file", Args: []string{dir}}}}}
}

// TestModuleConfigStartupFailureCleansResources 验证新配置路径失败后释放已建立的共享和局部资源。
func TestModuleConfigStartupFailureCleansResources(t *testing.T) {
	for _, stage := range []string{"shared", "runtime", "endpoint"} {
		t.Run(stage, func(t *testing.T) {
			counts := map[string]int{}
			stopped := false
			reg := registry.NewRegistry()
			_ = reg.NodeManager.Register(&lifetimeResource{BaseNode: types.NewBaseNode("test/lifetime", types.NodeMetadata{}), destroyed: counts})
			_ = reg.NodeManager.Register(&lifetimeEndpoint{lifetimeResource: lifetimeResource{BaseNode: types.NewBaseNode("test/lifetime-endpoint", types.NodeMetadata{}), destroyed: counts}, stopped: &stopped})
			files := map[string]string{"shared/resources.json": `{"ruleChain":{"id":"resources"},"metadata":{"nodes":[{"id":"resource","type":"test/lifetime"}]}}`}
			if stage == "shared" {
				files["shared/resources.json"] = `{"ruleChain":{"id":"resources"},"metadata":{"nodes":[{"id":"resource","type":"test/lifetime"},{"id":"bad","type":"test/lifetime","configuration":{"fail":true}}]}}`
			} else {
				files["rulechains/flow.json"] = `{"ruleChain":{"id":"flow"},"metadata":{"nodes":[{"id":"local","type":"test/lifetime"}]}}`
				if stage == "runtime" {
					files["rulechains/flow.json"] = `{"ruleChain":{"id":"flow"},"metadata":{"nodes":[{"id":"local","type":"test/lifetime"},{"id":"bad","type":"test/lifetime","configuration":{"fail":true}}]}}`
				}
				if stage == "endpoint" {
					files["endpoints/entry.json"] = `{"id":"entry","type":"test/lifetime-endpoint"}`
					files["rulechains/flow.json"] = `{"ruleChain":{"id":"flow"},"metadata":{"nodes":[{"id":"resource","type":"test/lifetime"},{"id":"local","type":"test/lifetime"}]}}`
				}
			}
			_, err := New(lifecycleConfig(t, files), WithRegistry(reg), WithModuleConfig("sample", fixtureReader("x")), WithDefaultConfigModule("sample"))
			if err == nil {
				t.Fatal("fixture must fail startup")
			}
			if strings.Contains(err.Error(), "private-") {
				t.Fatal("startup error leaked raw configuration cause")
			}
			if counts["resource"] != 1 {
				t.Fatalf("shared resource destroyed %d times", counts["resource"])
			}
			if stage != "shared" && counts["local"] != 1 {
				t.Fatalf("runtime resource destroyed %d times", counts["local"])
			}
			if stage == "endpoint" && (!stopped || counts["entry"] != 1) {
				t.Fatal("partially started endpoint was not stopped and destroyed")
			}
			if stage != "endpoint" && counts["bad"] != 1 {
				t.Fatalf("failed resource destroyed %d times", counts["bad"])
			}
		})
	}
}

// TestActivationPlanStartupFailureCleansResources 验证仅激活计划场景也使用私有池并回滚已初始化资源。
func TestActivationPlanStartupFailureCleansResources(t *testing.T) {
	counts := map[string]int{}
	reg := registry.NewRegistry()
	if err := reg.NodeManager.Register(&activationLifetimeResource{
		BaseNode:  types.NewBaseNode("test/activation-lifetime", types.NodeMetadata{}),
		destroyed: counts,
	}); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"shared/a.json": `{"ruleChain":{"id":"a"},"metadata":{"nodes":[{"id":"created","type":"test/activation-lifetime"}]}}`,
		"shared/b.json": `{"ruleChain":{"id":"b"},"metadata":{"nodes":[{"id":"rejected","type":"test/activation-lifetime"}]}}`,
	}
	_, err := New(
		lifecycleConfig(t, files),
		WithRegistry(reg),
		WithSharedNodeActivationPlan(func(def types.NodeDef) (bool, error) {
			if def.ID == "rejected" {
				return false, errors.New("activation failed")
			}
			return true, nil
		}),
	)
	if err == nil {
		t.Fatal("activation plan failure was accepted")
	}
	if counts["created"] != 1 {
		t.Fatalf("created resource destroyed %d times", counts["created"])
	}
	if _, exists := reg.SharedNodePool.Get("created"); exists {
		t.Fatal("activation-plan resource leaked into caller registry")
	}
}

// TestActivationPlanEndpointFailureStopsStartedEndpoints 验证纯激活计划在主动端点部分启动后仍完整回滚。
func TestActivationPlanEndpointFailureStopsStartedEndpoints(t *testing.T) {
	counts := map[string]int{}
	stopped := map[string]int{}
	reg := registry.NewRegistry()
	if err := reg.NodeManager.Register(&activationLifetimeEndpoint{
		activationLifetimeResource: activationLifetimeResource{
			BaseNode:  types.NewBaseNode("test/activation-endpoint", types.NodeMetadata{}),
			destroyed: counts,
		},
		stopped: stopped,
	}); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"endpoints/a.json": `{"id":"started","type":"test/activation-endpoint"}`,
		"endpoints/b.json": `{"id":"failed","type":"test/activation-endpoint","configuration":{"fail":true}}`,
	}
	_, err := New(
		lifecycleConfig(t, files),
		WithRegistry(reg),
		WithSharedNodeActivationPlan(func(types.NodeDef) (bool, error) { return true, nil }),
	)
	if err == nil {
		t.Fatal("active endpoint startup failure was accepted")
	}
	for _, id := range []string{"started", "failed"} {
		if stopped[id] != 1 {
			t.Fatalf("endpoint %q stopped %d times", id, stopped[id])
		}
		if counts[id] != 1 {
			t.Fatalf("endpoint %q destroyed %d times", id, counts[id])
		}
	}
}
