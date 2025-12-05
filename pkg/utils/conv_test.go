package utils

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/NeohetJ/Matrix/pkg/cnst"
)

func TestToMap(t *testing.T) {
	type User struct {
		ID   int
		Name string
		Addr *struct {
			City string
		}
	}

	user := &User{
		ID:   1,
		Name: "test",
		Addr: &struct{ City string }{City: "beijing"},
	}

	expected := map[string]any{
		"ID":   float64(1), // json unmarshal to float64 for numbers
		"Name": "test",
		"Addr": map[string]any{
			"City": "beijing",
		},
	}

	result, err := ToMap(user)
	if err != nil {
		t.Fatalf("ToMap failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ToMap result not as expected. got: %v, want: %v", result, expected)
	}

	// Test nil input
	result, err = ToMap(nil)
	if err != nil {
		t.Fatalf("ToMap(nil) failed: %v", err)
	}
	if result != nil {
		t.Errorf("ToMap(nil) should return nil, but got: %v", result)
	}
}

func TestToMapSlice(t *testing.T) {
	type User struct {
		Name string
	}
	users := []User{{Name: "user1"}, {Name: "user2"}}

	expected := []map[string]any{
		{"Name": "user1"},
		{"Name": "user2"},
	}

	result, err := ToMapSlice(users)
	if err != nil {
		t.Fatalf("ToMapSlice failed: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ToMapSlice result not as expected. got: %v, want: %v", result, expected)
	}
}

// TestToMap_ErrorCases tests error cases for the ToMap function.
// 测试函数: ToMap
// 测试点: 覆盖输入为无法JSON序列化的类型时的错误场景。
func TestToMap_ErrorCases(t *testing.T) {
	// Test with a type that cannot be marshaled to JSON
	data := struct {
		FuncField func()
	}{
		FuncField: func() {},
	}

	_, err := ToMap(data)
	if err == nil {
		t.Fatal("ToMap() did not return an error for unmarshalable type, but expected one")
	}
	if !strings.Contains(err.Error(), "failed to marshal data to json") {
		t.Errorf("ToMap() error = %q, want error containing 'failed to marshal data to json'", err)
	}
}

// TestToMapSlice_ErrorCases tests error cases for the ToMapSlice function.
// 测试函数: ToMapSlice
// 测试点: 覆盖输入为非切片或切片中包含无法JSON序列化的元素时的错误场景。
func TestToMapSlice_ErrorCases(t *testing.T) {
	// Test with a non-slice type that will cause a json.Marshal error
	_, err := ToMapSlice(123) // json.Marshal supports this, but the result won't unmarshal to a slice of maps
	if err == nil {
		t.Fatal("ToMapSlice() did not return an error for non-slice type, but expected one")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal json to map slice") {
		t.Errorf("ToMapSlice() error = %q, want error containing 'failed to unmarshal json to map slice'", err)
	}

	// Test with a slice containing an unmarshalable type
	data := []any{
		struct{ Name string }{"user1"},
		struct{ FuncField func() }{FuncField: func() {}},
	}
	_, err = ToMapSlice(data)
	if err == nil {
		t.Fatal("ToMapSlice() did not return an error for slice with unmarshalable element, but expected one")
	}
	if !strings.Contains(err.Error(), "failed to marshal slice to json") {
		t.Errorf("ToMapSlice() error = %q, want error containing 'failed to marshal slice to json'", err)
	}
}

func TestDecode(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	data := map[string]any{
		"ID":   123,
		"Name": "test_user",
	}

	var user User
	err := Decode(data, &user)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if user.ID != 123 || user.Name != "test_user" {
		t.Errorf("Decode result not as expected. got: %+v", user)
	}
}

// TestDecode_ErrorCases tests the error handling of the Decode function.
// 测试函数: Decode
// 测试点: 覆盖输入数据为nil、目标非指针、目标类型不正确以及数据类型不匹配等错误场景。
func TestDecode_ErrorCases(t *testing.T) {
	type User struct {
		ID int
	}

	tests := []struct {
		name          string
		data          map[string]any
		target        any
		wantErrString string
	}{
		{
			name:          "nil data map",
			data:          nil,
			target:        &User{},
			wantErrString: "input data map is nil",
		},
		{
			name:          "nil target",
			data:          map[string]any{"ID": 1},
			target:        nil,
			wantErrString: "target must be a pointer",
		},
		{
			name:          "non-pointer target",
			data:          map[string]any{"ID": 1},
			target:        User{},
			wantErrString: "target must be a pointer",
		},
		{
			name:          "target is not a struct or map pointer",
			data:          map[string]any{"ID": 1},
			target:        new(int),
			wantErrString: "target must be a pointer to a struct or a map",
		},
		{
			name:          "mismatched type",
			data:          map[string]any{"ID": "not-an-int"},
			target:        &User{},
			wantErrString: "failed to decode map to struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Decode(tt.data, tt.target)
			if err == nil {
				t.Fatalf("Decode() did not return an error, but expected one containing %q", tt.wantErrString)
			}
			if !strings.Contains(err.Error(), tt.wantErrString) {
				t.Errorf("Decode() error = %q, want error containing %q", err, tt.wantErrString)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	// String to Int
	val, err := Convert("123", cnst.INT)
	if err != nil || val.(int) != 123 {
		t.Errorf("Convert string to int failed. got: %v, err: %v", val, err)
	}

	// Int to String
	val, err = Convert(456, cnst.STRING)
	if err != nil || val.(string) != "456" {
		t.Errorf("Convert int to string failed. got: %v, err: %v", val, err)
	}

	// String to Bool
	val, err = Convert("true", cnst.BOOL)
	if err != nil || val.(bool) != true {
		t.Errorf("Convert string to bool failed. got: %v, err: %v", val, err)
	}
}

func TestConvert_SliceToStringSlice(t *testing.T) {
	// This test simulates the core issue from the http_endpoint bug.
	// 1. The http_endpoint receives a []string from query parameters.
	sourceSlice := []any{1, 1.01, "id3"}

	// 2. It calls Convert to ensure the type is correct for the target struct.
	convertedVal, err := Convert(sourceSlice, cnst.LIST_PREFIX+cnst.STRING)
	if err != nil {
		t.Fatalf("Convert from []string to []string failed: %v", err)
	}

	// 3. The result is placed into a map, which is then decoded into a struct.
	dataMap := map[string]any{
		"ArrayParam": convertedVal,
	}

	targetStruct := struct {
		ArrayParam []string
	}{}

	err = Decode(dataMap, &targetStruct)
	if err != nil {
		t.Fatalf("Decode failed after conversion: %v", err)
	}

	// 4. Assert that the final struct has the correct slice.
	expected := []string{"1", "1.01", "id3"}
	if !reflect.DeepEqual(targetStruct.ArrayParam, expected) {
		t.Errorf("Expected ArrayParam to be %v, but got %v", expected, targetStruct.ArrayParam)
	}
}

// TestConvert_Comprehensive provides a comprehensive test suite for the Convert function.
// 测试函数: Convert
// 测试点: 覆盖全面的类型转换，包括nil、数字、布尔、字符串、map/slice等。
func TestConvert_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		value         any
		targetType    cnst.MType
		want          any
		wantErr       bool
		wantErrString string // New field to check for specific error messages
	}{
		// Nil conversions
		{"nil to string", nil, cnst.STRING, "", false, ""},
		{"nil to int", nil, cnst.INT, 0, false, ""},
		{"nil to int64", nil, cnst.INT64, int64(0), false, ""},
		{"nil to float", nil, cnst.FLOAT, 0.0, false, ""},
		{"nil to bool", nil, cnst.BOOL, false, false, ""},
		{"nil to map", nil, cnst.MAP, nil, false, ""},
		{"nil to slice", nil, cnst.LIST_PREFIX + cnst.STRING, nil, false, ""},

		// Numeric conversions
		{"int to float", 123, cnst.FLOAT, float64(123), false, ""},
		{"float to int", 123.7, cnst.INT, 123, false, ""},
		{"uint to int", uint(456), cnst.INT, 456, false, ""},
		{"int to int64", 123, cnst.INT64, int64(123), false, ""},

		// Bool conversions
		{"true to int", true, cnst.INT, 1, false, ""},
		{"false to int", false, cnst.INT, 0, false, ""},
		{"true to int64", true, cnst.INT64, int64(1), false, ""},
		{"false to int64", false, cnst.INT64, int64(0), false, ""},
		{"true to string", true, cnst.STRING, "true", false, ""},
		{"int 1 to bool", 1, cnst.BOOL, true, false, ""},
		{"int 0 to bool", 0, cnst.BOOL, false, false, ""},

		// String to other types
		{"string to float", "123.45", cnst.FLOAT, 123.45, false, ""},
		{"string to int error", "not-a-number", cnst.INT, 0, true, `strconv.Atoi: parsing "not-a-number": invalid syntax`},
		{"string to int64", "123", cnst.INT64, int64(123), false, ""},
		{"string to int64 error", "not-a-number", cnst.INT64, int64(0), true, `strconv.ParseInt: parsing "not-a-number": invalid syntax`},
		{"string 'true' to bool", "true", cnst.BOOL, true, false, ""},
		{"string '1' to bool", "1", cnst.BOOL, true, false, ""},
		{"string 'false' to bool", "false", cnst.BOOL, false, false, ""},
		{"string '0' to bool", "0", cnst.BOOL, false, false, ""},
		{"string empty to bool", "", cnst.BOOL, false, false, ""},
		{"string bool error", "not-a-bool", cnst.BOOL, nil, true, `can't convert string 'not-a-bool' to bool`},

		// JSON string conversions
		{"map to json string", map[string]any{"a": 1}, cnst.STRING, `{"a":1}`, false, ""},
		{"slice to json string", []any{1, "b"}, cnst.STRING, `[1,"b"]`, false, ""},
		{"json string to map", `{"a":1}`, cnst.MAP, map[string]any{"a": float64(1)}, false, ""}, // JSON numbers are float64
		{"json string to slice", `[1,"b"]`, cnst.LIST_PREFIX + cnst.OBJECT, []any{float64(1), "b"}, false, ""},

		// Slice conversions
		{"[]any to []int", []any{1, 2}, cnst.LIST_PREFIX + cnst.INT, []int{1, 2}, false, ""},
		{"[]any to []int64", []any{1, 2}, cnst.LIST_PREFIX + cnst.INT64, []int64{1, 2}, false, ""},
		{"[]any to []string", []any{"a", "b"}, cnst.LIST_PREFIX + cnst.STRING, []string{"a", "b"}, false, ""},
		{"mixed []any to []string", []any{1, "b"}, cnst.LIST_PREFIX + cnst.STRING, []string{"1", "b"}, false, ""},
		{"mixed []any to []int error", []any{1, "b"}, cnst.LIST_PREFIX + cnst.INT, nil, true, "error converting slice element at index 1"},
		{"unsupported type", 123, "unsupported", nil, true, "unsupported target type: unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert(tt.value, tt.targetType)

			if (err != nil) != tt.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.wantErrString != "" && !strings.Contains(err.Error(), tt.wantErrString) {
					t.Errorf("Convert() error = %q, want error containing %q", err, tt.wantErrString)
				}
				// If we expect an error, we often don't need to check the returned value.
				return
			}

			// For json string to map/slice, we need to handle the order difference
			if tt.name == "map to json string" {
				var gotMap, wantMap map[string]any
				json.Unmarshal([]byte(got.(string)), &gotMap)
				json.Unmarshal([]byte(tt.want.(string)), &wantMap)
				if !reflect.DeepEqual(gotMap, wantMap) {
					t.Errorf("Convert() = %v, want %v", got, tt.want)
				}
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Convert() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestExtractByPath(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "john",
			"age":  float64(30),
		},
		"items": []any{"a", "b"},
	}

	// Test map path
	name, found, err := ExtractByPath(data, "user.name")
	if err != nil || !found || name.(string) != "john" {
		t.Errorf("ExtractByPath for map failed. got name: %v, found: %v, err: %v", name, found, err)
	}

	// Test struct path
	type User struct{ Name string }
	structData := struct{ User User }{User: User{Name: "jane"}}
	name, found, err = ExtractByPath(structData, "User.Name")
	if err != nil || !found || name.(string) != "jane" {
		t.Errorf("ExtractByPath for struct failed. got name: %v, found: %v, err: %v", name, found, err)
	}
}

// TestExtractByPath_Boundaries tests boundary conditions for the ExtractByPath function.
// 测试函数: ExtractByPath
// 测试点: 覆盖路径不存在、中间节点为nil、访问非导出字段和操作不支持的类型等边界情况。
func TestExtractByPath_Boundaries(t *testing.T) {
	type unexportedStruct struct {
		name string
	}

	tests := []struct {
		name          string
		data          any
		path          string
		wantVal       any
		wantFound     bool
		wantErrString string
	}{
		{
			name:          "path does not exist",
			data:          map[string]any{"user": map[string]any{"name": "john"}},
			path:          "user.age",
			wantVal:       nil,
			wantFound:     false,
			wantErrString: "",
		},
		{
			name:          "intermediate node is nil",
			data:          map[string]any{"user": nil},
			path:          "user.name",
			wantVal:       nil,
			wantFound:     false,
			wantErrString: "",
		},
		{
			name:          "access unexported field",
			data:          unexportedStruct{name: "john"},
			path:          "name",
			wantVal:       nil,
			wantFound:     false,
			wantErrString: "cannot access unexported field: name",
		},
		{
			name:          "unsupported type for extraction",
			data:          "a string",
			path:          "some.path",
			wantVal:       nil,
			wantFound:     false,
			wantErrString: "unsupported type for path extraction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotFound, err := ExtractByPath(tt.data, tt.path)

			if (err != nil) != (tt.wantErrString != "") {
				t.Errorf("ExtractByPath() error = %v, wantErr %v", err, tt.wantErrString != "")
				return
			}
			if tt.wantErrString != "" && !strings.Contains(err.Error(), tt.wantErrString) {
				t.Errorf("ExtractByPath() error = %q, want error containing %q", err, tt.wantErrString)
			}

			if gotFound != tt.wantFound {
				t.Errorf("ExtractByPath() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
			if !reflect.DeepEqual(gotVal, tt.wantVal) {
				t.Errorf("ExtractByPath() gotVal = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func TestEvalAndExtract(t *testing.T) {
	env := map[string]any{
		"a": 5,
		"b": 10,
	}
	result, err := EvalAndExtract(env, "a + b")
	if err != nil {
		t.Fatalf("EvalAndExtract failed: %v", err)
	}
	if result.(int) != 15 {
		t.Errorf("EvalAndExtract result not as expected. got: %v, want: 15", result)
	}
}

// TestGetValueFromMapByPath_SimplePathSuccess tests a simple path can be successfully retrieved.
// 测试函数: GetValueFromMapByPath
// 测试点: 简单路径能成功取值。
func TestGetValueFromMapByPath_SimplePathSuccess(t *testing.T) {
	data := map[string]any{"status": "active"}
	path := "status"

	val, ok := GetValueFromMapByPath(data, path)

	if !ok {
		t.Errorf("Expected to find value, but ok was false")
	}
	if val != "active" {
		t.Errorf("Expected value 'active', but got %v", val)
	}
}

// TestGetValueFromMapByPath_NestedPathSuccess tests a nested path can be successfully retrieved.
// 测试函数: GetValueFromMapByPath
// 测试点: 嵌套路径能成功取值。
func TestGetValueFromMapByPath_NestedPathSuccess(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"details": map[string]any{"age": 30},
		},
	}
	path := "user.details.age"

	val, ok := GetValueFromMapByPath(data, path)

	if !ok {
		t.Errorf("Expected to find value, but ok was false")
	}
	if val != 30 {
		t.Errorf("Expected value 30, but got %v", val)
	}
}

// TestGetValueFromMapByPath_NotFoundFinalPart tests behavior when the final part of the path does not exist.
// 测试函数: GetValueFromMapByPath
// 测试点: 路径的最后一部分不存在。
func TestGetValueFromMapByPath_NotFoundFinalPart(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"details": map[string]any{"age": 30},
		},
	}
	path := "user.details.city"

	val, ok := GetValueFromMapByPath(data, path)

	if ok {
		t.Errorf("Expected not to find value, but ok was true")
	}
	if val != nil {
		t.Errorf("Expected value to be nil, but got %v", val)
	}
}

// TestGetValueFromMapByPath_NotFoundIntermediatePart tests behavior when an intermediate part of the path does not exist.
// 测试函数: GetValueFromMapByPath
// 测试点: 路径的中间部分不存在。
func TestGetValueFromMapByPath_NotFoundIntermediatePart(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"details": map[string]any{"age": 30},
		},
	}
	path := "user.address.street"

	val, ok := GetValueFromMapByPath(data, path)

	if ok {
		t.Errorf("Expected not to find value, but ok was true")
	}
	if val != nil {
		t.Errorf("Expected value to be nil, but got %v", val)
	}
}

// TestGetValueFromMapByPath_NilMap tests behavior when the input map is nil.
// 测试函数: GetValueFromMapByPath
// 测试点: 输入的 map 为 nil。
func TestGetValueFromMapByPath_NilMap(t *testing.T) {
	path := "a.b"

	val, ok := GetValueFromMapByPath(nil, path)

	if ok {
		t.Errorf("Expected not to find value, but ok was true")
	}
	if val != nil {
		t.Errorf("Expected value to be nil, but got %v", val)
	}
}

// TestGetValueFromMapByPath_EmptyPath tests behavior when the path is an empty string.
// 测试函数: GetValueFromMapByPath
// 测试点: 路径为空字符串。
func TestGetValueFromMapByPath_EmptyPath(t *testing.T) {
	data := map[string]any{"status": "active"}
	path := ""

	val, ok := GetValueFromMapByPath(data, path)

	if ok {
		t.Errorf("Expected not to find value for empty path, but ok was true")
	}
	if val != nil {
		t.Errorf("Expected value to be nil for empty path, but got %v", val)
	}
}
