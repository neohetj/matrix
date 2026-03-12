# Generic Adapter Example

## Scenario

目标函数：`domain/normalize_items`

目标是把节点拆成：

1. 函数定义层：`NormalizeItemsFuncObj`
2. Matrix 适配层：`NormalizeItems(ctx,msg)`
3. 纯业务实现层：`NormalizeItemsImpl(context.Context, bizlog.Logger, []domain.Item, NormalizeOptions)`

## Adapter Layer

```go
func NormalizeItems(ctx types.NodeCtx, msg types.RuleMsg) {
	assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))

	items, err := helper.GetParam[[]domain.Item](assetCtx, paramItems)
	if err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(fmt.Errorf("failed to get items: %w", err)))
		return
	}

	trimSpaces, _ := helper.GetConfigAsset[bool](assetCtx, cfgTrimSpaces)
	opts := NormalizeOptions{TrimSpaces: trimSpaces}
	logger := adaptNodeLogger(ctx)

	output, err := NormalizeItemsImpl(ctx.GetContext(), logger, items, opts)
	if err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(fmt.Errorf("failed to normalize items: %w", err)))
		return
	}

	if _, err := helper.SetParam(assetCtx, paramNormalized, output); err != nil {
		ctx.HandleError(msg, types.InternalError.Wrap(fmt.Errorf("failed to set normalized items: %w", err)))
		return
	}

	ctx.TellSuccess(msg)
}
```

## Pure Business Impl

```go
func NormalizeItemsImpl(
	ctx context.Context,
	logger bizlog.Logger,
	items []domain.Item,
	opts NormalizeOptions,
) ([]domain.Item, error) {
	result := make([]domain.Item, 0, len(items))

	for _, item := range items {
		if opts.TrimSpaces {
			item.Name = strings.TrimSpace(item.Name)
		}
		result = append(result, item)
	}

	logger.Info("normalized items", "count", len(result))
	return result, nil
}
```

## Why This Shape Matters

1. `NormalizeItemsImpl` 可以被 service、CLI、测试直接调用。
2. Matrix 依赖被局限在 adapter 层。
3. logger 也是业务层契约，不强绑框架类型。
4. DSL 仍然只需要对接 `NormalizeItemsFuncObj` 的 I/O 契约。
