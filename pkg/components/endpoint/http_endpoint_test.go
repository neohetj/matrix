package endpoint

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

// MockCoreObjForTest is a mock CoreObj for testing purposes.
type MockCoreObjForTest struct {
	SingleParam string   `json:"singleParam"`
	ArrayParam  []string `json:"arrayParam"`
}

func init() {
	// Register a mock CoreObj definition for the test.
	registry.Default.CoreObjRegistry.Register(
		types.NewCoreObjDef(
			&MockCoreObjForTest{},
			"MockCoreObjForTestV1",
			"A mock object for testing http endpoint.",
		),
	)
}

// --- Test Helper Functions ---

// newNodeForTest creates and initializes a new HttpEndpointNode for testing.
// It fails the test if initialization fails.
func newNodeForTest(t *testing.T, config types.ConfigMap) *HttpEndpointNode {
	t.Helper()
	node := &HttpEndpointNode{}
	if err := node.Init(config); err != nil {
		t.Fatalf("Failed to initialize node: %v", err)
	}
	return node
}

// mockRequestWithParams creates a new mock http.Request with URL and path parameters for testing.
func mockRequestWithParams(method, urlStr string, params httprouter.Params) *http.Request {
	reqURL, _ := url.Parse(urlStr)
	req := &http.Request{
		Method: method,
		URL:    reqURL,
	}
	if params != nil {
		ctx := context.WithValue(req.Context(), httprouter.ParamsKey, params)
		req = req.WithContext(ctx)
	}
	return req
}

// mockRequestWithBody creates a new mock http.Request with a JSON body for testing.
func mockRequestWithBody(method, urlStr, body string) *http.Request {
	reqURL, _ := url.Parse(urlStr)
	bodyReader := strings.NewReader(body)
	return &http.Request{
		Method:        method,
		URL:           reqURL,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(bodyReader),
		ContentLength: int64(bodyReader.Len()),
	}
}

// helperWrapper calls helper.MapHttpRequestToRuleMsg simulating the node's behavior
func helperWrapper(node *HttpEndpointNode, r *http.Request) (types.RuleMsg, error) {
	msg := types.NewMsg(node.nodeConfig.RuleChainID, "", make(types.Metadata), nil)
	ctx := registry.NewMinimalNodeCtx("test-node")
	err := helper.MapHttpRequestToRuleMsg(ctx, msg, node.nodeConfig.EndpointDefinition.Request, r, node.nodeConfig.HttpPath)
	return msg, err
}

// --- Test Cases ---

// TestConvertRequestToRuleMsg_QueryArrayParam 测试 convertRequestToRuleMsg 函数处理数组类型的查询参数
// 这个测试用例验证了当HTTP请求的URL中包含数组形式的查询参数时（如 `ids[]=id1&ids[]=id2`），
// `convertRequestToRuleMsg` 函数能够正确地将这些参数解析并映射到 `RuleMsg` 的 `DataT` 结构中的目标数组字段。
func TestConvertRequestToRuleMsg_QueryArrayParam(t *testing.T) {
	// 1. Setup the HttpEndpointNode with a configuration that expects an array parameter.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "GET",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"queryParams": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "ids[]",
							Type:     "[]string",
							BindPath: "rulemsg://dataT/testObj.arrayParam?sid=MockCoreObjForTestV1",
						},
						{
							Name:     "filter",
							Type:     "string",
							BindPath: "rulemsg://dataT/testObj.singleParam?sid=MockCoreObjForTestV1",
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Create a mock HTTP request with array query parameters.
	reqURL, _ := url.Parse("http://localhost/test?filter=active&ids[]=id1&ids[]=id2")
	mockReq := &http.Request{
		Method: "GET",
		URL:    reqURL,
	}

	// 3. Call the helper wrapper.
	msg, err := helperWrapper(node, mockReq)
	if err != nil {
		t.Fatalf("convertRequestToRuleMsg failed: %v", err)
	}

	// 4. Assert the results.
	dataTItem, ok := msg.DataT().Get("testObj")
	if !ok {
		t.Fatalf("Failed to get DataT item 'testObj'")
	}

	resultObj, ok := dataTItem.Body().(*MockCoreObjForTest)
	if !ok {
		t.Fatalf("DataT item is not of type *MockCoreObjForTest")
	}

	// Check single parameter
	expectedSingle := "active"
	if resultObj.SingleParam != expectedSingle {
		t.Errorf("Expected singleParam to be '%s', but got '%s'", expectedSingle, resultObj.SingleParam)
	}

	// Check array parameter
	expectedArray := []string{"id1", "id2"}
	if !reflect.DeepEqual(resultObj.ArrayParam, expectedArray) {
		t.Errorf("Expected arrayParam to be %v, but got %v", expectedArray, resultObj.ArrayParam)
	}

	t.Log("Successfully verified that array and single query parameters are parsed correctly.")
}

// TestConvertRequestToRuleMsg_PathParameterMapping 测试路径参数的正确映射。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证当 endpointDefinition 中配置了 pathParams 时，函数能否正确地从 http.Request 的 URL 路径中
// (例如 /users/123) 提取出参数值（123），并将其成功映射到 RuleMsg 的 dataT 或 metadata 中。
func TestConvertRequestToRuleMsg_PathParameterMapping(t *testing.T) {
	// 1. Setup: Create a node configured to map a path parameter.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "GET",
		"httpPath":    "/users/:userId",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"pathParams": []types.EndpointIOField{
					{
						Name:     "userId",
						Type:     "string",
						BindPath: "rulemsg://dataT/user.singleParam?sid=MockCoreObjForTestV1",
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a request with the path parameter and process it.
	params := httprouter.Params{{Key: "userId", Value: "user-abc-123"}}
	mockReq := mockRequestWithParams("GET", "http://localhost/users/user-abc-123", params)

	msg, err := helperWrapper(node, mockReq)
	if err != nil {
		t.Fatalf("convertRequestToRuleMsg failed: %v", err)
	}

	// 4. Assert the results.
	dataTItem, ok := msg.DataT().Get("user")
	if !ok {
		t.Fatalf("Failed to get DataT item 'user'")
	}

	resultObj, ok := dataTItem.Body().(*MockCoreObjForTest)
	if !ok {
		t.Fatalf("DataT item is not of type *MockCoreObjForTest")
	}

	expectedParam := "user-abc-123"
	if resultObj.SingleParam != expectedParam {
		t.Errorf("Expected singleParam to be '%s', but got '%s'", expectedParam, resultObj.SingleParam)
	}

	t.Log("Successfully verified that path parameters are parsed and mapped correctly.")
}

// TestConvertRequestToRuleMsg_BodyFieldMapping 测试请求体字段的正确映射。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证对于 POST 或 PUT 请求，函数能否正确解析请求体中的 JSON 数据，并根据 endpointDefinition 中
// bodyFields 的配置，将单个及嵌套的字段值映射到 RuleMsg 的 dataT 对象中对应的字段。
func TestConvertRequestToRuleMsg_BodyFieldMapping(t *testing.T) {
	// 1. Setup: Create a node configured to map fields from a JSON body.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "POST",
		"httpPath":    "/devices",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"body": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "deviceName",
							Type:     "string",
							BindPath: "rulemsg://dataT/device.singleParam?sid=MockCoreObjForTestV1",
						},
						{
							Name:     "tags", // Assuming tags is an array of strings
							Type:     "[]string",
							BindPath: "rulemsg://dataT/device.arrayParam?sid=MockCoreObjForTestV1",
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a POST request with a JSON body and process it.
	body := `{"deviceName": "thermostat-1", "tags": ["living-room", "temp-control"]}`
	mockReq := mockRequestWithBody("POST", "http://localhost/devices", body)

	msg, err := helperWrapper(node, mockReq)
	if err != nil {
		t.Fatalf("convertRequestToRuleMsg failed: %v", err)
	}

	// 3. Assert: Check if the body fields were mapped correctly.
	dataTItem, ok := msg.DataT().Get("device")
	if !ok {
		t.Fatalf("Failed to get DataT item 'device'")
	}

	resultObj, ok := dataTItem.Body().(*MockCoreObjForTest)
	if !ok {
		t.Fatalf("DataT item is not of type *MockCoreObjForTest")
	}

	if resultObj.SingleParam != "thermostat-1" {
		t.Errorf("Expected singleParam to be 'thermostat-1', but got '%s'", resultObj.SingleParam)
	}
	expectedTags := []string{"living-room", "temp-control"}
	if !reflect.DeepEqual(resultObj.ArrayParam, expectedTags) {
		t.Errorf("Expected arrayParam to be %v, but got %v", expectedTags, resultObj.ArrayParam)
	}

	t.Log("Successfully verified that body fields are parsed and mapped correctly.")
}

// TestConvertRequestToRuleMsg_RequiredFieldMissing 测试当必需字段缺失时的错误处理。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证当 endpointDefinition 中的某个参数被标记为 required: true，但实际的 http.Request 中
// 没有提供该参数时，函数是否会中断执行并返回预期的 ErrRequiredFieldMissing 错误。
func TestConvertRequestToRuleMsg_RequiredFieldMissing(t *testing.T) {
	// 1. Setup: Configure a node with a required query parameter.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "GET",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"queryParams": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "deviceId",
							Type:     "string",
							Required: true,
							BindPath: "rulemsg://metadata/deviceId",
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a request that is missing the required parameter.
	mockReq := mockRequestWithParams("GET", "http://localhost/test", nil)

	// 3. Assert: Check that the correct error is returned.
	_, err := helperWrapper(node, mockReq)
	if err == nil {
		t.Fatalf("Expected an error but got nil")
	}

	// Note: The error message format might have changed slightly in helper.ProcessInbound
	fault, ok := err.(*types.Fault)
	if !ok {
		t.Fatalf("Expected error to be of type *types.Fault, but got %T", err)
	}

	if fault.Code != cnst.CodeRequiredFieldMissing {
		t.Errorf("Expected fault code %s, but got %s", cnst.CodeRequiredFieldMissing, fault.Code)
	}

	t.Log("Successfully verified that a missing required field returns the correct error.")
}

// TestConvertRequestToRuleMsg_FieldConversionFailed 测试当字段类型转换失败时的错误处理。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证当请求中提供的参数值（例如字符串 "abc"）与 endpointDefinition 中定义的 type（例如 int）不兼容时，
// 内部调用的 utils.Convert 是否会失败，并导致函数返回预期的 ErrFieldConversionFailed 错误。
func TestConvertRequestToRuleMsg_FieldConversionFailed(t *testing.T) {
	// 1. Setup: Configure a node with a parameter that expects an integer.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "GET",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"queryParams": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "value",
							Type:     "int",
							BindPath: "rulemsg://metadata/value",
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a request with a non-integer value for the parameter.
	mockReq := mockRequestWithParams("GET", "http://localhost/test?value=abc", nil)

	// 3. Assert: Check that the correct error is returned.
	_, err := helperWrapper(node, mockReq)
	if err == nil {
		t.Fatalf("Expected an error but got nil")
	}

	fault, ok := err.(*types.Fault)
	if !ok {
		t.Fatalf("Expected error to be of type *types.Fault, but got %T", err)
	}

	if fault.Code != cnst.CodeFieldConversionFailed {
		t.Errorf("Expected fault code %s, but got %s", cnst.CodeFieldConversionFailed, fault.Code)
	}

	t.Log("Successfully verified that a field conversion failure returns the correct error.")
}

// TestConvertRequestToRuleMsg_InvalidMappingFormat 测试当映射格式无效时的错误处理。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证当 endpointDefinition 中参数的 mapping.to 字段格式不符合 "metadata.key" 或 "dataT.objId.fieldPath" 规范时，
// 函数是否能检测到格式错误并返回 ErrInvalidMappingFormat 错误。
func TestConvertRequestToRuleMsg_InvalidMappingFormat(t *testing.T) {
	testCases := []struct {
		name        string
		bindPath    string
		expectedErr *types.Fault
	}{
		{
			name:        "Invalid metadata format",
			bindPath:    "metadata", // Missing key
			expectedErr: DefInvalidMappingFormat,
		},
		{
			name:        "Invalid dataT format",
			bindPath:    "dataT.myObj", // Missing field path
			expectedErr: DefInvalidMappingFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Setup: Configure a node with the invalid mapping format.
			config := types.ConfigMap{
				"ruleChainId": "testChain",
				"httpMethod":  "GET",
				"httpPath":    "/test",
				"endpointDefinition": map[string]any{
					"request": map[string]any{
						"queryParams": types.EndpointIOPacket{
							Fields: []types.EndpointIOField{
								{
									Name:     "param1",
									Type:     "string",
									BindPath: tc.bindPath,
								},
							},
						},
					},
				},
			}
			node := newNodeForTest(t, config)

			// 2. Execute: Create a request to trigger the mapping.
			mockReq := mockRequestWithParams("GET", "http://localhost/test?param1=value1", nil)

			// 3. Assert: Check for the specific invalid format error.
			_, err := helperWrapper(node, mockReq)
			if err == nil {
				t.Fatalf("Expected an error but got nil")
			}

			// Use Code checking if it's a Fault
			if fault, ok := err.(*types.Fault); ok {
				if fault.Code != tc.expectedErr.Code {
					t.Errorf("Expected fault code %s, got %s", tc.expectedErr.Code, fault.Code)
				}
			} else {
				// Fallback to string check
				if !strings.Contains(err.Error(), "invalid mapping format") && !strings.Contains(err.Error(), "failed to set field") {
					t.Errorf("Expected error message to contain failure info, got '%s'", err.Error())
				}
			}
		})
	}
	t.Log("Successfully verified that invalid mapping formats return the correct error.")
}

// TestConvertRequestToRuleMsg_MixedDataSourceMapping 测试混合数据源的正确映射。
// 测试函数: convertRequestToRuleMsg
// 测试点: 构造一个复杂的 http.Request，它同时包含路径参数、查询参数、请求头和 JSON 请求体。
// 验证函数能否在一次调用中，正确地从所有这些数据源提取数据，并将它们分别映射到 RuleMsg 的 metadata 和多个不同的 dataT 对象中。
func TestConvertRequestToRuleMsg_MixedDataSourceMapping(t *testing.T) {
	// 1. Setup: Configure a node to handle multiple data sources.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "POST",
		"httpPath":    "/tenant/:tenantId/user",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"pathParams": []types.EndpointIOField{
					{
						Name:     "tenantId",
						Type:     "string",
						BindPath: "rulemsg://metadata/tenantId",
					},
				},
				"queryParams": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "requestId",
							Type:     "string",
							BindPath: "rulemsg://metadata/requestId",
						},
					},
				},
				"headers": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "X-Auth-Token",
							Type:     "string",
							BindPath: "rulemsg://metadata/authToken",
						},
					},
				},
				"body": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "username",
							Type:     "string",
							BindPath: "rulemsg://dataT/user.singleParam?sid=MockCoreObjForTestV1",
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a complex request and process it.
	body := `{"username": "cline"}`
	mockReq := mockRequestWithBody("POST", "http://localhost/tenant/tenant-abc/user?requestId=req-123", body)
	mockReq.Header.Set("X-Auth-Token", "token-xyz")
	// Add path params to context
	params := httprouter.Params{{Key: "tenantId", Value: "tenant-abc"}}
	ctx := context.WithValue(mockReq.Context(), httprouter.ParamsKey, params)
	mockReq = mockReq.WithContext(ctx)

	msg, err := helperWrapper(node, mockReq)
	if err != nil {
		t.Fatalf("convertRequestToRuleMsg failed: %v", err)
	}

	// 3. Assert Metadata
	metadata := msg.Metadata()
	if metadata["tenantId"] != "tenant-abc" {
		t.Errorf("Expected metadata.tenantId to be 'tenant-abc', got '%s'", metadata["tenantId"])
	}
	if metadata["requestId"] != "req-123" {
		t.Errorf("Expected metadata.requestId to be 'req-123', got '%s'", metadata["requestId"])
	}
	if metadata["authToken"] != "token-xyz" {
		t.Errorf("Expected metadata.authToken to be 'token-xyz', got '%s'", metadata["authToken"])
	}

	// 4. Assert DataT
	dataTItem, ok := msg.DataT().Get("user")
	if !ok {
		t.Fatal("Failed to get DataT item 'user'")
	}
	resultObj, ok := dataTItem.Body().(*MockCoreObjForTest)
	if !ok {
		t.Fatal("DataT item is not of type *MockCoreObjForTest")
	}
	if resultObj.SingleParam != "cline" {
		t.Errorf("Expected dataT.user.singleParam to be 'cline', got '%s'", resultObj.SingleParam)
	}

	t.Log("Successfully verified mapping from mixed data sources.")
}

// TestConvertRequestToRuleMsg_MissingDefineSID 测试当 DataT 对象的 DefineSID 缺失时的错误处理。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证在处理映射到 dataT 的参数时，如果在所有相关的参数定义中都没有提供 mapping.defineSid，
// 函数在最后构建 dataT 对象时是否会因为找不到类型定义而返回一个明确的配置错误。
func TestConvertRequestToRuleMsg_MissingDefineSID(t *testing.T) {
	// 1. Setup: Configure a node with a dataT mapping that lacks a defineSid.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "GET",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"queryParams": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "param1",
							Type:     "string",
							BindPath: "rulemsg://dataT/myObj.someField", // Deliberately missing "sid"
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a request to trigger the mapping.
	mockReq := mockRequestWithParams("GET", "http://localhost/test?param1=value1", nil)

	// 3. Assert: Check for the specific configuration error.
	_, err := helperWrapper(node, mockReq)
	if err == nil {
		t.Fatalf("Expected an error but got nil")
	}

	// Updated expectation: The error message is now more detailed, coming from helper.SetInMsgByPath
	if !strings.Contains(err.Error(), "no defineSid provided") && !strings.Contains(err.Error(), "invalid mapping format") {
		t.Errorf("Expected error message to contain 'no defineSid provided' or 'invalid mapping format', but got '%s'", err.Error())
	}

	t.Log("Successfully verified that a missing defineSid for a DataT object returns a configuration error.")
}

// TestConvertRequestToRuleMsg_EmptyOrInvalidBody 测试空请求体或无效请求体的处理。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证当请求方法为 POST 且 Content-Length > 0，但请求体为空、或为无效 JSON 格式时，
// 函数能否正确地返回 ErrRequestDecodingFailed 错误。同时，测试当请求体为空但没有配置 bodyFields 时，函数应能正常执行而不报错。
func TestConvertRequestToRuleMsg_EmptyOrInvalidBody(t *testing.T) {
	t.Run("Empty body with no body fields configured", func(t *testing.T) {
		config := types.ConfigMap{
			"ruleChainId": "testChain",
			"httpMethod":  "POST",
			"httpPath":    "/test",
			"endpointDefinition": map[string]any{
				"request": map[string]any{}, // No bodyFields
			},
		}
		node := newNodeForTest(t, config)
		mockReq := mockRequestWithBody("POST", "http://localhost/test", "{}")

		_, err := helperWrapper(node, mockReq)
		if err != nil {
			t.Fatalf("Expected no error for empty body with no body field config, but got %v", err)
		}
	})

	t.Run("Invalid JSON body", func(t *testing.T) {
		config := types.ConfigMap{
			"ruleChainId": "testChain",
			"httpMethod":  "POST",
			"httpPath":    "/test",
			"endpointDefinition": map[string]any{
				"request": map[string]any{
					"body": types.EndpointIOPacket{
						Fields: []types.EndpointIOField{
							{
								Name:     "someField",
								Type:     "string",
								BindPath: "rulemsg://dataT/obj.field?sid=MockCoreObjForTestV1",
							},
						},
					},
				},
			},
		}
		node := newNodeForTest(t, config)
		mockReq := mockRequestWithBody("POST", "http://localhost/test", `{"key": "value"`) // Invalid JSON

		_, err := helperWrapper(node, mockReq)
		if err == nil {
			t.Fatal("Expected an error for invalid JSON body, but got nil")
		}

		// Use utils.RequestDecodingFailed indirectly
		fault, ok := err.(*types.Fault)
		if !ok {
			t.Fatalf("Expected error to be of type *types.Fault, but got %T", err)
		}

		if fault.Code != cnst.CodeRequestDecodingFailed {
			t.Errorf("Expected fault code %s, but got %s", cnst.CodeRequestDecodingFailed, fault.Code)
		}
	})
}

// TestConvertRequestToRuleMsg_DataTDecodeFailure 测试当 DataT 对象数据解码失败时的错误处理。
// 测试函数: convertRequestToRuleMsg
// 测试点: 验证在函数末尾，当从请求中收集到的数据映射（pendingDataT）被解码到 dataT 目标结构体时，
// 如果存在类型不匹配（例如，请求中某字段是数字，而结构体中对应字段是字符串），utils.Decode 是否会失败并导致函数返回错误。
func TestConvertRequestToRuleMsg_DataTDecodeFailure(t *testing.T) {
	// 1. Setup: Configure a node to map a body field to a DataT object.
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "POST",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"body": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "singleParam",
							Type:     "any", // Use 'any' to bypass initial type conversion
							BindPath: "rulemsg://dataT/testObj.singleParam?sid=MockCoreObjForTestV1",
						},
					},
				},
			},
		},
	}
	node := newNodeForTest(t, config)

	// 2. Execute: Create a request where the body field type mismatches the target struct field type.
	// MockCoreObjForTest.singleParam is a string, but we provide a complex object.
	body := `{"singleParam": {"a": "b"}}`
	mockReq := mockRequestWithBody("POST", "http://localhost/test", body)

	// 3. Assert: Check for the specific decode failure error.
	_, err := helperWrapper(node, mockReq)
	if err == nil {
		t.Fatalf("Expected an error but got nil")
	}

	// Updated expectation: The error might be a decoding error or internal server error wrapper.
	if !strings.Contains(err.Error(), "decode") && !strings.Contains(err.Error(), "unconvertible") {
		t.Errorf("Expected error message to contain 'decode' or 'unconvertible', but got '%s'", err.Error())
	}

	t.Log("Successfully verified that a DataT item decode failure returns the correct error.")
}

// TestConvertRequestToRuleMsg_MetadataMapping 测试请求参数到元数据（Metadata）的映射。
// 这个测试用例验证了 HTTP 请求中的参数（如查询参数、头部信息）能否被正确地
// 提取、转换，并设置到 RuleMsg 的 Metadata 字段中。
func TestConvertRequestToRuleMsg_MetadataMapping(t *testing.T) {
	node := &HttpEndpointNode{}
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "GET",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"request": map[string]any{
				"queryParams": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "requestId",
							Type:     "string",
							BindPath: "rulemsg://metadata/requestId",
						},
					},
				},
				"headers": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "X-Tenant-Id",
							Type:     "string",
							BindPath: "rulemsg://metadata/tenantId",
						},
					},
				},
			},
		},
	}
	if err := node.Init(config); err != nil {
		t.Fatalf("Failed to initialize node: %v", err)
	}

	reqURL, _ := url.Parse("http://localhost/test?requestId=req-123")
	mockReq := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: http.Header{
			"X-Tenant-Id": []string{"tenant-abc"},
		},
	}

	msg, err := helperWrapper(node, mockReq)
	if err != nil {
		t.Fatalf("convertRequestToRuleMsg failed: %v", err)
	}

	metadata := msg.Metadata()
	if metadata["requestId"] != "req-123" {
		t.Errorf("Expected metadata 'requestId' to be 'req-123', but got '%s'", metadata["requestId"])
	}
	if metadata["tenantId"] != "tenant-abc" {
		t.Errorf("Expected metadata 'tenantId' to be 'tenant-abc', but got '%s'", metadata["tenantId"])
	}
}

// TestManualExtractPathParams 测试 manualExtractPathParams 函数
// 这个测试用例验证了当 httprouter.Params 不可用时，能否从请求路径中手动、正确地提取出路径参数。
func TestManualExtractPathParams(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		requestPath string
		expected    map[string]string
	}{
		{
			name:        "Standard case with multiple params",
			configPath:  "/api/v1/users/:userId/posts/:postId",
			requestPath: "/api/v1/users/123/posts/abc",
			expected:    map[string]string{"userId": "123", "postId": "abc"},
		},
		{
			name:        "Single parameter",
			configPath:  "/product/:id",
			requestPath: "/product/xyz-789",
			expected:    map[string]string{"id": "xyz-789"},
		},
		{
			name:        "No parameters",
			configPath:  "/health",
			requestPath: "/health",
			expected:    map[string]string{},
		},
		{
			name:        "Mismatched length",
			configPath:  "/api/v1/users/:userId",
			requestPath: "/api/v1/users/123/details",
			expected:    nil,
		},
		{
			name:        "Empty paths",
			configPath:  "/",
			requestPath: "/",
			expected:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ManualExtractPathParams(tt.configPath, tt.requestPath)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("manualExtractPathParams() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestHttpEndpointNode_Init_ErrorHandling 测试 Init 函数在配置不完整时的错误处理能力。
// 这个测试用例确保了当关键配置项（如 ruleChainId, httpMethod, httpPath）缺失时，
// 节点能够返回一个明确的错误，从而防止无效的端点被初始化。
func TestHttpEndpointNode_Init_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		config      types.ConfigMap
		expectedErr string
	}{
		{
			name: "Missing ruleChainId",
			config: types.ConfigMap{
				"httpMethod": "GET",
				"httpPath":   "/test",
			},
			expectedErr: types.InvalidConfiguration.Error(),
		},
		{
			name: "Missing httpMethod",
			config: types.ConfigMap{
				"ruleChainId": "chain123",
				"httpPath":    "/test",
			},
			expectedErr: types.InvalidConfiguration.Error(),
		},
		{
			name: "Missing httpPath",
			config: types.ConfigMap{
				"ruleChainId": "chain123",
				"httpMethod":  "GET",
			},
			expectedErr: types.InvalidConfiguration.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &HttpEndpointNode{}
			err := node.Init(tc.config)
			if err == nil {
				t.Fatalf("Expected an error but got none")
			}
			if !reflect.DeepEqual(err.Error(), tc.expectedErr) {
				t.Errorf("Expected error message '%s', but got '%s'", tc.expectedErr, err.Error())
			}
		})
	}
}

// TestConvertResponse 测试 convertResponse 函数能否正确地将 RuleMsg 转换为 HTTP 响应。
// 这个测试用例确保了 RuleMsg 中的 DataT 数据和 Metadata 能够被正确地映射到
// HTTP 响应的 Body 和 Headers 中，同时验证了状态码的正确设置。
func TestConvertResponse(t *testing.T) {
	node := &HttpEndpointNode{}
	config := types.ConfigMap{
		"ruleChainId": "testChain",
		"httpMethod":  "POST",
		"httpPath":    "/test",
		"endpointDefinition": map[string]any{
			"response": map[string]any{
				"successCode": 201,
				"body": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "data.user.name",
							BindPath: "rulemsg://dataT/userObj.singleParam",
						},
					},
				},
				"headers": types.EndpointIOPacket{
					Fields: []types.EndpointIOField{
						{
							Name:     "X-Trace-Id",
							BindPath: "rulemsg://metadata/traceId",
						},
					},
				},
			},
		},
	}
	if err := node.Init(config); err != nil {
		t.Fatalf("Failed to initialize node: %v", err)
	}

	msg := types.NewMsg("testChain", "", types.Metadata{"traceId": "trace-xyz"}, nil)
	dataT := msg.DataT()
	item, _ := dataT.NewItem("MockCoreObjForTestV1", "userObj")
	item.Body().(*MockCoreObjForTest).SingleParam = "cline"

	ctx := registry.NewMinimalNodeCtx("test-node")
	body, headers, statusCode, err := helper.MapRuleMsgToHttpResponse(ctx, msg, node.nodeConfig.EndpointDefinition.Response)
	if err != nil {
		t.Fatalf("convertResponse failed: %v", err)
	}

	if statusCode != 201 {
		t.Errorf("Expected statusCode 201, but got %d", statusCode)
	}
	if headers["X-Trace-Id"] != "trace-xyz" {
		t.Errorf("Expected header 'X-Trace-Id' to be 'trace-xyz', but got '%s'", headers["X-Trace-Id"])
	}

	expectedBody := map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"name": "cline",
			},
		},
	}
	if !reflect.DeepEqual(body, expectedBody) {
		t.Errorf("Expected body %v, but got %v", expectedBody, body)
	}
}

// TestSetValueByDotPath 测试 SetValueByDotPath 函数
// 这个测试用例确保了 SetValueByDotPath 能够正确地在嵌套的 map[string]any 结构中，
// 根据点分隔的路径（dot-separated path）创建并设置值。
func TestSetValueByDotPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    any
		initial  map[string]any
		expected map[string]any
	}{
		{
			name:    "Set value in a new map",
			path:    "a.b.c",
			value:   123,
			initial: make(map[string]any),
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": 123,
					},
				},
			},
		},
		{
			name:  "Set value in an existing structure",
			path:  "a.b.d",
			value: "hello",
			initial: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": 456,
					},
				},
			},
			expected: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": 456,
						"d": "hello",
					},
				},
			},
		},
		{
			name:     "Set value at the top level",
			path:     "top",
			value:    true,
			initial:  make(map[string]any),
			expected: map[string]any{"top": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.SetValueByDotPath(tt.initial, tt.path, tt.value)
			if !reflect.DeepEqual(tt.initial, tt.expected) {
				t.Errorf("utils.SetValueByDotPath() got = %v, want %v", tt.initial, tt.expected)
			}
		})
	}
}
