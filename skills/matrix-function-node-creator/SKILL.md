---
name: matrix-function-node-creator
description: 实现或改造 Matrix 函数节点（types.NodeFuncObject），采用“函数定义层 + Matrix 适配层 + 纯业务实现层”的模式。用于新增 functionName、重构旧节点、保持注册与 DSL functions 接线一致，并让 `XxxImpl` 能在非 Matrix 场景下独立调用。
---

# Matrix Function Node Creator

## Overview

实现 Matrix 函数节点时，默认采用三层职责：

1. 函数定义层：`var XxxFuncObj = &types.NodeFuncObject{...}`
2. Matrix 适配层：`func Xxx(ctx types.NodeCtx, msg types.RuleMsg)`
3. 纯业务实现层：`func XxxImpl(ctx context.Context, logger bizlog.Logger, ...)`

目标不是把所有逻辑都塞进节点函数，而是让：

- `Xxx` 只负责 Matrix 取参、配置读取、错误出口和输出写回
- `XxxImpl` 只依赖标准库、领域对象、基础类型、options struct 和仓库自有 logger 契约

Skill 触发口令：`$matrix-function-node-creator`

## Mandatory Rules

1. `XxxFuncObj` 只定义元信息、I/O 与 Business 配置，不放业务代码。
2. `Xxx(ctx,msg)` 可以依赖 Matrix 对象与 helper，例如 `types.NodeCtx`、`types.RuleMsg`、`asset`、`helper`。
3. `XxxImpl` 不得依赖 Matrix 包，不得接收 `types.NodeCtx`、`types.RuleMsg`、`asset`、`helper` 相关对象。
4. `XxxImpl` 的输入输出只使用 `context.Context`、领域对象、标量/切片/map、options struct、仓库自有 logger interface。
5. logger 契约必须是仓库拥有的业务接口；`Xxx` 负责把 Matrix logger / node context 适配成这个接口，不把框架 logger 类型泄漏到 `XxxImpl`。
6. 保持 `FuncObject.ID` 与 DSL `configuration.functionName` 完全一致。
7. 默认不定义 `XxxInputs/XxxOutputs`、`loadXxxInputs/saveXxxOutputs`；只有项目已有明确约定时才跟随。

## Workflow

1. 选择最近似函数节点作为基线，优先复用同类参数和输出结构。
2. 先定义边界：哪些参数由 Matrix adapter 负责读取，哪些配置收敛为 `Options` struct，logger 适配点放在哪里。
3. 编写 `XxxFuncObj` 并声明完整 `Inputs/Outputs/Business`。
4. 编写 `Xxx(ctx, msg)`：取参/读配置 -> 构造 `Options` -> 适配 logger -> 调 `XxxImpl` -> 写回输出 -> `TellSuccess`。
5. 编写 `XxxImpl(context.Context, bizlog.Logger, ...)`，保证业务逻辑可被非 Matrix 场景直接调用。
6. 在项目的函数注册入口注册 `XxxFuncObj`。
7. 在 rulechain DSL 中新增或更新 `type: functions` 节点，设置 `functionName`、`inputs`、`outputs`、`defineSid`。
8. 使用 `references/integration-checklist.md` 完成自检。
9. 需要补充日志边界或示例时，读取：
   - `references/logger-boundary.md`
   - `references/generic-adapter-example.md`

## Minimal Template

```go
// bizlog is a repo-owned logging contract, not a Matrix package.
type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}

type Options struct {
	Limit int
}

const (
	idXxx      = "domain/xxx"
	paramInput = "input"
	paramOut   = "output"
	cfgLimit   = "limit"
)

var XxxFuncObj = &types.NodeFuncObject{
	Func: Xxx,
	FuncObject: types.FuncObject{
		ID:   idXxx,
		Name: "Xxx",
		// ...
	},
}

func Xxx(ctx types.NodeCtx, msg types.RuleMsg) {
	assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))

	input, err := helper.GetParam[*domain.Input](assetCtx, paramInput)
	if err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(err))
		return
	}

	limit, _ := helper.GetConfigAsset[int](assetCtx, cfgLimit)
	opts := Options{Limit: limit}
	logger := adaptNodeLogger(ctx) // project-owned adapter to bizlog.Logger

	output, err := XxxImpl(ctx.GetContext(), logger, input, opts)
	if err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(err))
		return
	}

	if _, err := helper.SetParam(assetCtx, paramOut, output); err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(err))
		return
	}

	ctx.TellSuccess(msg)
}

func XxxImpl(
	ctx context.Context,
	logger bizlog.Logger,
	input *domain.Input,
	opts Options,
) (*domain.Output, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	logger.Info("processing input", "limit", opts.Limit)
	return &domain.Output{}, nil
}
```

## Anti-Patterns

```go
func XxxImpl(ctx types.NodeCtx, msg types.RuleMsg, input *domain.Input) (*domain.Output, error) {
	// 违规：把 Matrix 对象直接下沉到业务实现层
	return nil, nil
}
```

还包括以下违规：

1. 在 `XxxImpl` 内直接调用 `helper.GetParam/helper.SetParam`。
2. 让 `XxxImpl` import Matrix 包，导致它无法在 service、orchestrator、测试里复用。
3. 不做 logger 适配，直接把 `NodeCtx` 或框架 logger 类型传进 `XxxImpl`。
4. 在 `Xxx(ctx,msg)` 中混入核心业务循环和复杂数据加工。

## References

- `references/function-layering-pattern.md`
- `references/logger-boundary.md`
- `references/integration-checklist.md`
- `references/generic-adapter-example.md`
