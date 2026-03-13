# Rule Summary

## Current Checks

### `function-not-found`

触发条件：
- 节点类型为 `functions`
- `configuration.functionName` 不在本地已注册函数目录里

为什么危险：
- 这不是运行期业务错误，而是 DSL 本身引用了不存在的函数
- UI 和运行期都会把它视为坏配置

建议修复：
- 修正 `functionName`
- 或确认对应函数是否真的已经注册到 `NodeFuncManager`

### `function-input-not-defined`

触发条件：
- 节点类型为 `functions`
- DSL 节点声明了函数签名里不存在的输入参数

为什么危险：
- 这类输入不会被函数读取
- 常见于旧 DSL 没清干净，或为了“保对象”硬塞一个函数根本不消费的 `task_context`
- UI 会直接显示配置问题

建议修复：
- 删除多余输入
- 如果函数确实应该读取它，改 `NodeFuncObject` 定义而不是只改 DSL

### `function-output-not-defined`

触发条件：
- 节点类型为 `functions`
- DSL 节点声明了函数签名里不存在的输出参数

为什么危险：
- 说明 DSL 和函数元数据已经漂移
- 下游看到的输出绑定并不真实可靠

建议修复：
- 删除多余输出
- 或补齐函数签名中的输出定义

### `function-required-input-missing`

触发条件：
- 节点类型为 `functions`
- 函数签名里的必填输入没有在 DSL 节点上绑定

为什么危险：
- 节点会在运行时直接因为缺参失败
- trace 往往只会看到 `failed to get <param>`，排障成本高

建议修复：
- 按函数签名补齐输入
- 如果参数不该是必填，回到函数定义修改 `Required`

### `function-input-sid-mismatch` / `function-output-sid-mismatch`

触发条件：
- 节点类型为 `functions`
- DSL 里的 `defineSid` 和函数签名里的 `DefineSID` 不一致

为什么危险：
- 字段名可能看起来没问题，但类型系统已经分叉
- 后续 `GetParam` / `SetParam`、UI 展示和运行期对象解析都会出现漂移
- 但函数签名如果显式使用 `Any`、`[]Any`、`MapStringInterface` 作为通用入口，DSL 绑定更具体的业务 SID 属于合法特化，不应误报

建议修复：
- 以函数签名为准修正 DSL `defineSid`
- 不要在 DSL 和 `NodeFuncObject` 各自维护不同类型

### `collection-sid-object-conversion`

触发条件：
- 节点会走 `helper.ProcessInbound` 或 `helper.ProcessOutbound`
- 字段 `type` 为 `"object"`
- 字段源 URI 或目标 URI 带 `sid=[]...`

为什么危险：
- `convertValue(..., object)` 只支持 `map`、`struct`、`*struct`、JSON string
- `[]T` 或 `*[]T` 会在运行期报 `can't convert ... to map[string]any`

建议修复：
- 如果是集合整体透传，删除 `type`
- 如果确实要变换结构，改成 list-aware 处理，不要走 `"object"`

### `typed-whole-object-cross-sid-conversion`

触发条件：
- 节点会走 `helper.ProcessInbound` 或 `helper.ProcessOutbound`
- 字段 `type` 为 `"object"`
- 字段源和目标都是 whole-object `rulemsg://dataT/...`
- 源和目标都是非泛型 typed SID，且 SID 不相同

为什么危险：
- 这会触发“把一个 typed 对象整体 decode 成另一个 typed 对象”的流程
- 一旦目标结构比源结构更窄，例如 `LeadAnalysisPatch_V1` 只接受 patch 字段，就会在运行期报 `invalid keys`
- `*Patch*_V*` 是最高风险场景，因为 patch 天然只应包含少量允许更新的字段

建议修复：
- 不要整对象跨 SID 透传
- 改成显式字段映射，只输出目标 SID 实际声明的字段
- 如果源是 JSON 字符串，保留 `type: "object"` 没问题；这条规则只针对 typed whole-object 到 typed whole-object

### `object-mapper-alias-copy`

触发条件：
- 节点类型为 `transform/object_mapper`
- `mappingDefinition` 只有一个字段
- 该字段只是把 whole-object `rulemsg://dataT/<obj>?sid=X` 映射到另一个 whole-object `rulemsg://dataT/<other>?sid=X`
- 字段未声明 `type`

为什么危险：
- 这种节点通常没有真实转换语义，只是额外制造一个中间 objId
- 会让 trace 更长，也更容易在后续维护时引入无效复制或错误类型声明

建议修复：
- 如果下游不需要独立快照，直接读取原 objId
- 如果确实需要快照，明确实现深拷贝语义，不要依赖别名式搬运

## Runtime Edge Cases

### `MapStringInterface` nested field writes on old runtimes

触发条件：
- trace 报 `assignment to entry in nil map`
- 出错节点通常是 `transform/object_mapper`
- `bindPath` 类似 `rulemsg://dataT/<obj>.<field>?sid=MapStringInterface`

为什么危险：
- 旧版 Matrix 在首次创建 `MapStringInterface` 对象后，若直接写嵌套字段，根 map 可能还是 nil
- 这会在 `rulemsg://dataT` 写入阶段直接 panic，后续错误处理链会被一起带崩

建议修复：
- 优先升级到包含 nil-map 初始化修复的 Matrix 运行时
- 如果短期不能升级，改用函数节点先输出完整 `MapStringInterface` 对象，再做 whole-object 写入

## Intentional Non-Checks

- `action/forEach` 的 `type` 不参与这一套校验
- 纯链内连接边不做 DataT 投影风险检查
- pipeline stage projection 不再通过 DSL 兜底，问题应在框架层排查
- 函数业务配置 `business.*` 目前不做签名一致性校验
