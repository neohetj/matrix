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

## Skill Handoff

- 只做 DSL 设计与编排：先走 `matrix-requirement-to-dsl`（项目内再接 `*-dsl-adapter`）。
- 需要落地/改造函数节点代码：使用本 skill。

## Mandatory Rules

1. `XxxFuncObj` 只定义元信息、I/O 与 Business 配置，不放业务代码。
2. `Xxx(ctx,msg)` 可以依赖 Matrix 对象与 helper，例如 `types.NodeCtx`、`types.RuleMsg`、`asset`、`helper`。
3. `XxxImpl` 不得依赖 Matrix 包，不得接收 `types.NodeCtx`、`types.RuleMsg`、`asset`、`helper` 相关对象。
4. `XxxImpl` 的输入输出只使用 `context.Context`、领域对象、标量/切片/map、options struct、仓库自有 logger interface。
5. logger 契约必须是仓库拥有的业务接口；`Xxx` 负责把 Matrix logger / node context 通过仓库通用 adapter 适配成这个接口，不把框架 logger 类型泄漏到 `XxxImpl`。
6. 保持 `FuncObject.ID` 与 DSL `configuration.functionName` 完全一致。
7. 默认不定义 `XxxInputs/XxxOutputs`、`loadXxxInputs/saveXxxOutputs`；只有项目已有明确约定时才跟随。
8. 读取列表型 DataT 输入时，优先使用 `helper.GetParam[*[]T](...)`，不要默认写成 `helper.GetParam[[]T](...)`；Matrix 运行时常把列表对象物化成 `*[]T`，直接按 `[]T` 读取会在链路里报 `expected []T, got *[]T`。
9. 对于“多节点编排”的节点域，`XxxImpl` 的核心业务实现应直接放在对应 `node_<capability>.go` 中；`support` 只保留跨节点共享能力（client/store/config/toolkit），不要把节点特有流程再次集中到单个 `support/*flow*.go`。
10. `node_<capability>.go` 文件必须同时包含该节点的 `NodeFuncObject` 与主入口实现（`Xxx`/`ExecuteXxx`/`XxxImpl`）；不要把节点主逻辑转调到 `support.ExecuteXxx` 或 `support.SomeStep`。
11. 不包含 `NodeFuncObject` 的文件不得使用 `node_` 前缀；多节点复用 helper 统一命名为 `<domain>_helpers.go` 或 `<capability>_helpers.go`。
12. 外部系统集成能力（HTTP/WS client、第三方 SDK 访问、数据库 CRUD/store）默认放 `infrastructure`；`support` 不承载 transport/persistence 细节。
13. 复杂业务流程必须拆成多个函数节点并通过 DSL rulechain 串联；禁止用“单节点 + 巨型 support flow”替代编排。
14. 测试默认使用外部包（`xxx_test`）；仅在必要时保留极小的测试缝隙，避免长期维护 `testing_exports.go` 这类生产代码污染。
15. `support` 不定义域错误 sentinel（`ErrXxx`）与存储实现（`*_store.go`）；错误定义放 `errors`（必要时 `errdefs`），存储实现放 `infrastructure`。
16. 需要对外暴露函数时，直接把实现函数定义为导出（`PascalCase`）；禁止新增“导出壳函数仅 `return privateFn(...)`”的双函数形态。

## Workflow

1. 选择最近似函数节点作为基线，优先复用同类参数和输出结构。
2. 先定义边界：哪些参数由 Matrix adapter 负责读取，哪些配置收敛为 `Options` struct，logger 适配点放在哪里；如果仓库已有通用 `AdaptNodeLogger/ResolveLogger/StdLogger`，优先直接复用。
3. 编写 `XxxFuncObj` 并声明完整 `Inputs/Outputs/Business`。
4. 编写 `Xxx(ctx, msg)`：取参/读配置 -> 构造 `Options` -> 通过通用 logger adapter 适配 logger -> 调 `XxxImpl` -> 写回输出 -> `TellSuccess`。
   列表输入如果要传给 `XxxImpl`，在 adapter 层先用 `GetParam[*[]T]` 取值并解引用，再把普通 `[]T` 传下去。
5. 编写 `XxxImpl(context.Context, bizlog.Logger, ...)`，保证业务逻辑可被非 Matrix 场景直接调用；该实现与入口函数放在同一个 `node_<capability>.go` 文件。
6. 在项目的函数注册入口注册 `XxxFuncObj`。
7. 在 rulechain DSL 中新增或更新 `type: functions` 节点，设置 `functionName`、`inputs`、`outputs`、`defineSid`。
8. 对多阶段流程先做节点切分（每个阶段一个函数节点），再落 DSL 串联；不要先写 support flow 再“包一层节点壳”。
9. 使用 `references/integration-checklist.md` 完成自检。
10. 完成后执行最小结构自检（可按仓库实际路径调整）：
   - `rg --files | rg '/nodes/node_.*_helpers\\.go$'` 结果应为空。
   - `rg 'return support\\.' <funcs-domain>/nodes` 结果应为空或仅剩跨节点真正共享工具调用。
   - `rg --files <funcs-domain> | rg '_impl\\.go$'` 在本次新增/重构范围内应为空。
   - `rg -n 'Err[A-Z]\\w+\\s*=' <funcs-domain>/support` 结果应为空（引用 `errdefs.ErrXxx` 不算违规）。
   - `rg -n 'model\\.(Create|List|Get|Update|Delete)Model' <funcs-domain>/support` 结果应为空。
   - `rg --files <funcs-domain>/support | rg '_store\\.go$'` 结果应为空。
   - `rg -n '^func [A-Z].*\\{\\s*return [a-z]' <funcs-domain>` 结果应为空（排除确有必要的兼容层并在评审中说明）。
   - `go test ./...` 至少覆盖本次变更所在包与其直接依赖包。
11. 需要补充日志边界或示例时，读取：
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
	logger := bizlog.AdaptNodeLogger(ctx) // repo-owned shared adapter to bizlog.Logger

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
5. 在单个节点包里重复定义私有 `adaptNodeLogger`、`noopLogger`；优先复用仓库级 logger adapter 和默认 logger。
6. 在 `node` 层只写 `return support.SomeStep(...)` 的空转 `Impl`，把节点特有业务继续堆回 `support`。
7. 把 HTTP/WS client 或数据库操作散落在 `support`，导致 `support` 与 `infrastructure` 职责混乱。
8. 用 `node_*.go` 命名纯 helper 文件（文件里没有任何 `NodeFuncObject`）。
9. 复杂链路只实现一个“万能节点”，再在节点内部顺序调用大量步骤函数，绕过 DSL 编排。
10. 同时保留 `foo()` 与 `Foo()`，且 `Foo()` 仅转发到 `foo()`，制造无意义双层函数。

## References

- `references/function-layering-pattern.md`
- `references/logger-boundary.md`
- `references/integration-checklist.md`
- `references/generic-adapter-example.md`
