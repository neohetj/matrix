# Test Scope Matrix

按改动类型选测试层级：

## Pure Business Logic

- 目标：验证算法、参数处理、领域规则
- 位置：函数节点的 `XxxImpl`、service、helper
- 建议：普通单测，不引入 Matrix 运行时

## Matrix Adapter Layer

- 目标：验证 `GetParam`、`SetParam`、配置读取、错误出口、`TellSuccess/TellFailure`
- 位置：函数节点入口、packet 组装、consumer 资源解析
- 建议：包级测试，必要时复用 `test/utils`

## Shared / Provider

- 目标：验证 provider 初始化、复用和 consumer 解析失败
- 建议：
  - fake client 或内存实现
  - 不默认连真实外部系统
  - 显式覆盖“缺配置”“初始化失败”“资源不存在”

## HTTP / Endpoint Mapping

- 目标：验证 query/path/header/body 到内部对象，以及内部对象到响应对象的边界
- 建议：
  - 至少一条正常 case
  - 容易踩坑时再补空值、类型错配、patch 构造失败
  - 配套运行 `matrix-rulechain-validator`

## Rulechain / Endpoint Integration

- 目标：验证多节点协作和最终行为
- 建议：
  - 只在单测覆盖不了真实风险时才升到这一层
  - 优先复用仓库已有 `test/utils` 和 `test/e2e_test` 结构

## Command Checklist

至少记录并尽量执行：

- 相关包 `go test`
- 涉及 DSL 时的 `matrix-rulechain-validator`
- 如果有 endpoint / shared 实际链路，再补一个最小 smoke 验证
