---
uuid: "6b8a1bd9-bb96-4690-a7f7-6a2ed6788e33"
type: "Plan"
title: "计划：拓扑驱动的多视图部署与自动分发平台实现"
status: "Draft"
owner: "neohetj"
version: "1.0.0"
tags:
  - "plan"
  - "implementation"
  - "ops"
  - "deployment"
  - "topology"
  - "morpheus"

relations:
  - type: "is_plan_for"
    target_uuid: "f1d4c850-1f6e-4dcb-8d51-1fe6a46701ae"
    description: "本实现计划将 RFC-0012 转化为可分阶段交付的工程蓝图。"
  - type: "is_guided_by"
    target_uuid: "fb5bdc48-90b9-49d7-8b66-8676a72fbfe3"
    description: "本计划遵循 Matrix 规则链执行保持 DAG 的核心约束。"
  - type: "is_guided_by"
    target_uuid: "becf62ab-9a79-483a-96d9-243928855ff9"
    description: "本计划复用运维实体建模与 DSL 扩展的已有决策。"
---

# Plan: 计划：拓扑驱动的多视图部署与自动分发平台实现 (TopologyDrivenDeploymentPlatformImplPlan)

## 1. 计划概述 (Overview)

本计划旨在把 RFC-0012 中定义的“拓扑驱动的多视图部署与自动分发平台”拆解成可实施的工程工作流。最终交付目标是：

1. 在 `Matrix` 中提供可复用的部署拓扑源模型与 `ops/*` 节点。
2. 在一个新的部署模块中实现基于 `Matrix` 的部署 Controller、部署 DSL 与 bundle 生成。
3. 提供独立的 `deploy-runner`，以 `docker-compose` 为首个执行器。
4. 在 `Morpheus` 中提供多视角拓扑/部署可视化与运维入口。

## 2. 核心约束 (Constraints)

### 2.1. 源拓扑允许有环，执行流必须保持 DAG

必须严格区分：

1. **源拓扑 / 运行依赖**
   - 可以有环。
   - 使用 `relations` 建模。
2. **规则链执行 / 部署计划**
   - 必须是 DAG。
   - 使用 `connections` 建模或由 Controller 生成执行计划。

这意味着平台不能尝试让 `Matrix` 规则链直接执行一个有环拓扑，而必须先将拓扑依赖压缩、分组并收敛成部署 DAG。

### 2.2. Matrix core 只承载通用能力

`Matrix` 仓库只负责：

1. 通用 `ops/*` 节点。
2. 拓扑 DSL、视图语义、解析与元数据。
3. 供上层平台消费的通用接口。

与业务部署强绑定的内容，如部署控制器、bundle 生成、Runner 协议，不应直接写入 `Matrix` core。

### 2.3. 非 Matrix 组件必须是一等公民

数据库、消息队列、普通 Compose 服务必须与 `Matrix` 服务一样可建模、可展示、可部署。不能把它们视为“附属注释”。

### 2.4. 部署执行必须是 typed deployment

禁止设计成“Controller 下发任意 shell 命令，Runner 直接执行”的模式。Controller 和 Runner 之间必须传输 typed bundle 和 typed job spec。

## 3. 代码归属与目录规划 (Code Placement)

### 3.1. Matrix

建议改动目录：

1. `Matrix/pkg/components/ops`
2. `Matrix/pkg/ops/model`
3. `Matrix/internal/parser`
4. `Matrix/pkg/types`
5. `Matrix/test` 或 `Matrix/pkg/.../*_test.go`

职责：

1. `ops/*` 节点实现。
2. 部署拓扑共享配置结构。
3. `imports / relations / viewType` 的解析与加载补齐。
4. 为图构建和上层控制器提供稳定的结构化模型。

### 3.2. 新平台模块：`deploy`

建议新增顶层工作区模块：`deploy`

参考 `sellitx` 当前按 `common / code / web / cmd / release / scripts` 分层的结构，建议 `deploy` 模块采用下面这种通用组织方式：

```text
deploy/
  init.go
  common/
    server/
    executors/
  code/
    defaults/
    dsl/
      topology/
      rulechains/
      endpoints/
      shared/
      prompts/
    executors/
      coreobjs/
      funcs/
    http/
    model/
    orchestrator/
  web/
    src/
  cmd/server/
  release/
  scripts/
```

职责：

1. 部署 Controller 的 DSL 和函数节点。
2. 部署计划、bundle、Runner Job 的 CoreObj 与业务函数。
3. 部署历史与状态查询接口。

### 3.3. deploy-runner

建议新增顶层工作区模块：`deploy-runner`

目录结构建议保持执行面导向，重点围绕 job 生命周期、执行器与工作目录管理组织：

```text
deploy-runner/
  cmd/runner/
  internal/app/
  internal/model/
  internal/executor/compose/
  internal/reporting/
  internal/storage/
```

职责：

1. 接收 bundle / job。
2. 渲染 release 工作目录。
3. 调用 `docker compose` 部署。
4. 回传状态、日志与健康检查结果。

### 3.4. Morpheus

建议改动目录：

1. `Morpheus/pkg/graph`
2. `Morpheus/pkg/handlers`
3. `Morpheus/web/src/api`
4. `Morpheus/web/src/views`
5. `Morpheus/web/src/components/rulechain/editor`

职责：

1. 多视角图构建 API。
2. 拓扑探索与部署计划可视化。
3. Runner 状态、部署历史、节点详情。

## 4. 增量能力清单 (Scope Breakdown)

### 4.1. Matrix core 需要新增/补齐

1. `ops/application`
2. `ops/service`
3. `ops/database`
4. `ops/message_queue`
5. `ops/network`
6. `ops/volume`
7. `ops/runner`
8. `ops/machine`
9. 统一部署配置结构
10. 源拓扑图加载与关系查询能力

### 4.2. Matrix core 已有能力，可直接复用

1. `NodeMetadata.Icon` 已存在，无需再次设计。
2. `RuleChainAttrs.ViewType`、`Imports` 与 `Metadata.Relations` 结构已在类型层存在。
3. 规则链 DAG 执行模型已经存在，应继续保持不变。

### 4.3. 需要新开发的平台能力

1. 部署 Controller
2. Bundle Generator
3. Runner 协议与 Runner 执行器
4. 部署状态存储与审计

## 5. 核心数据结构计划 (Core Structures)

### 5.1. Matrix 侧共享拓扑配置

建议新增 `Matrix/pkg/ops/model/types.go`，承载所有 `ops/*` 节点共享配置结构。

```go
package model

type RuntimeProfile string

const (
    RuntimeProfileMatrixService   RuntimeProfile = "matrix-service"
    RuntimeProfileComposeService  RuntimeProfile = "compose-service"
    RuntimeProfileComposeDatabase RuntimeProfile = "compose-database"
    RuntimeProfileComposeMQ       RuntimeProfile = "compose-mq"
    RuntimeProfileExternalManaged RuntimeProfile = "external-managed"
)

type DeployAdapter string

const (
    DeployAdapterDockerCompose DeployAdapter = "docker-compose"
    DeployAdapterNone          DeployAdapter = "none"
)

type DeployableSpec struct {
    Deployable   bool          `json:"deployable"`
    RuntimeProfile RuntimeProfile `json:"runtimeProfile"`
    DeployAdapter  DeployAdapter  `json:"deployAdapter"`
    RunnerRef      string         `json:"runnerRef,omitempty"`
    ArtifactRef    string         `json:"artifactRef,omitempty"`
    NetworkRefs    []string       `json:"networkRefs,omitempty"`
    VolumeRefs     []string       `json:"volumeRefs,omitempty"`
    EnvRefs        []string       `json:"envRefs,omitempty"`
    SecretRefs     []string       `json:"secretRefs,omitempty"`
}

type ServiceImplementationSpec struct {
    RuleChainRefs []string `json:"ruleChainRefs,omitempty"`
    EndpointRefs  []string `json:"endpointRefs,omitempty"`
}
```

目标：

1. 避免 8 个节点各自维护一套相似字段。
2. 便于上层 Controller 统一读取。
3. 便于 Morpheus 统一渲染部署属性。

### 5.2. Deploy 模块 CoreObj

建议在 `deploy/code/executors/coreobjs` 新增：

1. `DeploymentRequest`
2. `DeploymentSelection`
3. `TopologySnapshot`
4. `DeploymentPlan`
5. `DeploymentWave`
6. `RunnerBundle`
7. `RunnerJob`
8. `RunnerResult`
9. `DeploymentRecord`

建议最小化先落以下对象：

```go
type DeploymentRequest struct {
    TopologyID       string   `json:"topologyId"`
    Env              string   `json:"env"`
    TargetNodeIDs    []string `json:"targetNodeIds"`
    IncludeInfra     bool     `json:"includeInfra"`
    VersionOverrides map[string]string `json:"versionOverrides,omitempty"`
}

type DeploymentWave struct {
    ID       string   `json:"id"`
    Phase    string   `json:"phase"`
    NodeIDs  []string `json:"nodeIds"`
    RunnerID string   `json:"runnerId"`
}

type DeploymentPlan struct {
    DeploymentID string           `json:"deploymentId"`
    Waves        []DeploymentWave `json:"waves"`
}

type RunnerBundle struct {
    BundleID      string `json:"bundleId"`
    DeploymentID  string `json:"deploymentId"`
    RunnerID      string `json:"runnerId"`
    WorkDir       string `json:"workDir,omitempty"`
    ComposePath   string `json:"composePath"`
    ManifestPath  string `json:"manifestPath"`
}
```

## 6. Workstream A: Matrix Core 改造

### A1. 落 `ops/*` 节点

建议目录：

```text
Matrix/pkg/components/ops/
  init.go
  common.go
  application_node.go
  service_node.go
  database_node.go
  message_queue_node.go
  network_node.go
  volume_node.go
  runner_node.go
  machine_node.go
```

执行要求：

1. 每个节点都实现 `types.Node`。
2. 节点保持轻逻辑，`OnMsg` 默认 no-op 或仅处理少量探测动作。
3. 元数据里补齐 `Icon`、`Description`、`Tags`。

### A2. 提炼共享配置结构

目标目录：

```text
Matrix/pkg/ops/model/
  types.go
  relation_labels.go
```

执行要求：

1. 将部署相关结构与节点配置分离。
2. 统一定义支持的 `RuntimeProfile`、`DeployAdapter`、关系 label 常量。
3. 为 Morpheus 和 Deploy 模块提供稳定的导入点。

### A3. 核查并补齐 parser / loader

虽然 `types.RuleChainAttrs`、`Metadata.Relations` 已存在，但仍需核实：

1. `internal/parser` 是否完整解析 `relations` 与 `imports`
2. `internal/loader` 是否支持按 `imports` 递归加载
3. 运行时对 `viewType=static-topology` 的实例化是否会产生副作用

验收要求：

1. `executable=false` 的拓扑文件可被加载。
2. `imports` 拓扑文件可被可执行工作流复用。
3. 对未执行拓扑不创建无意义运行时副作用。

### A4. 增加拓扑查询辅助 API

建议新增一个轻量查询包：

```text
Matrix/pkg/ops/query/
  topology.go
  selection.go
  projection.go
```

功能：

1. 通过 `RuleChainDef` 提取拓扑节点。
2. 按类型过滤节点。
3. 按 `relations` 查询一跳/多跳邻居。
4. 导出供 Controller 使用的标准拓扑快照。

## 7. Workstream B: 新增 Deploy 模块

### B1. 搭建 Deploy 模块骨架

目录目标：

```text
deploy/
  init.go
  common/
  code/
  web/
  cmd/server/
```

建议采用与 `sellitx` 同类的分层方式：

1. `common` 放共享 server / executor / 公共能力。
2. `code` 放 DSL、执行器、orchestrator、HTTP 适配与模型。
3. `web` 放平台前端。
4. `cmd/server` 放独立启动入口。

### B2. 实现 Controller 函数节点

建议函数节点列表：

1. `deploy/load_topology`
2. `deploy/validate_topology`
3. `deploy/select_targets`
4. `deploy/build_runtime_dependency_graph`
5. `deploy/build_deployment_plan`
6. `deploy/partition_by_runner`
7. `deploy/generate_runner_bundle`
8. `deploy/dispatch_bundle`
9. `deploy/poll_runner_status`
10. `deploy/finalize_record`

实现方式：

1. 先按 `matrix-function-node-creator` 模式落三层结构。
2. 每个函数保持“纯业务实现层 + Matrix 适配层”。
3. 便于后续在非 Matrix 场景复用。

### B3. 设计 Controller 规则链

建议最小链路：

1. `rc-create-deployment`
2. `rc-build-plan`
3. `rc-dispatch-runner-bundles`
4. `rc-reconcile-runner-results`

其中：

* `rc-build-plan` 负责把可有环拓扑收敛成 DAG。
* `rc-dispatch-runner-bundles` 负责按 Runner 并发下发。

### B4. 收敛算法实现

部署计划算法拆成三步：

1. 从拓扑 `relations` 中提取部署相关边。
2. 对依赖图做强连通分量压缩，得到 SCC group。
3. 叠加部署相位约束，生成最终 waves。

相位建议：

1. `prepare`
2. `infra`
3. `stateful`
4. `stateless`
5. `verify`

### B5. 生成 Bundle

输出目录规范：

```text
bundle/
  manifest.json
  docker-compose.yml
  env/
  secrets/
  matrix/
```

具体规则：

1. `matrix-service` 额外写 `matrix/<service>/dsl/**`
2. `compose-service` 只生成 Compose 所需配置
3. `external-managed` 不生成部署资源，只出现在 manifest 中供校验和可视化

### B6. 状态存储

建议最小落库对象：

1. Deployment
2. DeploymentWave
3. RunnerDispatch
4. RunnerExecutionLog

存储层可先复用现有 Mongo / JsonDB 模式，但生产建议优先 Mongo。

## 8. Workstream C: deploy-runner

### C1. Runner 协议

建议最小 HTTP 协议：

1. `POST /api/runner/jobs`
2. `GET /api/runner/jobs/:id`
3. `POST /api/runner/jobs/:id/cancel`
4. `GET /healthz`

`POST /api/runner/jobs` 请求体最小包含：

```json
{
  "jobId": "job-001",
  "bundleUrl": "https://...",
  "bundleChecksum": "sha256:...",
  "deploymentId": "dep-001",
  "runnerId": "runner/prod/sh-01"
}
```

### C2. Runner 内部模块

建议结构：

1. `app`: 生命周期与 worker 管理
2. `model`: job / result / log event
3. `executor/compose`: compose 适配器
4. `reporting`: 回传控制面
5. `storage`: release 工作目录与运行态缓存

### C3. Compose 执行器

最小命令序列：

1. `docker compose pull`
2. `docker compose up -d --remove-orphans`
3. `docker compose ps`
4. 可选：`docker compose logs`

约束：

1. release 目录要按 `deploymentId/bundleId` 隔离。
2. secret 文件必须以最小权限写入。
3. 需要留出 rollback hook，但首期可以只记录 rollback metadata。

### C4. 健康检查

支持三种最小检查模式：

1. `http`
2. `tcp`
3. `container-status`

先由 Controller 写入 bundle manifest，Runner 只执行。

## 9. Workstream D: Morpheus 集成

### D1. 后端图查询扩展

建议新增一个部署拓扑 Graph API 组，而不是强行复用当前仅面向 execution/data 的接口语义。

建议接口：

1. `GET /graph/topology?id=...`
2. `GET /graph/runtime-dependency?id=...`
3. `GET /graph/deployment-target?id=...`
4. `GET /graph/implementation?id=...`
5. `GET /graph/deployment-plan?id=...&deploymentId=...`

### D2. 图构建逻辑

建议在 `Morpheus/pkg/graph` 下新增独立子域：

```text
Morpheus/pkg/graph/ops/
  topology.go
  runtime_dependency.go
  deployment_target.go
  implementation.go
  deployment_plan.go
```

目标：

1. 避免污染现有 rulechain execution/data graph 构建逻辑。
2. 允许新视图采用不同的边过滤与布局策略。

### D3. 前端视图

建议新增部署拓扑页面，而不是直接塞进当前 RuleChain Manager。

候选页面：

1. `TopologyExplorerView.vue`
2. `DeploymentPlanView.vue`
3. `RunnerDashboardView.vue`
4. `DeploymentHistoryView.vue`

### D4. 布局策略

1. `Topology` / `Runtime` 视图允许有环，布局不要复用纯 DAG longest-path 规则。
2. `Deployment Plan` 视图必须走 DAG 布局。
3. `network`、`volume` 默认折叠，不在主画布铺满。

### D5. 详情面板

点击 `ops/service` 时，详情面板至少展示：

1. 基础部署属性
2. `runnerRef`
3. `runtimeProfile`
4. `endpointRefs`
5. `ruleChainRefs`
6. 最近部署状态

## 10. Workstream E: 视图与执行的对齐规则

必须明确以下边界，避免后续语义漂移：

### E1. 源拓扑

* 输入：`ops/*` 节点 + `relations`
* 可有环
* 是单一事实来源

### E2. 运行依赖图

* 从源拓扑过滤而来
* 可有环
* 仅用于可视化与影响分析

### E3. 部署计划图

* 从源拓扑推导而来
* 必须是 DAG
* 仅由 Controller 生成，不手工维护

### E4. Runner 执行图

* 是部署计划图按 Runner 分区后的结果
* 由 Controller 按 wave + runner 生成

## 11. 里程碑与顺序 (Milestones)

### M1. Matrix Topology Foundation

交付：

1. `ops/*` 节点
2. 共享部署配置结构
3. 拓扑 DSL 可加载

完成标准：

1. 能定义一个 `executable=false` 的部署拓扑 DSL
2. 能通过代码读取其节点和 `relations`

### M2. Deploy Controller MVP

交付：

1. `deploy` 模块骨架
2. `load_topology -> build_plan -> generate_bundle` 链路
3. 部署记录最小持久化

完成标准：

1. 能根据单个应用拓扑生成 bundle
2. 能输出有相位顺序的部署计划

### M3. Runner MVP

交付：

1. `deploy-runner` HTTP 服务
2. Compose 执行器
3. 日志与状态回传

完成标准：

1. 能接 bundle 并执行一个 Compose 项目
2. 能回传成功/失败状态

### M4. Morpheus Visualization MVP

交付：

1. `Topology` 视图
2. `Deployment Target` 视图
3. Runner 状态页

完成标准：

1. 可查看部署架构
2. 可定位组件部署在哪个 Runner / Machine

### M5. Matrix Auto-Distribution

交付：

1. `matrix-service` 的 DSL 工件导出
2. `endpointRefs / ruleChainRefs` 自动打包
3. `Implementation` 与 `Deployment Plan` 视图

完成标准：

1. 修改 `ops/service` 关联配置后，可自动产出对应 Matrix 服务 bundle

## 12. 测试与验收 (Validation)

### 12.1. Matrix

1. `go test ./...` 覆盖新增 `pkg/components/ops`、`internal/parser`
2. 新增拓扑 DSL 测试样例
3. 验证 `imports` + `relations` 的解析与加载

### 12.2. Deploy 模块

1. Controller 纯业务层单测
2. 拓扑到部署计划的算法测试
3. Bundle 目录快照测试

### 12.3. deploy-runner

1. Runner API 单测
2. Compose 命令封装测试
3. Bundle 渲染到 release 工作目录的集成测试

### 12.4. Morpheus

1. Graph API 合约测试
2. 前端图渲染与视图切换测试
3. 部署计划视图的 DAG 可视化回归测试

## 13. 风险与缓解 (Risks)

### 13.1. Matrix core 污染风险

风险：

部署业务逻辑过早进入 `Matrix` core。

缓解：

1. `Matrix` 只做通用模型。
2. Controller / bundle / runner 协议放到新 `deploy` 模块。

### 13.2. 视图语义漂移

风险：

前端和控制器对“同一份拓扑”的解释不同。

缓解：

1. 统一使用 `RuntimeProfile`、relation label 常量。
2. 把视图过滤规则固化成后端投影函数与测试样例。

### 13.3. 有环拓扑无法稳定收敛

风险：

某些拓扑关系无法直接生成部署顺序。

缓解：

1. 强连通分量压缩。
2. 额外引入部署相位屏障。
3. 对无法自动推导的边给出显式校验错误。

## 14. 任务拆解 (Task Breakdown)

### T1. Matrix 基础节点

1. 新建 `pkg/components/ops`
2. 实现 8 个节点
3. 编写单元测试

### T2. Matrix 拓扑模型

1. 新建 `pkg/ops/model`
2. 提炼共享部署配置
3. 落 relation label 常量

### T3. Matrix 拓扑加载核查

1. 核查 parser / loader 对 `imports`、`relations` 支持
2. 补缺失测试
3. 固化最小拓扑样例

### T4. Deploy 模块骨架

1. 创建 `deploy/`
2. 搭建 `code/common/web/cmd` 目录
3. 打通 server 启动

### T5. Controller 纯业务层

1. 拓扑读取
2. SCC 压缩
3. phase 分组
4. bundle 生成

### T6. Controller Matrix 适配层

1. 新增 CoreObj
2. 新增函数节点
3. 新增 rulechain / endpoint

### T7. deploy-runner

1. Runner HTTP API
2. Bundle 下载与解包
3. Compose 执行器
4. 结果回传

### T8. Morpheus 视图

1. 图 API
2. 视图切换
3. 详情面板
4. Runner 状态页

## 15. 建议的首轮交付范围 (First Slice)

为降低不确定性，首轮只实现以下最小闭环：

1. 一个应用
2. 一个 `matrix-service`
3. 一个 `compose-database`
4. 一个 `ops/runner`
5. 一个 `ops/machine`
6. 一个 `deploy-runner`
7. 两个 Morpheus 视图：
   - `Topology`
   - `Deployment Target`

首轮完成后，再扩展：

1. `message_queue`
2. `Implementation` 视图
3. 自动打包 `endpointRefs / ruleChainRefs`
4. 完整部署计划视图
