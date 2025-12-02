package test_utils

import (
	"errors"

	"gitlab.com/neohet/matrix/pkg/types"
)

// NewTestRuleMsg creates a simple RuleMsg for testing.
func NewTestRuleMsg() types.RuleMsg {
	return types.NewMsg("test", "", nil, nil)
}

// GetRootError traverses the error chain and returns the root ErrorObj.
func GetRootError(err error) *types.ErrorObj {
	var errObj *types.ErrorObj
	if errors.As(err, &errObj) {
		for {
			unwrapped := errors.Unwrap(errObj)
			if unwrapped == nil {
				return errObj
			}
			if nextErrObj, ok := unwrapped.(*types.ErrorObj); ok {
				errObj = nextErrObj
			} else {
				break
			}
		}
	}
	return errObj
}
