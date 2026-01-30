# 如何增加基础 SID (语义 ID)

本文档指南说明了如何在 Matrix 框架中增加一个新的基础语义 ID (SID)。基础 SID 代表系统数据契约中使用的基本数据类型（例如 String, Int64, SliceAny）。

## 步骤

### 1. 定义常量

在 `Matrix/pkg/cnst/constant.go` 中添加新的 SID 常量字符串。

```go
// Matrix/pkg/cnst/constant.go

const (
    // ... existing constants
    SID_SLICE_ANY = "SliceAny" // 示例
)
```

### 2. 实现数据处理逻辑

在 `Matrix/pkg/utils/datat.go` 中更新 `trySetBodyBySID` 函数，以处理新 SID 的类型转换和赋值逻辑。

```go
// Matrix/pkg/utils/datat.go

func trySetBodyBySID(obj types.CoreObj, value any, sid string) (bool, error) {
    switch sid {
    // ... existing cases
    case cnst.SID_SLICE_ANY:
        // 处理切片值或指针赋值
        if v, ok := value.([]any); ok {
            return true, obj.SetBody(&v)
        } else if v, ok := value.(*[]any); ok {
            return true, obj.SetBody(v)
        }
    }
    return false, nil
}
```

### 3. 注册定义

在 `Matrix/internal/registry/registry.go` 中为新的 SID 注册默认的 `CoreObjDef`。这使得系统能够识别该 SID 并创建其实例。

```go
// Matrix/internal/registry/registry.go

func init() {
    // ...
    Default.CoreObjRegistry.Register(
        // ...
        contract.NewDefaultCoreObjDef(
            []any{},            // 原型值（零值）
            cnst.SID_SLICE_ANY, // SID 常量
            "基本类型：任意类型切片", // 描述
        ),
    )
}
```

### 4. 添加单元测试

通过添加单元测试确保行为正确。

*   **Utils 测试**: 在 `Matrix/pkg/utils/datat_test.go`（如不存在则创建）中添加测试用例，验证 `SetCoreObjBody` 逻辑。

*   **Asset 测试**: 在 `Matrix/pkg/asset/rulemsg_asset_set_test.go` 中添加测试用例，验证与 `RuleMsg` 数据访问的集成。

```go
// 测试用例示例
func TestRuleMsgAssetSet_DataTSliceAnyTypes(t *testing.T) {
    // ... setup context ...
    anySlice := []any{"a", 123, true}
    sliceAsset := asset.Asset[any]{URI: "rulemsg://dataT/mixed?sid=" + cnst.SID_SLICE_ANY}
    err := sliceAsset.Set(ctx, anySlice)
    assert.NoError(t, err)
    // ... assertions ...
}
```
