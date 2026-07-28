---
uuid: "a18a0c1b-d599-4f8d-a949-51a60182873d"
type: "Reference"
title: "参考：Matrix 内部错误码编码规范"
status: "Stable"
owner: "neohetj"
version: "1.0.0"
tags:
  - "matrix"
  - "reference"
  - "error-handling"
  - "error-code"
  - "development-standard"
relations:
  - type: "is_part_of"
    target_uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
    description: "本规范属于 Matrix 参考文档库。"
  - type: "specifies"
    target_uuid: "8f7e6a5d-1b2c-4e3f-9a0b-1c2d3e4f5a6b"
    description: "本规范承接 Unified Error Handling RFC 中 Fault.Code 的当前编码事实与研发约束。"
---

# Matrix 内部错误码编码规范

## 1. 定位与权威边界

本文是 Matrix core 内部研发错误码的权威规范。新增或修改 Matrix core `Fault.Code` 时，必须使用 `aabbbcccc` 格式的 9 位数字字符串。

本规范适用于：

- `platform/Matrix` 内部节点、runtime、helper、component 和 protocol adapter 产生的 `Fault.Code`。
- Matrix core 在 `pkg/cnst/constant.go` 中统一登记的 `cnst.ErrCode`。
- `Fault.Code -> FailureInfo.Code -> errorMappings` 链路中的 Matrix 内部错误码。

本规范不适用于：

- HTTP status、gRPC status 或 Stream 处置结果等外部协议状态。
- IdentityX、UsageX、PaymentX 等业务模块对外公开的产品错误码，例如 `PAYMENTX_BAD_REQUEST`。
- 上游供应商的原始错误码、业务状态码或数据字段中的 reason code。

`FailureInfo.Code` 继续使用 `string`。它是结构化错误码的传输容器，不得因 Matrix 内部码为数字形式而改为整数类型。

## 2. `aabbbcccc` 编码结构

Matrix 内部错误码必须满足正则 `^20[0-9]{7}$`，并按以下位段解释：

| 位段 | 位置 | 取值 | 含义 |
| --- | --- | --- | --- |
| `aa` | 第 1-2 位 | 固定为 `20` | Matrix 所属软件产品研发域。 |
| `bbb` | 第 3-5 位 | `000`-`999` | Matrix core 内的模块或能力域编号。 |
| `cccc` | 第 6-9 位 | `0001`-`9999` | 模块内唯一的具体错误标识。`0000` 保留，不得分配。 |

示例：

| 错误码 | `aa` | `bbb` | `cccc` | 含义 |
| --- | --- | --- | --- | --- |
| `200000001` | `20` | `000` | `0001` | Matrix 全局内部错误。 |
| `200010004` | `20` | `001` | `0004` | Runtime client 未初始化。 |
| `202504003` | `20` | `250` | `4003` | HTTP client 代理配置无效。 |

错误码不具有算术语义。代码、JSON、metadata 和 DSL 中必须始终以字符串保存和比较，禁止转为整数后传播、排序或判断。

## 3. 当前模块号分配

`bbb` 是 Matrix core 内部受控命名空间。当前已分配范围如下：

| `bbb` | 能力域 | 当前示例 |
| --- | --- | --- |
| `000` | 全局通用错误 | `200000001`-`200000003` |
| `001` | Runtime | `200010001`-`200010004` |
| `002` | Probe / Tooling | `200020001` |
| `003` | Asset | `200030001`-`200030007` |
| `250` | 历史内建组件与集成域 | `202501001`-`202506004` |

新增能力域时，必须先在本表分配未占用的 `bbb`，再在 `pkg/cnst/constant.go` 中增加常量。不得仅通过搜索“当前没有重复数值”就自行占用新模块号。

`250` 是已有实现中的聚合能力域，其 `cccc` 已包含 endpoint、Redis、DB、HTTP client、pipeline 和 Redis Stream 等历史分段。新的独立能力域不应默认继续扩展 `250`，应优先分配独立 `bbb`。

## 4. 定义与传播规范

### 4.1 定义

Matrix core 内部错误码必须：

1. 在 `pkg/cnst/constant.go` 中定义唯一的 `cnst.ErrCode` 常量。
2. 使用能表达错误语义的 Go 常量名，不得在节点内散落硬编码数字字符串。
3. 在对应 `bbb` 下分配尚未使用的 `cccc`。
4. 一旦发布就不得重新分配或改变语义；废弃错误码应保留历史含义。

```go
// pkg/cnst/constant.go
const (
    CodeExampleUnavailable ErrCode = "200040001"
)

// 错误产生方
var DefExampleUnavailable = &types.Fault{
    Code:    cnst.CodeExampleUnavailable,
    Message: "example capability is unavailable",
}
```

上例只演示结构。使用 `bbb=004` 前仍必须先在本文的模块号分配表中完成登记。

### 4.2 传播

- 节点和 helper 应返回或包装 `types.Fault`，不得依赖错误文案恢复 code。
- Runtime 必须将 `Fault.Code` 原样传播到 `FailureInfo.Code`。
- Endpoint DSL 通过 `errorMappings` 把内部 code 映射为外部协议状态，不得把 HTTP status 当成 `Fault.Code`。
- `ServiceErrorAspect` 和产品外壳必须基于结构化 `FailureInfo.Code` 做转换，不得解析 `error.Error()` 或公开文案。

## 5. Matrix 内部码与产品码的边界

Matrix 内部码和产品公开码可以同时使用 `FailureInfo.Code` 这个字符串容器，但它们属于不同命名空间：

| 类型 | 所有者 | 格式 | 定义位置 |
| --- | --- | --- | --- |
| Matrix core 内部错误码 | Matrix | `aabbbcccc` 9 位数字字符串 | `pkg/cnst/constant.go` |
| 产品公开错误码 | 业务模块 | 由模块公开契约定义，可使用 `IDENTITYX_*` 等命名型 code | 模块 `code/orchestrator/**` 或对外契约 |
| 上游供应商错误码 | Provider adapter | 依供应商协议 | Provider 边界类型，由 Matrix / 产品 Fault 包装 |

业务模块不得在 Matrix `pkg/cnst/constant.go` 中登记产品错误码，也不得占用 `20` 开头的 Matrix core 内部命名空间作为新的产品公开 code。

## 6. 兼容边界

当前 Matrix core 仍存在 `MATRIX_FOR_EACH_001` 等命名型内部 `Fault.Code`。这些是本规范生效前的存量兼容码，不得作为新错误码示例，也不得继续扩展同类命名系列。

存量 code 迁移必须单独建立兼容方案，并同步检查：

- DSL `errorMappings`。
- 日志、trace 和监控查询。
- 宿主 `ServiceErrorAspect`。
- 调用方对错误码的分支逻辑。

在完成兼容评估前，不得为了形式统一直接替换已发布 code。

## 7. 评审清单

新增或修改 Matrix core 错误码时，评审者必须确认：

- [ ] code 是满足 `^20[0-9]{7}$` 的 9 位数字字符串。
- [ ] `bbb` 已在本文的模块号分配表登记。
- [ ] `cccc` 在对应模块内唯一，且不是 `0000`。
- [ ] code 使用 `cnst.ErrCode` 常量定义，没有散落硬编码。
- [ ] code 没有复用已发布错误的编号或改变旧 code 语义。
- [ ] Fault 已按节点注册和传播规范接入，并有覆盖 code 传播的测试。
- [ ] 需要对外协议状态时，DSL `errorMappings` 已显式登记。

## 8. 实现依据

- `pkg/cnst/types.go`：`cnst.ErrCode` 的字符串类型定义。
- `pkg/cnst/constant.go`：Matrix core 预定义错误码和当前编号分配。
- `pkg/types/logger.go`：`Fault` 和 `FailureInfo` 结构。
- `internal/builtin/nodes/endpoint/http_endpoint.go`：`FailureInfo.Code` 到外部 HTTP status 的 DSL 映射。

## 9. 相关文档

- [统一错误处理指南](../guides/unified-error-handling-guide.md)
- [HTTP Endpoint 深度解析](10_http_endpoint_deep_dive.md)
- [Unified Error Handling RFC](../designs/rfc/0010_unified_error_handling_rfc.md)
