# Shared Node Checklist

把一个能力收敛成 shared/provider 之前，先逐项确认：

1. 这是长生命周期资源，而不是一次性业务动作。
2. 资源是否已有现成 provider，可直接复用 `external/sqlClient`、`external/redisClient`、`external/mongoClient`、`external/httpClient` 这类模式。
3. shared DSL 是否已经明确：
   - 资源名
   - provider 类型
   - `configuration.business` 或 `ref://` 配置来源
   - consumer 会在哪些节点里使用
4. provider 是否有清晰的初始化和失败边界：
   - 配置缺失
   - 连接失败
   - 重复创建
5. consumer 是否通过 `asset.Asset` 或仓库 helper 解析资源，而不是直接访问运行时内部 pool。
6. 是否需要给函数节点再补一层 adapter，把 shared client 和纯业务实现层隔开。
7. 验证是否覆盖：
   - provider 初始化成功
   - provider 初始化失败
   - consumer 解析成功
   - consumer 缺资源时报错

优先看的代码基线：

- `internal/builtin/base/shareable.go`
- `pkg/asset/asset_core.go`
- `pkg/components/external/sql_client_node.go`
- `pkg/components/external/redis_client_node.go`
- `pkg/components/external/mongo_client_node.go`
- `pkg/components/external/http_client_node.go`
