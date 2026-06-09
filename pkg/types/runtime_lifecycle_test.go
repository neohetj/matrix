package types

import "testing"

func TestRuntimeLifecycleRequestValidate(t *testing.T) {
	valid := RuntimeLifecycleRequest{
		RuntimeID: "order_submit",
		Owner:     RuntimeLifecycleOwnerEngine,
		Operation: RuntimeLifecycleOperationReload,
		Reason:    "rulechain definition changed",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid lifecycle request, got %v", err)
	}

	missingOwner := valid
	missingOwner.Owner = ""
	if err := missingOwner.Validate(); err == nil {
		t.Fatalf("expected missing owner to be rejected")
	}

	invalidOperation := valid
	invalidOperation.Operation = RuntimeLifecycleOperation("restart")
	if err := invalidOperation.Validate(); err == nil {
		t.Fatalf("expected invalid operation to be rejected")
	}
}
