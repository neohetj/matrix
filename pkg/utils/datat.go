package utils

import (
	"fmt"
	"reflect"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

// SetCoreObjBody tries to assign a value to a CoreObj body with flexible type handling.
// It supports:
// - direct assignment when the value type matches the body type
// - basic type coercion (string/int64/float64/bool, including pointer targets)
// - decoding map[string]any into a struct/map body
// - direct map[string]string replacement for header-like bodies
func SetCoreObjBody(obj types.CoreObj, value any, sid string) (bool, error) {
	if obj == nil {
		return false, fmt.Errorf("nil core object")
	}

	if value == nil {
		return true, nil
	}

	body := obj.Body()
	if body == nil {
		return false, fmt.Errorf("core object body is nil")
	}

	// 1. 尝试 SID 基础类型赋值
	if ok, err := trySetBodyBySID(obj, value, sid); ok || err != nil {
		return ok, err
	}

	// 3. 尝试直接赋值 (Moved up for priority and performance)
	valueType := reflect.TypeOf(value)
	bodyType := reflect.TypeOf(body)

	if valueType == bodyType {
		err := obj.SetBody(value)
		return err == nil, err
	}

	// 4. 处理 value 是 slice，而 body 是指向 slice 的指针的情况 (Moved up)
	if valueType.Kind() == reflect.Slice && bodyType.Kind() == reflect.Pointer && bodyType.Elem().Kind() == reflect.Slice {
		if valueType == bodyType.Elem() {
			// 创建一个新的 body slice 的指针，并将 value 复制过去
			newSlicePtr := reflect.New(valueType)
			newSlicePtr.Elem().Set(reflect.ValueOf(value))
			err := obj.SetBody(newSlicePtr.Interface())
			return err == nil, err
		}
	}

	// 4.1 处理 value 是 struct，而 body 是指向 struct 的指针的情况
	if valueType.Kind() == reflect.Struct && bodyType.Kind() == reflect.Pointer && bodyType.Elem().Kind() == reflect.Struct {
		if valueType == bodyType.Elem() {
			// 创建一个新的 body struct 的指针，并将 value 复制过去
			newPtr := reflect.New(valueType)
			newPtr.Elem().Set(reflect.ValueOf(value))
			err := obj.SetBody(newPtr.Interface())
			return err == nil, err
		}
	}

	// 2. 尝试 Decode (Map or Slice -> Struct/Slice)
	if valueType.Kind() == reflect.Map || valueType.Kind() == reflect.Slice {
		if err := Decode(value, body); err != nil {
			return false, fmt.Errorf("failed to decode %v to body: %w", valueType.Kind(), err)
		}
		return true, nil
	}

	// 5. Special handling for Pointer to Slice -> Decode
	// This handles cases where value is *[]Struct and body is *[]Any
	if valueType.Kind() == reflect.Pointer && valueType.Elem().Kind() == reflect.Slice {
		// Try to decode the slice value (dereferenced) into the body
		if err := Decode(reflect.ValueOf(value).Elem().Interface(), body); err == nil {
			return true, nil
		}
	}

	// 6. Special handling for Pointer to Map -> Decode
	// This handles cases where value is *map[string]any and body is Struct or Map
	if valueType.Kind() == reflect.Pointer && valueType.Elem().Kind() == reflect.Map {
		// Try to decode the map value (dereferenced) into the body
		if err := Decode(reflect.ValueOf(value).Elem().Interface(), body); err == nil {
			return true, nil
		}
	}

	// 7. Special handling for Struct or Pointer to Struct -> Decode
	// This handles cases where value is Struct or *Struct and body is Struct or Map
	if valueType.Kind() == reflect.Struct || (valueType.Kind() == reflect.Pointer && valueType.Elem().Kind() == reflect.Struct) {
		valToDecode := value
		if valueType.Kind() == reflect.Pointer {
			valToDecode = reflect.ValueOf(value).Elem().Interface()
		}
		if err := Decode(valToDecode, body); err == nil {
			return true, nil
		}
	}

	return false, nil
}

func trySetBodyBySID(obj types.CoreObj, value any, sid string) (bool, error) {
	switch sid {
	case cnst.SID_STRING:
		if v, ok := value.(string); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*string); ok {
			return true, obj.SetBody(v)
		}
		if converted, err := Convert(value, cnst.STRING); err == nil {
			if v, ok := converted.(string); ok {
				return true, obj.SetBody(&v)
			}
		}
	case cnst.SID_INT64:
		if v, ok := value.(int64); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*int64); ok {
			return true, obj.SetBody(v)
		}
		if converted, err := Convert(value, cnst.INT64); err == nil {
			if v, ok := converted.(int64); ok {
				return true, obj.SetBody(&v)
			}
		}
	case cnst.SID_FLOAT64:
		if v, ok := value.(float64); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*float64); ok {
			return true, obj.SetBody(v)
		}
		if converted, err := Convert(value, cnst.FLOAT); err == nil {
			if v, ok := converted.(float64); ok {
				return true, obj.SetBody(&v)
			}
		}
	case cnst.SID_BOOL:
		if v, ok := value.(bool); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*bool); ok {
			return true, obj.SetBody(v)
		}
		if converted, err := Convert(value, cnst.BOOL); err == nil {
			if v, ok := converted.(bool); ok {
				return true, obj.SetBody(&v)
			}
		}
	case cnst.SID_MAP_STRING_STRING:
		if v, ok := value.(map[string]string); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*map[string]string); ok {
			return true, obj.SetBody(v)
		}
	case cnst.SID_MAP_STRING_INTERFACE:
		if v, ok := value.(map[string]any); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*map[string]any); ok {
			return true, obj.SetBody(v)
		}
	case cnst.SID_SLICE_STRING:
		if v, ok := value.([]string); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*[]string); ok {
			return true, obj.SetBody(v)
		}
	case cnst.SID_SLICE_INT64:
		if v, ok := value.([]int64); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*[]int64); ok {
			return true, obj.SetBody(v)
		}
	case cnst.SID_SLICE_ANY:
		if v, ok := value.([]any); ok {
			return true, obj.SetBody(&v)
		} else if v, ok := value.(*[]any); ok {
			return true, obj.SetBody(v)
		}
	}
	return false, nil
}
