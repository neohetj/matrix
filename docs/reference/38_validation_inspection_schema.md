---
uuid: "66415a13-ae2e-4d05-b791-542005c946ec"
type: "Reference"
title: "参考：Matrix Validation 与 Inspection 输出模型"
status: "Draft"
owner: "neohetj"
version: "0.1.0"
tags:
  - "matrix"
  - "validation"
  - "inspection"
  - "schema"
relations:
  - type: "is_part_of"
    target_uuid: "c5d6e7f8-a9b0-c1d2-e3f4-a5b6c7d8e9f0"
    description: "本文档属于 Matrix 参考文档库。"
  - type: "supports"
    target_uuid: "86e6d134-004f-4483-a97a-396d2ab79a37"
    description: "本文档记录 Matrix core interface boundary refactor 的 validation / inspection 输出事实。"
  - type: "references"
    target_uuid: "ce131041-d7a5-494c-b4f3-dace15eff644"
    description: "本文档承接 Plan 0015-1 Stage 0.5 的第一批低影响边界优化。"
---

# Matrix Validation 与 Inspection 输出模型

本文档记录 Matrix 当前新增的 validation report 与 inspection snapshot 输出模型。它们是 Stage 0.5 的第一批低影响边界优化：先稳定可被 CLI、CI、Morpheus、rulechain validator 和文档审计工具消费的结构化输出，不改变 runtime、loader、endpoint 或业务模块行为。

## 1. 实现位置

| 包 | 当前职责 |
| --- | --- |
| `pkg/validation` | 定义 `Report`、`Issue`、`Target`、`Scope`、severity、mode 和 issue code。 |
| `pkg/validation` | 提供 `ValidateLoaderResources(...)`，以 report-only 方式扫描 DSL loader 输入。 |
| `pkg/inspection` | 定义 `InspectionSnapshot` 与 `RuntimeFactDescriptor`，用于承载 runtime、rulechain、endpoint、function、shared resource 等事实描述。 |

当前实现是 schema foundation 和 loader report-only scanner，不替换现有 runtime / loader 路径。

## 2. ValidationReport

`pkg/validation.Report` 的 JSON 输出字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `schemaVersion` | string | 输出 schema 版本；默认值是 `matrix.validation.report/v1`。 |
| `mode` | string | `report-only` 或 `strict`。 |
| `scope` | object | 本次校验范围，例如 rulechain、engine、loader。 |
| `issues` | array | 结构化 issue 列表。 |
| `hasErrors` | bool | 根据 `issues[].severity == "error"` 派生。 |
| `errorCount` | number | error issue 数量，序列化时派生。 |
| `warningCount` | number | warning issue 数量，序列化时派生。 |

`Issue` 当前字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | string | 稳定 issue code，例如 `dangling_connection`、`unknown_function`、`loader_failure`。 |
| `severity` | string | `info`、`warning` 或 `error`。 |
| `message` | string | 面向开发者的简短说明。 |
| `target` | object | 问题对象，例如 node、connection、endpoint、function、shared ref。 |
| `details` | object | 可选细节，不作为主判断字段。 |

## 3. Loader Report-Only Scanner

`pkg/validation.ValidateLoaderResources(provider, paths)` 当前会扫描指定的 rulechain、endpoint 和 shared DSL 目录，并返回 `Report`。

输入：

| 字段 | 说明 |
| --- | --- |
| `LoaderPaths.RuleChains` | rulechain DSL 目录列表。 |
| `LoaderPaths.Endpoints` | endpoint DSL 目录列表。 |
| `LoaderPaths.Shared` | shared DSL 目录列表。 |

模块级扫描必须传入该模块实际加载的完整 DSL 根。对于同时存在 `code/dsl` 与 `common/dsl` 的模块，`RuleChains`、`Endpoints`、`Shared` 应同时包含两个根下对应目录；只扫描 `code/dsl` 会把 `common/dsl/shared` 中定义的 shared resource 误报为 `missing_shared_ref`。

`pkg/validation.DiscoverLoaderPaths(provider, dslRoots)` 用于从 DSL root 列表推导 `LoaderPaths`。当 `dslRoots` 为空时，默认扫描：

1. `code/dsl`
2. `common/dsl`

它只返回实际存在的 `rulechains`、`endpoints`、`shared` 子目录，并按输入 root 顺序去重。

## 4. Report-Only CLI

`cmd/matrix-validate` 是当前 report-only 验证入口：

```bash
go run ./cmd/matrix-validate --module-root <module-repo-root>
```

默认行为：

1. `--module-root` 默认为当前目录。
2. 默认 DSL root 为 `code/dsl` 与 `common/dsl`。
3. 可重复传入 `--dsl-root <path>` 覆盖默认 DSL root。
4. 输出 `ValidationReport` JSON 到 stdout。
5. 不实例化 node，不调用 `Node.Init(...)`，不注册 endpoint trigger，不影响 startup。

当前覆盖：

| issue code | severity | 触发条件 |
| --- | --- | --- |
| `loader_failure` | `error` | JSON 文件读取或解析失败。 |
| `dangling_connection` | `error` | rulechain connection 的 `fromId` 或 `toId` 找不到对应 node。 |
| `missing_endpoint_target` | `error` | endpoint 的 `configuration.ruleChainId` 不在已加载 rulechain ID 集合中。 |
| `missing_shared_ref` | `error` | node / endpoint configuration 中的 `ref://...` 找不到 shared node ID。 |
| `optional_fallback` | `warning` | `ref://...` 找不到 shared node ID，但同一配置对象声明了 `optional: true`、`fallback`、`fallbackUri`、`fallbackURI`、`default` 或 `defaultValue`。 |

该 scanner 不会实例化 node，不会调用 `Node.Init(...)`，也不会注册 runtime trigger。它用于在 Stage 0.5 先建立 report-only 输出，后续再决定是否接入启动流程和 strict mode。

## 5. InspectionSnapshot

`pkg/inspection.InspectionSnapshot` 的 JSON 输出字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `schemaVersion` | string | 输出 schema 版本；默认值是 `matrix.inspection.snapshot/v1`。 |
| `engineId` | string | 可选 engine 标识，由宿主或调用方提供。 |
| `ruleChains` | array | rulechain 事实描述。 |
| `runtimes` | array | runtime 事实描述。 |
| `endpoints` | array | endpoint 事实描述。 |
| `sharedResources` | array | shared resource 事实描述。 |
| `functions` | array | function 事实描述。 |
| `validation` | object | 可选 validation report。 |

`RuntimeFactDescriptor` 当前字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `kind` | string | `rulechain`、`runtime`、`endpoint`、`function` 或 `shared_resource`。 |
| `id` | string | 稳定事实 ID。 |
| `name` | string | 可选展示名。 |
| `type` | string | DSL node type、endpoint type 或事实分类。 |
| `sourcePath` | string | 可选 DSL 或配置来源路径。 |
| `refs` | array | 可选依赖引用，例如 target rulechain 或 `ref://...`。 |
| `metadata` | object | 可选补充元数据。 |

## 6. 当前限制

1. 当前 loader scanner 只做静态 JSON / DSL 结构扫描，不做 node type registry、function routing、endpoint IO contract 或 DAG cycle validation。
2. 当前不保证 `details` / `metadata` 内部字段稳定；消费者应优先依赖顶层字段和 issue code。
3. Morpheus 仍未迁移到 inspection API，本模型只是后续 Stage 6 的输入契约基础。
4. strict validation 还未打开；Stage 0.5 后续切片需要继续接入 rulechain validator 和 startup pipeline。

## 7. 验证

当前聚焦测试：

```bash
go test ./pkg/validation ./pkg/inspection ./pkg/runtimebridge ./cmd/matrix-validate
```
