package runtime

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/types"
)

func init() {
	types.NewMsg = func(msgType, data string, metadata types.Metadata, dataT types.DataT) types.RuleMsg {
		if dataT == nil {
			dataT = contract.NewDataT()
		}
		return contract.NewDefaultRuleMsg(msgType, data, metadata, dataT)
	}
	types.NewDataT = func() types.DataT {
		return contract.NewDataT()
	}
	types.CloneMsgWithDataT = func(msg types.RuleMsg, dataT types.DataT) types.RuleMsg {
		if cloner, ok := msg.(types.RuleMsgDataTCloner); ok {
			return cloner.CloneWithDataT(dataT)
		}
		return contract.NewDefaultRuleMsg(msg.Type(), string(msg.Data()), msg.Metadata().Copy(), dataT).WithDataFormat(msg.DataFormat())
	}
}

func TestCloneMsgForEdgeKeepsAllObjectsWithinRuleChain(t *testing.T) {
	dataT := types.NewDataT()
	dataT.Set("image_push_items", &projectionRuntimeCoreObj{key: "image_push_items", body: "keep"})
	dataT.Set("ttscrapedposts", &projectionRuntimeCoreObj{key: "ttscrapedposts", body: "keep-too"})

	r := &DefaultRuntime{}
	ctx := &DefaultNodeCtx{
		runtime: r,
		selfDef: &types.NodeDef{ID: "node_a"},
	}

	msg := types.NewMsg("test", "", nil, dataT)
	cloned, err := ctx.cloneMsgForEdge(msg, "node_b")
	if err != nil {
		t.Fatalf("cloneMsgForEdge failed: %v", err)
	}

	if _, ok := cloned.DataT().Get("image_push_items"); !ok {
		t.Fatalf("expected cloned message to keep image_push_items")
	}
	if _, ok := cloned.DataT().Get("ttscrapedposts"); !ok {
		t.Fatalf("expected cloned message to keep ttscrapedposts")
	}
	if _, ok := msg.DataT().Get("ttscrapedposts"); !ok {
		t.Fatalf("expected source message to remain unchanged")
	}
}

type projectionRuntimeCoreObj struct {
	key  string
	body any
}

func (o *projectionRuntimeCoreObj) Key() string { return o.key }
func (o *projectionRuntimeCoreObj) Definition() types.CoreObjDef {
	return &projectionRuntimeCoreObjDef{}
}
func (o *projectionRuntimeCoreObj) Body() any { return o.body }
func (o *projectionRuntimeCoreObj) SetBody(body any) error {
	o.body = body
	return nil
}
func (o *projectionRuntimeCoreObj) DeepCopy() (types.CoreObj, error) {
	return &projectionRuntimeCoreObj{key: o.key, body: o.body}, nil
}

type projectionRuntimeCoreObjDef struct{}

func (d *projectionRuntimeCoreObjDef) SID() string                     { return "projection-runtime-test" }
func (d *projectionRuntimeCoreObjDef) New() any                        { return map[string]any{} }
func (d *projectionRuntimeCoreObjDef) Description() string             { return "" }
func (d *projectionRuntimeCoreObjDef) Schema() string                  { return "{}" }
func (d *projectionRuntimeCoreObjDef) OpenAPISchema() *openapi3.Schema { return &openapi3.Schema{} }
func (d *projectionRuntimeCoreObjDef) MarshalJSON() ([]byte, error)    { return []byte(`{}`), nil }
