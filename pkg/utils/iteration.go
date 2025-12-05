package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// RangeCount extracts an integer count for RANGE iteration from common numeric types.
// It accepts ints, floats, and json.Number to be flexible with decoded payloads.
func RangeCount(source any) (int, error) {
	switch v := source.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), nil
		}
		return 0, fmt.Errorf("invalid json number: %v", v)
	default:
		return 0, fmt.Errorf("unsupported type %T for range count", source)
	}
}

// SliceValue normalizes list-like values for LIST iteration.
// It supports json.RawMessage containing a JSON array and pointer-to-slice values.
func SliceValue(source any) (reflect.Value, error) {
	if rawMessage, ok := source.(json.RawMessage); ok {
		var sliceOfItems []any
		if err := json.Unmarshal(rawMessage, &sliceOfItems); err != nil {
			return reflect.Value{}, fmt.Errorf("failed to unmarshal list from json.RawMessage: %w", err)
		}
		source = sliceOfItems
	}

	itemsVal := reflect.ValueOf(source)
	if itemsVal.Kind() == reflect.Ptr {
		itemsVal = itemsVal.Elem()
	}
	if itemsVal.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("did not return a slice, but a %s", itemsVal.Kind())
	}
	return itemsVal, nil
}
