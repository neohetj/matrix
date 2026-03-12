# Design Output Contract

开始改 DSL 之前，先产出下面这些设计结果。

## A. Entry Decision

- 这是新增入口还是扩展已有入口？
- 入口文件预计落在哪个目录？

## B. Rulechain Plan

按链路列出：

- Rulechain id
- 职责
- 上游输入
- 下游输出
- 是否复用已有 chain

## C. Stage Plan

如果是 pipeline，至少列出：

- stage 名称
- processor chain id
- 输入 channel
- 输出 channel
- 并发和 buffer 的选择理由

## D. Object Mapping Table

至少列出关键对象：

- `objId`
- `SID`
- 来源节点
- 去向节点
- 是否持久化后继续流转

## E. Node Inventory

- 复用节点
- 新增 DSL 节点
- 需要新增的 Go function 节点
- 需要新增的 prompt / shared 资源

## F. Risk List

- 契约风险：SID、函数签名、跨链对象引用
- 运行风险：异步顺序、失败链、数据缺失
- 验证计划：准备跑哪些命令和 trace
