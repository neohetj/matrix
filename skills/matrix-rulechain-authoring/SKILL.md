---
name: matrix-rulechain-authoring
description: 设计和改造 Matrix rulechain DSL。用于组织 `ruleChain.id`、`connections`、`inputs/outputs`、`configuration.business`、`objId`、stage 数据流和 shared ref namespace，并在需要时串联 `matrix-http-io-designer` 与 `matrix-rulechain-validator`。
---

# Matrix Rulechain Authoring

这个 skill 负责“rulechain 怎么组织”的作者视角，不负责项目专属目录知识，也不替代需求收敛与静态校验。

适用场景：

- 新增或重构 `dsl/rulechains/**.json`
- 把单体流程拆成多个函数节点和转换节点
- 排查 rulechain 的连接方式、数据流、命名空间和 stage 边界
- 审查现有 rulechain 是否有伪输入、错误连接、无意义 mapper 或错误 channel 写回

## Skill Handoff

- 需求还没收敛：先走 `matrix-requirement-to-dsl`
- 涉及 `endpoint/http`、`external/httpClient`、packet 边界：并行使用 `matrix-http-io-designer`
- 涉及函数节点实现：追加 `matrix-function-node-creator`
- 需要静态校验或风险扫描：收尾使用 `matrix-rulechain-validator`
- 行为变更需要补验证：实现完成后使用 `matrix-test-author`

## Mandatory Rules

1. `ruleChain.id` 必须使用 `namespace/id` 形式，不要写裸 ID。
2. `connections` 只使用 `fromId`、`toId`，不要回退到 `fromIndex`、`toIndex`。
3. `functions` 节点的业务配置必须放在 `configuration.business`，不要把自定义业务字段平铺到配置根层。
4. DSL 节点 `inputs/outputs` 必须与函数签名一致，不允许引入伪输入来“保活”上下文对象。
5. 新增 `objId` 必须使用稳定的可识别缩写，统一全小写连续命名且不要下划线；固定为 `12` 位，建议遵循 `[a-z][a-z0-9]{11}`。
6. 如果流程依赖模块私有 shared ref，对应 rulechain ID、endpoint ID、shared 顶层 ID 和 HTTP path 都必须模块命名空间化。
7. Pipeline stage 场景下，不要手动把结果再推回同一个 `outputChannel`。
8. `transform/object_mapper` 只在真正需要字段转换时引入；不要为了 alias copy 或整体搬运强行加 mapper。
9. 设计完成后必须跑 `matrix-rulechain-validator`，不要只靠肉眼判断。

## Workflow

1. 先定 `ruleChain.id` 和命名空间。
   - 以模块、领域或稳定能力为 namespace
   - 不要把实验性命名写进正式链路

2. 再画执行主干。
   - 哪些节点是步骤节点
   - 哪些节点是转换节点
   - 哪些节点是共享资源消费者
   - 哪些节点只是错误处理或收尾

3. 再定节点契约。
   - `functions` 节点的 `inputs/outputs` 必须和函数定义一致
   - `ParamName`、`defineSid`、`objId` 三者一起看
   - endpoint 或 stage 注入的入口对象要明确，不要靠“顺手经过”保活

4. 再定连接。
   - Success / Failure 分支要显式
   - 普通 rulechain 不要制造多余并发入口
   - 入口链有多个零入度节点时，回到 `matrix-http-io-designer` 明确 `startNodeId`

5. 再查数据流。
   - 中间对象只在真正需要复用时保留
   - stage 边界只传真正需要的对象
   - 保存后要消费保存后的对象，不要继续读保存前的临时对象

6. 最后做收口。
   - 如果 packet / bindPath 不清楚，交给 `matrix-http-io-designer`
   - 如果函数签名、typed whole-object、mapper 风险不清楚，交给 `matrix-rulechain-validator`

## Anti-Patterns

1. 为了保对象，往函数节点上硬塞一个函数根本不读取的输入。
2. `connections` 继续使用 `fromIndex` / `toIndex`。
3. `functions` 节点把业务配置平铺到 `configuration` 根层。
4. 同一个 stage 里手动推回当前 `outputChannel`，制造重复消息。
5. 用 `transform/object_mapper` 只做同 SID 的 objId 改名或整体搬运。
6. 规则链已经依赖模块私有 shared ref，但 ID 仍然使用通用命名空间。

## References

- `references/rulechain-authoring-checklist.md`
- `references/data-flow-smells.md`
