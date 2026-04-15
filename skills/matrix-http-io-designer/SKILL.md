---
name: matrix-http-io-designer
description: 设计和修复 Matrix `endpoint/http`、`external/httpClient` 与相关 packet 映射。用于收敛 `EndpointIOField.BindPath`、`EndpointIOPacket`、`mapAll`、`fields`、`body`、query/path/header 绑定、`ruleChainId/startNodeId`、同步 list contract，以及避免 typed whole-object 跨 SID、集合对象误配和 patch/map 写入风险。
---

# Matrix HTTP IO Designer

## Overview

这个 skill 负责把 HTTP 出入参边界设计清楚，避免 DSL 写出来以后才在 `ProcessInbound` / `ProcessOutbound` 阶段爆炸。

重点覆盖：

- `endpoint/http` 请求入参绑定
- `endpoint/http` 响应包体输出
- `external/httpClient` 请求/响应 packet
- `endpoint/http` 入口 rulechain 与 `startNodeId` 选择
- 与 `transform/object_mapper` 相邻的对象边界设计

Skill 触发口令：`$matrix-http-io-designer`

## Skill Handoff

- 先做需求收敛和整体 DSL 草案：先走 `matrix-requirement-to-dsl`
- rulechain 主干、节点职责或 stage 数据流不清楚：并行使用 `matrix-rulechain-authoring`
- 映射设计完成后做静态扫描：收尾使用 `matrix-rulechain-validator`
- 映射背后需要 shared HTTP client / DB client：并行使用 `matrix-shared-node-creator`
- 行为变更需要补测试：实现完成后使用 `matrix-test-author`

## Mandatory Rules

1. 当前 HTTP 映射统一使用 `EndpointIOField.BindPath` 与 `EndpointIOPacket`，不要再写 `mapping.to`、`mapping.defineSid`、`bodyFields` 这套旧语法。
2. `mapAll` 只用于同形状、同语义或泛型对象的整体透传；一旦跨业务 SID、跨 patch 边界或跨 typed object，就改成显式 `fields`。
3. 带集合 SID 的字段不要再声明 `"type": "object"`。
4. typed whole-object 不得跨不同 SID 直接搬运；尤其是 `Patch`、`MapStringInterface`、聚合响应对象，默认做显式字段映射。
5. `bindPath` 必须明确来源位置，例如 query、path、header、body 内部字段；不要依赖隐式字段名猜测。
6. 同步查询类 HTTP list 接口默认遵守统一契约：`GET + page/pageSize + data/total [+ meta]`。
7. endpoint 层默认保留业务对象 SID；如果下游函数只吃 `MapStringInterface` 或 patch map，优先在 rulechain / mapper 边界显式降级，不要在 HTTP 边界过早降级。
8. 如果目标对象是 `MapStringInterface` patch，先确认运行时是否支持嵌套路径自动初始化；不确定时优先用函数节点先组完整 map。
9. 设计完成后必须跑 `matrix-rulechain-validator`，不要只靠肉眼判断。

## Workflow

1. 先列清楚边界：请求有哪些 query/path/header/body 字段，响应有哪些 body/status/header 字段。
2. 先确认入口 rulechain：`ruleChainId` 必须存在；只要入口链有分叉、多个零入度节点或后续会演化，默认显式写 `startNodeId`，不要赌隐式起点。
3. 再列 rulechain 内部对象：每个字段会落到哪个 `rulemsg://dataT/...` URI、SID 是什么、是否需要 whole-object 透传。
4. 设计 inbound packet：能显式写字段就显式写字段；只有结构完全一致时才考虑 `mapAll`。如果下游需要 `MapStringInterface`，优先在 rulechain 内显式转换，不要在 endpoint 边界把业务对象提前降级。
5. 设计 outbound packet：优先返回稳定响应对象，不要把内部业务对象原样暴露给 HTTP。同步 list 接口默认返回 `data`、`total`，必要时再补 `meta`。
6. 如需 `transform/object_mapper`，确认它是真正做字段转换，而不是仅仅换一个 objId。
7. 跑 `matrix-rulechain-validator`，修掉 typed whole-object、collection/object、函数签名等问题。
8. 行为变更后，用 `matrix-test-author` 补至少一条能覆盖请求/响应边界的验证。

## Anti-Patterns

1. 继续沿用 `mapping.to`、`mapping.defineSid`、`bodyFields` 旧格式。
2. 把 `Lead_V1`、`Task_V1` 这类完整业务对象直接塞进 `LeadPatch_V1`、`TaskPatch_V1`。
3. 为了图省事，对 `[]DomainObject` 字段写 `"type": "object"`。
4. response 直接暴露内部暂存对象，导致外部契约和内部对象演化强耦合。
5. 用 `transform/object_mapper` 只做相同 SID 的 alias copy，却引入额外节点和类型风险。
6. endpoint 只是为了迎合下游 `MapStringInterface`，就在 HTTP 边界把业务对象整体降级。

## References

- `references/http-io-checklist.md`
- `references/packet-patterns.md`
