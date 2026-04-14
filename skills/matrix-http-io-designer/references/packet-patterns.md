# Packet Patterns

## Explicit Field Mapping First

优先写成显式字段映射：

```yaml
request:
  fields:
    - bindPath: query.page
      to: rulemsg://dataT/pagination.page?sid=Int
    - bindPath: query.pageSize
      to: rulemsg://dataT/pagination.page_size?sid=Int
```

这种写法最适合：

- query/path/header 到内部 typed object
- patch 对象构造
- 外部 DTO 到内部对象

## `mapAll` Use Cases

只在这些场景考虑 `mapAll`：

- 源和目标语义一致
- 对象形状稳定
- 不是跨业务 SID 的 typed whole-object 转换

一旦出现下面任一情况，就回退到显式字段映射：

- `Patch`
- `MapStringInterface`
- 列表包装对象
- 内外部对象字段名不同
- 只需要其中一部分字段

## Response Pattern

返回给 HTTP 的对象优先是“对外契约对象”，而不是内部暂存对象。必要时用 `transform/object_mapper` 或函数节点先组装响应对象，再交给 `endpoint/http` 输出。
