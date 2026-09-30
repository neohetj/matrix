package runtime_test

import (
	"testing"

	"github.com/neohetj/matrix/internal/builtin/base"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/internal/runtime"
	"github.com/neohetj/matrix/internal/scheduler"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
)

type metadataEngine struct {
	types.MatrixEngine
	registry *registry.Registry
}

func (e metadataEngine) NodeManager() types.NodeManager         { return e.registry.NodeManager }
func (e metadataEngine) NodeFuncManager() types.NodeFuncManager { return e.registry.NodeFuncManager }
func (e metadataEngine) SharedNodePool() types.NodePool         { return e.registry.SharedNodePool }
func (e metadataEngine) Logger() types.Logger                   { return nil }

func TestRuntimeFunctionMetadataUsesOwningEngine(t *testing.T) {
	sch, err := scheduler.NewAntsScheduler(2)
	if err != nil {
		t.Fatal(err)
	}
	defer sch.Stop()
	for _, sid := range []string{"First_V1", "Second_V1"} {
		reg := registry.NewRegistry()
		if err := reg.NodeManager.Register(base.FunctionsNodePrototype); err != nil {
			t.Fatal(err)
		}
		reg.NodeFuncManager.Register(&types.NodeFuncObject{Func: func(ctx types.NodeCtx, msg types.RuleMsg) { ctx.TellSuccess(msg) }, FuncObject: types.FuncObject{ID: "instance-metadata", Configuration: types.FuncObjConfiguration{
			Inputs:   []types.IOObject{{ParamName: "input", DefineSID: sid, Required: true}},
			Business: []types.DynamicConfigField{{ID: "enabled", Type: cnst.BOOL, Desc: sid, Default: false, NotEditable: true}},
			Errors:   []*types.Fault{{Code: "sample", Message: sid}},
		}}})
		def := &types.RuleChainDef{RuleChain: types.RuleChainData{ID: "metadata"}, Metadata: types.MetadataData{Nodes: []types.NodeDef{{ID: "fn", Type: "functions", Configuration: types.ConfigMap{"functionName": "instance-metadata"}, Inputs: map[string]any{"input": map[string]any{"objId": "value", "defineSid": sid}}}}}}
		rt, err := runtime.NewDefaultRuntime(sch, def, runtime.WithEngine(metadataEngine{registry: reg}))
		if err != nil {
			t.Fatal(err)
		}
		node, _ := rt.GetChainInstance().GetNode("fn")
		if reads := node.DataContract().Reads; len(reads) != 1 || reads[0] != "rulemsg://dataT/value?sid="+sid {
			t.Fatalf("wrong engine contract: %v", reads)
		}
		field := node.ConfigSchema().Properties["enabled"].Value
		if field.Description != sid || field.Default != false || !field.ReadOnly {
			t.Fatalf("wrong engine schema: %#v", field)
		}
		if faults := node.Errors(); len(faults) != 1 || faults[0].Message != sid {
			t.Fatalf("wrong engine errors: %#v", faults)
		}
	}
}
