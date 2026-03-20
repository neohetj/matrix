package endpoint

import (
	"net/http"
	"testing"

	"github.com/neohetj/matrix/pkg/types"
)

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
}
