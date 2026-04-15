# Rulechain Data Flow Smells

## 1. 伪输入保活

症状：

- 函数节点上挂了一个函数定义里根本没有的输入参数
- 只是为了让某个对象继续出现在链路里

处理：

- 删除伪输入
- 重新设计真正的入口对象和中间对象保留点

## 2. alias copy mapper

症状：

- `transform/object_mapper` 只有一个字段
- 只是把同一个 SID 的 whole-object 从一个 objId 搬到另一个 objId

处理：

- 优先删除 mapper
- 让下游节点直接读取原对象

## 3. stage 回推同一 channel

症状：

- Pipeline stage 里把结果再手动推回当前 `outputChannel`
- 出现重复处理或顺序混乱

处理：

- 让 stage 只负责当前阶段输出
- 需要额外分支时改 stage 设计，不要回推原 channel

## 4. 共享资源命名空间泄漏

症状：

- 流程已经依赖模块私有 shared ref
- rulechain / endpoint / shared 顶层 ID 仍然使用通用命名

处理：

- 将相关 ID 一并切换到模块命名空间
- 避免多个模块落地后互相碰撞
