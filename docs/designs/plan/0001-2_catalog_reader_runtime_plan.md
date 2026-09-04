---
uuid: "281fac8e-19ca-4fbd-94d0-5829f462c1a4"
type: "Plan"
title: "Plan：Matrix Catalog Reader 与配置实例装配"
status: "Draft"
owner: "neohetj"
version: "1.0.0"
updated_at: "2026-09-04"
scope: "workspace"
tags: ["configuration", "runtime", "module-boundary"]
relations:
  - type: "is_part_of"
    target_uuid: "fde86d18-74ce-4350-9137-553929ee1261"
    description: "本文档属于模块配置运行时治理文档集。"
  - type: "is_plan_for"
    target_uuid: "7f76ade4-d0ad-4fc4-909f-5a5f3eab0387"
    description: "本仓承接跨仓配置运行时 RFC 的对应实现边界。"
---

# Plan：Matrix Catalog Reader 与配置实例装配

## 计划概述与范围

承接 [跨仓 RFC](../../../../../docs/cross-repo/module-configuration-runtime/designs/rfc/0001_module_configuration_runtime_rfc.md)，只负责通用能力，不在框架源码、fixtures 或模板中写入任何真实业务 key、路径、module id。执行步骤和完整行为用例见 [跨仓 Plan](../../../../../docs/cross-repo/module-configuration-runtime/designs/plan/0001-1_module_configuration_runtime_plan.md#task-2matrix-建立-catalog-reader-唯一入口)。

## 本仓写集

- pkg/config/config_resolver.go：复用来源逻辑、安全错误，不再添加第二套 source resolution。
- pkg/config/catalog/reader.go、duration.go、asset_context.go 及同名测试：typed Reader、明确单位、AssetContext 适配。
- pkg/types/config_reader.go：可选配置领域接口；不得把 config/catalog 引入 types 造成循环。
- module_config.go、module_config_test.go 与 matrix.go：实例 option、共享初始化前注入。
- node clone→Init 的确切文件：由 G1 源码盘点在写集冻结后列明，未冻结不实现此部分。
- 若需旧语义兼容，限定独立 compat 文件并记录引用和删除 gate，不影响新 Reader 默认契约。

## 可衡量目标

业务消费者只依赖稳定包 API；两个 Engine 相同模块名不串值；未装配配置 consumer 明确失败。没有新增进程全局默认 Reader、workspace 路径发现或远程资源管理依赖。

## 实现与验收

- [ ] G1 冻结 API、输入快照、可选接口与真正 Init 前注入位置；必要时重新批准写集。
- [ ] TDD 建立 env/alias/business/default、Secret env-only、false/0/int64、duration 溢出、unknown/missing/source failure 用例。
- [ ] 构建 Reader 后所有入口调用现有 ConfigResolver/Catalog 核心，字段读取不重跑全局 schema。
- [ ] 按跨仓 Task 3 增加真实 Engine 双实例、shared Init、AssetContext、旧接口 mock 编译测试。
- [ ] 执行 go test ./pkg/config/... -count=1、go test -race ./pkg/config/... -count=1、go test ./... -run '^$' 及冻结的 Engine 行为包。
- [ ] 发布可引用的 Matrix 兼容版本后再允许下游移除旧读取入口；不自动 git push 或发 release。

## 风险与收口

ConfigResolver 旧错误包含值、Secret whitespace、旧 bool/时长宽松规则都必须在测试中分离。不得借重构改变全仓 behavior 或强制所有 Node 实现新接口。

实现后更新现有配置 URI Guide，并在本仓 Reference 增补 ConfigResolver/Catalog Reader 当前 API 与生命周期（沿本仓编号分配，不提前创建 Stable 事实）。同步 reference/00_decision_traceability.md。

当前实施中：Reader、Snapshot、Decoder、AssetContext 适配及 shared/runtime Init 前注入已经实现。真实注入点为 `internal/registry/shared_node_pool.go` 与 `internal/runtime/runtime.go`；`internal/builder/builder.go` 传播新配置消费者的安全错误。配置分支为 `codex/refactor/module-config-runtime`，不合并 main，不发布版本。

最近验证：`go test -race . ./pkg/config/... ./pkg/asset -count=1` 所涉包通过。builder 的重复定义测试存在 map 遍历顺序导致的 SourcePath 断言不稳定，需单独报告，不能据此宣称全仓全绿；另外 registry SID 命名及 runtime 未注册 log 的既有失败已经单独记录在跨仓 Plan。全模块迁移、最终事实文档和远端版本消费仍未关闭。
