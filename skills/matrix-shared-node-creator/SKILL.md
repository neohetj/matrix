---
name: matrix-shared-node-creator
description: 设计与实现 Matrix shared/provider 节点及其消费者。用于新增或重构 `shared` DSL、`external/*` 共享客户端、`types.ShareData` / `base.Shareable` 提供者，以及让函数节点、endpoint 或 orchestrator 通过 `asset.Asset` 消费共享资源。
---

# Matrix Shared Node Creator

## Overview

这个 skill 负责把“可复用资源”从业务逻辑里拆出来，收敛成稳定的 Matrix shared/provider 模式：

- provider 节点负责初始化和复用 client、pool、store、session manager 等长生命周期资源
- shared DSL 负责暴露资源与配置引用
- consumer 节点负责通过 `asset.Asset` 或等价 helper 解析资源，而不是直接碰全局变量或私有 pool

Skill 触发口令：`$matrix-shared-node-creator`

## Skill Handoff

- 只做 DSL 入口和链路编排：先走 `matrix-requirement-to-dsl`
- shared 资源由函数节点消费：并行使用 `matrix-function-node-creator`
- shared 资源影响 endpoint/http 或 httpClient 入参设计：并行使用 `matrix-http-io-designer`
- 实现后要补回归覆盖：收尾使用 `matrix-test-author`

## Mandatory Rules

1. shared/provider 节点只负责资源生命周期，不承担业务流程编排。
2. 共享资源优先通过 `type: "shared"` DSL 容器暴露，不要新增隐藏的全局单例。
3. provider 配置优先走 `configuration.business`、显式字段或 `ref://`，不要把关键配置散落在业务函数内部即时读取。
4. consumer 侧优先通过 `asset.Asset`、`asset.WithNodeCtx`、`asset.WithRuleMsg` 或仓库已有 helper 取资源，不要把 `NodePool` 直接下沉到业务实现层。
5. 如果 Matrix 已有同类 provider，例如 `external/sqlClient`、`external/redisClient`、`external/mongoClient`、`external/httpClient`，默认优先复用现有模式，不重复造一套生命周期。
6. shared/provider 节点要能安全处理初始化失败、重复获取和关闭时机；不要把“每次调用都重新建连接”伪装成 shared。
7. consumer 节点如果只是做业务动作，不要再把 client 初始化逻辑塞回函数节点或 orchestrator。
8. 任何 shared 改动，至少覆盖 provider 初始化失败、consumer 解析失败和 happy path 三类验证。

## Workflow

1. 先判断资源边界：它究竟是“共享 client/pool/store”，还是“单次业务动作”。只有前者才应该进 shared/provider。
2. 找最近似基线，优先看已有 `external/*` 节点或同类 shared DSL。
3. 设计 shared DSL：资源名、配置引用、chain 入口、上下游依赖关系。
4. 落 provider 代码：优先复用 `base.Shareable[T]` 或仓库现有生命周期封装。
5. 落 consumer 代码：通过 `asset.Asset` 或 helper 解析资源，把业务处理留在函数/服务实现里。
6. 如果 consumer 是函数节点，再用 `matrix-function-node-creator` 收敛 adapter 层与纯业务实现层。
7. 用 `matrix-test-author` 设计最小验证集，并至少执行 provider + consumer 两侧的关键测试。

## Anti-Patterns

1. 在函数节点里偷偷缓存全局 client，却没有 shared DSL 或 provider 声明。
2. provider 节点里顺手执行业务 SQL、发 HTTP、改业务对象，导致资源生命周期和业务流程耦合。
3. consumer 直接 import 私有 pool、registry 或运行时内部 map，而不是走 `asset` / helper。
4. 同一类数据库/缓存客户端在多个模块各自造一套 shared 生命周期，无法复用也无法统一排障。

## References

- `references/shared-node-checklist.md`
- `references/consumer-patterns.md`
