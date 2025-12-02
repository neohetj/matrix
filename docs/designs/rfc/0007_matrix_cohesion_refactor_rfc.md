---
uuid: "9e38d28b-cd7f-49b6-b099-23fdee01329e"
type: "RFC"
title: "需求：重构Matrix框架以提升内聚性"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "rfc"
  - "matrix"
  - "refactor"
  - "cohesion"
---

# RFC: 重构Matrix框架以提升内聚性 (RefactorMatrixForCohesion)

## 1. 阐述重构动机 (Motivation)

当前 `Matrix` 框架的核心初始化逻辑与其“宿主”应用 `Trinity` 存在严重耦合。具体表现在：
- **DSL加载与解析逻辑外置**: `Matrix` 框架的统一入口 `matrix.New()` 期望调用方（`Trinity`）提前加载并解析好所有规则链定义（`DefMap`），这违反了封装原则。`matrix.go` 中的 `TODO` 注释也明确指出了这一问题。
- **通用能力散落**: 大量本应属于框架核心的通用能力，如 DSL 加载器 (`fs`)、引擎适配器 (`adapter`)、追踪 (`trace`) 等，目前都实现在 `trinity/matrixext` 目录下，导致 `Matrix` 框架自身不完整、无法独立演进。
- **应用层结构臃肿**: `Trinity` 项目被迫承担了过多的框架级职责，其 `matrixext` 目录和 `wire` 配置变得异常复杂，模糊了作为“框架使用者”的纯粹定位。

本次重构旨在解决以上痛点，通过将通用能力上移至框架核心，提升 `Matrix` 的内聚性和独立性，并简化 `Trinity` 的实现。

## 2. 提出核心重构方案 (Proposal)

### 2.1. 概括核心思想 (Summary)

本提案建议对 `Matrix` 框架进行一次内聚性重构。核心思想是将 `trinity/matrixext` 目录下的通用模块上移并整合进 `Matrix` 框架。重构后的 `matrix.New()` 将成为真正的统一入口，它将自行负责 DSL 的加载、解析和运行时环境的创建，对调用者屏蔽所有内部复杂性，最终返回一个功能完备、接口稳定的 `MatrixEngine` 实例。

### 2.2. 定义设计与实现要点 (KeyDesignAndImplementationPoints)

- **要点一: 上移DSL加载器**
    - **动作**: 将 `matrix/pkg/loader/` 整个目录移动到 `matrix/pkg/loader/`。
    - **收益**: `Matrix` 获得独立加载 DSL 资源（来自文件系统、embed、http等）的能力。

- **要点二: 内化DSL解析逻辑**
    - **动作**:
        1.  重构 `matrix.New()` 函数，使其接受一个 `loader.ResourceProvider` 和一个配置对象（例如 `LoaderConfig{BasePaths: []string{"..."}}`）作为输入，而不是 `DefMap`。
        2.  将 `trinity/matrixext/provider/rulechain_provider.go` 中遍历、读取、解析 DSL 文件的逻辑，整合进 `matrix.New()` 函数内部。
    - **收益**: `matrix.New()` 实现了 `TODO` 中描述的目标，对调用者隐藏了 DSL 加载和解析的细节。

- **要点三: 抽象并返回标准引擎实例**
    - **动作**:
        1.  在 `matrix` 包中定义一个新的 `MatrixEngine` 结构体，它将作为所有 `Matrix` 核心能力的统一门面，至少包含 `RuntimePool`。
        2.  `matrix.New()` 函数在完成所有初始化后，返回 `*MatrixEngine` 实例。
        3.  移除 `trinity/matrixext/adapter/engine.go`，其聚合管理器的功能由新的 `MatrixEngine` 替代。
    - **收益**: `Trinity` 不再需要手动聚合各种 `Manager`，而是直接与 `Matrix` 框架提供的标准引擎实例交互。

- **要点四: 上移通用适配器**
    - **动作**: 有选择地将 `trinity/matrixext` 下的其他通用模块（如 `trace`, `adapter/endpoint`）的核心逻辑上移到 `matrix/pkg/` 下的新目录（如 `matrix/pkg/adapters/`）。
    - **解耦**: 这些适配器在 `Matrix` 框架中应被设计为**可选模块**。`MatrixEngine` 可以提供 `RegisterTracer(provider)` 或 `RegisterEndpointManager(manager)` 等方法，允许 `Trinity` 在启动时将具体的实现（如 go-zero 的 server）注入进来。
    - **收益**: 增强了 `Matrix` 框架的可扩展性，同时彻底解除了其与 `go-zero` 等具体技术的硬编码依赖。

## 3. 评估备选方案 (Alternatives)

- **方案A**: 保持现状，仅在 `Trinity` 内部进行小范围重构。
  - **未选择原因**: 无法从根本上解决 `Matrix` 框架缺乏独立性和内聚性的核心问题，技术债务会持续累积。

## 4. 评估潜在影响 (ImpactAssessment)

- **正面影响**:
    - `Matrix` 框架将变得高度内聚和独立，可单独进行测试、演进和分发。
    - `Trinity` 应用将得到极大简化，其职责更清晰，`wire` 依赖注入配置将大幅减少。
    - 整个项目的架构分层将更加合理，符合“核心能力层”与“业务实现层”的设计哲学。
- **负面影响**:
    - 这是一项侵入式重构，需要修改 `Trinity` 的 `wire` 配置文件和启动流程以适应新的 `Matrix` API。
    - 需要投入集中的开发时间来确保重构的正确性和稳定性。

<!-- qa_section_start -->
<!-- qa_section_end -->
