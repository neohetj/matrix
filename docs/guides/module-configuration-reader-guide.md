---
uuid: "2a29782b-caa7-4d25-927e-ec123b770d94"
type: "Guide"
title: "指南：接入模块配置 Reader"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
updated_at: "2026-09-08"
tags: ["matrix", "configuration", "guide"]
relations:
  - type: "is_part_of"
    target_uuid: "b4c5d6e7-f8a9-0b1c-2d3e-4f5a6b7c8d9e"
    description: "属于 Matrix 指南文档库。"
  - type: "uses_reference"
    target_uuid: "1f8831e5-2dc6-42a1-a5df-37db4a549efe"
    description: "遵循 Reader 和实例注入契约。"
---

# 接入模块配置 Reader

前提：依赖版本或本地 replace 必须包含 [Reader 契约](../reference/41_module_configuration_reader.md) 中的 API。本指南不执行发布或重启。

1. 在模块 Catalog 声明键、类型、来源策略及唯一默认值；节点 Schema 保留形状约束，不重复声明默认值。
2. 从显式嵌入文件系统或目录加载 Catalog，用模块自己的 business 和 env 来源创建 ConfigResolver，再调用 NewReader。不要共享可变全局 resolver。
3. 在同一 Reader 上执行 ValidateProvided；能力初始化使用 Read/ReadNode/ReadDuration 或 Decoder 装配 DTO，检查全部错误后再产生副作用。跨字段完整性使用完整 Catalog 校验或模块明确的业务校验。
4. 实现 ConfigReaderAware 的资源在 Init 中消费已注入 Reader；宿主在 matrix.New 前登记 Reader 和节点、规则链归属。WhiteRoom 托管模块先同步 configuration capability，不在模块内复制解析算法。
5. 用两个 Engine、同名配置键、不同来源验证隔离；覆盖 false/0、Secret 覆盖拒绝、错误来源、时长溢出与初始化失败。用合成值，不读取真实凭据。

在 Matrix 仓库执行：

```sh
go test ./pkg/config/... ./pkg/types/...
go test . -run ModuleConfig
```

消费方接入验证独立于 Matrix 包测试。失败时修复来源或业务声明；不要吞错后补默认值，也不要修改进程环境绕过门禁。Reader 快照更新通过新实例完成，不提供原地热更新接口。
