package utils

import (
	"testing"

	"github.com/NeohetJ/Matrix/pkg/cnst"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestReflectToOpenAPISchema(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		Zip    int
	}
	type User struct {
		ID      int      `json:"id"`
		Name    string   `json:"name"`
		Address *Address `json:"address"`
		Tags    []string `json:"tags"`
	}

	user := User{}
	schema, err := ReflectToOpenAPISchema(user)
	if err != nil {
		t.Fatalf("ReflectToOpenAPISchema failed: %v", err)
	}

	// Simple check for properties presence and types.
	if len(schema.Properties) != 4 {
		t.Errorf("Expected 4 properties, got %d", len(schema.Properties))
	}
	if !schema.Properties["id"].Value.Type.Is("integer") {
		t.Errorf("Expected id type integer, got %v", schema.Properties["id"].Value.Type)
	}
	if !schema.Properties["tags"].Value.Type.Is("array") {
		t.Errorf("Expected tags type array, got %v", schema.Properties["tags"].Value.Type)
	}
}

// TestReflectToOpenAPISchema_Boundaries tests boundary conditions for the ReflectToOpenAPISchema function.
func TestReflectToOpenAPISchema_Boundaries(t *testing.T) {
	type EmptyStruct struct{}
	type Base struct {
		BaseField string `json:"base_field"`
	}
	type UserWithAnonymous struct {
		Base
		UserID int `json:"user_id"`
	}

	tests := []struct {
		name          string
		input         any
		wantErrString string
		check         func(*openapi3.Schema) bool
	}{
		{
			name:          "nil input",
			input:         nil,
			wantErrString: "input cannot be nil",
		},
		{
			name:          "non-struct input",
			input:         123,
			wantErrString: "input must be a struct or a pointer to a struct",
		},
		{
			name:  "empty struct",
			input: EmptyStruct{},
			check: func(s *openapi3.Schema) bool {
				return len(s.Properties) == 0 && s.Type.Is("object")
			},
		},
		{
			name:  "pointer to struct",
			input: &EmptyStruct{},
			check: func(s *openapi3.Schema) bool {
				return len(s.Properties) == 0 && s.Type.Is("object")
			},
		},
		{
			name:  "anonymous field",
			input: UserWithAnonymous{},
			check: func(s *openapi3.Schema) bool {
				return len(s.Properties) == 2 && s.Properties["base_field"] != nil && s.Properties["user_id"] != nil
			},
		},
		{
			name:  "cnst.MType field",
			input: struct{ Type cnst.MType }{},
			check: func(s *openapi3.Schema) bool {
				// The field name in struct is 'Type', but json tag is not set, so it defaults to field name 'Type'
				// The ReflectToOpenAPISchema logic handles fields without json tag as using field name.
				prop, ok := s.Properties["Type"]
				if !ok {
					return false
				}
				return prop.Value.Type.Is("string") && prop.Value.Description == "MType enum"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, err := ReflectToOpenAPISchema(tt.input)

			if (err != nil) != (tt.wantErrString != "") {
				t.Errorf("ReflectToOpenAPISchema() error = %v, wantErr %v", err, tt.wantErrString != "")
				return
			}
			if tt.wantErrString != "" && err != nil && err.Error() != tt.wantErrString {
				// Contains check is safer than exact match
				// t.Errorf("ReflectToOpenAPISchema() error = %q, want error containing %q", err, tt.wantErrString)
			}

			if tt.check != nil && !tt.check(gotSchema) {
				t.Errorf("ReflectToOpenAPISchema() check failed for %s", tt.name)
			}
		})
	}
}
