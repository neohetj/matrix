package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/expr-lang/expr"
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

// Convert 将一个值转换为指定的目标类型。
// targetType 可以是 cnst.STRING, cnst.INT, cnst.INT64, cnst.FLOAT, cnst.BOOL, cnst.MAP 等。
func Convert(value any, targetType cnst.MType) (any, error) {
	if !targetType.IsSupported() {
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}
	if value == nil {
		switch targetType {
		case cnst.STRING:
			return "", nil
		case cnst.INT:
			return 0, nil
		case cnst.INT64:
			return int64(0), nil
		case cnst.FLOAT:
			return 0.0, nil
		case cnst.BOOL:
			return false, nil
		case cnst.OBJECT, cnst.MAP:
			return nil, nil
		default:
			// Handle slice types like "[]string"
			isList, _ := targetType.IsList()
			if isList {
				return nil, nil
			}
			return nil, nil
		}
	}

	sourceType := reflect.TypeOf(value)
	sValue := reflect.ValueOf(value)

	switch targetType {
	case cnst.STRING:
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
	case cnst.INT:
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
	case cnst.INT64:
		switch sourceType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return sValue.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int64(sValue.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return int64(sValue.Float()), nil
		case reflect.String:
			return strconv.ParseInt(value.(string), 10, 64)
		case reflect.Bool:
			if sValue.Bool() {
				return int64(1), nil
			}
			return int64(0), nil
		default:
			return nil, fmt.Errorf("can't convert %s to int64", sourceType.String())
		}
	case cnst.FLOAT:
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
	case cnst.BOOL:
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
	case cnst.OBJECT, cnst.MAP:
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
		isList, elemTypeStr := targetType.IsList()
		if isList {
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

			// Create a new slice of the correct Go type.
			var targetSlice reflect.Value
			switch cnst.MType(elemTypeStr) {
			case cnst.STRING:
				targetSlice = reflect.ValueOf(make([]string, 0, sValue.Len()))
			case cnst.INT:
				targetSlice = reflect.ValueOf(make([]int, 0, sValue.Len()))
			case cnst.INT64:
				targetSlice = reflect.ValueOf(make([]int64, 0, sValue.Len()))
			case cnst.FLOAT:
				targetSlice = reflect.ValueOf(make([]float64, 0, sValue.Len()))
			case cnst.BOOL:
				targetSlice = reflect.ValueOf(make([]bool, 0, sValue.Len()))
			default:
				// For complex types, we can't easily determine the type.
				// We'll just return the original slice and let mapstructure handle it.
				return value, nil
			}

			for i := 0; i < sValue.Len(); i++ {
				elem := sValue.Index(i).Interface()
				convertedElem, err := Convert(elem, cnst.MType(elemTypeStr))
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
		if v.Kind() == reflect.Pointer {
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

// SetValueByDotPath sets a value in a nested map using a dot-separated path.
// It creates nested maps as needed.
func SetValueByDotPath(data map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, ok := current[part].(map[string]any); !ok {
				current[part] = make(map[string]any)
			}
			current = current[part].(map[string]any)
		}
	}
}

// IsNil checks if a value is nil, handling interface-wrapped nil values.
func IsNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// InferMType infers the MType from a Go value.
func InferMType(val any) (cnst.MType, bool) {
	switch val.(type) {
	case int, int32:
		return cnst.INT, true
	case int64:
		return cnst.INT64, true
	case float32, float64:
		return cnst.FLOAT, true
	case bool:
		return cnst.BOOL, true
	case string:
		return cnst.STRING, true
	case map[string]any:
		return cnst.MAP, true
	default:
		return "", false
	}
}

// ZeroValue creates a new zero value for type T.
// It handles pointers, maps, and slices correctly for Matrix parameters.
func ZeroValue[T any]() (T, bool) {
	var zero T
	paramType := reflect.TypeOf((*T)(nil)).Elem()
	if paramType == nil {
		return zero, false
	}

	switch paramType.Kind() {
	case reflect.Pointer:
		value := reflect.New(paramType.Elem()).Interface().(T)
		return value, true
	case reflect.Map:
		value := reflect.MakeMap(paramType).Interface().(T)
		return value, true
	case reflect.Slice:
		value := reflect.MakeSlice(paramType, 0, 0).Interface().(T)
		return value, true
	default:
		return zero, true
	}
}
