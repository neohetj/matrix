package builder_test

import (
	"embed"
	"testing"

	"github.com/neohetj/matrix/internal/builder"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

// TestNewSchedulerFromConfig 测试从配置创建调度器
// 该测试覆盖了以下场景：
// 1. 成功创建：提供有效的调度器类型和配置
// 2. 未知类型：提供不支持的调度器类型，预期返回错误
func TestNewSchedulerFromConfig(t *testing.T) {
	// 测试点：成功创建一个 "ants" 类型的调度器
	t.Run("success", func(t *testing.T) {
		cfg := config.SchedulerConfig{
			Type:     "ants",
			PoolSize: 100,
		}
		scheduler, err := builder.NewSchedulerFromConfig(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, scheduler)
	})

	// 测试点：尝试创建一个未知类型的调度器，预期失败
	t.Run("unknown type", func(t *testing.T) {
		cfg := config.SchedulerConfig{
			Type: "unknown",
		}
		_, err := builder.NewSchedulerFromConfig(cfg)
		assert.Error(t, err)
	})
}

// TestNewLoaderFromConfig 测试从配置创建资源加载器
// 该测试覆盖了以下场景：
// 1. 文件提供者成功：正确配置本地文件路径
// 2. 嵌入文件提供者成功：正确配置嵌入文件系统
// 3. 文件提供者参数缺失：缺少路径参数
// 4. 嵌入文件提供者参数缺失：缺少 embed.FS 实例
// 5. 未知提供者类型：配置不支持的提供者类型
func TestNewLoaderFromConfig(t *testing.T) {
	// 测试点：成功创建一个基于本地文件的加载器
	t.Run("file provider success", func(t *testing.T) {
		cfg := config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{"/tmp"},
				},
			},
		}
		loader, err := builder.NewLoaderFromConfig(cfg, nil)
		assert.NoError(t, err)
		assert.NotNil(t, loader)
	})

	// 测试点：成功创建一个基于 embed.FS 的加载器
	t.Run("embed provider success", func(t *testing.T) {
		cfg := config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "embed",
				},
			},
		}
		embedFS := embed.FS{}
		loader, err := builder.NewLoaderFromConfig(cfg, nil, embedFS)
		assert.NoError(t, err)
		assert.NotNil(t, loader)
	})

	// 测试点：文件加载器缺少路径参数，预期失败
	t.Run("file provider missing args", func(t *testing.T) {
		cfg := config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
				},
			},
		}
		_, err := builder.NewLoaderFromConfig(cfg, nil)
		assert.Error(t, err)
	})

	// 测试点：嵌入加载器缺少 embed.FS 参数，预期失败
	t.Run("embed provider not enough embed.FS", func(t *testing.T) {
		cfg := config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "embed",
				},
			},
		}
		_, err := builder.NewLoaderFromConfig(cfg, nil)
		assert.Error(t, err)
	})

	// 测试点：尝试创建未知类型的加载器，预期失败
	t.Run("unknown provider type", func(t *testing.T) {
		cfg := config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "unknown",
				},
			},
		}
		_, err := builder.NewLoaderFromConfig(cfg, nil)
		assert.Error(t, err)
	})
}

// TestLoadDefs 测试从资源提供者加载规则链定义
// 该测试覆盖了以下场景：
// 1. 成功加载：正确加载多个 JSON 格式的规则链定义
// 2. 重复 ID：两个定义文件包含相同的规则链 ID，预期失败
// 3. 无效 JSON：包含格式错误的 JSON 文件，应跳过错误文件并加载有效文件
func TestLoadDefs(t *testing.T) {
	// 测试点：成功加载两个有效的规则链定义
	t.Run("success", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"rulechains/chain1.json": {Content: `{"ruleChain":{"id":"chain1"}, "metadata":{}}`},
				"rulechains/chain2.json": {Content: `{"ruleChain":{"id":"chain2"}, "metadata":{}}`},
			},
		}
		defs, err := builder.LoadDefs(provider, []string{"rulechains"})
		assert.NoError(t, err)
		assert.Len(t, defs, 2)
		assert.Contains(t, defs, "chain1")
		assert.Contains(t, defs, "chain2")
	})

	// 测试点：检测到重复的规则链 ID，预期返回错误
	t.Run("duplicate chain id", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"rulechains/chain1.json": {Content: `{"ruleChain":{"id":"chain1"}, "metadata":{}}`},
				"rulechains/chain2.json": {Content: `{"ruleChain":{"id":"chain1"}, "metadata":{}}`},
			},
		}
		_, err := builder.LoadDefs(provider, []string{"rulechains"})
		assert.Error(t, err)
	})

	// 测试点：处理无效的 JSON 文件，应忽略错误并继续加载其他有效文件
	t.Run("invalid json", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"rulechains/chain1.json":  {Content: `{"ruleChain":{"id":"chain1"}, "metadata":{}}`},
				"rulechains/invalid.json": {Content: `invalid json`},
			},
		}
		defs, err := builder.LoadDefs(provider, []string{"rulechains"})
		assert.NoError(t, err)
		assert.Len(t, defs, 1)
		assert.Contains(t, defs, "chain1")
	})
}

// TestLoadEndpoints 测试从资源提供者加载 Endpoint 定义
// 该测试覆盖了以下场景：
// 1. 成功加载：正确加载并注册 Endpoint
// 2. 无效定义：遇到格式错误的定义文件，应忽略
func TestLoadEndpoints(t *testing.T) {
	// 测试点：成功加载一个有效的 Endpoint 定义
	t.Run("success", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"endpoints/ep1.json": {Content: `{"id": "ep1", "type": "endpoint"}`},
			},
		}
		nodeMgr := &utils.MockNodeManager{
			NodePrototypes: map[string]types.Node{
				"endpoint": &utils.MockEndpoint{},
			},
		}
		nodePool := &utils.MockNodePool{Nodes: make(map[string]types.NodeCtx)}
		runtimePool := &utils.MockRuntimePool{}
		err := builder.LoadEndpoints(provider, []string{"endpoints"}, nodeMgr, nodePool, runtimePool)
		assert.NoError(t, err)
		_, ok := nodePool.GetNodeContext("ep1")
		assert.True(t, ok)
	})

	// 测试点：处理无效的 Endpoint 定义文件，应忽略且不报错
	t.Run("invalid node def", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"endpoints/invalid.json": {Content: `invalid json`},
			},
		}
		nodeMgr := &utils.MockNodeManager{}
		nodePool := &utils.MockNodePool{Nodes: make(map[string]types.NodeCtx)}
		runtimePool := &utils.MockRuntimePool{}
		err := builder.LoadEndpoints(provider, []string{"endpoints"}, nodeMgr, nodePool, runtimePool)
		assert.NoError(t, err)
		assert.Len(t, nodePool.Nodes, 0)
	})
}

// TestLoadSharedNodes 测试加载共享节点定义
// 该测试覆盖了以下场景：
// 1. 成功加载：正确加载包含节点的规则链定义，并实例化共享节点
// 2. 无效 JSON：遇到格式错误的文件，应忽略
func TestLoadSharedNodes(t *testing.T) {
	// 测试点：成功从规则链定义中加载并实例化一个共享节点
	t.Run("success", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"shared/nodes.json": {Content: `{"metadata":{"nodes":[{"id":"shared1", "type":"some_node"}]}}`},
			},
		}
		nodeMgr := &utils.MockNodeManager{
			NodePrototypes: map[string]types.Node{
				"some_node": &utils.MockNode{},
			},
		}
		nodePool := &utils.MockNodePool{Nodes: make(map[string]types.NodeCtx)}

		err := builder.LoadSharedNodes(provider, []string{"shared"}, nodeMgr, nodePool)
		assert.NoError(t, err)
		_, ok := nodePool.GetNodeContext("shared1")
		assert.True(t, ok)
	})

	// 测试点：处理无效的共享节点定义文件，应忽略
	t.Run("invalid json", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"shared/invalid.json": {Content: `invalid json`},
			},
		}
		nodeMgr := &utils.MockNodeManager{}
		nodePool := &utils.MockNodePool{Nodes: make(map[string]types.NodeCtx)}
		err := builder.LoadSharedNodes(provider, []string{"shared"}, nodeMgr, nodePool)
		assert.NoError(t, err)
		assert.Len(t, nodePool.Nodes, 0)
	})
}

// TestMerger 测试规则链定义的合并逻辑（继承与覆盖）
// 该测试覆盖了以下场景：
// 1. 简单合并：一个规则链继承另一个，验证节点合并
// 2. 嵌套合并：多层继承（A -> B -> C），验证所有节点合并
// 3. 循环引用：检测规则链之间的循环依赖，预期报错
// 4. 引用不存在：引用的规则链不存在，预期报错
func TestMerger(t *testing.T) {
	// 测试点：简单的单层继承合并
	t.Run("simple merge", func(t *testing.T) {
		defs := builder.DefMap{
			"base": {
				Metadata: types.MetadataData{
					Nodes: []types.NodeDef{{ID: "node1"}},
				},
			},
			"overlay": {
				RuleChain: types.RuleChainData{
					Attrs: types.RuleChainAttrs{Imports: []string{"base"}},
				},
				Metadata: types.MetadataData{
					Nodes: []types.NodeDef{{ID: "node2"}},
				},
			},
		}
		merger := builder.NewMerger(defs)
		merged, err := merger.Merge("overlay")
		assert.NoError(t, err)
		assert.Len(t, merged.Metadata.Nodes, 2)
	})

	// 测试点：多层嵌套继承合并
	t.Run("nested merge", func(t *testing.T) {
		defs := builder.DefMap{
			"base": {
				Metadata: types.MetadataData{
					Nodes: []types.NodeDef{{ID: "node1"}},
				},
			},
			"middle": {
				RuleChain: types.RuleChainData{
					Attrs: types.RuleChainAttrs{Imports: []string{"base"}},
				},
				Metadata: types.MetadataData{
					Nodes: []types.NodeDef{{ID: "node2"}},
				},
			},
			"top": {
				RuleChain: types.RuleChainData{
					Attrs: types.RuleChainAttrs{Imports: []string{"middle"}},
				},
				Metadata: types.MetadataData{
					Nodes: []types.NodeDef{{ID: "node3"}},
				},
			},
		}
		merger := builder.NewMerger(defs)
		merged, err := merger.Merge("top")
		assert.NoError(t, err)
		assert.Len(t, merged.Metadata.Nodes, 3)
	})

	// 测试点：检测循环引用（A 引用 B，B 引用 A），预期失败
	t.Run("circular import", func(t *testing.T) {
		defs := builder.DefMap{
			"a": {
				RuleChain: types.RuleChainData{
					Attrs: types.RuleChainAttrs{Imports: []string{"b"}},
				},
			},
			"b": {
				RuleChain: types.RuleChainData{
					Attrs: types.RuleChainAttrs{Imports: []string{"a"}},
				},
			},
		}
		merger := builder.NewMerger(defs)
		_, err := merger.Merge("a")
		assert.Error(t, err)
	})

	// 测试点：引用不存在的规则链，预期失败
	t.Run("import not found", func(t *testing.T) {
		defs := builder.DefMap{
			"a": {
				RuleChain: types.RuleChainData{
					Attrs: types.RuleChainAttrs{Imports: []string{"nonexistent"}},
				},
			},
		}
		merger := builder.NewMerger(defs)
		_, err := merger.Merge("a")
		assert.Error(t, err)
	})
}

// TestDiscoverComponentPaths 测试组件目录结构发现
// 该测试验证是否能正确识别组件内的特定目录（如 rulechains, endpoints, shared）
func TestDiscoverComponentPaths(t *testing.T) {
	// 测试点：扫描组件目录，验证是否正确返回各类配置文件的路径列表
	t.Run("discover paths", func(t *testing.T) {
		provider := &utils.MockResourceProvider{
			Files: map[string]struct {
				Content string
				IsDir   bool
			}{
				"components/common/rulechains": {IsDir: true},
				"components/comp1/endpoints":   {IsDir: true},
				"components/comp2/shared":      {IsDir: true},
			},
		}

		rulechainPaths, endpointPaths, sharedNodePaths := builder.DiscoverComponentPaths(
			provider,
			"components",
			[]string{"comp1", "comp2"},
		)

		assert.Equal(t, []string{"components/common/rulechains"}, rulechainPaths)
		assert.Equal(t, []string{"components/comp1/endpoints"}, endpointPaths)
		assert.Equal(t, []string{"components/comp2/shared"}, sharedNodePaths)
	})
}
