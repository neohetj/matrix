---
# === Node Properties: 定义文档节点自身 ===
uuid: "c8f2b1d9-e5a6-4c7b-8d9e-0f1a2b3c4d5e"
type: "ComponentGuide"
title: "组件指南：日志记录 (action/log)"
status: "Stable"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "action"
  - "log"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part_of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本指南是Matrix项目文档体系的一部分。"

---

# 1. 功能概述 (Overview)

`action/log` 节点用于在规则链执行过程中记录日志。它不仅支持输出静态文本，还集成了强大的表达式引擎，允许开发者动态提取消息上下文中的数据，并将其格式化到日志内容中。

该节点主要用于：
*   **调试 (Debugging)**: 在开发和测试阶段，打印中间数据以验证逻辑。
*   **审计 (Auditing)**: 记录关键业务流程的执行情况。
*   **监控 (Monitoring)**: 输出特定格式的日志供外部系统采集和分析。

# 2. 如何配置 (Configuration)

该节点在 DSL 的 `configuration` 块中支持以下参数：

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `level` | 日志级别 | 指定输出日志的级别。可选值：`debug`, `info`, `warn`, `error`。 | string | 否 | `info` |
| `message` | 消息模板 | 日志消息的格式化模板。支持 Go 语言 `fmt.Sprintf` 的占位符（如 `%s`, `%d`, `%v`, `%+v`）。 | string | 是 | - |
| `args` | 参数列表 | 一个表达式字符串列表。每个表达式都会在当前消息上下文中被求值，结果将按顺序填充到 `message` 的占位符中。 | []string | 否 | `[]` |

### 配置示例 (Configuration Example)

```json
{
  "id": "node-log-example",
  "type": "action/log",
  "name": "记录设备状态",
  "configuration": {
    "level": "INFO",
    "message": "收到设备 %s (ID: %s) 的数据，温度: %.2f，原始数据: %+v",
    "args": [
      "data.deviceName",
      "metadata.deviceId",
      "data.temperature",
      "data"
    ]
  }
}
```

# 3. 数据契约 (DataContract)

`action/log` 节点本身不修改消息流中的数据，它以只读方式访问消息上下文。

### 输入上下文 (Input Context)

节点在执行时会构建一个包含当前消息数据的上下文环境 (`env`)，供 `args` 中的表达式使用。环境包含以下顶层变量：

| 变量名 | 类型 | 描述 | 示例表达式 |
| :--- | :--- | :--- | :--- |
| `data` | any | 消息体数据。如果是 JSON 格式，会自动解析为 Map 或 Array；否则为原始字符串。 | `data.temperature` |
| `metadata` | map[string]string | 消息的元数据。 | `metadata.deviceId` |
| `dataT` | map[string]any | 结构化数据容器，包含所有已解析的 CoreObj。键为对象的 ID。 | `dataT.parsedAlert.summary` 或 `dataT["uuid"].summary` |

### 输出 (Output)

*   **Success**: 日志记录成功（或即使失败也会尝试继续），消息原样传递给下一个节点。

# 4. 错误处理 (ErrorHandling)

该节点主要关注日志记录，通常不会阻断规则链的执行。但在配置错误时会产生警告日志。

| 错误类型 | 描述 | 处理方式 |
| :--- | :--- | :--- |
| **表达式编译错误** | `args` 中的表达式语法错误，或引用了不存在的变量。 | 节点会捕获错误，在日志中打印警告，并将该参数位置替换为 `<!表达式!>` 形式的错误标记，继续执行。 |
| **格式化错误** | `message` 中的占位符与 `args` 计算出的值类型不匹配，或数量不一致。 | Go 的 `fmt.Sprintf` 会自动处理，通常会在日志中输出 `%!v(MISSING)` 或类似标记。 |

# 5. 问答环节 (FAQ)

<!-- qa_section_start -->

### Q: 如何访问 ID 为 UUID 格式的 dataT 对象？
**A:** 如果 `dataT` 中的对象 ID 是 UUID（例如 `4D034A28...`）或以数字开头，直接使用点号访问（如 `dataT.4D034...`）会导致表达式解析错误（`bad number syntax`）。
**解决方法**：请使用方括号和引号的语法来访问，例如：
```
dataT["4D034A28-DE78-4395-BFC2-5130D8DC182A"].summary
```

### Q: 如果我不配置 `args` 会发生什么？
**A:** 如果未配置 `args`，节点将直接输出 `message` 的内容，不会进行任何格式化填充。这意味着 `message` 中的 `%s` 等占位符将原样输出。

### Q: 支持哪些表达式语法？
**A:** 节点使用的是 `expr` 语言引擎。它支持常见的算术运算、逻辑运算、字符串操作以及访问 Map/Struct 的字段。详细语法可参考 [expr-lang 文档](https://expr-lang.org/)。

<!-- qa_section_end -->
