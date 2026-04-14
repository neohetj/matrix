---
name: matrix-test-author
description: 为 Matrix DSL、函数节点、shared/provider 节点和 HTTP 映射补充最小有效测试。用于选择单测/集成测试/E2E 边界、复用 `test/utils`、覆盖映射失败与共享资源失败路径，并把验证命令收敛成可执行清单。
---

# Matrix Test Author

## Overview

这个 skill 负责把“我们应该怎么验证这次改动”收敛成最小、可执行、能挡回归的测试方案。

重点覆盖：

- 函数节点 adapter 层和纯业务实现层
- shared/provider 生命周期
- `endpoint/http` / `external/httpClient` packet 边界
- rulechain / endpoint 级集成验证

Skill 触发口令：`$matrix-test-author`

## Skill Handoff

- 函数节点改动：与 `matrix-function-node-creator` 配合使用
- shared/provider 改动：与 `matrix-shared-node-creator` 配合使用
- HTTP IO 映射改动：与 `matrix-http-io-designer` 配合使用
- 需要先整理需求和 DSL 边界：先走 `matrix-requirement-to-dsl`

## Mandatory Rules

1. 先选最小测试层级，不要一上来就写笨重的 E2E。
2. 纯业务实现优先写普通单测；Matrix adapter 层再补参数读取、输出写回和错误路径。
3. shared/provider 改动至少覆盖初始化成功、初始化失败、consumer 缺资源三类路径。
4. HTTP packet 改动至少覆盖一个正常请求/响应边界；如果容易踩坑，再补一个失败或空值 case。
5. 改的是 DSL 配置但没有现成自动化时，至少执行 `matrix-rulechain-validator` 并记录剩余风险。
6. 默认避免真实外部网络、数据库和第三方服务；优先 fake、stub、mock 或仓库既有 test utils。
7. 不要只测日志；优先断言返回值、写回对象、路由结果和错误类型。

## Workflow

1. 先归类改动：函数逻辑、DSL 映射、shared 生命周期，还是端到端链路。
2. 为每类改动选最小测试层级：
   - 纯函数逻辑 -> 单测
   - Matrix adapter/packet 边界 -> 包级测试
   - 多节点协作或 endpoint 行为 -> 集成测试
   - 真实链路 smoke -> E2E 或人工验证
3. 优先复用 `test/utils`、mock node、mock logger 和现有 helper。
4. 补失败路径，不只测 happy path。
5. 执行最小相关命令，确认测试和静态校验都能跑通。
6. 如果仍有无法自动化的风险，显式写出来，而不是假装“已覆盖”。

## Anti-Patterns

1. 只补一个 happy path，然后把映射失败、缺配置、缺资源都留给线上。
2. 本该测纯业务实现，却把断言全写在笨重的端到端里。
3. shared/provider 改动直接连真实数据库或 Redis，导致测试脆弱且不可复现。
4. DSL 改动完全不跑静态校验，也没有任何样例请求验证。

## References

- `references/test-scope-matrix.md`
