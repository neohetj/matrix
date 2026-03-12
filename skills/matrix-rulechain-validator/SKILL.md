---
name: matrix-rulechain-validator
description: "校验基于 Matrix 的 DSL 规则链、HTTP endpoint 与函数节点配置。用于排查规则链执行失败、审查 DSL 改动、批量扫描 `EndpointIOPacket` 风险映射、识别函数输入输出与签名不一致、`collection SID + type: object`、typed whole-object 跨 SID 转换，以及发现无意义的 `transform/object_mapper` 整体搬运节点。"
---

# Matrix Rulechain Validator

## Overview

复用工作区内的静态校验器，优先发现两类高频 Matrix DSL 问题：
- 会在运行期触发 `ProcessInbound` / `ProcessOutbound` 转换错误的映射配置
- 会在 UI 或运行期表现为配置不合法的函数节点签名问题

遇到 trace 失败时，先用这个 skill 做静态扫描，再结合 trace 定位首个有效失败节点，避免被级联报错带偏。

适用前提：
- 工作区里有 Matrix 风格 DSL，通常位于 `code/dsl/`
- 最好已有仓库本地校验脚本，例如 `scripts/validate_rulechain_mappings.py`
- 如果项目还没有校验脚本，可以先沿用本 skill 的规则和模式补一个

## Workflow

1. 从工作区根目录运行仓库本地的 `scripts/validate_rulechain_mappings.py`，或用本 skill 的 `scripts/run_validator.py` 自动定位工作区。
2. 先处理 `function-*` 规则：
   `function-input-not-defined` / `function-output-not-defined` 往往说明 DSL 还留着伪输入、旧字段名或 UI 会直接报警的配置。
   `function-required-input-missing` 说明节点少绑了必填参数。
   `function-*-sid-mismatch` 说明 DSL `defineSid` 和函数签名已经漂移。
3. 再处理映射规则。如果命中 `collection-sid-object-conversion`，优先检查是否在 helper-based packet 中把集合 SID 显式写成了 `"type": "object"`。
4. 如果命中 `typed-whole-object-cross-sid-conversion`，优先检查是不是把完整业务对象直接灌进另一个 typed SID，尤其是 `*Patch*_V*`；这类场景必须显式按字段映射。
5. 如果命中 `object-mapper-alias-copy`，评估这个 `transform/object_mapper` 是否只是把同一个 SID 换了个 objId；能直读原对象就直接删掉。
6. 如果静态检查干净但运行仍失败，再查 trace，确认首个真正失败节点，而不是后续错误处理链的连锁报错。

## Covered Rules

- `function-not-found`
  只检查 `type=functions` 节点。
  当 `configuration.functionName` 不在本地已注册函数目录里时，判为错误。

- `function-input-not-defined`
  只检查 `type=functions` 节点。
  当 DSL 节点声明了函数签名里不存在的输入参数时，判为错误。
  典型场景是为了“保活”对象，往节点上硬塞一个函数根本不读取的 `task_context`。

- `function-output-not-defined`
  只检查 `type=functions` 节点。
  当 DSL 节点声明了函数签名里不存在的输出参数时，判为错误。

- `function-required-input-missing`
  只检查 `type=functions` 节点。
  当函数签名里的必填输入没有在 DSL 节点上绑定时，判为错误。

- `function-input-sid-mismatch` / `function-output-sid-mismatch`
  只检查 `type=functions` 节点。
  当 DSL 里的 `defineSid` 和函数签名声明的 `DefineSID` 不一致时，判为错误。
  这类问题即使字段名对了，也会在运行期制造类型漂移。
  通用签名如 `Any`、`[]Any`、`MapStringInterface` 允许 DSL 绑定更具体的业务 SID，不视为错误。

- `collection-sid-object-conversion`
  只检查真正走 `helper.ProcessInbound` / `helper.ProcessOutbound` 的节点：
  `transform/object_mapper`、`endpoint/http`、`external/httpClient`。
  当字段源或目标 URI 带 `sid=[]...`，但字段类型写成 `"object"` 时，判为错误。

- `object-mapper-alias-copy`
  只检查 `transform/object_mapper`。
  当节点只有一个字段，且只是把一个 whole-object `rulemsg://dataT/...` 原样搬到另一个相同 SID 的 whole-object 上时，判为警告。

- `typed-whole-object-cross-sid-conversion`
  检查 `transform/object_mapper`、`endpoint/http`、`external/httpClient` 中 `type: "object"` 的字段。
  当字段源和目标都是 whole-object `rulemsg://dataT/...`，且源/目标都是非泛型 typed SID、两者 SID 不同，则判为错误。
  典型错误是把 `InstagramLead_V1` 整体灌进 `LeadAnalysisPatch_V1`。

## Intentional Exclusions

- `action/forEach` 不在本技能的 `type: object` 风险范围内。
  它的 `inputMapping/outputMapping` 是自定义实现，不走 `convertValue`，不能按同一规则误报。
- 链内普通节点连线不会裁剪 DataT。
  如果问题表现为对象在 stage 之间丢失，应回头检查 pipeline / channel 边界，而不是普通连接边。

## Fix Patterns

- 函数节点多余输入/输出：删除 DSL 上不在函数签名中的绑定，不要把“保对象”当成函数参数传。
- 函数节点缺少必填输入：按函数签名补齐参数；如果函数确实不该必填，改函数定义而不是靠 DSL 猜测。
- 函数节点 SID 不一致：以函数签名为准修正 `defineSid`，不要让 DSL 和 `NodeFuncObject` 各写各的。
- 集合整体透传：去掉字段上的 `"type"`，让 typed DataT 原样流转。
- 字符串 JSON 解析成对象：保留 `"type": "object"`，但确保源是 `sid=String` 或原始 JSON 字符串。
- 跨 SID 的 typed whole-object 转换：不要整对象透传，改成显式字段映射。
- 目标是 `Patch` SID：只映射 patch 所需字段，不要把完整业务对象直接塞进去。
- 仅换 objId 的 mapper：优先删除中间 `object_mapper`，直接让下游节点读取原对象。

## Resources

- `scripts/run_validator.py`
  自动定位包含 `code/dsl` 与 `scripts/validate_rulechain_mappings.py` 的工作区，并转调仓库本地校验器。
- `references/rules.md`
  汇总当前静态校验规则和常见修复方式。

遇到复杂 case 时再读 [references/rules.md](references/rules.md)。
