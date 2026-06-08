---
uuid: "f2a1b0c9-d8e7-4f6a-9b5c-4d3e2f1a0b9c"
type: "Reference"
title: "学习Matrix框架测试策略"
status: "Draft"
owner: "neohetj"
version: "1.1.0"
tags:
  - "testing"
  - "strategy"
  - "unit-test"
  - "integration-test"
relations:
  - type: "supports"
    target_uuid: "60c07c47-df0e-4b76-9ed9-62fabe2e2add"
    description: "参考-08 会引导开发者查阅当前测试策略。"
---

# 如何为 Matrix 框架贡献测试

本文档定义当前仓库的测试分层、推荐工具和提交流程。目标不是“测试越多越好”，而是让每次变更至少有与风险等级匹配的验证。

## 1. 测试分层

Matrix 推荐三层测试：

1. **单元测试**：验证单个节点、helper、映射器或注册器的局部逻辑。
2. **集成测试**：验证规则链接线、`inputs/outputs`、relation、共享资源或运行时装配是否正确。
3. **端到端测试**：验证接近真实入口的链路行为，例如 HTTP 入口、文件上传、告警处理。

当前仓库中的典型位置：

- 单元/集成：各包内 `*_test.go`
- 公共测试工具：`test/utils/`
- E2E 样例：`test/e2e_test/alert/`、`test/e2e_test/image_upload/`

## 2. 单元测试建议

### 2.1. 隔离外部依赖

- 不要在单元测试里发真实 HTTP 请求。
- 不要连真实数据库、Redis、Mongo。
- 不要依赖本机环境状态或外部文件系统布局。

如果节点依赖外部系统，优先把依赖抽成接口或可替换 helper。例如 `external/httpClient` 使用 `httpDoer`，从而可以在测试中注入 mock client。

### 2.2. 优先复用现有测试工具

仓库已经提供了通用测试工具，优先复用：

- `test/utils/MockNodeCtx`
- `test/utils/TestLogger`
- `test/utils/MockLogger`
- `test/utils/NewTestRuleMsg`
- `test/utils/GetRootError`
- `test/utils/MockNodeManager`
- `test/utils/MockEndpoint`
- `test/utils/MockRuntimePool`

这些工具已经覆盖了节点执行、日志断言、builder/runtime 注入等高频场景，避免每个测试文件重复手写一套 mock。

## 3. 集成测试建议

以下变更应优先补集成测试，而不是只写单测：

1. 修改了 DSL `inputs` / `outputs` 绑定。
2. 修改了 `EndpointIOPacket`、`BindPath`、`MapAll` 等映射规则。
3. 修改了运行时组装逻辑，如 `builder`、`runtime`、`SharedNodePool`。
4. 修改了 sub-chain、pipeline、forEach、object mapper 等跨节点编排能力。

编写集成测试时，建议：

- 用最小 DSL 只覆盖本次改动链路。
- 显式断言 `DataT`、`Metadata`、relation 路由结果。
- 若改动的是函数节点，单测函数实现，集成测 `type: "functions"` 的接线。

## 4. 端到端测试建议

只有在入口行为、序列化协议或跨模块链路发生变化时，才需要补 E2E。例如：

- `endpoint/http` 的入参与回包语义变化
- 文件上传、二进制处理
- 真实告警/回调链路

E2E 的目标不是替代单测，而是验证“前面所有拼图拼在一起后仍然成立”。

## 5. 提交前检查清单

- [ ] 是否为改动所在包补了至少一个边界或错误场景。
- [ ] 是否为 DSL / runtime / mapping 变更补了最小集成测试。
- [ ] 是否复用了 `test/utils`，而不是复制 mock。
- [ ] 是否避免真实网络、真实数据库和脆弱环境依赖。
- [ ] 如果修改了共享资源或映射规则，是否覆盖了失败路径。

<!-- qa_section_start -->
> **问：新节点应该先写哪类测试？**
> **答：** 先写单元测试锁住局部逻辑，再补一个最小集成测试验证接线、契约和 relation。如果变更了入口或跨模块行为，再补 E2E。
<!-- qa_section_end -->
