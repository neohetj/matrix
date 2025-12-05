package asset

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/NeohetJ/Matrix/pkg/cnst"
)

var placeholderRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ---------------- Builder ------------------

// DataURI creates a URI for accessing RuleMsg.Data with explicit format.
func DataURI(format cnst.MFormat) string {
	return RuleMsgAsset{}.Data().Format(format).Build()
}

// MetadataURI creates a URI for accessing RuleMsg.Metadata.
// key is the metadata key (e.g., "trace_id").
func MetadataURI(key string) string {
	return RuleMsgAsset{}.Metadata().Key(key).Build()
}

// DataTURI creates a URI for accessing DataT with explicit SID.
func DataTURI(objId string, fieldPath string, sid string) string {
	return RuleMsgAsset{}.DataT().Obj(objId).Field(fieldPath).SID(sid).Build()
}

// ValidateURI validates a URI using the registered scheme handler.
func ValidateURI(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid uri: %w", err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("missing uri scheme")
	}
	h := GetHandler(u.Scheme)
	if h == nil {
		return fmt.Errorf("no handler registered for scheme: %s", u.Scheme)
	}
	if validator, ok := h.(ValidatingHandler); ok {
		return validator.Validate(u)
	}
	return nil
}

// NormalizeURI normalizes a URI using the registered scheme handler.
func NormalizeURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme == "" {
		return uri
	}
	h := GetHandler(u.Scheme)
	if h == nil {
		return uri
	}
	return h.NormalizeAssetURI(uri)
}

// ---------------- Template Render ------------------

// CollectTemplateAssets extracts unique, valid asset URIs from placeholders like ${...}.
func CollectTemplateAssets(template string) []string {
	if !IsTemplate(template) {
		return nil
	}

	matches := placeholderRegex.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}

	assets := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		uri := strings.TrimSpace(match[1])
		if uri == "" {
			continue
		}
		if ruleMsgAsset, err := ParseRuleMsg(uri); err == nil {
			normalized := ruleMsgAsset.NormalizeAssetURI(uri)
			if normalized == "" {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			assets = append(assets, normalized)
			continue
		}
		if err := ValidateURI(uri); err != nil {
			continue
		}
		normalized := uri
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		assets = append(assets, normalized)
	}

	return assets
}

// ReplacePlaceholders replaces placeholders using a custom resolver function.
// Copied from pkg/utils to avoid circular dependencies if pkg/utils imports pkg/asset.
func ReplacePlaceholders(template string, resolver func(path string) (any, error)) (string, error) {
	if !IsTemplate(template) {
		return template, nil
	}

	var err error
	result := placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
		if err != nil {
			return match
		}
		// Extract the path from the placeholder
		path := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")

		value, resolveErr := resolver(path)
		if resolveErr != nil {
			err = resolveErr
			return match
		}

		// Convert the found value to a string for replacement.
		return fmt.Sprintf("%v", value)
	})

	if err != nil {
		return "", err
	}

	return result, nil
}

// IsTemplate checks if the string contains potential template placeholders.
func IsTemplate(s string) bool {
	return strings.Contains(s, "${")
}

func IsURI(uri string, schemas ...string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if u.Scheme == "" {
		return false
	}
	if GetHandler(u.Scheme) == nil {
		return false
	}
	if len(schemas) > 0 {
		return slices.Contains(schemas, u.Scheme)
	}
	return true
}

// RenderTemplate renders a template string by resolving any embedded URIs using Asset logic.
// It supports placeholders like ${config:///key}, ${rulemsg://data/path} or ${rulemsg://dataT/path}.
func RenderTemplate(template string, ctx *AssetContext) (string, error) {
	if !IsTemplate(template) {
		return template, nil
	}

	// The resolver function for ReplacePlaceholders.
	// It treats the path inside ${...} as a URI.
	resolver := func(uri string) (any, error) {
		a := Asset[any]{URI: uri}
		return a.Resolve(ctx)
	}

	return ReplacePlaceholders(template, resolver)
}
