package helper

import (
	"fmt"
	"strings"

	"github.com/NeohetJ/Matrix/internal/builtin/base"
	"github.com/NeohetJ/Matrix/internal/contract"
	"github.com/NeohetJ/Matrix/pkg/asset"
	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/types"
	"github.com/NeohetJ/Matrix/pkg/utils"
)

// GetParam retrieves a parameter by name from DataT and converts it to the specified type T.
func GetParam[T any](ctx *asset.AssetContext, name string) (T, error) {
	var zero T
	if ctx == nil || ctx.RuleMsg() == nil {
		return zero, fmt.Errorf("rule message is required to resolve param '%s'", name)
	}

	objID, sid, err := resolveParamBinding(ctx, name, "inputs")
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

	objID, sid, err := resolveParamBinding(ctx, name, "outputs")
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
// It constructs a config:// URI, resolves it via Asset[string],
// and handles template rendering and type conversion if necessary.
func GetConfigAsset[T any](ctx *asset.AssetContext, key string) (T, error) {
	var zero T

	uri, err := buildConfigAssetURI[T](ctx, key)
	if err != nil {
		return zero, err
	}

	asset.ClearConfigSearchedScopes(ctx)
	defer asset.ClearConfigSearchedScopes(ctx)

	// Resolve once as string, then render (if template) and convert.
	strVal, err := asset.Asset[string]{URI: uri}.Resolve(ctx)
	if err != nil {
		return zero, err
	}

	rendered := strVal
	if asset.IsTemplate(strVal) {
		innerURI := strings.TrimSuffix(strings.TrimPrefix(strVal, "${"), "}")
		configAsset, pErr := asset.ParseConfig(innerURI)
		if pErr != nil {
			return zero, pErr
		}

		newURI := configAsset.BuildWithRemainingScopes(ctx)
		rendered, err = asset.Asset[string]{URI: newURI}.Resolve(ctx)
		if err != nil {
			return zero, err
		}
	}

	// If T is string, we are done.
	if v, ok := any(rendered).(T); ok {
		return v, nil
	}

	// Convert string to T using utils.Convert.
	var result T
	var targetType cnst.MType
	var ok bool

	targetType, ok = utils.InferMType(result)
	if !ok {
		return zero, fmt.Errorf("unsupported type %T for auto conversion", result)
	}

	converted, err := utils.Convert(rendered, targetType)
	if err != nil {
		return zero, err
	}

	if v, ok := converted.(T); ok {
		return v, nil
	}

	// Handle case where Convert returns a compatible type but not exact T.
	return zero, fmt.Errorf("converted value %v (type %T) cannot be asserted to %T", converted, converted, result)
}

func buildConfigAssetURI[T any](ctx *asset.AssetContext, key string) (string, error) {
	var zero T

	defaultValue, definedType := resolveConfigFieldMeta(ctx, key)
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

func resolveConfigFieldMeta(ctx *asset.AssetContext, key string) (string, string) {
	nodeCtx := ctx.NodeCtx()
	if nodeCtx == nil {
		return "", ""
	}
	fn, ok := nodeCtx.GetNode().(*base.FunctionsNode)
	if !ok {
		return "", ""
	}

	var mgr types.NodeFuncManager
	if r := nodeCtx.GetRuntime(); r != nil && r.GetEngine() != nil {
		mgr = r.GetEngine().NodeFuncManager()
	}
	if mgr == nil {
		return "", ""
	}

	funcDef, found := mgr.Get(fn.FuncConfig.FunctionName)
	if !found || funcDef.FuncObject.Configuration.Business == nil {
		return "", ""
	}

	for _, field := range funcDef.FuncObject.Configuration.Business {
		if field.ID != key {
			continue
		}
		defaultValue := ""
		if field.Default != nil {
			defaultValue = fmt.Sprintf("%v", field.Default)
		}
		return defaultValue, string(field.Type)
	}

	return "", ""
}

func resolveParamBinding(ctx *asset.AssetContext, name string, io string) (string, string, error) {
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
