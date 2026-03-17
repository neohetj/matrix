# Logger Boundary

## Goal

让 `XxxImpl` 保持可复用，不把 Matrix 或项目框架对象直接带进业务实现层。

## Recommended Contract

在业务层定义一个仓库自有 logger interface，例如：

```go
package bizlog

type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}
```

这只是一个示例。你的仓库也可以使用别的命名或方法集合；关键点是：

- 这个接口由业务层拥有
- `XxxImpl` 依赖这个接口，而不是 Matrix 类型

## Adapter Responsibility

`Xxx(ctx,msg)` 负责把 Matrix 运行时里的 logger 或 node context 适配成业务 logger。
如果仓库已经提供通用 logger adapter，优先直接复用，不要在每个节点包里重复定义一套。

示意：

```go
func AdaptNodeLogger(ctx types.NodeCtx) bizlog.Logger {
	if ctx == nil {
		return StdLogger{}
	}
	return &nodeLoggerAdapter{ctx: ctx}
}

func ResolveLogger(logger bizlog.Logger) bizlog.Logger {
	if logger != nil {
		return logger
	}
	return StdLogger{}
}
```

这里的 `AdaptNodeLogger/ResolveLogger/StdLogger` 只是表达形状；具体命名由项目决定，但应放在仓库级公共位置统一复用。

## What To Avoid

- 在 `XxxImpl` 中直接接收 `types.NodeCtx`
- 在 `XxxImpl` 中直接接收 Matrix logger 类型
- 在 `XxxImpl` 中为了兼容框架日志到处写特殊分支
- 在单个节点包里重复定义私有 `noopLogger` 或一次性的 logger adapter

## Nil / Default Strategy

如果项目允许空 logger 或需要默认 logger：

- 由仓库通用 adapter / resolver 提供默认实现
- 默认实现优先使用一个简单的标准输出 logger，便于排查问题
- 非必须时不要为每个节点单独定义 `noopLogger`

不要把这类框架兼容逻辑散落到每个 `XxxImpl` 中。
