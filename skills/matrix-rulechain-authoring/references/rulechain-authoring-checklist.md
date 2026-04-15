# Rulechain Authoring Checklist

## 1. ID 与命名空间

- `ruleChain.id` 是否采用 `namespace/id`
- namespace 是否稳定、可复用、不是一次性实验命名
- 新增 `objId` 是否为全小写连续命名、长度稳定、可读

## 2. 节点职责

- 步骤节点、转换节点、共享资源消费者是否区分清楚
- 是否存在只为“保对象”而存在的无意义节点
- 是否把业务配置错误地平铺到 `configuration` 根层，而不是 `configuration.business`

## 3. 输入输出契约

- `functions` 节点 `inputs/outputs` 是否与函数签名一致
- `ParamName`、`defineSid`、`objId` 是否成组一致
- 入口对象是否来自明确的 endpoint / stage 注入，而不是隐式流经

## 4. 连接方式

- `connections` 是否只使用 `fromId`、`toId`
- Success / Failure 分支是否明确
- 是否存在多个零入度节点但没有显式入口控制

## 5. 数据流与 stage

- 中间对象是否只在真正需要复用时保留
- 保存后的流程是否消费持久化后的对象，而不是保存前对象
- Pipeline stage 场景下是否错误地回推同一个 `outputChannel`
- stage 边界是否只传递真正需要的对象

## 6. 收尾校验

- packet / bindPath 问题是否交给 `matrix-http-io-designer`
- 函数签名、typed whole-object、mapper 风险是否交给 `matrix-rulechain-validator`
- 行为变化后是否补了验证
