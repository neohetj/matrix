package runtimebridge

import (
	"context"
	"testing"

	"github.com/neohetj/matrix/pkg/facotry"
	"github.com/neohetj/matrix/pkg/types"
	tu "github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/mock"
)

func init() {
	types.NewMsg = facotry.NewMsg
	types.NewDataT = facotry.NewDataT
}

func TestExecuteRuleChainRunsRuntimeWithMetadataAndFill(t *testing.T) {
	const ruleChainID = "identityx/rc-auth-sync-user-after-login"
	runtime := &tu.MockRuntime{}
	var gotMsg types.RuleMsg
	returnedMsg := types.NewMsg("returned", "", nil, types.NewDataT())
	runtime.On("ExecuteAndWait", mock.Anything, "start", mock.MatchedBy(func(msg types.RuleMsg) bool {
		gotMsg = msg
		return msg != nil
	}), mock.Anything).Return(returnedMsg, nil)
	engine := &tu.MockEngine{
		RuntimePoolValue: &tu.MockRuntimePool{
			Runtimes: map[string]types.Runtime{ruleChainID: runtime},
		},
	}

	finalMsg, err := ExecuteRuleChain(
		context.Background(),
		engine,
		ruleChainID,
		func(msg types.RuleMsg) error {
			msg.Metadata()["filled"] = "yes"
			return nil
		},
		WithStartNodeID("start"),
		WithExecutionID("exec-1"),
	)
	if err != nil {
		t.Fatalf("ExecuteRuleChain returned error: %v", err)
	}
	if finalMsg != returnedMsg {
		t.Fatalf("expected runtime result to be returned")
	}
	if gotMsg.Type() != ruleChainID {
		t.Fatalf("message type = %q, want %q", gotMsg.Type(), ruleChainID)
	}
	if gotMsg.Metadata()[types.ExecutionIDKey] != "exec-1" {
		t.Fatalf("execution id metadata missing: %#v", gotMsg.Metadata())
	}
	if gotMsg.Metadata()[types.ExecutionStartRuleChainIDKey] != ruleChainID {
		t.Fatalf("start rulechain metadata missing: %#v", gotMsg.Metadata())
	}
	if gotMsg.Metadata()["filled"] != "yes" {
		t.Fatalf("fill metadata missing: %#v", gotMsg.Metadata())
	}
}

func TestExecuteRuleChainMissingRuntime(t *testing.T) {
	engine := &tu.MockEngine{RuntimePoolValue: &tu.MockRuntimePool{}}
	if _, err := ExecuteRuleChain(context.Background(), engine, "missing/rc", nil); err == nil {
		t.Fatalf("expected missing runtime error")
	}
}
