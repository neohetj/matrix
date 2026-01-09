---
# === Node Properties: 定义文档节点自身 ===
uuid: "37175778-9583-4e9e-9400-f3bdfacc9872"
type: "ADR"
title: "设计：Matrix框架重构的抽象层与适配器"
status: "Draft"
owner: "@cline"
version: "1.3.0"
tags:
  - "adr"
  - "matrix"
  - "refactor"
  - "architecture"
  - "api-design"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "realizes"
    target_uuid: "9e38d28b-cd7f-49b6-b099-23fdee01329e" # -> RFC-0007
    description: "本ADR为RFC-0007中提出的重构方案提供了详细的接口和结构设计。"

---

# ADR: 设计：Matrix框架重构的抽象层与适配器 (MatrixRefactorAdaptersAndAbstractions)

## 1. 决策背景 (Context)

本架构决策记录（ADR）旨在为 **[RFC-0007: 重构Matrix框架以提升内聚性](../rfc/0007_matrix_cohesion_refactor_rfc.md)** 提供详细的技术设计。RFC 确立了重构的必要性和高阶目标，本 ADR 则专注于定义重构后具体的**抽象层（核心API）**和**适配层（可插拔组件）**的接口与交互方式，以指导后续的编码实现。

## 2. 核心设计决策 (Decision)

我们将设计一套新的、清晰的 API 和接口，将 `Matrix` 的核心能力与外部依赖（如 `Trinity` 的 Web 框架、日志、追踪系统）彻底解耦。核心思想是将 `MatrixEngine` 从一个简单的状态容器，提升为一个**能力工厂**。

### 2.1. 设计新的框架入口与工厂 (NewEntryPointAndFactory)

`matrix.New` 函数将演变为一个全功能的框架引导程序，并返回一个作为“能力工厂”的 `MatrixEngine` 实例。

<!-- finetune_role: code_generation_example -->
<!-- finetune_instruction: "展示最终版的 MatrixEngine 定义，它通过方法而不是公共字段来暴露核心组件。" -->
```go
// file: matrix/matrix.go

// MatrixEngine 是重构后框架的统一门面。
// 它内部持有对 registry 的引用，并通过方法暴露其能力。
type MatrixEngine struct {
	registry types.Registry // 内部持有 registry 实例
}

// RuntimePool 返回引擎持有的 RuntimePool 实例。
func (e *MatrixEngine) RuntimePool() types.RuntimePool {
	return e.registry.RuntimePool()
}

// SharedNodePool 返回引擎持有的 SharedNodePool 实例。
func (e *MatrixEngine) SharedNodePool() types.NodePool {
	return e.registry.SharedNodePool()
}

// NodeManager 返回引擎持有的 NodeManager 实例。
func (e *MatrixEngine) NodeManager() types.NodeManager {
	return e.registry.NodeManager()
}

// NodeFuncManager 返回引擎持有的 NodeFuncManager 实例。
func (e *MatrixEngine) NodeFuncManager() types.NodeFuncManager {
	return e.registry.NodeFuncManager()
}

// New 是重构后的唯一框架入口。
func New(cfg Config, dslLoader types.ResourceProvider) (*MatrixEngine, error) {
    // ... (加载、解析、创建 runtime 并注册到 registry.Default.RuntimePool) ...

    // 初始化 MatrixEngine 实例，并注入 registry。
    // 当前版本，我们注入全局单例 registry.Default。
    engine := &MatrixEngine{
        registry: registry.Default,
    }

    return engine, nil
}
```

### 2.2. 将 Loader 类型定义移入 `pkg/types` (LoaderTypesInCore)

为了将所有核心类型定义内聚管理，`ResourceProvider` 接口及其相关的 `Resource` 结构体将被定义在 `pkg/types` 包中。

<!-- finetune_role: code_generation_example -->
<!-- finetune_instruction: "展示 ResourceProvider 接口在 types 包中的最终定义。" -->
```go
// file: matrix/pkg/types/loader.go (新文件)

package types

import "io/fs"

// Resource ...
type Resource struct {
    Content []byte
    Source  string
}

// ResourceProvider ...
type ResourceProvider interface {
	fs.ReadDirFS
	fs.StatFS
	WalkDir(root string, fn fs.WalkDirFunc) error
	ReadFile(name string) (*Resource, error)
}

// file: matrix/pkg/loader/file_provider.go

// NewFileProvider 创建一个基于物理文件系统的加载器。
func NewFileProvider(root string) ResourceProvider {
    // ... 实现 ...
}

// file: matrix/pkg/loader/embed_provider.go

// NewEmbedProvider 创建一个基于嵌入式文件系统的加载器。
func NewEmbedProvider(embedFS fs.FS) ResourceProvider {
    // ... 实现 ...
}
```

#### 关于 `embed.FS` 的安全实践 (EmbedFSSecurityPractices)

为了避免将不必要的 Go 源码文件嵌入到最终的二进制文件中，我们将在 `Trinity` 应用中采取特定实践：

```go
// file: trinity/internal/dsl/embed.go

package dsl

import "embed"

//go:embed all:rulechains
var RulechainsFS embed.FS

//go:embed all:components
var ComponentsFS embed.FS

// 通过在专门的包（如 `trinity/internal/dsl`）中定义 `embed.FS`，
// 并使用 `all:` 指令精确指定要嵌入的目录（如 `rulechains` 和 `components`），
// 我们可以确保只有这些目录下的静态资源（*.json）被嵌入。
// 在构建时，Trinity 会将这些 `embed.FS` 实例传递给 Matrix 的 `EmbedProvider`。
```

### 2.3. 定义适配器接口与注入机制 (AdapterInterfaces)

*(本节设计保持不变，定义了日志、追踪和端点适配器的接口。)*

#### 2.3.1. 日志与追踪适配器 (LoggingAndTracingAdapters)
#### 2.3.2. 端点适配器 (EndpointAdapters)

## 3. 决策结果 (Consequences)

- **优点**:
    - **清晰的边界**: `Matrix` 的核心 API 变得非常清晰和稳定。
    - **工厂模式**: `MatrixEngine` 提供了灵活的能力获取方式，为未来扩展（如多租户）奠定了基础。
    - **终极解耦**: `Trinity` 应用的 `import` 列表中将不再出现 `github.com/neohetj/matrix/pkg/registry` 或其他内部实现包。

- **缺点**:
    - `Trinity` 的启动逻辑需要进行较大规模的重构。

## 4. 未来方向 (Future Work)

### 4.1. 支持可注入的注册中心 (InjectableRegistry)

当前设计为了简化迁移，`matrix.New` 内部仍然依赖 `registry.Default`。未来可以移除此硬编码依赖，允许调用者通过 `matrix.Config` 将一个 `registry` 实例注入进来，从而支持多个独立的 `Matrix` 实例。

<!-- qa_section_start -->
<!-- qa_section_end -->
