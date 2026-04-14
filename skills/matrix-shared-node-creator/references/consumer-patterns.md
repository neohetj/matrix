# Consumer Patterns

## Shared DSL Skeleton

```yaml
id: shared.sql.main
type: shared
configuration:
  chainId: "shared/sql/main"
```

```yaml
id: sqlClient
type: external/sqlClient
configuration:
  business:
    dsn: ref://env/DB_DSN
    maxOpenConns: 10
```

## Provider Pattern

- provider 节点负责把配置转成可复用资源
- provider 节点返回或登记 `types.ShareData`
- 生命周期封装优先复用 `base.Shareable[T]`

## Consumer Pattern

```go
assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))
client, err := helper.GetSharedResource[*sql.DB](assetCtx, "sqlClient")
```

如果仓库没有通用 helper，也应保持同样边界：

- 解析资源发生在 Matrix adapter 层
- 纯业务实现层接收具体 client/store interface
- 不把 `NodeCtx`、`RuleMsg`、`NodePool` 继续下沉

## When Not To Use Shared

- 只是单次业务动作，没有复用价值
- 资源初始化非常便宜，而且生命周期必须严格绑定单次请求
- 同类 provider 已存在，你真正需要的是复用 consumer，而不是新增 provider
