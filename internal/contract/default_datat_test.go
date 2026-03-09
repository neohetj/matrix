package contract

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/types"
)

func TestDefaultDataT_ProjectKeepsOnlyRequestedObjects(t *testing.T) {
	dataT := NewDataT()
	dataT.Set("image_push_items", &projectTestCoreObj{key: "image_push_items", body: "keep"})
	dataT.Set("ttscrapedposts", &projectTestCoreObj{key: "ttscrapedposts", body: "drop"})

	projected, err := dataT.Project([]string{"image_push_items"})
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}

	if _, ok := projected.Get("image_push_items"); !ok {
		t.Fatalf("expected projected DataT to keep image_push_items")
	}
	if _, ok := projected.Get("ttscrapedposts"); ok {
		t.Fatalf("expected projected DataT to drop ttscrapedposts")
	}
	if _, ok := dataT.Get("ttscrapedposts"); !ok {
		t.Fatalf("expected source DataT to remain unchanged")
	}
}

type projectTestCoreObj struct {
	key  string
	body any
}

func (o *projectTestCoreObj) Key() string                  { return o.key }
func (o *projectTestCoreObj) Definition() types.CoreObjDef { return &projectTestCoreObjDef{} }
func (o *projectTestCoreObj) Body() any                    { return o.body }
func (o *projectTestCoreObj) SetBody(body any) error {
	o.body = body
	return nil
}
func (o *projectTestCoreObj) DeepCopy() (types.CoreObj, error) {
	return &projectTestCoreObj{key: o.key, body: o.body}, nil
}

type projectTestCoreObjDef struct{}

func (d *projectTestCoreObjDef) SID() string                     { return "project-test" }
func (d *projectTestCoreObjDef) New() any                        { return map[string]any{} }
func (d *projectTestCoreObjDef) Description() string             { return "" }
func (d *projectTestCoreObjDef) Schema() string                  { return "{}" }
func (d *projectTestCoreObjDef) OpenAPISchema() *openapi3.Schema { return &openapi3.Schema{} }
func (d *projectTestCoreObjDef) MarshalJSON() ([]byte, error)    { return []byte(`{}`), nil }
