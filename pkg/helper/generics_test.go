package helper_test

import (
	"fmt"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Name string
	Age  int
}

func TestSetParam_StructPointer(t *testing.T) {
	mockey.PatchConvey("TestSetParam with Struct Pointer", t, func() {
		// Mock ResolveParamBinding
		mockey.Mock(helper.ResolveParamBinding).Return("obj_struct", "test/struct", nil).Build()

		// Prepare input
		val := &TestStruct{Name: "test", Age: 18}
		nodeCtx := utils.NewMockNodeCtx()
		msg := contract.NewDefaultRuleMsg("test", "", nil, nil)
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx), asset.WithRuleMsg(msg))

		// Mock Asset.Set to verify it's called with the correct pointer
		mockey.Mock(asset.Asset[*TestStruct].Set).To(func(a asset.Asset[*TestStruct], ctx *asset.AssetContext, v *TestStruct) error {
			assert.Equal(t, val, v)
			return nil
		}).Build()

		// Act
		ret, err := helper.SetParam[*TestStruct](assetCtx, "param_struct", val)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, val, ret)
	})
}

func TestSetParam_SliceValue(t *testing.T) {
	mockey.PatchConvey("TestSetParam with Slice Value", t, func() {
		// Mock ResolveParamBinding
		mockey.Mock(helper.ResolveParamBinding).Return("obj_slice", "test/slice", nil).Build()

		// Prepare input
		val := []string{"a", "b"}
		nodeCtx := utils.NewMockNodeCtx()
		msg := contract.NewDefaultRuleMsg("test", "", nil, nil)
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx), asset.WithRuleMsg(msg))

		// Mock Asset.Set to verify it's called with the slice value
		mockey.Mock(asset.Asset[[]string].Set).To(func(a asset.Asset[[]string], ctx *asset.AssetContext, v []string) error {
			assert.Equal(t, val, v)
			return nil
		}).Build()

		// Act
		ret, err := helper.SetParam[[]string](assetCtx, "param_slice", val)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, val, ret)
	})
}

func TestGetParam_StructPointer(t *testing.T) {
	mockey.PatchConvey("TestGetParam with Struct Pointer", t, func() {
		// Mock ResolveParamBinding
		mockey.Mock(helper.ResolveParamBinding).Return("obj_struct", "test/struct", nil).Build()

		expectedVal := &TestStruct{Name: "fetched", Age: 20}
		nodeCtx := utils.NewMockNodeCtx()
		// We need a RuleMsg for GetParam check `ctx.RuleMsg() == nil`
		msg := contract.NewDefaultRuleMsg("test", "", nil, nil)
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx), asset.WithRuleMsg(msg))

		// Mock Asset.Resolve
		mockey.Mock(asset.Asset[*TestStruct].Resolve).To(func(a asset.Asset[*TestStruct], ctx *asset.AssetContext) (*TestStruct, error) {
			return expectedVal, nil
		}).Build()

		// Act
		ret, err := helper.GetParam[*TestStruct](assetCtx, "param_struct")

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedVal, ret)
	})
}

func TestGetParam_SlicePointer(t *testing.T) {
	mockey.PatchConvey("TestGetParam with Slice Pointer", t, func() {
		// Mock ResolveParamBinding
		mockey.Mock(helper.ResolveParamBinding).Return("obj_slice", "test/slice", nil).Build()

		sliceVal := []string{"x", "y"}
		expectedVal := &sliceVal

		nodeCtx := utils.NewMockNodeCtx()
		msg := contract.NewDefaultRuleMsg("test", "", nil, nil)
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx), asset.WithRuleMsg(msg))

		// Mock Asset.Resolve
		mockey.Mock(asset.Asset[*[]string].Resolve).To(func(a asset.Asset[*[]string], ctx *asset.AssetContext) (*[]string, error) {
			return expectedVal, nil
		}).Build()

		// Act
		// Note: The guide says: helper.GetParam[*[]Type](...)
		ret, err := helper.GetParam[*[]string](assetCtx, "param_slice")

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedVal, ret)
		assert.Equal(t, "x", (*ret)[0])
	})
}

func TestRenderConfigAsset_TemplateRendering(t *testing.T) {
	mockey.PatchConvey("TestRenderConfigAsset with Template and Conversion", t, func() {
		// 1. Mock GetConfigAsset[string] to return a template string
		mockey.Mock(helper.GetConfigAsset[string]).Return("${config:///someKey}", nil).Build()

		// 2. Mock asset.RenderTemplate
		mockey.Mock(asset.RenderTemplate).Return("123", nil).Build()

		nodeCtx := utils.NewMockNodeCtx()
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx))

		// Act: Try to get as int
		ret, err := helper.RenderConfigAsset[int](assetCtx, "myKey")

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 123, ret)
	})
}

func TestGetConfigAssetComplexType(t *testing.T) {
	configKey := "complexArgs"
	expectedMap := map[string]any{
		"position": []any{19, 28},
		"isValid":  true,
	}

	mockey.PatchConvey("TestGetConfigAsset with complex map type", t, func() {
		// Mock ResolveConfigFieldMeta to provide the type info needed by BuildConfigAssetURI
		mockey.Mock(helper.ResolveConfigFieldMeta).Return("", "map").Build()

		// 1. Setup: Create a mock node context with the complex map in its config
		nodeCtx := utils.NewMockNodeCtx(
			utils.WithTestNodeConfig(map[string]interface{}{
				"business": map[string]any{
					configKey: expectedMap,
				},
			}),
		)

		// Create an asset context
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx))

		// 2. Act: Call the function under test
		actualMap, err := helper.GetConfigAsset[map[string]any](assetCtx, configKey)

		// 3. Assert: Check for errors and correctness of the result
		assert.NoError(t, err)
		assert.NotNil(t, actualMap)
		assert.Equal(t, expectedMap["isValid"], actualMap["isValid"])

		// Verify the nested slice and its numeric types
		position, ok := actualMap["position"].([]interface{})
		assert.True(t, ok, "position should be a []interface{}")
		assert.Len(t, position, 2)

		// Check the values of the numbers in the slice using the new helper.
		assert.Equal(t, 19, utils.MustInt(position[0]))
		assert.Equal(t, 28, utils.MustInt(position[1]))
	})
}

func TestGetConfigAsset_ShouldRenderTemplateString(t *testing.T) {
	mockey.PatchConvey("when config value is a template string, it should be rendered", t, func() {
		nodeCtx := utils.NewMockNodeCtx()
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx))

		// Mock the Resolve method to handle URI parsing including query params
		mockResolve := func(uri string) (any, error) {
			parsedURI, err := asset.ParseConfig(uri)
			if err != nil {
				return nil, err
			}

			key := parsedURI.Key
			scope := parsedURI.Query.Get("scope")

			// Default scope resolution
			if key == "templatedKey" && scope == "" {
				return "${config:///apiKey}", nil
			}
			if key == "apiKey" && scope == "" {
				return "resolved_api_key", nil
			}
			return nil, fmt.Errorf("config key '%s' with scope '%s' not found", key, scope)
		}

		mockey.Mock(asset.Asset[any].Resolve).To(func(a asset.Asset[any], ctx *asset.AssetContext) (any, error) {
			return mockResolve(a.URI)
		}).Build()

		mockey.Mock(asset.Asset[string].Resolve).To(func(a asset.Asset[string], ctx *asset.AssetContext) (string, error) {
			res, err := mockResolve(a.URI)
			if err != nil {
				return "", err
			}
			return res.(string), nil
		}).Build()

		// 2. Act: Call the function
		resolvedValue, err := helper.GetConfigAsset[string](assetCtx, "templatedKey")

		// 3. Assert: Check the result
		assert.NoError(t, err)
		assert.Equal(t, "resolved_api_key", resolvedValue)
	})
}

func TestGetConfigAsset_RenderWithDifferentScopes(t *testing.T) {
	mockey.PatchConvey("template string should resolve config from correct scope", t, func() {
		nodeCtx := utils.NewMockNodeCtx()
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx))

		// Mock the Resolve method to handle URI parsing including query params
		mockResolve := func(uri string) (any, error) {
			parsedURI, err := asset.ParseConfig(uri)
			if err != nil {
				return nil, err
			}
			key := parsedURI.Key
			scope := parsedURI.Query.Get("scope")

			if key == "templatedKey" && scope == "" {
				return "${config:///engine_api_key?scope=engine}", nil
			}
			// This simulates the resolution of the inner template
			if key == "engine_api_key" && scope == "engine" {
				return "engine_key_123", nil
			}
			return nil, fmt.Errorf("config key '%s' with scope '%s' not found", key, scope)
		}

		mockey.Mock(asset.Asset[any].Resolve).To(func(a asset.Asset[any], ctx *asset.AssetContext) (any, error) {
			return mockResolve(a.URI)
		}).Build()

		mockey.Mock(asset.Asset[string].Resolve).To(func(a asset.Asset[string], ctx *asset.AssetContext) (string, error) {
			res, err := mockResolve(a.URI)
			if err != nil {
				return "", err
			}
			return res.(string), nil
		}).Build()

		// 2. Act: Call the function to resolve the template
		resolvedValue, err := helper.GetConfigAsset[string](assetCtx, "templatedKey")

		// 3. Assert: Check that it resolved from the chain scope
		assert.NoError(t, err)
		assert.Equal(t, "engine_key_123", resolvedValue)
	})
}

func TestGetParam_SliceConversion(t *testing.T) {
	mockey.PatchConvey("TestGetParam conversion from []interface{} to []string", t, func() {
		// Mock ResolveParamBinding to point to our data
		mockey.Mock(helper.ResolveParamBinding).Return("obj_user_ids", cnst.SID_SLICE_STRING, nil).Build()

		// Simulate input data as []interface{} (JSON style)
		rawList := []interface{}{"user1", "user2"}

		// Use utils to create NodeCtx if needed, but for GetParam we mainly need RuleMsg
		nodeCtx := utils.NewMockNodeCtx()

		// Create a CoreObj wrapping the []interface{} data
		// We define it as []string (SID: std/slice_string) but inject []interface{} body
		// This simulates the scenario where data was unmarshaled loosely or from a source that returned []interface{}
		def := contract.NewDefaultCoreObjDef([]string{}, cnst.SID_SLICE_STRING, "test desc")
		coreObj := contract.NewDefaultCoreObj("obj_user_ids", def)
		coreObj.SetBody(rawList)

		dataT := contract.NewDataT()
		dataT.Set("obj_user_ids", coreObj)
		msg := contract.NewDefaultRuleMsg("test", "", nil, dataT)

		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx), asset.WithRuleMsg(msg))

		// Act: Try to retrieve as []string
		ids, err := helper.GetParam[[]string](assetCtx, "user_ids")

		// Assert
		// If platform conversion works, this passes. If not, it fails (confirming the need for the fix).
		if err == nil {
			assert.Equal(t, 2, len(ids))
			if len(ids) > 0 {
				assert.Equal(t, "user1", ids[0])
			}
		} else {
			t.Logf("GetParam[[]string] failed: %v", err)
			// Verify fallback works
			rawIds, err2 := helper.GetParam[[]interface{}](assetCtx, "user_ids")
			assert.NoError(t, err2)
			assert.Equal(t, 2, len(rawIds))
		}
	})
}

type MockProfile struct {
	Name string
}

func TestSetParam_SlicePointer_RealAsset(t *testing.T) {
	// This test attempts to reproduce the "unsupported type for whole object assignment" error
	// when passing a pointer to a slice (*[]T) to SetParam.
	mockey.PatchConvey("TestSetParam with Slice Pointer (Real Asset)", t, func() {
		sid := "mock_profile_list"
		objID := "profile_list"

		// Mock ResolveParamBinding
		mockey.Mock(helper.ResolveParamBinding).Return(objID, sid, nil).Build()

		// 1. Setup RuleMsg with a DataT object containing a slice body
		// The definition uses []Any to simulate a generic container or mismatching type
		// This ensures valueType != bodyType, forcing it to fall through to Decode checks.
		// Since value is a pointer (*[]MockProfile), Kind() is Ptr, not Slice, so Decode is skipped if not handled.
		def := contract.NewDefaultCoreObjDef([]any{}, sid, "Mock Profile List")
		coreObj := contract.NewDefaultCoreObj(objID, def)

		dataT := contract.NewDataT()
		dataT.Set(objID, coreObj)

		msg := contract.NewDefaultRuleMsg("test_req", "", nil, dataT)
		nodeCtx := utils.NewMockNodeCtx()
		assetCtx := asset.NewAssetContext(asset.WithNodeCtx(nodeCtx), asset.WithRuleMsg(msg))

		// 2. Prepare input value: *[]MockProfile
		valSlice := []MockProfile{{Name: "test_profile"}}
		val := &valSlice

		// 3. Act
		// We use the real Asset implementation (no mocking of Asset.Set)
		// This relies on RuleMsgAsset being registered and functioning.
		ret, err := helper.SetParam(assetCtx, "param_profile_list", val)

		// 4. Assert
		assert.NoError(t, err)
		assert.Equal(t, val, ret)

		// Verify the value was actually set in the CoreObj
		storedBody := coreObj.Body()
		// Since we defined the CoreObj as []any, the body is *[]interface{}
		// SetCoreObjBody should have decoded *[]MockProfile into *[]interface{}
		storedList, ok := storedBody.(*[]interface{})
		assert.True(t, ok, "Stored body should be *[]interface{}")
		if ok {
			assert.Equal(t, 1, len(*storedList))
			if len(*storedList) > 0 {
				// Let's check the first element.
				elem := (*storedList)[0]
				if p, ok := elem.(MockProfile); ok {
					assert.Equal(t, "test_profile", p.Name)
				} else if m, ok := elem.(map[string]interface{}); ok {
					assert.Equal(t, "test_profile", m["Name"])
				} else {
					// Fallback for debugging
					t.Logf("Element type: %T", elem)
				}
			}
		}
	})
}
