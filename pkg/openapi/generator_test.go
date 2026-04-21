package openapi

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/internal/contract"
	"github.com/neohetj/matrix/pkg/types"
)

const testProfilePatchSID = "OpenAPITestProfilePatchV1"

type testCoreObjRegistry struct {
	defs map[string]types.CoreObjDef
}

func newTestCoreObjRegistry(defs ...types.CoreObjDef) *testCoreObjRegistry {
	registry := &testCoreObjRegistry{defs: make(map[string]types.CoreObjDef)}
	registry.Register(defs...)
	return registry
}

func (r *testCoreObjRegistry) Register(defs ...types.CoreObjDef) {
	for _, def := range defs {
		r.defs[def.SID()] = def
	}
}

func (r *testCoreObjRegistry) Get(sid string) (types.CoreObjDef, bool) {
	def, ok := r.defs[sid]
	return def, ok
}

func (r *testCoreObjRegistry) GetAll() []types.CoreObjDef {
	defs := make([]types.CoreObjDef, 0, len(r.defs))
	for _, def := range r.defs {
		defs = append(defs, def)
	}
	return defs
}

type testProfilePatch struct {
	Nickname   string          `json:"nickname"`
	AgreeTerms *bool           `json:"agree_terms,omitempty"`
	Settings   testSettingsObj `json:"settings"`
}

type testSettingsObj struct {
	Enabled bool `json:"enabled"`
}

func newOpenAPITestGenerator() *Generator {
	return &Generator{
		registry: newTestCoreObjRegistry(
			contract.NewDefaultCoreObjDef(&testProfilePatch{}, testProfilePatchSID, "profile patch"),
		),
	}
}

func newOpenAPITestSpec() *openapi3.T {
	return &openapi3.T{
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}
}

func TestBuildSchemaFromPacketFieldsUseBoundCoreObjPropertySchema(t *testing.T) {
	gen := newOpenAPITestGenerator()
	spec := newOpenAPITestSpec()

	schemaRef, err := gen.buildSchemaFromPacket(spec, types.EndpointIOPacket{
		Fields: []types.EndpointIOField{
			{
				Name:     "agree_terms",
				BindPath: "rulemsg://dataT/profilePatch.agree_terms?sid=" + testProfilePatchSID,
			},
			{
				Name:     "nickname",
				BindPath: "rulemsg://dataT/profilePatch.nickname?sid=" + testProfilePatchSID,
			},
		},
	})
	if err != nil {
		t.Fatalf("buildSchemaFromPacket returned error: %v", err)
	}
	if schemaRef == nil || schemaRef.Value == nil {
		t.Fatal("expected inline fields schema")
	}

	agreeTerms := schemaRef.Value.Properties["agree_terms"]
	if agreeTerms == nil || agreeTerms.Value == nil {
		t.Fatal("expected agree_terms to use an inline property schema")
	}
	if agreeTerms.Ref != "" {
		t.Fatalf("expected agree_terms not to reference the whole object, got ref %q", agreeTerms.Ref)
	}
	if !agreeTerms.Value.Type.Is(openapi3.TypeBoolean) {
		t.Fatalf("expected agree_terms to be boolean, got %v", agreeTerms.Value.Type)
	}
	if len(agreeTerms.Value.Properties) != 0 {
		t.Fatalf("expected agree_terms to be a scalar field, got nested properties: %v", agreeTerms.Value.Properties)
	}

	nickname := schemaRef.Value.Properties["nickname"]
	if nickname == nil || nickname.Value == nil || !nickname.Value.Type.Is(openapi3.TypeString) {
		t.Fatalf("expected nickname to be string, got %#v", nickname)
	}
}

func TestBuildSchemaFromPacketFieldsKeepWholeCoreObjRefForObjectBinding(t *testing.T) {
	gen := newOpenAPITestGenerator()
	spec := newOpenAPITestSpec()

	schemaRef, err := gen.buildSchemaFromPacket(spec, types.EndpointIOPacket{
		Fields: []types.EndpointIOField{
			{
				Name:     "profile",
				BindPath: "rulemsg://dataT/profilePatch?sid=" + testProfilePatchSID,
			},
		},
	})
	if err != nil {
		t.Fatalf("buildSchemaFromPacket returned error: %v", err)
	}
	if schemaRef == nil || schemaRef.Value == nil {
		t.Fatal("expected inline fields schema")
	}

	profile := schemaRef.Value.Properties["profile"]
	if profile == nil {
		t.Fatal("expected profile property")
	}
	expectedRef := "#/components/schemas/" + testProfilePatchSID
	if profile.Ref != expectedRef {
		t.Fatalf("expected profile to keep whole object ref %q, got %q", expectedRef, profile.Ref)
	}
}
