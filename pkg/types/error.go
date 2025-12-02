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
	// ErrInternal is a generic internal error.
	ErrInternal = &ErrorObj{Code: int32(CodeInternalError), Message: "internal server error"}
	// ErrInvalidParams indicates that the provided parameters are invalid.
	ErrInvalidParams = &ErrorObj{Code: int32(CodeInvalidParams), Message: "invalid parameters"}
	// ErrInvalidConfiguration indicates that a component's configuration is invalid or missing required fields.
	ErrInvalidConfiguration = &ErrorObj{Code: int32(CodeInvalidConfiguration), Message: "invalid configuration"}
	// ErrNodeNotFound indicates that a requested node was not found in the chain.
	ErrNodeNotFound = &ErrorObj{Code: int32(CodeNodeNotFound), Message: "node not found"}
	// ErrFuncNotFound indicates that a requested function was not found in the registry.
	ErrFuncNotFound = &ErrorObj{Code: int32(CodeFuncNotFound), Message: "function not found"}
)

// ErrorObj represents a standardized error object in the Matrix engine.
type ErrorObj struct {
	Code    int32
	Message string
	Cause   error
}

// Error implements the standard error interface.
func (e *ErrorObj) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap allows the use of errors.Is and errors.As with ErrorObj.
func (e *ErrorObj) Unwrap() error {
	return e.Cause
}

// Wrap creates a new ErrorObj, wrapping an existing error.
func (e *ErrorObj) Wrap(cause error) *ErrorObj {
	return &ErrorObj{
		Code:    e.Code,
		Message: e.Message,
		Cause:   cause,
	}
}

// ErrorRegistry is the interface for managing a collection of predefined ErrorObj.
type ErrorRegistry interface {
	// Register adds a new error definition.
	Register(errs ...*ErrorObj)
	// Get retrieves an error definition by its code.
	Get(code int32) (*ErrorObj, bool)
}

// ChainError is an alias for a map that carries rich error context from the rule chain.
type ChainError map[string]any

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
