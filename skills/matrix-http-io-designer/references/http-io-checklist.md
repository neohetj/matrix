# HTTP IO Checklist

设计或审查 `endpoint/http`、`external/httpClient` 时，按这个顺序过一遍：

1. 请求边界
   - query 参数有哪些
   - path 参数有哪些
   - header 里是否有业务字段
   - body 是整对象还是字段子集
2. 内部对象边界
   - 每个输入字段落到哪个 `rulemsg://dataT/...`
   - 是否真的需要 new SID
   - 是否存在 patch / map / typed whole-object 风险
3. 响应边界
   - body 返回哪个对象
   - `status` 是否固定
   - 是否需要 headers/meta
4. packet 设计
   - 默认优先 `fields`
   - `mapAll` 只在同形状整体透传时使用
   - 集合 SID 不写 `"type": "object"`
5. 验证
   - 跑 `matrix-rulechain-validator`
   - 行为有变化时补请求/响应验证

同步查询类 list 接口默认再加一轮人工检查：

- `GET`
- `page`
- `pageSize`
- `data`
- `total`
- `meta` 仅放分页无关信息
