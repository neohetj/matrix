# Acceptance Checklist

## Before Editing

- 需求已经整理成结构化模板
- 设计草案已经列出入口、chain、stage、对象映射和风险
- 已经找到最近似的现有 DSL 基线

## After Editing

- JSON 可以解析
- 跨文件引用完整
- `SID` 和函数签名一致
- 没有跨 `SID` 整对象透传
- `Patch` 映射是逐字段的
- 没有引入伪输入保活对象

## Validation

- 运行项目内 DSL validator
- 运行项目内测试或最小相关测试
- 如果有真实执行链，检查 trace 是否符合预期

## Final Output

最终说明至少包含：

- 需求如何映射成 DSL
- 改了哪些文件
- 复用了哪些现有链路或节点
- 还存在哪些风险或后续验证项
