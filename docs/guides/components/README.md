---
uuid: "a2b1c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
type: "Reference"
title: "README: Matrix 组件文档编写规范"
status: "Stable"
owner: "neohetj"
version: "1.1.0"
tags:
  - "matrix"
  - "component"
  - "documentation"
  - "guide"
  - "readme"
relations:
  - type: "is_part_of"
    target_uuid: "b4c5d6e7-f8a9-0b1c-2d3e-4f5a6b7c8d9e"
    description: "本规范属于 Matrix 指南文档体系。"
  - type: "references"
    target_uuid: "245d6065-8675-4614-8d7a-d3c97afb72f8"
    description: "组件指南模板与占位规则由 templates README 统一说明。"
---

# Matrix 组件文档编写规范

本目录存放可复用组件与节点的详细使用指南。共享规则见 [Matrix 文档总览](../../README.md)，模板规则见 [模板库说明](../../templates/README.md)。

## 1. 组件指南特定规则

1. 文件名应遵循 `<component_type>_<component_name>_guide.md`。
2. 组件指南应从 [node_guide_template.md](../../templates/node_guide_template.md) 起草。
3. 每份组件指南都应明确说明：
   - 功能概述
   - 关键配置项
   - 输入输出或数据契约
   - 错误处理与常见问题
4. 与组件实现直接相关的源码路径、DSL 类型与关系名，应尽量写到文档里，便于审计和搜索。

## 2. 文档索引

### Action

- [action_aggregator_guide.md](./action_aggregator_guide.md)
- [action_expr_switch_guide.md](./action_expr_switch_guide.md)
- [action_flow_guide.md](./action_flow_guide.md)
- [action_for_each_guide.md](./action_for_each_guide.md)
- [action_functions_guide.md](./action_functions_guide.md)
- [action_log_guide.md](./action_log_guide.md)

### Ops

- [ops_topology_nodes_guide.md](./ops_topology_nodes_guide.md)

### Endpoint

- [endpoint_http_guide.md](./endpoint_http_guide.md)
- [endpoint_inotify_guide.md](./endpoint_inotify_guide.md)
- [endpoint_redis_stream_guide.md](./endpoint_redis_stream_guide.md)

### External

- [external_db_client_guide.md](./external_db_client_guide.md)
- [external_http_client_guide.md](./external_http_client_guide.md)
- [external_redis_client_guide.md](./external_redis_client_guide.md)

### Functions

- [functions_parse_validate_guide.md](./functions_parse_validate_guide.md)
- [functions_redis_command_guide.md](./functions_redis_command_guide.md)
- [functions_sql_query_guide.md](./functions_sql_query_guide.md)
- [functions_transaction_management_guide.md](./functions_transaction_management_guide.md)
