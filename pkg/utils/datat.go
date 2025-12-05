package utils

import (
	"fmt"
	"reflect"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/NeohetJ/Matrix/pkg/types"
)

// SetCoreObjBody tries to assign a value to a CoreObj body with flexible type handling.
// It supports:
// - direct assignment when the value type matches the body type
// - basic type coercion (string/int64/float64/bool, including pointer targets)
// - decoding map[string]any into a struct/map body
// - direct map[string]string replacement for header-like bodies
func SetCoreObjBody(obj types.CoreObj, value any) (bool, error) {
	if obj == nil {
		return false, fmt.Errorf("nil core object")
	}

	if trySetBodyByType(obj, value) {
		return true, nil
	}

	if headerMap, ok := value.(map[string]string); ok {
		if target, ok := obj.Body().(*map[string]string); ok {
			*target = headerMap
			return true, nil
		}
	}

	if bodyMap, ok := value.(map[string]any); ok {
		if err := Decode(bodyMap, obj.Body()); err != nil {
			return true, err
		}
		return true, nil
	}

	return false, nil
}

func trySetBodyByType(obj types.CoreObj, value any) bool {
	currentBody := obj.Body()
	if currentBody == nil {
		return false
	}

	currentType := reflect.TypeOf(currentBody)
	valueType := reflect.TypeOf(value)

	if currentType.Kind() == reflect.Pointer {
		elemKind := currentType.Elem().Kind()
		isBasic := elemKind == reflect.String || elemKind == reflect.Int64 || elemKind == reflect.Float64 || elemKind == reflect.Bool
		if !isBasic && valueType == currentType {
			if err := obj.SetBody(value); err == nil {
				return true
			}
		}
	} else if valueType == currentType {
		if err := obj.SetBody(value); err == nil {
			return true
		}
	}

	if coerced, ok := convertBasicTypeByKind(value, currentType); ok {
		if err := obj.SetBody(coerced); err == nil {
			return true
		}
	}

	return false
}

func convertBasicTypeByKind(value any, targetType reflect.Type) (any, bool) {
	isPointer := targetType.Kind() == reflect.Pointer
	if isPointer {
		targetType = targetType.Elem()
	}

	var targetMType cnst.MType
	switch targetType.Kind() {
	case reflect.String:
		targetMType = cnst.STRING
	case reflect.Int64:
		targetMType = cnst.INT64
	case reflect.Float64:
		targetMType = cnst.FLOAT
	case reflect.Bool:
		targetMType = cnst.BOOL
	default:
		return nil, false
	}

	converted, err := Convert(value, targetMType)
	if err != nil {
		return nil, false
	}

	if isPointer {
		ptr := reflect.New(targetType)
		ptr.Elem().Set(reflect.ValueOf(converted).Convert(targetType))
		return ptr.Interface(), true
	}

	return converted, true
}
