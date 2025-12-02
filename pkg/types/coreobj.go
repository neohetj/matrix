/*
 * Copyright 2025 The Matrix Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// CoreObjDef holds the definition of a business object, including its OpenAPI schema.
type CoreObjDef struct {
	sid             string
	desc            string
	schema          *openapi3.Schema
	schemaJSONCache string
	objType         reflect.Type
}

// SID returns the unique type identifier for the object definition.
func (d *CoreObjDef) SID() string {
	return d.sid
}

// New returns a new instance of the business object.
func (d *CoreObjDef) New() any {
	if d.objType.Kind() == reflect.Ptr {
		return reflect.New(d.objType.Elem()).Interface()
	}
	return reflect.New(d.objType).Interface()
}

// Description returns a human-readable description of the object.
func (d *CoreObjDef) Description() string {
	return d.desc
}

// Schema returns the JSON schema of the object's fields in OpenAPI format.
func (d *CoreObjDef) Schema() string {
	return d.schemaJSONCache
}

// OpenAPISchema returns the raw openapi3.Schema object.
func (d *CoreObjDef) OpenAPISchema() *openapi3.Schema {
	return d.schema
}

// MarshalJSON implements the json.Marshaler interface to provide a custom JSON representation.
func (d *CoreObjDef) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SID    string           `json:"sid"`
		Name   string           `json:"name"`
		Desc   string           `json:"desc"`
		Schema *openapi3.Schema `json:"schema"`
	}{
		SID:    d.sid,
		Name:   d.sid, // Using SID as Name for now, can be improved
		Desc:   d.desc,
		Schema: d.schema,
	})
}

// NewCoreObjDef creates a new CoreObjDef from a prototype instance.
// It uses reflection to generate the OpenAPI schema.
func NewCoreObjDef(prototype any, sid, desc string) *CoreObjDef {
	objType := reflect.TypeOf(prototype)
	def := &CoreObjDef{
		sid:     sid,
		desc:    desc,
		objType: objType,
	}

	schema := schemaFromStruct(objType)
	schema.Description = desc

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		log.Printf("Error: Failed to serialize OpenAPI schema for %s: %v. Schema will be empty.", sid, err)
		def.schema = &openapi3.Schema{}
		def.schemaJSONCache = "{}"
	} else {
		def.schema = schema
		def.schemaJSONCache = string(schemaBytes)
	}
	return def
}

// schemaFromStruct recursively generates an OpenAPI schema from a Go struct type.
func schemaFromStruct(t reflect.Type) *openapi3.Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeString := goTypeToOpenAPIType(t)
	schema := &openapi3.Schema{
		Type: &openapi3.Types{typeString},
	}

	switch t.Kind() {
	case reflect.Struct:
		schema.Properties = make(map[string]*openapi3.SchemaRef)
		schema.Required = make([]string, 0)

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			jsonTagParts := strings.Split(jsonTag, ",")
			fieldName := jsonTagParts[0]
			if fieldName == "" {
				fieldName = field.Name
			}

			fieldSchemaRef := openapi3.NewSchemaRef("", schemaFromStruct(field.Type))
			fieldSchemaRef.Value.Description = field.Tag.Get("description")

			if enumTag := field.Tag.Get("enum"); enumTag != "" {
				enumValues := strings.Split(enumTag, ",")
				for _, v := range enumValues {
					fieldSchemaRef.Value.Enum = append(fieldSchemaRef.Value.Enum, v)
				}
			}

			schema.Properties[fieldName] = fieldSchemaRef

			if requiredTag := field.Tag.Get("required"); requiredTag == "true" {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	case reflect.Slice, reflect.Array:
		elemType := t.Elem()
		schema.Items = openapi3.NewSchemaRef("", schemaFromStruct(elemType))
	}

	return schema
}

// goTypeToOpenAPIType converts a Go type to an OpenAPI type string.
func goTypeToOpenAPIType(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return openapi3.TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return openapi3.TypeInteger
	case reflect.Float32, reflect.Float64:
		return openapi3.TypeNumber
	case reflect.Bool:
		return openapi3.TypeBoolean
	case reflect.Struct, reflect.Map:
		return openapi3.TypeObject
	case reflect.Slice, reflect.Array:
		return openapi3.TypeArray
	default:
		// Fallback for interface{}, etc.
		return openapi3.TypeObject
	}
}

// CoreObj is the interface for a concrete business object instance.
type CoreObj interface {
	Key() string
	Definition() *CoreObjDef
	Body() any
	SetBody(body any) error
	DeepCopy() (CoreObj, error)
}

// CoreObjRegistry is the interface for managing a collection of CoreObjDef.
type CoreObjRegistry interface {
	Register(defs ...*CoreObjDef)
	Get(sid string) (*CoreObjDef, bool)
	GetAll() []*CoreObjDef
}
