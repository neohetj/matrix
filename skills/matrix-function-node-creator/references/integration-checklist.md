# Matrix Function Node Integration Checklist

## Implementation Checks

- [ ] 节点文件位于当前项目约定的函数节点目录。
- [ ] 存在 `XxxFuncObj`，并完整声明 `ID/Name/Desc/Dimension/Version/Inputs/Outputs`。
- [ ] 存在 `Xxx(ctx,msg)`，并只做取参、读配置、构造 options、适配 logger、调用 `XxxImpl`、写回输出。
- [ ] 存在 `XxxImpl(context.Context, bizlog.Logger, ...)`。
- [ ] `XxxImpl` 不 import Matrix 包，不依赖 `NodeCtx/RuleMsg/helper.GetParam/helper.SetParam`。
- [ ] `XxxImpl` 的参数只包含领域对象、标量/切片/map、options struct、仓库自有 logger interface。
- [ ] logger 适配发生在 `Xxx`，不是在 `XxxImpl` 内部。
- [ ] 入口函数未混入复杂业务循环和数据加工。
- [ ] 默认未定义 `XxxInputs/XxxOutputs`、`loadXxxInputs/saveXxxOutputs`，除非项目已有明确要求。

## Registration Checks

- [ ] 在项目的函数注册入口调用 `mgr.Register(XxxFuncObj)`。
- [ ] 新增 SID 时，已补充对应的对象注册。
- [ ] 函数 ID 全局唯一，命名遵循项目约定（常见形式为 `<domain>/<action>`）。

## DSL Wiring Checks

- [ ] Rulechain 节点类型为 `functions`。
- [ ] `configuration.functionName` 与 `FuncObject.ID` 完全一致。
- [ ] `inputs` 键名与 `IOObject.ParamName` 一致。
- [ ] `outputs` 键名与 `IOObject.ParamName` 一致。
- [ ] `defineSid` 与 Go 侧类型 SID 一致。

## Reuse Checks

- [ ] `XxxImpl` 可以被非 DSL 场景直接调用，例如 service、orchestrator 或单元测试。
- [ ] `XxxImpl` 没有隐含 Matrix 运行时前提。
- [ ] 如果项目允许空 logger，no-op 或默认 logger 由 adapter 提供，而不是在 `XxxImpl` 中到处做框架分支。

## Test Focus

1. 行为检查：`XxxImpl` 可独立单测。
2. 边界检查：`Xxx` 只做 Matrix adapter 职责。
3. 接线检查：`FuncObject.ID`、注册项、DSL `functionName` 一致。
4. 日志检查：logger 契约来自业务层，框架日志类型没有泄漏进 `XxxImpl`。
