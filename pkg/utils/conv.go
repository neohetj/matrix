package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
)

// ToMap 将任意结构体或结构体指针转换为 map[string]any。
// 底层使用 json.Marshal 和 json.Unmarshal 实现，通用性强。
func ToMap(data any) (map[string]any, error) {
	if data == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to json: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json to map: %w", err)
	}
	return result, nil
}

// ToMapSlice 将任意结构体切片或指针切片转换为 []map[string]any。
// 底层同样使用 json.Marshal 和 json.Unmarshal 实现。
func ToMapSlice(data any) ([]map[string]any, error) {
	if data == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slice to json: %w", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json to map slice: %w", err)
	}
	return result, nil
}

// Decode 将 map[string]any 的数据解码到一个强类型的结构体指针中。
// 使用 `mitchellh/mapstructure` 库，它功能强大且性能优于纯反射。
// targetStructPtr 必须是一个指向结构体的指针。
func Decode(data map[string]any, targetStructPtr any) error {
	if data == nil {
		return fmt.Errorf("input data map is nil")
	}
	v := reflect.ValueOf(targetStructPtr)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	switch v.Elem().Kind() {
	case reflect.Struct, reflect.Map:
		// Allowed kinds
	default:
		return fmt.Errorf("target must be a pointer to a struct or a map, but got %s", v.Elem().Kind())
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           targetStructPtr,
		WeaklyTypedInput: true, // 允许在解码时进行类型转换
	})
	if err != nil {
		return fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}

	if err := decoder.Decode(data); err != nil {
		return fmt.Errorf("failed to decode map to struct: %w", err)
	}
	return nil
}

// Convert 将一个值转换为指定的目标类型。
// targetType 可以是 "string", "int", "float", "bool", "map", "slice"。
func Convert(value any, targetType string) (any, error) {
	if value == nil {
		switch strings.ToLower(targetType) {
		case "string":
			return "", nil
		case "int", "integer":
			return 0, nil
		case "float", "double", "number":
			return 0.0, nil
		case "bool", "boolean":
			return false, nil
		case "object", "map":
			return nil, nil
		default:
			// Handle slice types like "[]string"
			if strings.HasPrefix(targetType, "[]") {
				return nil, nil
			}
			return nil, nil
		}
	}

	sourceType := reflect.TypeOf(value)
	sValue := reflect.ValueOf(value)
	lowerTargetType := strings.ToLower(targetType)

	switch lowerTargetType {
	case "string":
		if sourceType.Kind() == reflect.String {
			return value.(string), nil
		}
		// If a slice or map is to be converted to a string, JSON marshal it.
		if sourceType.Kind() == reflect.Slice || sourceType.Kind() == reflect.Map {
			bytes, err := json.Marshal(value)
			if err == nil {
				return string(bytes), nil
			}
		}
		return fmt.Sprintf("%v", value), nil
	case "int", "integer":
		switch sourceType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(sValue.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(sValue.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return int(sValue.Float()), nil
		case reflect.String:
			return strconv.Atoi(value.(string))
		case reflect.Bool:
			if sValue.Bool() {
				return 1, nil
			}
			return 0, nil
		default:
			return nil, fmt.Errorf("can't convert %s to int", sourceType.String())
		}
	case "float", "double", "number":
		switch sourceType.Kind() {
		case reflect.Float32, reflect.Float64:
			return sValue.Float(), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(sValue.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(sValue.Uint()), nil
		case reflect.String:
			return strconv.ParseFloat(value.(string), 64)
		default:
			return nil, fmt.Errorf("can't convert %s to float", sourceType.String())
		}
	case "bool", "boolean":
		switch sourceType.Kind() {
		case reflect.Bool:
			return value.(bool), nil
		case reflect.String:
			s := strings.ToLower(value.(string))
			if s == "true" || s == "1" {
				return true, nil
			}
			if s == "false" || s == "0" || s == "" {
				return false, nil
			}
			return nil, fmt.Errorf("can't convert string '%s' to bool", value.(string))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return sValue.Int() != 0, nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return sValue.Uint() != 0, nil
		case reflect.Float32, reflect.Float64:
			return sValue.Float() != 0, nil
		default:
			return nil, fmt.Errorf("can't convert %s to bool", sourceType.String())
		}
	case "object", "map":
		if sourceType.Kind() == reflect.Map {
			return ToMap(value)
		}
		if sourceType.Kind() == reflect.Struct || (sourceType.Kind() == reflect.Ptr && sourceType.Elem().Kind() == reflect.Struct) {
			return ToMap(value)
		}
		if sourceType.Kind() == reflect.String {
			var result map[string]any
			if err := json.Unmarshal([]byte(value.(string)), &result); err == nil {
				return result, nil
			}
		}
		return nil, fmt.Errorf("can't convert %s to map[string]any", sourceType.String())
	default:
		// Handle slice types like "[]string", "[]int", etc.
		if strings.HasPrefix(lowerTargetType, "[]") {
			// If the source is a string, try to unmarshal it as a JSON array.
			if sourceType.Kind() == reflect.String {
				var result []any
				if err := json.Unmarshal([]byte(value.(string)), &result); err == nil {
					// If unmarshal is successful, recurse with the new slice value.
					return Convert(result, targetType)
				}
			}

			if sValue.Kind() != reflect.Slice {
				return nil, fmt.Errorf("can't convert non-slice type %s to %s", sourceType.String(), targetType)
			}

			elemTypeStr := strings.TrimPrefix(lowerTargetType, "[]")

			// Create a new slice of the correct Go type.
			var targetSlice reflect.Value
			switch elemTypeStr {
			case "string":
				targetSlice = reflect.ValueOf(make([]string, 0, sValue.Len()))
			case "int", "integer":
				targetSlice = reflect.ValueOf(make([]int, 0, sValue.Len()))
			case "float", "double", "number":
				targetSlice = reflect.ValueOf(make([]float64, 0, sValue.Len()))
			case "bool", "boolean":
				targetSlice = reflect.ValueOf(make([]bool, 0, sValue.Len()))
			default:
				// For complex types, we can't easily determine the type.
				// We'll just return the original slice and let mapstructure handle it.
				return value, nil
			}

			for i := 0; i < sValue.Len(); i++ {
				elem := sValue.Index(i).Interface()
				convertedElem, err := Convert(elem, elemTypeStr)
				if err != nil {
					return nil, fmt.Errorf("error converting slice element at index %d: %w", i, err)
				}
				targetSlice = reflect.Append(targetSlice, reflect.ValueOf(convertedElem))
			}
			return targetSlice.Interface(), nil
		}
		return value, nil
	}
}

// ExtractByPath 从一个复杂的对象（map 或 struct）中，通过点路径提取值。
// 支持通过点（.）和方括号（[]）访问嵌套字段和数组成员。
func ExtractByPath(data any, path string) (any, bool, error) {
	if data == nil {
		return nil, false, nil
	}
	if path == "" {
		return data, true, nil
	}

	// 简单的点路径解析
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		v := reflect.ValueOf(current)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		// 如果当前值是无效的（例如，前一步的结果是nil），则无法继续
		if !v.IsValid() {
			return nil, false, nil
		}

		switch v.Kind() {
		case reflect.Map:
			// 假设 key 是 string 类型
			mapKeys := v.MapKeys()
			var found bool
			for _, key := range mapKeys {
				if key.Kind() == reflect.String && key.String() == part {
					current = v.MapIndex(key).Interface()
					found = true
					break
				}
			}
			if !found {
				return nil, false, nil
			}
		case reflect.Struct:
			field := v.FieldByName(part)
			if !field.IsValid() {
				return nil, false, nil
			}
			// Add check for unexported fields to prevent panic.
			if !field.CanInterface() {
				return nil, false, fmt.Errorf("cannot access unexported field: %s", part)
			}
			current = field.Interface()
		default:
			return nil, false, fmt.Errorf("unsupported type for path extraction: %T", current)
		}
	}
	return current, true, nil
}

// EvalAndExtract 使用表达式从环境中提取值。
func EvalAndExtract(env map[string]any, expression string) (any, error) {
	if env == nil {
		env = make(map[string]any)
	}
	return expr.Eval(expression, env)
}

// GetValueFromMapByPath 从 map[string]any 中按点分隔的路径取值。
func GetValueFromMapByPath(data map[string]any, path string) (any, bool) {
	if data == nil {
		return nil, false
	}
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// ReflectToSchema 将一个 Go 结构体类型转换为一个描述其结构的 map。
// 这是对“场景五”的封装。
// structPtrOrInstance 可以是一个结构体实例，或一个指向结构体的指针。
func ReflectToSchema(structPtrOrInstance any) (map[string]any, error) {
	if structPtrOrInstance == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	t := reflect.TypeOf(structPtrOrInstance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or a pointer to a struct")
	}

	return reflectStruct(t), nil
}

func reflectStruct(t reflect.Type) map[string]any {
	schema := make(map[string]any)
	schema["type"] = "object"
	properties := make(map[string]any)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 {
				fieldName = parts[0]
			}
		}

		if field.Anonymous {
			// For anonymous fields, merge their properties into the current schema.
			anonymousSchema := reflectStruct(field.Type)
			if anonProps, ok := anonymousSchema["properties"].(map[string]any); ok {
				for k, v := range anonProps {
					properties[k] = v
				}
			}
		} else {
			properties[fieldName] = reflectType(field.Type)
		}
	}
	schema["properties"] = properties
	return schema
}

func reflectType(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": reflectType(t.Elem()),
		}
	case reflect.Map:
		// Assuming string keys for simplicity
		return map[string]any{
			"type":                 "object",
			"additionalProperties": reflectType(t.Elem()),
		}
	case reflect.Struct:
		return reflectStruct(t)
	case reflect.Ptr:
		return reflectType(t.Elem())
	default:
		return map[string]any{"type": "any"}
	}
}
