# Requirement Template

把原始需求先整理成下面的结构，再进入 DSL 设计：

## 1. Goal

- 这条需求最终要让系统做到什么？

## 2. Entry

- 入口类型：`endpoint/http`、`endpoint/pipeline`、已有 rulechain 扩展、定时触发
- 入口输入：请求参数、消息体、上游对象、外部事件

## 3. Outputs

- 最终响应给谁？
- 输出对象是什么？
- 是否需要推送到 channel 或下游 pipeline？

## 4. Data And Persistence

- 关键业务对象有哪些？
- 哪些对象必须先保存再继续流转？
- 哪些对象只是中间态？

## 5. External Dependencies

- 是否依赖 LLM、抓取器、数据库、第三方 HTTP 服务、shared resource？

## 6. Failure Semantics

- 失败时是直接返回错误、写失败状态、重试，还是走专门失败链？

## 7. Async Boundaries

- 哪些步骤需要拆成 pipeline stage？
- 哪些步骤必须串行？

## 8. Acceptance

- 完成后最小可验证结果是什么？
- 需要哪些命令、测试、trace 证明链路正确？
