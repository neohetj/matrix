---
# === Node Properties: 定义文档节点自身 ===
uuid: "f9e8d7c6-b5a4-4993-8262-3c1d0a9b8e7f"
type: "ComponentGuide"
title: "组件指南：Inotify文件监听端点 (endpoint/inotify)"
status: "Draft"
owner: "@cline"
version: "1.0.0"
tags:
  - "matrix"
  - "component"
  - "endpoint"
  - "file"
  - "inotify"

# === Node Relations: 定义与其他文档节点的关系 ===
relations:
  - type: "is_part of"
    target_uuid: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"
    description: "本节点是Matrix规则链的主动型入口点之一。"
  - type: "references"
    target_uuid: "81080378-a3e9-41ee-86ed-807193d45bce"
    description: "本文档遵循语义化文档规范编写。"
---

# 1. 功能概述 (FunctionalOverview)

`endpoint/inotify` 是一个**主动型 (Active)** 的端点节点。与被动等待外部请求的 `http` 端点不同，`inotify` 端点在启动后会主动监听指定的文件系统路径，当该路径下发生文件事件（如创建、写入、删除等）时，它会自动读取文件内容，将其封装成一个 `RuleMsg`，并触发一个指定的规则链。

这使 `Matrix` 具备了与文件系统进行事件驱动式集成的能力，非常适用于处理日志文件、配置文件变更、数据文件落地等场景。

# 2. 如何配置 (Configuration)

| 配置键 (ID) | 名称 | 描述 | 类型 | 是否必须 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `ruleChainId` | 规则链ID | 当文件事件发生时，要触发的规则链的ID。 | `string` | 是 | N/A |
| `startNodeId` | 起始节点ID | (可选) 指定规则链从哪个节点开始执行。**如果为空，则会从所有入度为0的节点并行开始执行**。<br/><br/>**⚠️ 警告**: 并行执行可能引发资源竞争问题，**强烈建议总是明确指定一个起始节点**。 | `string` | 否 | `""` |
| `path` | 监听路径 | 要监听的文件或目录的绝对或相对路径。 | `string` | 是 | N/A |
| `events` | 监听事件类型 | (可选) 一个字符串数组，用于过滤监听的事件类型。如果为空或不提供，则监听所有事件。 | `string[]` | 否 | `[]` |
| `description` | 描述 | 对该端点功能的简短描述。 | `string` | 否 | `""` |

### `events` 可用值

`events` 数组中的值不区分大小写。可用的事件类型包括：
*   `CREATE`: 文件或目录被创建。
*   `WRITE`: 文件被写入。
*   `REMOVE`: 文件或目录被删除。
*   `RENAME`: 文件或目录被重命名。
*   `CHMOD`: 文件或目录的权限被修改。

# 3. 配置示例 (Example)

假设我们需要监听 `/var/log/app/` 目录下所有新创建或被写入的日志文件，并由 `rc-log-processing` 规则链进行处理。

```json
{
  "id": "ep-log-watcher",
  "type": "endpoint/inotify",
  "name": "监听应用日志文件",
  "configuration": {
    "ruleChainId": "rc-log-processing",
    "startNodeId": "node-parse-log",
    "path": "/var/log/app/",
    "events": [
      "CREATE",
      "WRITE"
    ],
    "description": "监听应用日志目录，当有新日志写入时触发处理链。"
  }
}
```
**流程解析**:
1.  `Matrix` 启动时，该节点会开始监听 `/var/log/app/` 目录。
2.  当一个新文件（如 `app-2025-09-09.log`）在该目录下被创建 (`CREATE`) 或被写入 (`WRITE`) 时，该节点会被激活。
3.  节点会立即读取该文件的**全部内容**。
4.  然后，它会创建一个 `RuleMsg`，并送往 `rc-log-processing` 规则链的 `node-parse-log` 节点开始执行。

# 4. 数据契约 (DataContract)

当一个文件事件被触发时，`inotify` 节点会创建一个新的 `RuleMsg`，其结构如下：

*   **`Data`**:
    *   **类型**: `string`
    *   **内容**: 被触发事件的文件的**完整文本内容**。
*   **`DataFormat`**:
    *   **值**: `TEXT`
*   **`Metadata`**:
    *   `source`: 固定为 `"inotify"`。
    *   `path`: 触发事件的文件的完整路径 (e.g., `/var/log/app/app-2025-09-09.log`)。
    *   `event`: 触发的事件类型的大写字符串 (e.g., `"WRITE"`)。
    *   `filename`: 不包含路径的文件名 (e.g., `app-2025-09-09.log`)。

# 5. 错误处理 (ErrorHandling)

作为一个主动型、长时间运行的端点，`inotify` 节点主要通过**日志**来报告其在运行过程中遇到的问题，例如：
*   监听的路径不存在或无权限访问。
*   读取事件文件失败。
*   找不到目标 `ruleChainId` 对应的规则链。
*   启动规则链执行失败。

这些错误通常不会导致 `Matrix` 进程崩溃，但会阻止该次文件事件被正确处理。

# 6. 问答环节 (FrequentlyAskedQuestions)
<!-- qa_section_start -->
> **问：`inotify` 端点能保证不丢事件吗？**
> **答：** `inotify` 依赖于操作系统底层的通知机制，在正常情况下是可靠的。但在极端情况下（如短时间内产生大量文件事件，超出系统内核队列的缓冲能力），事件可能会丢失。因此，它适用于准实时的文件处理场景，但不应用于要求100%数据不丢失的严格场景。
<!-- qa_section_end -->

<!-- 链接定义区域 -->
[Guide-MatrixOverview-2b3c4d]: ../00_matrix_guide.md
[Ref-SemanticDoc-d45bce]: ../../reference/04_semantic_documentation_standard.md
