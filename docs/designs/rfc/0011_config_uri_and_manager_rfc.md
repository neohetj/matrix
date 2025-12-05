# RFC 0011: Config URI 协议与规则链统一配置视图

## 概要
本文档旨在规范化节点配置中的变量引用方式，引入专用的 `config://` URI 协议，将配置引用与数据引用 (`rulemsg://`) 分离。同时，定义了基于作用域（Scope）的配置解析逻辑，并设计了一个“规则链统一配置视图”，用于集中管理和配置规则链中分散的配置项。

## 1. 问题背景
1.  **语法模糊**：目前的 `PromptNode` 混合使用了 `${data...}` 和 `${config...}`，缺乏明确的命名空间区分。
2.  **解析逻辑僵化**：配置查找逻辑硬编码，缺乏灵活性。用户需要能够定义回退策略（例如：“优先使用节点配置，如果不存在则使用引擎全局配置”）。
3.  **配置分散**：在复杂的规则链中，关键配置参数（如 Prompt 模板、模型 ID、API Key 等）分散在各个节点中。用户缺乏一个全局视图来统一查看和修改这些参数。

## 2. `config://` URI 协议设计

我们引入一个新的顶级协议 `config://`，专门用于配置项的注入和解析。

### 2.1 URI 格式
```
config://<path>?scope=<scope_list>&default=<default_value>
```

*   **path**: 配置项的路径，使用点号（`.`）分隔层级。这直接映射到 JSON/YAML 配置文件的嵌套结构。
    *   例如：`llm.openai.api_key` 对应如下结构：
        ```yaml
        llm:
          openai:
            api_key: "sk-..."
        ```
*   **scope**: (可选) 定义查找顺序的逗号分隔列表。
    *   `node`: 在当前节点实例的配置中查找（通常是节点的 `Variables` 或自定义配置区）。
    *   `engine`: 在全局引擎配置 (`BizConfig`) 中查找。
    *   `env`: 在系统环境变量中查找。
    *   *默认值*: `node,engine` (即优先查节点，查不到查引擎)。
*   **default**: (可选) 如果在所有作用域中都未找到，则使用的默认值。

### 2.2 使用示例
*   **基础用法**: `${config://llm.model}` (使用默认作用域 `node,engine`)
*   **指定回退策略**: `${config://llm.token?scope=node,engine,env}`
    1.  首先检查节点配置中是否存在 `llm` 对象及其 `token` 字段。
    2.  如果不存在，检查全局引擎配置中的 `llm.token`。
    3.  如果仍不存在，检查名为 `llm_token` (自动转换命名规范) 的环境变量。
*   **仅环境变量**: `${config://SECRET_KEY?scope=env}`

## 3. 配置作用域与存储

### 3.1 节点作用域 (Node Scope)
*   **定义**: 仅对当前节点实例生效的配置。
*   **实现**: 为了支持任意键值的“节点级覆盖”，建议在节点的配置结构体（如 `PromptNodeConfiguration`）中增加一个通用的 `Variables` 字段。
    *   结构示例：
        ```go
        type PromptNodeConfiguration struct {
            // ... 原有字段
            Variables map[string]any `json:"variables,omitempty"`
        }
        ```
    *   当解析 `${config://llm.model}` 时，解析器会尝试在 `Variables` map 中查找 key 为 `llm.model` 的值（或支持嵌套查找）。

### 3.2 引擎作用域 (Engine Scope)
*   **定义**: 整个应用或运行时共享的配置，通常由 `config.yaml` 加载。
*   **访问**: 通过 `ctx.GetRuntime().GetEngine().BizConfig()` 获取。这是一个多层级的 Map 结构，支持通过点号路径（如 `llm.token`）进行深层访问。

### 3.3 环境变量作用域 (Env Scope)
*   **定义**: 操作系统级别的环境变量。

## 4. 规则链统一配置视图 (RuleChain Configuration View)

为了解决“配置分散”的问题，将在规则链管理界面增加一个“配置视图”或“配置概览”面板。

### 4.1 功能概念
该视图自动扫描规则链中所有节点，识别出所有使用的 `config://` 变量，并提供一个集中的表格来查看和管理这些值。

### 4.2 工作流
1.  **扫描 (Scan)**: 后端（或前端）遍历规则链的 JSON 定义。
2.  **提取 (Extract)**: 解析所有节点配置中的字符串字段，正则匹配 `${config://...}` 模式。
3.  **聚合 (Aggregate)**: 汇总出该规则链依赖的所有配置 Key（去重）。
4.  **展示 (Display)**:
    *   **配置项 (Key)**: 例如 `llm.temperature`。
    *   **来源节点**: 显示哪些节点使用了该配置（例如：“PromptNode A, CheckNode B”）。
    *   **当前值**:
        *   **引擎值**: 显示全局配置中的默认值。
        *   **节点覆盖值**: 如果某个节点定义了覆盖值，在此处高亮显示。
    *   **操作**: 提供“编辑节点覆盖值”的入口。

### 4.3 界面布局示意

**标题**: 规则链配置概览

| 配置键 (Key) | 引擎默认值 (Global) | 节点级覆盖 (Node Override) | 来源节点 |
| :--- | :--- | :--- | :--- |
| `llm.model` | `gpt-3.5-turbo` | `gpt-4` (仅 Node A) | Node A (GenAI), Node B (Review) |
| `llm.token` | `******` | - | Node A, Node B |
| `retry.count` | `3` | `5` | Node C (Http) |

> **注意**: 在此视图修改“节点级覆盖”，本质上是批量修改对应节点的 `Variables` 字段。

## 5. 实现计划

### 5.1 后端 (`Matrix`)
1.  **URI 解析器**: 在 `pkg/helper` 或 `pkg/config` 中实现 `ConfigURIResolver`。
    *   需传入 `NodeContext` 以访问 Engine 配置。
    *   需传入节点配置对象（Map形式）以访问 Node 作用域。
    *   实现基于点号 (`.`) 的嵌套 Map 查找逻辑（用于支持 `llm.token` 这种多层级结构）。
2.  **字符串工具**: 升级 `ReplacePlaceholders`，使其支持传入自定义的回调函数进行变量解析。

### 5.2 前端 / API
1.  **配置提取 API**: 新增接口，接收规则链 JSON，返回其依赖的 `config://` 键列表。
    *   `POST /api/chains/analyze-config`
2.  **配置页面**: 实现 4.3 描述的聚合视图。

### 5.3 节点适配
1.  重构 `PromptNode` (及其他需要配置注入的节点) 的 `OnMsg` 方法，对接新的解析逻辑。
2.  在节点配置 Schema 中增加 `variables` 字段，用于存储节点级的配置覆盖。
