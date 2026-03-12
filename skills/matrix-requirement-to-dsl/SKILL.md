---
name: matrix-requirement-to-dsl
description: 将产品或业务需求转换为 Matrix DSL 实现方案与实际改动。用于把自然语言需求落成 `endpoint/http`、`endpoint/pipeline`、`rulechain`、`prompt`、`shared` 等 DSL 文件，并强制执行需求收敛、设计草案、契约映射和实现后验证。
---

# Matrix Requirement To DSL

这个 skill 负责“需求 -> Matrix DSL”的通用过程控制，不包含任何项目专属目录或对象知识。

如果当前仓库存在 `skills/*-dsl-adapter/`，在需求收敛后必须继续读取对应 adapter。  
如果当前仓库存在 `matrix-rulechain-validator` 或等价校验入口，实现完成前必须执行它。

## Workflow

1. 先用 `references/requirement-template.md` 把自然语言需求整理成结构化输入。
2. 再用 `references/design-output-contract.md` 产出 DSL 设计草案，再决定是否开始改文件。
3. 再读取项目 adapter，找到最接近的现有 DSL 作为基线。
4. 按 `references/implementation-workflow.md` 做最小化实现。
5. 按 `references/acceptance-checklist.md` 做校验，并补充项目自己的验证命令。

## Non-Negotiable Rules

- 不要从自然语言直接跳到 JSON 编辑。
- 优先复用现有 endpoint、pipeline、rulechain、prompt、shared 结构。
- 不允许跨 `SID` 的整对象透传。
- 目标为 `Patch` 类型时，必须显式字段映射。
- DSL 节点 `inputs/outputs` 必须与函数签名一致。
- 不允许引入伪输入来“保活”上下文对象。
- 如果 stage 依赖持久化后的对象，必须消费保存后的对象，而不是保存前的临时对象。

## Required Deliverables

在真正修改 DSL 之前，至少明确这些内容：

- 需求摘要
- 入口类型和入口文件
- Rulechain / stage 拆分
- 关键 `objId -> SID` 对照
- 复用节点与新增节点清单
- 风险点和待确认项

## References

- `references/requirement-template.md`
- `references/design-output-contract.md`
- `references/mapping-rules.md`
- `references/implementation-workflow.md`
- `references/acceptance-checklist.md`
