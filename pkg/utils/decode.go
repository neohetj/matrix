package utils

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// Decode 将 map[string]any 或 []any 的数据解码到一个强类型的结构体指针或切片指针中。
// 使用 `mitchellh/mapstructure` 库，它功能强大且性能优于纯反射。
// targetStructPtr 必须是一个指向结构体/切片的指针。
func Decode(data any, targetStructPtr any) error {
	if data == nil {
		return fmt.Errorf("input data is nil")
	}
	v := reflect.ValueOf(targetStructPtr)
	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("target must be a pointer")
	}
	switch v.Elem().Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice:
		// Allowed kinds
	default:
		return fmt.Errorf("target must be a pointer to a struct, map or slice, but got %s", v.Elem().Kind())
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           targetStructPtr,
		WeaklyTypedInput: true, // 允许在解码时进行类型转换
		TagName:          "json",
		ErrorUnused:      true, // 如果输入包含目标结构体未定义的字段，则报错
		ZeroFields:       true, // 全量替换：清空已有字段（尤其是切片/Map）后再写入
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(textUnmarshalHook, jsonUnmarshalHook),
	})
	if err != nil {
		return fmt.Errorf("failed to create mapstructure decoder: %w", err)
	}

	if err := decoder.Decode(data); err != nil {
		return fmt.Errorf("failed to decode map to struct: %w", err)
	}
	return nil
}

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func textUnmarshalHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if from == nil || to == nil {
		return data, nil
	}
	if from.Kind() != reflect.String {
		return data, nil
	}

	if to.Kind() == reflect.Pointer {
		if to.Implements(textUnmarshalerType) {
			v := reflect.New(to.Elem())
			if err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(data.(string))); err != nil {
				return nil, err
			}
			return v.Interface(), nil
		}
		return data, nil
	}

	if reflect.PointerTo(to).Implements(textUnmarshalerType) {
		v := reflect.New(to)
		if err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(data.(string))); err != nil {
			return nil, err
		}
		return v.Elem().Interface(), nil
	}

	return data, nil
}

var jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

func jsonUnmarshalHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if from == nil || to == nil {
		return data, nil
	}

	// Determine if 'to' or '*to' implements Unmarshaler
	isPtr := to.Kind() == reflect.Pointer
	checkType := to
	if !isPtr {
		checkType = reflect.PointerTo(to)
	}

	if !checkType.Implements(jsonUnmarshalerType) {
		return data, nil
	}

	// Marshal source data to JSON
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data for UnmarshalJSON: %w", err)
	}

	// Create target instance
	var val reflect.Value
	if isPtr {
		val = reflect.New(to.Elem()) // *T
	} else {
		val = reflect.New(to) // *T
	}

	// Call UnmarshalJSON
	if u, ok := val.Interface().(json.Unmarshaler); ok {
		if err := u.UnmarshalJSON(bytes); err != nil {
			return nil, err // Return validation error
		}
		if isPtr {
			return val.Interface(), nil
		}
		return val.Elem().Interface(), nil
	}

	return data, nil
}
