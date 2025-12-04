package test_utils

import (
	"errors"

	"gitlab.com/neohet/matrix/pkg/types"
)

// NewTestRuleMsg creates a simple RuleMsg for testing.
func NewTestRuleMsg() types.RuleMsg {
	return types.NewMsg("test", "", nil, nil)
}

// GetRootError traverses the error chain and returns the root Fault.
func GetRootError(err error) *types.Fault {
	var fault *types.Fault
	if errors.As(err, &fault) {
		for {
			unwrapped := errors.Unwrap(fault)
			if unwrapped == nil {
				return fault
			}
			if nextFault, ok := unwrapped.(*types.Fault); ok {
				fault = nextFault
			} else {
				break
			}
		}
	}
	return fault
}
