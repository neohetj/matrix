---
uuid: "1f8831e5-2dc6-42a1-a5df-37db4a549efe"
type: "Reference"
title: "参考：模块配置 Reader 与实例注入契约"
status: "Stable"
owner: "neohetj"
version: "1.1.2"
updated_at: "2026-10-12"
tags: ["matrix", "configuration", "runtime"]
relations:
  - type: "is_part_of"
    target_uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
    description: "属于 Matrix 参考文档库。"
---

# 模块配置 Reader 与实例注入契约

本文描述配置分支中已实现的 API；Stable 表示契约已核实，不表示已发布远端版本或完成全部业务模块迁移。

## 配置来源与生命周期

`pkg/config/catalog.Load(ctx, source)` 从显式 CatalogSource 编译定义；FSSource 只读取指定文件系统根目录的 `*_catalog.yaml`。不探测父目录、不发现 workspace、不获取部署凭据。

`NewReader(definition, resolver)` 对声明键及 alias 建立一次来源快照。读取顺序沿用 ConfigResolver：普通占位值为 env → business → Catalog default；节点非 Secret 显式覆盖优先。Secret 拒绝节点覆盖，只允许受支持的环境来源，不接受 business/default。Reader 不在读取时重新访问环境，修改环境或 Catalog 必须新建 Reader。

同一次快照中，已经是大写下划线形式的环境键不会因规范化名称相同而再查询一次；首次缺失也是冻结事实。只有规范化名称不同才继续读取兼容名称，再按既有 alias 顺序解析。回归 `TestSnapshotFreezesAbsentCanonicalEnvOnce` 与 `TestSnapshotKeepsDistinctNormalizedEnvName` 覆盖缺失、alias 和点号名称兼容。

来源中明确提供的空字符串与缺失不同。ConfigResolver 对 env、YAML alias、命名来源和转换结果保留空字符串，不继续读取低优先级 alias、business 或 default；required 字符串不能为空，枚举、格式与空白约束仍由 Catalog 声明校验。可选字符串若允许空值，就保留空值；希望采用默认时必须省略该输入。点号环境键仍保留 ConfigAsset 既有的大写下划线名称兼容。此修复仅作用于 ConfigResolver/Catalog，公共 `config://` Asset 的历史空值回退契约不变。

完整执行校验同时覆盖 Schema 中命中的条件 `required`、`dependentRequired` 与 `dependentSchemas`：仅在当前断言分支要求某字符串时拒绝空值，不将可选字段全局设为非空，也不改写 `if/not` 的条件判断。该约束在编译副本中补齐，原始声明和冻结摘要不变。Reader 的单字段预检继续延后根条件，条件完整性必须由 `Catalog.Resolve/Validate` 执行。

| API | 合同 |
| --- | --- |
| `Read[T]` / `ReadNode[T]` | 返回值、是否存在和错误；未知键、类型错误、必填缺失均为错误。ConfigOverride.Present 和来源快照保留 false/0/空字符串的显式语义；空字符串接受 required 及字段 Schema 校验。 |
| `ReadDuration` | 裸整数必须由调用方指定单位；支持 Go 时长，检查乘法溢出，不根据键名猜单位。 |
| `NewDecoder` | 汇总 DTO 装配错误；调用方在网络、文件和业务副作用之前检查 Err。 |
| `Reader.ValidateProvided` | 使用同一快照检查已提供值和默认值；延后缺失必填及根条件约束。 |
| `Catalog.ValidateProvided` | 草稿门禁，允许尚未完成的空字符串输入；该规则不适用于运行来源或 Reader 预检，也不能替代完整执行校验。 |
| `Catalog.Validate` / `Resolve` | 校验完整类型化配置及根条件规则；执行门禁不能由 ValidateProvided 替代。 |
| `Freeze` / `Restore` | 只保存定义，校验格式和摘要；不保存有效值，也不自动升级旧快照。 |

Reader 返回集合副本，格式化和 JSON 输出隐藏值；Issue 使用键、代码和路径，不回显被拒绝的配置原值。int64 通过保留十进制精度的路径处理，拒绝有损浮点整数和溢出。v1 文档的兼容归一化仅发生在 Decode，Restore 不重写冻结内容。

## Engine 与节点边界

```mermaid
flowchart LR
  C[显式 CatalogSource] --> D[不可变定义]
  S[模块 env 和 business] --> R[来源快照 Reader：保留空串、false、0]
  D --> R
  R --> V[已提供字段及默认值预检]
  V --> O[Engine options 与显式归属]
  O --> N[克隆节点并注入 Reader 和 NodePool]
  N --> I[Init 装配 DTO 并检查完整业务约束]
  I --> E[启动副作用]
```

宿主在 `matrix.New` 前通过 `WithModuleConfig` 注册 Reader，以 `WithNodeConfigOwners` / `WithRuleChainConfigOwners` 声明归属。配置注入依据显式 module ID，不能从全局环境推导模块身份。

实现 `types.ConfigReaderAware` 的节点在 Init 前收到对应 Reader；实现 NodePoolAware 的节点使用 Engine 自有资源池。启用模块配置的 Engine 中，共享资源加载与注入失败会阻止创建，避免启动半初始化实例。没有启用模块配置的旧 Engine 保留兼容路径。

`FromAssetContext` 依据显式模块选择 Reader；`RenderString` 复用配置占位语法，但 env/engine 拼写不改变 Reader 的来源顺序，其他资产 scheme 和节点局部上下文不由此入口处理。

## 可选的宿主激活边界

`WithDeferredActiveEndpoints()` 使 `New` 完成定义、共享资源 Init 和 Runtime 创建后返回，不启动主动端点。该模式即使没有模块 Reader 也使用 Engine 私有共享资源/Runtime 池。默认 `New` 仍自动激活，既有模块独立服务器无需迁移。

```mermaid
flowchart LR
  P[宿主公开配置与业务预检] --> N[New + DeferredActiveEndpoints]
  N --> S[宿主完成模块 setup 与迁移]
  S --> A[StartActiveEndpoints]
  A --> R[接受业务流量]
  A -->|失败| X[取消上下文并 Stop 本轮尝试的端点]
  S -->|失败| C[Close]
  X --> C
  R --> D[宿主排空请求]
  D --> C
```

`StartActiveEndpoints(ctx)` 和 `StopActiveEndpoints()` 串行保护生命周期；重复 Start 不重复激活，部分启动失败立即取消并逆序 Stop 本轮已尝试的端点，未启用的端点不被 Stop。`Close()` 幂等停止端点和 scheduler、释放 Engine 私有资源与 Runtime，并停止 trace maintenance goroutine；不删除历史 trace 文件，不清空调用方/全局资源池。Close 后禁止再次 Start。宿主须在 Close 前完成请求排空；模块私有 setup Runtime 的停止仍由宿主负责。

`CoreObjRegistry()` 与 NodeFuncManager 等 getter 一样返回本 Engine 关联的定义 registry；私有运行池不会复制或修改注册定义。

实现：`matrix.go`、`module_config.go`、`pkg/trace`。回归包括 `TestDeferredActiveEndpointsLifecycle`、`TestDeferredEndpointFailureRollsBackBeforeClose` 和 `TestManagerCloseStopsMaintenanceWithoutDeletingPayload`，仅使用合成节点及临时 trace 数据。

### Engine Trace 快照的完成与保留

Engine 在 `trace.enableAop` 开启时创建实例自己的内存 Trace store，默认保留期为 24 小时。HTTP 入口保留自身请求完成逻辑；宿主通过 `runtimebridge.ExecuteRootRuleChain` 明确声明独立根执行，由该入口统一 defer 完成快照。`ExecuteRuleChain` 保留原语义，不自动结束继承的外层 Trace，不能根据 ID 存在或链名相同推断根所有权。

根入口要求非空独立 Execution ID，根链身份必须等于目标 rulechain；后续 option 不得覆盖已声明 ID/根链，`fill` 不得更换或删除身份。runtime 查找、option 校验或 fill 失败时不制造快照；准备成功即将调用 runtime 时才获取当前 Engine 可选 `TraceManager()` 并注册一次完成动作。业务 `WithEndFunc` 与返回消息/错误原样保留，Trace 关闭或 Engine 没有该可选 API 时正常执行。公开必需接口不变。

```mermaid
flowchart LR
  L[节点 After 日志] --> A[原子创建或取得同一 ExecutionStatus]
  F[ExecuteRootRuleChain 返回时 defer FinalizeSnapshot] --> A
  A --> U[状态锁：追加日志 / 设置 EndTs]
  U --> C[每分钟扫描]
  C -->|EndTs 非零且 LastUpdated 超过 24 小时| D[删除快照]
  C -->|未结束或尚未过期| K[保留]
```

原生 `InMemoryStore` 的首次创建使用 `sync.Map.LoadOrStore`，避免节点日志和完成动作并发时覆盖彼此的状态。迟到日志保留已有 `EndTs` 并刷新 `LastUpdated`；重复完成不重置结束时间。公共 `types.Store` 的接口不变，自定义 store 保留原有 `Get/Set` 合同，不继承原生 store 的原子创建保证。关闭 Engine 只停止维护任务；已经运行的旧二进制不会自动采用修复。

回归：`pkg/trace/lifecycle_test.go` 中的并发日志/完成测试与已完成过期快照回收测试。保留期限制历史时间，不是记录数或内存字节上限；不对长时间未结束的执行实施强制清除。

根调用超时或取消是停止等待，不证明异步任务已经物理停止。保留期内迟到的 After 日志可以追加并刷新 `LastUpdated`，不能清空已有 `EndTs`；已被 TTL 真正删除的 ID 与从未建立的 ID 在现有 Store 中不可区分，本轮不新增 tombstone 或第二套保留策略。根入口、执行前错误、普通子链、options/fill 身份保护及迟到日志回归见 `pkg/runtimebridge/root_rulechain_test.go`；使用说明见 `pkg/runtimebridge/README.md`。

## 权责与证据

Matrix 拥有解析、类型转换、快照及注入机制；模块拥有 Catalog、DTO 映射和业务条件校验；WhiteRoom 生成装配薄层；Morpheus 在创建 Engine 前调用公开模块 provider。Catalog default 是节点配置默认值的唯一声明源，节点 Schema 不应物化另一套默认值。

实现：`pkg/config/catalog/reader.go`、`reader_validation.go`、`model.go`、`source.go`、`schema.go`、`values.go`，以及 Engine 的 module config options。验证：配置包测试及 `module_config_*_test.go`；操作见 [接入指南](../guides/module-configuration-reader-guide.md)。
