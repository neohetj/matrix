# Matrix Function Node Layering Pattern

## Purpose

实现 `types.NodeFuncObject` 时，默认采用“函数定义层 + Matrix 适配层 + 纯业务实现层”模式，保证：

- DSL 接线稳定
- 业务逻辑可单测
- 同一份业务实现可以被 Matrix 以外的场景复用

## Layer Contract

1. **函数定义层**
- 放置常量与 `var XxxFuncObj`。
- 只定义元信息、I/O、Business 配置。

2. **Matrix 适配层**
- 使用 `func Xxx(ctx types.NodeCtx, msg types.RuleMsg)`。
- 负责读取 Matrix 输入、配置、资产与运行时上下文。
- 负责把这些输入收敛成业务参数、`Options` struct 和业务 logger。
- 只做 `取参 -> 调 Impl -> 写回 -> TellSuccess`。

3. **纯业务实现层**
- 使用 `func XxxImpl(context.Context, bizlog.Logger, ...)`。
- 不 import Matrix 包。
- 不接触 `NodeCtx`、`RuleMsg`、`asset`、`helper`。
- 只处理业务逻辑与业务错误。

## Recommended Signature Shape

```go
func XxxImpl(
	ctx context.Context,
	logger bizlog.Logger,
	input *domain.Input,
	opts Options,
) (*domain.Output, error)
```

## Logger Boundary

- `bizlog.Logger` 是仓库自有契约，可以是接口或轻量包装。
- `Xxx` 负责把 Matrix logger / node context 适配成 `bizlog.Logger`。
- `XxxImpl` 只依赖这个业务 logger，不依赖 `types.Logger` 或 `types.NodeCtx`。

详细说明见 `references/logger-boundary.md`。

## Registration And DSL Contract

1. 在项目的函数注册入口注册 `XxxFuncObj`。
2. 在 rulechain DSL 中使用 `type: "functions"` 节点。
3. 保持 `configuration.functionName == FuncObject.ID`。
4. 保持 `inputs/outputs` 键名与 `IOObject.ParamName` 一致。
5. 保持 `defineSid` 与 Go 侧 SID 一致。

## Error Handling Contract

1. 入口函数统一使用项目约定的错误出口，例如 `ctx.HandleError(...)`。
2. 参数读取与输出写回错误在 adapter 层处理。
3. `XxxImpl` 只返回业务错误，不直接处理 `RuleMsg`。
