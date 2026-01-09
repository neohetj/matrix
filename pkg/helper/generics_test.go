package helper

import (
	"fmt"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetConfigAssetComplexType(t *testing.T) {
	configKey := "complexArgs"
	expectedMap := map[string]any{
		"position": []any{19, 28},
		"isValid":  true,
	}

	mockey.PatchConvey("TestGetConfigAsset with complex map type", t, func() {
		// Mock ResolveConfigFieldMeta to provide the type info needed by BuildConfigAssetURI
		mockey.Mock(ResolveConfigFieldMeta).Return("", "map").Build()

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
		actualMap, err := GetConfigAsset[map[string]any](assetCtx, configKey)

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
		resolvedValue, err := GetConfigAsset[string](assetCtx, "templatedKey")

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
		resolvedValue, err := GetConfigAsset[string](assetCtx, "templatedKey")

		// 3. Assert: Check that it resolved from the chain scope
		assert.NoError(t, err)
		assert.Equal(t, "engine_key_123", resolvedValue)
	})
}
