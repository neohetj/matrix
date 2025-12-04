/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import "fmt"

var (
	// DefInternalError defines a generic internal error.
	DefInternalError = &Fault{Code: int32(CodeInternalError), Message: "internal server error"}
	// DefInvalidParams defines an error for invalid parameters.
	DefInvalidParams = &Fault{Code: int32(CodeInvalidParams), Message: "invalid parameters"}
	// DefInvalidConfiguration defines an error for invalid component configuration.
	DefInvalidConfiguration = &Fault{Code: int32(CodeInvalidConfiguration), Message: "invalid configuration"}
	// DefNodeNotFound defines an error for a missing node.
	DefNodeNotFound = &Fault{Code: int32(CodeNodeNotFound), Message: "node not found"}
	// DefFuncNotFound defines an error for a missing function.
	DefFuncNotFound = &Fault{Code: int32(CodeFuncNotFound), Message: "function not found"}
)

// ServiceError represents a standardized error object returned by a service endpoint.
// It is the outermost error layer, intended for consumption by external clients.
type ServiceError struct {
	// ResponseCode is the protocol-specific error code (e.g., HTTP status code 4xx/5xx, gRPC code)
	// that should be returned to the client.
	ResponseCode int32
	// UserMessage is a human-readable, safe-to-display message for the end user.
	UserMessage string
	// Cause represents a system-level or execution error that occurred outside of the rule chain logic.
	// It is typically a Go error (e.g., network failure, JSON marshalling error).
	Cause error
	// FailureInfo represents a business logic error or a specific node failure within the rule chain.
	// It contains structured information about which node failed and why, derived from a Fault.
	FailureInfo *FailureInfo
}

// Error implements the standard error interface.
func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.UserMessage, e.Cause)
	}
	return e.UserMessage
}

// Unwrap allows the use of errors.Is and errors.As with ServiceError.
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// Wrap creates a new ServiceError, wrapping an existing error.
func (e *ServiceError) Wrap(cause error) *ServiceError {
	return &ServiceError{
		ResponseCode: e.ResponseCode,
		UserMessage:  e.UserMessage,
		Cause:        cause,
		FailureInfo:  e.FailureInfo,
	}
}

// WithFailureInfo creates a new ServiceError with the provided FailureInfo.
func (e *ServiceError) WithFailureInfo(failureInfo *FailureInfo) *ServiceError {
	return &ServiceError{
		ResponseCode: e.ResponseCode,
		UserMessage:  e.UserMessage,
		Cause:        e.Cause,
		FailureInfo:  failureInfo,
	}
}

// FaultRegistry is the interface for managing a collection of predefined Fault.
type FaultRegistry interface {
	// Register adds a new error definition.
	Register(errs ...*Fault)
	// Get retrieves an error definition by its code.
	Get(code int32) (*Fault, bool)
}

// FailureInfo is a struct that carries rich error context from the rule chain.
// It represents a specific failure instance at runtime, including which node failed and when.
type FailureInfo struct {
	Error     string `json:"error"`
	NodeID    string `json:"error_node_id"`
	NodeName  string `json:"error_node_name"`
	Timestamp string `json:"error_timestamp"`
	Code      string `json:"error_code"`
}

// Fault represents a static, predefined specification of an error condition.
// It is defined at development time and includes a unique code and a message format.
// A Fault is the root cause that can lead to a runtime Failure.
type Fault struct {
	Code    int32
	Message string
	Wrapped error
}

// Error implements the error interface for Fault.
func (e *Fault) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("error code: %d, message: %s, cause: %v", e.Code, e.Message, e.Wrapped)
	}
	return fmt.Sprintf("error code: %d, message: %s", e.Code, e.Message)
}

// Unwrap returns the cause of the fault.
func (e *Fault) Unwrap() error {
	return e.Wrapped
}

// Wrap creates a new Fault instance with the same code and message, but with the cause set.
func (e *Fault) Wrap(err error) *Fault {
	return &Fault{
		Code:    e.Code,
		Message: e.Message,
		Wrapped: err,
	}
}

const (
	// MetaError is the key for the raw error object in the metadata.
	MetaError = "error"
	// MetaErrorNodeID is the key for the ID of the node where the error occurred.
	MetaErrorNodeID = "error_node_id"
	// MetaErrorNodeName is the key for the name of the node where the error occurred.
	MetaErrorNodeName = "error_node_name"
	// MetaErrorTimestamp is the key for the timestamp when the error occurred.
	MetaErrorTimestamp = "error_timestamp"
	// MetaErrorCode is the key for the structured error code.
	MetaErrorCode = "error_code"
)
