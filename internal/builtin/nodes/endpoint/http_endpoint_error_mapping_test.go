package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

type recordingServiceErrorAspect struct {
	received *types.ServiceError
}

func (a *recordingServiceErrorAspect) Handle(err *types.ServiceError) error {
	a.received = err
	return fmt.Errorf("mapped product error: %w", &types.ServiceError{
		ResponseCode: http.StatusUnprocessableEntity,
		UserMessage:  "product data is invalid",
	})
}

func TestWriteResponse_HidesServiceErrorCause(t *testing.T) {
	node := &HttpEndpointNode{}
	recorder := httptest.NewRecorder()
	serviceErr := &types.ServiceError{
		ResponseCode: http.StatusBadRequest,
		UserMessage:  "invalid request",
		Cause:        errors.New("invalid URL format: asfsdf"),
	}

	node.writeResponse(recorder, http.StatusBadRequest, nil, nil, serviceErr)

	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != "invalid request" {
		t.Fatalf("expected safe message, got %q", response.Message)
	}
	if strings.Contains(recorder.Body.String(), "asfsdf") {
		t.Fatalf("response exposed internal cause: %s", recorder.Body.String())
	}
}

func TestWriteResponse_HidesPlainInternalError(t *testing.T) {
	node := &HttpEndpointNode{}
	recorder := httptest.NewRecorder()

	node.writeResponse(recorder, http.StatusInternalServerError, nil, nil, errors.New("database password leaked"))

	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != "internal server error" {
		t.Fatalf("expected generic message, got %q", response.Message)
	}
	if strings.Contains(recorder.Body.String(), "password") {
		t.Fatalf("response exposed internal error: %s", recorder.Body.String())
	}
}

func TestHandleHttpRequest_RequestMappingUsesStructuredSafeError(t *testing.T) {
	node := &HttpEndpointNode{
		nodeConfig: types.HttpEndpointNodeConfiguration{
			RuleChainID: "unused",
			HttpMethod:  http.MethodPost,
			HttpPath:    "/products",
			EndpointDefinition: types.HttpEndpointDef{
				Request: types.HttpRequestDef{
					Body: types.EndpointIOPacket{
						Fields: []types.EndpointIOField{
							{
								Name:     "product",
								Type:     "string",
								BindPath: "rulemsg://metadata/product",
							},
						},
					},
				},
			},
		},
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(`{"product":`))
	request.Header.Set("Content-Type", "application/json")
	aspect := &recordingServiceErrorAspect{}

	if err := node.HandleHttpRequest(recorder, request, types.WithErrorAspect(aspect)); err != nil {
		t.Fatalf("handle request: %v", err)
	}
	if aspect.received == nil || aspect.received.FailureInfo == nil {
		t.Fatal("expected the aspect to receive structured failure info")
	}
	if aspect.received.FailureInfo.Code != string(cnst.CodeRequestDecodingFailed) {
		t.Fatalf("expected request decoding code, got %q", aspect.received.FailureInfo.Code)
	}
	if aspect.received.UserMessage != "invalid request" {
		t.Fatalf("expected safe request message, got %q", aspect.received.UserMessage)
	}
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected aspect response code %d, got %d", http.StatusUnprocessableEntity, recorder.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != "product data is invalid" {
		t.Fatalf("expected mapped public message, got %q", response.Message)
	}
}

func TestCreateServiceErrorFromExecErr_MapsFaultCode(t *testing.T) {
	node := &HttpEndpointNode{
		faultCodeMap: map[string]int32{
			"40005000": http.StatusBadRequest,
		},
		defaultErrorCode: http.StatusBadGateway,
	}

	serviceErr := node.createServiceErrorFromExecErr(types.NewFault("40005000", "bad request from flow"))
	if serviceErr == nil {
		t.Fatalf("expected service error, got nil")
	}
	if serviceErr.ResponseCode != http.StatusBadRequest {
		t.Fatalf("expected response code %d, got %d", http.StatusBadRequest, serviceErr.ResponseCode)
	}
	if serviceErr.FailureInfo == nil {
		t.Fatalf("expected failure info to be set")
	}
	if serviceErr.FailureInfo.Code != "40005000" {
		t.Fatalf("expected failure code 40005000, got %q", serviceErr.FailureInfo.Code)
	}
	// 结构化 fault 的 Message 应作为面向用户的公开文案透出，而非默认兜底文案。
	if serviceErr.UserMessage != "bad request from flow" {
		t.Fatalf("expected fault message as public message, got %q", serviceErr.UserMessage)
	}
}

func TestCreateServiceErrorFromExecErr_KeepsDefaultMessageForPlainError(t *testing.T) {
	node := &HttpEndpointNode{
		defaultErrorCode: http.StatusBadGateway,
	}

	serviceErr := node.createServiceErrorFromExecErr(errors.New("internal db timeout"))
	if serviceErr == nil {
		t.Fatalf("expected service error, got nil")
	}
	if serviceErr.ResponseCode != http.StatusBadGateway {
		t.Fatalf("expected response code %d, got %d", http.StatusBadGateway, serviceErr.ResponseCode)
	}
	if serviceErr.FailureInfo == nil || serviceErr.FailureInfo.Code != string(cnst.CodeInternalError) {
		t.Fatalf("expected internal error code, got %+v", serviceErr.FailureInfo)
	}
	// 普通内部错误不泄露原因，保持默认兜底文案。
	if serviceErr.UserMessage != "service unavailable" {
		t.Fatalf("expected safe default message, got %q", serviceErr.UserMessage)
	}
}

func TestCreateServiceErrorFromMsg_PrefersFaultUserMessage(t *testing.T) {
	node := &HttpEndpointNode{
		faultCodeMap: map[string]int32{
			"REDEMPTION_CODE_INCORRECT": http.StatusUnprocessableEntity,
		},
		defaultErrorCode: http.StatusBadGateway,
	}

	msg := types.NewMsg("test", "", types.Metadata{
		types.MetaErrorCode:        "REDEMPTION_CODE_INCORRECT",
		types.MetaErrorMessage:     "兑换码输入错误",
	}, nil)

	serviceErr := node.createServiceErrorFromMsg(msg, "boom")
	if serviceErr == nil {
		t.Fatalf("expected service error, got nil")
	}
	if serviceErr.ResponseCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected response code %d, got %d", http.StatusUnprocessableEntity, serviceErr.ResponseCode)
	}
	if serviceErr.FailureInfo.Code != "REDEMPTION_CODE_INCORRECT" {
		t.Fatalf("expected failure code, got %q", serviceErr.FailureInfo.Code)
	}
	if serviceErr.UserMessage != "兑换码输入错误" {
		t.Fatalf("expected fault user message, got %q", serviceErr.UserMessage)
	}
}

func TestCreateServiceErrorFromMsg_MapsQuotedFaultCode(t *testing.T) {
	node := &HttpEndpointNode{
		faultCodeMap: map[string]int32{
			"40005000": http.StatusBadRequest,
		},
		defaultErrorCode: http.StatusBadGateway,
	}

	msg := types.NewMsg("test", "", types.Metadata{
		types.MetaErrorCode: "\"40005000\"",
	}, nil)

	serviceErr := node.createServiceErrorFromMsg(msg, "boom")
	if serviceErr == nil {
		t.Fatalf("expected service error, got nil")
	}
	if serviceErr.ResponseCode != http.StatusBadRequest {
		t.Fatalf("expected response code %d, got %d", http.StatusBadRequest, serviceErr.ResponseCode)
	}
	if serviceErr.FailureInfo == nil {
		t.Fatalf("expected failure info to be set")
	}
	if serviceErr.FailureInfo.Code != "40005000" {
		t.Fatalf("expected normalized code 40005000, got %q", serviceErr.FailureInfo.Code)
	}
	if serviceErr.UserMessage != "invalid request" {
		t.Fatalf("expected safe public message, got %q", serviceErr.UserMessage)
	}
}
