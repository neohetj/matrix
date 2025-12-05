package utils

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// Decode 将 map[string]any 的数据解码到一个强类型的结构体指针中。
// 使用 `mitchellh/mapstructure` 库，它功能强大且性能优于纯反射。
// targetStructPtr 必须是一个指向结构体的指针。
func Decode(data map[string]any, targetStructPtr any) error {
	if data == nil {
		return fmt.Errorf("input data map is nil")
	}
	v := reflect.ValueOf(targetStructPtr)
	if v.Kind() != reflect.Pointer {
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
		TagName:          "json",
		ErrorUnused:      true, // 如果输入包含目标结构体未定义的字段，则报错
		ZeroFields:       true, // 全量替换：清空已有字段（尤其是切片/Map）后再写入
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(textUnmarshalHook),
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
