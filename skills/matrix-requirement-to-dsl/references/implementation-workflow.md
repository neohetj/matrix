# Implementation Workflow

## 1. Pick A Baseline

- 先找同域、同入口类型、同数据流形态的现有 DSL。
- 优先复制结构，再最小化修改。
- 如果是同步查询类 list 接口，先确认基线是否已经遵守 skill 主文档里的 list contract，不要继承旧接口的资源专属返回结构。

## 2. Change The Smallest Set Of Files

- 只改真正需要的 endpoint、pipeline、rulechain、prompt、shared 文件。
- 如果需要新增 Go function，和 DSL 一起补，不要留下悬空引用。

## 3. Update References Immediately

- 新增入口时，立刻确认引用的 rulechain 已存在。
- 新增 channel push 时，立刻确认目标 pipeline 已存在。
- 新增 prompt 时，立刻确认模板文件路径正确。

## 4. Keep Design And Runtime Aligned

- 设计阶段认定的 `objId/SID` 映射，改文件时不要随意漂移。
- 如果实现过程中发现设计不成立，先回到设计草案修正，再继续改 DSL。

## 5. Validate Before Claiming Done

- 至少完成语法、引用、契约三类校验。
- 如果新增或改造了 list endpoint，额外核对 `GET + page/pageSize + data/total [+ meta]` 是否成立。
- 如果能触发真实链路，再补一轮 trace 验证。
