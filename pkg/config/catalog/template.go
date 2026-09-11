package catalog

import (
	"context"
	"net/url"
	"strings"

	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/types"
)

// RenderString 为初始化期配置模板复用 Asset 语法与实例 Reader，不读取进程环境。
// 旧 env/engine scope 仅兼容语法；来源顺序由 Reader 唯一定义。
// 表达式 default 只填补可选配置缺失，并经 Reader 字段校验；Secret 不接受该默认值。
func RenderString(ctx context.Context, reader types.ConfigReader, template string) (string, error) {
	if !asset.IsTemplate(template) {
		return template, nil
	}
	// ReplacePlaceholders 允许普通文本；此入口先拒绝不完整的配置表达式。
	if strings.Count(template, "${") != strings.Count(template, "}") {
		return "", problem("", "config_template", "")
	}
	result, err := asset.ReplacePlaceholders(template, func(raw string) (any, error) {
		uri, err := url.Parse(raw)
		if err != nil || uri.Scheme != "config" || uri.Host != "" || uri.Fragment != "" || uri.User != nil {
			return nil, problem("", "config_template", "")
		}
		query, err := url.ParseQuery(uri.RawQuery)
		if err != nil {
			return nil, problem("", "config_template", "")
		}
		for key, entries := range query {
			if len(entries) != 1 || (key != "scope" && key != "default" && key != "type") {
				return nil, problem("", "config_template", "")
			}
		}
		for _, scope := range strings.Split(query.Get("scope"), ",") {
			if scope != "" && scope != "env" && scope != "engine" {
				return nil, problem("", "config_template_scope", "")
			}
		}
		key := strings.TrimPrefix(uri.Path, "/")
		if key == "" {
			return nil, problem("", "config_template", "")
		}
		value, found, err := Read[any](ctx, reader, key)
		if err != nil || found {
			return value, err
		}
		if fallback, ok := query["default"]; ok {
			value, _, err := ReadNode[any](ctx, reader, key, types.ConfigOverride{Present: true, Value: fallback[0]})
			return value, err
		}
		return nil, problem(key, "required", "")
	})
	if err != nil {
		return "", err
	}
	if asset.IsTemplate(result) {
		return "", problem("", "config_template", "")
	}
	return result, nil
}
