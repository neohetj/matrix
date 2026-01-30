# How to Add a Basic SID (Semantic ID)

This guide outlines the steps required to add a new basic Semantic ID (SID) to the Matrix framework. A Basic SID represents a fundamental data type (e.g., String, Int64, SliceAny) used within the system's data contract.

## Steps

### 1. Define Constant

Add the new SID constant string in `Matrix/pkg/cnst/constant.go`.

```go
// Matrix/pkg/cnst/constant.go

const (
    // ... existing constants
    SID_SLICE_ANY = "SliceAny" // Example
)
```

### 2. Implement Data Handling

Update `trySetBodyBySID` in `Matrix/pkg/utils/datat.go` to handle type coercion and assignment for the new SID.

```go
// Matrix/pkg/utils/datat.go

func trySetBodyBySID(obj types.CoreObj, value any, sid string) (bool, error) {
    switch sid {
    // ... existing cases
    case cnst.SID_SLICE_ANY:
        // Handle pointer to slice or value assignment
        if v, ok := value.([]any); ok {
            return true, obj.SetBody(&v)
        } else if v, ok := value.(*[]any); ok {
            return true, obj.SetBody(v)
        }
    }
    return false, nil
}
```

### 3. Register Definition

Register the default `CoreObjDef` for the new SID in `Matrix/internal/registry/registry.go`. This allows the system to recognize the SID and create instances of it.

```go
// Matrix/internal/registry/registry.go

func init() {
    // ...
    Default.CoreObjRegistry.Register(
        // ...
        contract.NewDefaultCoreObjDef(
            []any{},            // Prototype value (zero value)
            cnst.SID_SLICE_ANY, // SID Constant
            "Basic Type: Slice of Any", // Description
        ),
    )
}
```

### 4. Add Unit Tests

Ensure correct behavior by adding unit tests.

*   **Utils Test**: Add a test case in `Matrix/pkg/utils/datat_test.go` (create if not exists) to verify `SetCoreObjBody` logic.

*   **Asset Test**: Add a test case in `Matrix/pkg/asset/rulemsg_asset_set_test.go` to verify integration with `RuleMsg` data access.

```go
// Example Test Case
func TestRuleMsgAssetSet_DataTSliceAnyTypes(t *testing.T) {
    // ... setup context ...
    anySlice := []any{"a", 123, true}
    sliceAsset := asset.Asset[any]{URI: "rulemsg://dataT/mixed?sid=" + cnst.SID_SLICE_ANY}
    err := sliceAsset.Set(ctx, anySlice)
    assert.NoError(t, err)
    // ... assertions ...
}
```
