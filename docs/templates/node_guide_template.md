---
uuid: "GENERATED_UUID"
type: "ComponentGuide"
title: "[节点名称] 使用指南"
status: "Draft"
owner: "@developer-name"
version: "1.0.0"
tags:
  - "component"
  - "guide"
  - "[功能分类标签, e.g., database, action]"
relations:
  - type: "is_governed_by"
    target_uuid: "a2c8d4e1-7b3e-4c2a-8f5d-9e1b3c4d5a6b" # -> Matrix节点/组件开发SOP
  - type: "relates_to"
    target_uuid: "[UUID of a related concept doc]"
    description: "This node implements the concepts described in the related document."
---

# 如何使用 [节点名称] 节点

## 1. 功能概述 (Overview)

*（用一两句话概括这个节点的核心功能。例如：“本节点用于向指定的数据库表中插入一条记录。”）*

## 2. 配置参数 (Configuration)

*（使用表格详细列出该节点的所有配置参数。）*

| 参数ID | 名称 | 类型 | 是否必需 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `dbRef` | 数据库引用 | `string` | 是 | 指向一个共享数据库连接节点的引用，格式为`ref://...`。 | |
| `tableName` | 表名 | `string` | 是 | 需要插入数据的数据库表名。 | |
| `timeout` | 超时（毫秒）| `integer` | 否 | 数据库操作的超时时间。 | `5000` |

## 3. 输入 (Input)

*（描述该节点期望从输入的`RuleMsg`中获取什么数据。）*

本节点期望输入的`RuleMsg`的`DataT`容器中，包含一个符合特定结构的对象。该对象的字段将用于构建插入数据库的记录。

**示例输入 `DataT` 对象:**
```json
{
  "user_to_insert": {
    "name": "John Doe",
    "email": "john.doe@example.com",
    "status": "active"
  }
}
```

## 4. 输出 (Output)

*（描述该节点处理成功或失败后，会对`RuleMsg`产生什么影响。）*

*   **成功 (Success)**:
    *   节点会将成功插入记录的ID，添加到`RuleMsg`的`DataT`中。
    *   **示例输出 `DataT` 对象:**
        ```json
        {
          "user_to_insert": { ... },
          "inserted_record_id": 12345
        }
        ```
    *   然后通过`Success`路径将消息传递下去。

*   **失败 (Failure)**:
    *   如果数据库操作失败，节点不会修改`RuleMsg`。
    *   它会将具体的数据库错误信息包装后，通过`Failure`路径传递下去。

## 5. 使用示例 (Example)

*（提供一个完整的、可直接复制使用的规则链JSON片段，展示如何配置和使用该节点。）*

```json
{
  "id": "my_insert_chain",
  "nodes": [
    {
      "id": "my_db_connection",
      "type": "database/mysql",
      "name": "MySQL Connection",
      "configuration": {
        "dsn": "user:password@tcp(127.0.0.1:3306)/mydb"
      }
    },
    {
      "id": "insert_user_node",
      "type": "[your_node_type]",
      "name": "Insert User Record",
      "configuration": {
        "dbRef": "ref://my_db_connection",
        "tableName": "users"
      }
    }
  ],
  "connections": [
    {
      "from": "source_node_id",
      "to": "insert_user_node",
      "type": "Success"
    }
  ]
}
```

## 6. 常见问题与解答 (FAQ)

<!-- qa_section_start -->
> **问：如果我需要插入的数据字段和数据库表字段不完全匹配怎么办？**
> **答：** 本节点目前只支持直接映射。对于复杂的字段映射或转换，你应该在本节点之前，增加一个“数据转换”节点（如`action/transform`），先将`RuleMsg`中的数据处理成符合数据库表结构的格式。
<!-- qa_section_end -->
