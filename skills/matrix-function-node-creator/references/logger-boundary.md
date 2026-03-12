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

示意：

```go
func adaptNodeLogger(ctx types.NodeCtx) bizlog.Logger {
	return nodelog.New(ctx) // project-owned adapter
}
```

这里的 `nodelog.New` 只是占位表达，具体实现由项目自己提供。

## What To Avoid

- 在 `XxxImpl` 中直接接收 `types.NodeCtx`
- 在 `XxxImpl` 中直接接收 Matrix logger 类型
- 在 `XxxImpl` 中为了兼容框架日志到处写特殊分支

## Nil / Default Strategy

如果项目允许空 logger 或需要 no-op logger：

- 由 adapter 提供默认实现
- 或由业务 logger 包提供 no-op logger

不要把这类框架兼容逻辑散落到每个 `XxxImpl` 中。
