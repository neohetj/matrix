# Mapping Rules

## Core Rules

- `SID` 必须来自已有对象定义，或与新增对象定义同步落地。
- `objId` 要稳定、可读，不使用无意义随机串。
- `endpoint/http`、`endpoint/pipeline`、`rulechain`、`prompt`、`shared` 之间的引用必须可追溯。
- 新增 prompt 文件时，`rel://` 路径必须从当前 rulechain 文件位置正确推导。

## Function Contract Rules

- `functions` 节点的 `functionName` 必须已注册。
- `inputs/outputs` 必须和函数签名一致。
- 不允许为保对象而附加函数签名里不存在的输入。

## Object Mapping Rules

- 跨 `SID` 不做整对象透传。
- `Patch` 目标必须逐字段映射。
- 集合对象的 `loopSource`、`inputMapping`、`outputs` 必须保持 `SID` 一致。
- 如果字段可能缺失，只在显式配置了 `defaultValue` 时才依赖缺字段兜底。

## Pipeline Rules

- stage 的输入输出 channel 要形成完整链路。
- 如果 stage 依赖持久化结果，消费保存后的对象或列表。
- 并发设置要和外部依赖特性相匹配，不要盲目复制别的 pipeline 数值。

## Anti-Patterns

- 直接从自然语言开始手写整份 JSON
- 为了跑通链路临时塞 `task_context` 等伪输入
- 用旧对象继续驱动后续 stage，而不是消费保存结果
- 用 `type: "object"` 把完整业务对象写进 patch
