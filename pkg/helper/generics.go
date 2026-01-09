package helper

import (
	"fmt"
	"strings"

	"github.com/neohetj/matrix/internal/builtin/base"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

// GetParam retrieves a parameter by name from DataT and converts it to the specified type T.
func GetParam[T any](ctx *asset.AssetContext, name string) (T, error) {
	var zero T
	if ctx == nil || ctx.RuleMsg() == nil {
		return zero, fmt.Errorf("rule message is required to resolve param '%s'", name)
	}

	objID, sid, err := ResolveParamBinding(ctx, name, "inputs")
	if err != nil {
		return zero, err
	}

	uri := asset.DataTURI(objID, "", sid)
	val, err := asset.Asset[T]{URI: uri}.Resolve(ctx)
	if err != nil {
		return zero, err
	}

	return val, nil
}

// SetParam creates a new item by parameter name and returns the typed body.
// If value is provided and not nil, it is used to set the parameter.
// Otherwise, a new zero value is created and set.
func SetParam[T any](ctx *asset.AssetContext, name string, value T) (T, error) {
	var zero T
	if ctx == nil || ctx.RuleMsg() == nil {
		return zero, fmt.Errorf("rule message is required to resolve param '%s'", name)
	}

	objID, sid, err := ResolveParamBinding(ctx, name, "outputs")
	if err != nil {
		return zero, err
	}

	uri := asset.DataTURI(objID, "", sid)
	var val T

	if !utils.IsNil(value) {
		val = value
	} else {
		var ok bool
		val, ok = utils.ZeroValue[T]()
		if !ok {
			return zero, fmt.Errorf("unsupported type %T for param '%s'", zero, name)
		}
	}

	a := asset.Asset[T]{URI: uri}
	if err := a.Set(ctx, val); err != nil {
		return zero, err
	}

	return val, nil
}

// RenderConfigAsset retrieves a config value as string, renders it as a template, and then converts to T.
// This is useful when the config value itself contains placeholders (e.g. "${config:///anotherKey}").
func RenderConfigAsset[T any](ctx *asset.AssetContext, key string) (T, error) {
	var zero T

	// 1. Get config value as string to support template rendering
	strVal, err := GetConfigAsset[string](ctx, key)
	if err != nil {
		return zero, err
	}

	// 2. Render template
	rendered, err := asset.RenderTemplate(strVal, ctx)
	if err != nil {
		return zero, err
	}

	// 3. Convert to target type T
	if v, ok := any(rendered).(T); ok {
		return v, nil
	}

	// 4. Try conversion if T is not string
	targetType, ok := utils.InferMType(zero)
	if !ok {
		return zero, fmt.Errorf("unsupported type %T for auto conversion", zero)
	}

	converted, err := utils.Convert(rendered, targetType)
	if err != nil {
		return zero, err
	}

	if v, ok := converted.(T); ok {
		return v, nil
	}

	return zero, fmt.Errorf("converted value %v (type %T) cannot be asserted to %T", converted, converted, zero)
}

// GetConfigAsset retrieves a config value using Asset mechanism.
// It resolves the asset to its raw type and then performs conversions or template rendering.
func GetConfigAsset[T any](ctx *asset.AssetContext, key string) (T, error) {
	var zero T

	uri, err := BuildConfigAssetURI[T](ctx, key)
	if err != nil {
		return zero, err
	}

	asset.ClearConfigSearchedScopes(ctx)
	defer asset.ClearConfigSearchedScopes(ctx)

	// Resolve the asset to its raw `any` type first.
	rawVal, err := asset.Asset[any]{URI: uri}.Resolve(ctx)
	if err != nil {
		return zero, err
	}

	//统一处理
	finalVal := rawVal
	// 优先判断是不是模版字符串，是则渲染
	if strVal, ok := rawVal.(string); ok && asset.IsTemplate(strVal) {
		innerURI := strings.TrimSuffix(strings.TrimPrefix(strVal, "${"), "}")
		configAsset, pErr := asset.ParseConfig(innerURI)
		if pErr != nil {
			return zero, pErr
		}

		newURI := configAsset.BuildWithRemainingScopes(ctx)
		rendered, err := asset.Asset[string]{URI: newURI}.Resolve(ctx)
		if err != nil {
			return zero, err
		}
		finalVal = rendered
	}

	// 尝试将最终值转换为目标类型
	if val, ok := finalVal.(T); ok {
		return val, nil
	}

	// 如果直接转换失败，尝试进行类型转换
	var result T
	targetType, ok := utils.InferMType(result)
	if !ok {
		return zero, fmt.Errorf("unsupported type %T for auto conversion", result)
	}
	converted, err := utils.Convert(finalVal, targetType)
	if err != nil {
		return zero, err
	}
	if v, ok := converted.(T); ok {
		return v, nil
	}

	return zero, asset.AssetTypeMismatch.Wrap(fmt.Errorf("expected %T, but got value '%v' of type %T", zero, finalVal, finalVal))
}

func BuildConfigAssetURI[T any](ctx *asset.AssetContext, key string) (string, error) {
	var zero T

	defaultValue, definedType := ResolveConfigFieldMeta(ctx, key)
	if definedType == "" {
		if mt, ok := utils.InferMType(zero); ok {
			if mt == cnst.INT64 {
				mt = cnst.INT
			}
			definedType = string(mt)
		}
	}

	configAsset := asset.ConfigAsset{}
	builder := configAsset.SetKey(key)
	if defaultValue != "" {
		builder = builder.Default(defaultValue)
	}
	if definedType != "" {
		builder = builder.Type(definedType)
	}

	return builder.Build(), nil
}

func ResolveConfigFieldMeta(ctx *asset.AssetContext, key string) (defaultValue string, definedType string) {
	nodeCtx := ctx.NodeCtx()
	if nodeCtx == nil {
		return defaultValue, definedType
	}
	fn, ok := nodeCtx.GetNode().(*base.FunctionsNode)
	if !ok {
		return defaultValue, definedType
	}

	var mgr types.NodeFuncManager
	if r := nodeCtx.GetRuntime(); r != nil && r.GetEngine() != nil {
		mgr = r.GetEngine().NodeFuncManager()
	}
	if mgr == nil {
		return defaultValue, definedType
	}

	funcDef, found := mgr.Get(fn.FuncConfig.FunctionName)
	if !found || funcDef.FuncObject.Configuration.Business == nil {
		return defaultValue, definedType
	}

	for _, field := range funcDef.FuncObject.Configuration.Business {
		if field.ID != key {
			continue
		}

		if field.Default != nil {
			defaultValue = fmt.Sprintf("%v", field.Default)
		}
		return defaultValue, string(field.Type)
	}

	return defaultValue, definedType
}

func ResolveParamBinding(ctx *asset.AssetContext, name string, io string) (string, string, error) {
	nodeCtx := ctx.NodeCtx()
	if nodeCtx == nil {
		return "", "", fmt.Errorf("node context is required to resolve param '%s'", name)
	}

	var (
		ioConfigs map[string]contract.IOConfig
		ok        bool
	)

	switch io {
	case "inputs":
		ioConfigs, ok = contract.GetInputs(nodeCtx)
		if !ok {
			return "", "", fmt.Errorf("inputs not found in node configuration for node %s", nodeCtx.NodeID())
		}
	case "outputs":
		ioConfigs, ok = contract.GetOutputs(nodeCtx)
		if !ok {
			return "", "", fmt.Errorf("outputs not found in node configuration for node %s", nodeCtx.NodeID())
		}
	default:
		return "", "", fmt.Errorf("invalid io type '%s'", io)
	}

	config, ok := ioConfigs[name]
	if !ok {
		return "", "", fmt.Errorf("parameter '%s' not found in %s configuration for node %s", name, io, nodeCtx.NodeID())
	}
	if config.ObjId == "" {
		return "", "", fmt.Errorf("objId is empty for parameter '%s' in node %s", name, nodeCtx.NodeID())
	}
	if config.DefineSID == "" {
		return "", "", fmt.Errorf("defineSid is empty for parameter '%s' in node %s", name, nodeCtx.NodeID())
	}

	return config.ObjId, config.DefineSID, nil
}
