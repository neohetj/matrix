package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/cnst"
)

// MTypeToOpenAPISchema converts a cnst.MType to an openapi3.Schema.
func MTypeToOpenAPISchema(mType cnst.MType) *openapi3.Schema {
	schema := &openapi3.Schema{}
	switch mType {
	case cnst.STRING:
		schema.Type = &openapi3.Types{"string"}
	case cnst.INT:
		schema.Type = &openapi3.Types{"integer"}
	case cnst.INT64:
		schema.Type = &openapi3.Types{"integer"}
		schema.Format = "int64"
	case cnst.FLOAT:
		schema.Type = &openapi3.Types{"number"}
	case cnst.BOOL:
		schema.Type = &openapi3.Types{"boolean"}
	case cnst.MAP, cnst.OBJECT:
		schema.Type = &openapi3.Types{"object"}
	default:
		// Check for array types (LIST_PREFIX)
		if strings.HasPrefix(string(mType), cnst.LIST_PREFIX) {
			elemType := strings.TrimPrefix(string(mType), cnst.LIST_PREFIX)
			schema.Type = &openapi3.Types{"array"}
			schema.Items = &openapi3.SchemaRef{
				Value: MTypeToOpenAPISchema(cnst.MType(elemType)),
			}
		} else {
			// Default to string/any
			schema.Type = &openapi3.Types{"string"}
		}
	}
	return schema
}

// ReflectToOpenAPISchema converts a Go struct type to an openapi3.Schema.
// structPtrOrInstance can be a struct instance or a pointer to a struct.
func ReflectToOpenAPISchema(structPtrOrInstance any) (*openapi3.Schema, error) {
	if structPtrOrInstance == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	t := reflect.TypeOf(structPtrOrInstance)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or a pointer to a struct")
	}

	return reflectStructToOpenAPI(t), nil
}

func reflectStructToOpenAPI(t reflect.Type) *openapi3.Schema {
	schema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(map[string]*openapi3.SchemaRef),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 {
				fieldName = parts[0]
			}
		}

		if field.Anonymous {
			// For anonymous fields, merge their properties into the current schema.
			anonymousSchema := reflectStructToOpenAPI(field.Type)
			for k, v := range anonymousSchema.Properties {
				schema.Properties[k] = v
			}
		} else {
			schema.Properties[fieldName] = &openapi3.SchemaRef{
				Value: reflectTypeToOpenAPI(field.Type),
			}
		}
	}
	return schema
}

func reflectTypeToOpenAPI(t reflect.Type) *openapi3.Schema {
	switch t.Kind() {
	case reflect.String:
		// Check if it's MType
		if t.Name() == "MType" {
			return &openapi3.Schema{Type: &openapi3.Types{"string"}, Description: "MType enum"}
		}
		return &openapi3.Schema{Type: &openapi3.Types{"string"}}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &openapi3.Schema{Type: &openapi3.Types{"integer"}}
	case reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &openapi3.Schema{Type: &openapi3.Types{"integer"}, Format: "int64"}
	case reflect.Float32, reflect.Float64:
		return &openapi3.Schema{Type: &openapi3.Types{"number"}}
	case reflect.Bool:
		return &openapi3.Schema{Type: &openapi3.Types{"boolean"}}
	case reflect.Slice, reflect.Array:
		return &openapi3.Schema{
			Type: &openapi3.Types{"array"},
			Items: &openapi3.SchemaRef{
				Value: reflectTypeToOpenAPI(t.Elem()),
			},
		}
	case reflect.Map:
		return &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			AdditionalProperties: openapi3.AdditionalProperties{
				Schema: &openapi3.SchemaRef{
					Value: reflectTypeToOpenAPI(t.Elem()),
				},
			},
		}
	case reflect.Struct:
		return reflectStructToOpenAPI(t)
	case reflect.Pointer:
		return reflectTypeToOpenAPI(t.Elem())
	default:
		return &openapi3.Schema{} // Any type
	}
}
