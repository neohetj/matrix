package rulechain

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/types"
)

func TestAnalyzeCoreObjProjection_RequiredInputsFlowAcrossUntouchedRelay(t *testing.T) {
	def := &types.RuleChainDef{
		Metadata: types.MetadataData{
			Nodes: []types.NodeDef{
				{ID: "node_a", Type: "functions"},
				{ID: "node_b", Type: "functions"},
				{ID: "node_c", Type: "functions"},
			},
			Connections: []types.Connection{
				{FromID: "node_a", ToID: "node_b", Type: "Success"},
				{FromID: "node_b", ToID: "node_c", Type: "Success"},
			},
		},
	}
	instance := &projectionTestChainInstance{
		nodes: map[string]types.Node{
			"node_a": &projectionTestNode{id: "node_a"},
			"node_b": &projectionTestNode{id: "node_b"},
			"node_c": &projectionTestNode{
				id: "node_c",
				contract: types.DataContract{
					Reads: []string{"rulemsg://dataT/finalleads?sid=%5B%5DLead_V1"},
				},
			},
		},
		nodeDefs: map[string]*types.NodeDef{
			"node_a": {ID: "node_a", Type: "functions"},
			"node_b": {ID: "node_b", Type: "functions"},
			"node_c": {ID: "node_c", Type: "functions"},
		},
		def:     def,
		rootIDs: []string{"node_a"},
	}

	analysis := AnalyzeCoreObjProjection(def, instance)

	if analysis.RequiredInputs.RetainAll {
		t.Fatalf("expected specific required inputs, got retain_all")
	}
	if len(analysis.RequiredInputs.ObjIDs) != 1 || analysis.RequiredInputs.ObjIDs[0] != "finalleads" {
		t.Fatalf("expected required input finalleads, got %#v", analysis.RequiredInputs.ObjIDs)
	}
	if got := analysis.LiveObjectsByEdge[LiveObjectsEdgeKey("node_a", "node_b")]; len(got.ObjIDs) != 1 || got.ObjIDs[0] != "finalleads" {
		t.Fatalf("expected edge node_a->node_b to keep finalleads, got %#v", got)
	}
	if got := analysis.LiveObjectsByEdge[LiveObjectsEdgeKey("node_b", "node_c")]; len(got.ObjIDs) != 1 || got.ObjIDs[0] != "finalleads" {
		t.Fatalf("expected edge node_b->node_c to keep finalleads, got %#v", got)
	}
}

func TestAnalyzeCoreObjProjection_KillsOverwrittenObjectOnUpstreamEdge(t *testing.T) {
	def := &types.RuleChainDef{
		Metadata: types.MetadataData{
			Nodes: []types.NodeDef{
				{ID: "writer_a", Type: "functions"},
				{ID: "writer_b", Type: "functions"},
				{ID: "reader_c", Type: "functions"},
			},
			Connections: []types.Connection{
				{FromID: "writer_a", ToID: "writer_b", Type: "Success"},
				{FromID: "writer_b", ToID: "reader_c", Type: "Success"},
			},
		},
	}
	instance := &projectionTestChainInstance{
		nodes: map[string]types.Node{
			"writer_a": &projectionTestNode{
				id: "writer_a",
				contract: types.DataContract{
					Writes: []string{"rulemsg://dataT/finalleads?sid=%5B%5DLead_V1"},
				},
			},
			"writer_b": &projectionTestNode{
				id: "writer_b",
				contract: types.DataContract{
					Writes: []string{"rulemsg://dataT/finalleads?sid=%5B%5DLead_V1"},
				},
			},
			"reader_c": &projectionTestNode{
				id: "reader_c",
				contract: types.DataContract{
					Reads: []string{"rulemsg://dataT/finalleads?sid=%5B%5DLead_V1"},
				},
			},
		},
		nodeDefs: map[string]*types.NodeDef{
			"writer_a": {ID: "writer_a", Type: "functions"},
			"writer_b": {ID: "writer_b", Type: "functions"},
			"reader_c": {ID: "reader_c", Type: "functions"},
		},
		def:     def,
		rootIDs: []string{"writer_a"},
	}

	analysis := AnalyzeCoreObjProjection(def, instance)

	if got := analysis.LiveObjectsByEdge[LiveObjectsEdgeKey("writer_a", "writer_b")]; got.RetainAll || len(got.ObjIDs) != 0 {
		t.Fatalf("expected writer_a->writer_b to drop overwritten object, got %#v", got)
	}
	if got := analysis.LiveObjectsByEdge[LiveObjectsEdgeKey("writer_b", "reader_c")]; len(got.ObjIDs) != 1 || got.ObjIDs[0] != "finalleads" {
		t.Fatalf("expected writer_b->reader_c to keep finalleads, got %#v", got)
	}
	if len(analysis.ProducedObjects.ObjIDs) != 1 || analysis.ProducedObjects.ObjIDs[0] != "finalleads" {
		t.Fatalf("expected produced object finalleads, got %#v", analysis.ProducedObjects.ObjIDs)
	}
}

type projectionTestNode struct {
	id       string
	name     string
	contract types.DataContract
}

func (n *projectionTestNode) New() types.Node                    { return n }
func (n *projectionTestNode) Type() types.NodeType               { return types.NodeType("test") }
func (n *projectionTestNode) Init(types.ConfigMap) error         { return nil }
func (n *projectionTestNode) OnMsg(types.NodeCtx, types.RuleMsg) {}
func (n *projectionTestNode) Destroy()                           {}
func (n *projectionTestNode) NodeMetadata() types.NodeMetadata   { return types.NodeMetadata{} }
func (n *projectionTestNode) DataContract() types.DataContract   { return n.contract }
func (n *projectionTestNode) ID() string                         { return n.id }
func (n *projectionTestNode) Name() string                       { return n.name }
func (n *projectionTestNode) SetID(id string)                    { n.id = id }
func (n *projectionTestNode) SetName(name string)                { n.name = name }
func (n *projectionTestNode) Errors() []*types.Fault             { return nil }
func (n *projectionTestNode) ConfigSchema() *openapi3.Schema     { return &openapi3.Schema{} }

type projectionTestChainInstance struct {
	nodes    map[string]types.Node
	nodeDefs map[string]*types.NodeDef
	def      *types.RuleChainDef
	rootIDs  []string
}

func (c *projectionTestChainInstance) GetNode(id string) (types.Node, bool) {
	node, ok := c.nodes[id]
	return node, ok
}

func (c *projectionTestChainInstance) GetNodeDef(id string) (*types.NodeDef, bool) {
	def, ok := c.nodeDefs[id]
	return def, ok
}

func (c *projectionTestChainInstance) GetConnections(fromNodeID string) []types.Connection {
	result := make([]types.Connection, 0)
	for _, conn := range c.def.Metadata.Connections {
		if conn.FromID == fromNodeID {
			result = append(result, conn)
		}
	}
	return result
}

func (c *projectionTestChainInstance) Definition() *types.RuleChainDef { return c.def }
func (c *projectionTestChainInstance) GetRootNodeIDs() []string        { return c.rootIDs }
func (c *projectionTestChainInstance) GetAllNodes() map[string]types.Node {
	return c.nodes
}
func (c *projectionTestChainInstance) Destroy() {}
