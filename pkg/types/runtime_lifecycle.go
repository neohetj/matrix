package types

import (
	"fmt"
	"strings"
)

// RuntimeLifecycleOwner identifies the component that is allowed to request a runtime lifecycle operation.
type RuntimeLifecycleOwner string

const (
	RuntimeLifecycleOwnerEngine      RuntimeLifecycleOwner = "engine"
	RuntimeLifecycleOwnerRuntimePool RuntimeLifecycleOwner = "runtime_pool"
	RuntimeLifecycleOwnerHost        RuntimeLifecycleOwner = "host"
	RuntimeLifecycleOwnerTest        RuntimeLifecycleOwner = "test"
)

// RuntimeLifecycleOperation identifies the lifecycle action being requested.
type RuntimeLifecycleOperation string

const (
	RuntimeLifecycleOperationReload  RuntimeLifecycleOperation = "reload"
	RuntimeLifecycleOperationStop    RuntimeLifecycleOperation = "stop"
	RuntimeLifecycleOperationDestroy RuntimeLifecycleOperation = "destroy"
)

// RuntimeLifecycleRequest is the auditable contract for reload / stop / destroy calls.
type RuntimeLifecycleRequest struct {
	RuntimeID string                    `json:"runtimeId"`
	Owner     RuntimeLifecycleOwner     `json:"owner"`
	Operation RuntimeLifecycleOperation `json:"operation"`
	Reason    string                    `json:"reason,omitempty"`
}

func (r RuntimeLifecycleRequest) Validate() error {
	if strings.TrimSpace(r.RuntimeID) == "" {
		return fmt.Errorf("runtime lifecycle request requires runtimeId")
	}
	switch r.Owner {
	case RuntimeLifecycleOwnerEngine, RuntimeLifecycleOwnerRuntimePool, RuntimeLifecycleOwnerHost, RuntimeLifecycleOwnerTest:
	default:
		return fmt.Errorf("runtime lifecycle request has invalid owner: %s", r.Owner)
	}
	switch r.Operation {
	case RuntimeLifecycleOperationReload, RuntimeLifecycleOperationStop, RuntimeLifecycleOperationDestroy:
	default:
		return fmt.Errorf("runtime lifecycle request has invalid operation: %s", r.Operation)
	}
	return nil
}
