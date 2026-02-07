package openapi

import (
	"fmt"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/types"
	"github.com/neohetj/matrix/pkg/utils"
)

// Generator is responsible for generating OpenAPI documentation from the Matrix node pool.
type Generator struct {
	nodePool types.NodePool
	registry types.CoreObjRegistry
}

// NewGenerator creates a new OpenAPI Generator.
func NewGenerator(pool types.NodePool, registry types.CoreObjRegistry) *Generator {
	return &Generator{
		nodePool: pool,
		registry: registry,
	}
}

// Generate builds the OpenAPI v3 specification.
func (g *Generator) Generate() (*openapi3.T, error) {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:       "Matrix Generated API",
			Version:     "1.0.0",
			Description: "API documentation generated automatically from Matrix HTTP Endpoints.",
		},
		Paths: openapi3.NewPaths(),
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}

	// Use GetEndpoints to retrieve all endpoint nodes
	endpoints := g.nodePool.GetEndpoints()
	for _, endpoint := range endpoints {
		// Filter for HttpEndpoint
		httpEndpoint, ok := endpoint.(types.HttpEndpoint)
		if !ok {
			continue
		}
		if err := g.addEndpoint(spec, httpEndpoint); err != nil {
			return nil, fmt.Errorf("failed to process endpoint %s: %w", endpoint.ID(), err)
		}
	}

	return spec, nil
}

func (g *Generator) addEndpoint(spec *openapi3.T, endpoint types.HttpEndpoint) error {
	config := endpoint.Configuration()

	// Convert Matrix path (/user/:id) to OpenAPI path (/user/{id})
	path := convertPath(config.HttpPath)
	method := strings.ToUpper(config.HttpMethod)

	// Ensure PathItem exists
	pathItem := spec.Paths.Find(path)
	if pathItem == nil {
		pathItem = &openapi3.PathItem{}
		spec.Paths.Set(path, pathItem)
	}

	operation := &openapi3.Operation{
		Summary:     config.Summary,
		Description: config.Description,
		Tags:        config.Tags,
		Parameters:  make(openapi3.Parameters, 0),
		Responses:   openapi3.NewResponses(),
	}
	if operation.Summary == "" {
		operation.Summary = endpoint.Name()
	}

	// Auto-tag with Namespace derived from RuleChainID
	if config.RuleChainID != "" {
		parts := strings.Split(config.RuleChainID, "/")
		if len(parts) > 0 {
			namespace := parts[0]
			// Check if already tagged
			if !slices.Contains(operation.Tags, namespace) {
				// Prepend namespace to tags
				operation.Tags = append([]string{namespace}, operation.Tags...)
			}
		}
	}

	// 1. Process Request Parameters (Path, Query, Header)
	reqDef := config.EndpointDefinition.Request

	// Path Params
	for _, field := range reqDef.PathParams {
		param := openapi3.NewPathParameter(field.Name)
		param.Description = field.Description
		param.Required = true // Path params are always required
		param.Schema = &openapi3.SchemaRef{Value: MTypeToOpenAPISchema(field.Type)}
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: param})
	}

	// Query Params
	for _, field := range reqDef.QueryParams.Fields {
		param := openapi3.NewQueryParameter(field.Name)
		param.Description = field.Description
		param.Required = field.Required
		param.Schema = &openapi3.SchemaRef{Value: MTypeToOpenAPISchema(field.Type)}
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: param})
	}

	// Header Params
	for _, field := range reqDef.Headers.Fields {
		param := openapi3.NewHeaderParameter(field.Name)
		param.Description = field.Description
		param.Required = field.Required
		param.Schema = &openapi3.SchemaRef{Value: MTypeToOpenAPISchema(field.Type)}
		operation.Parameters = append(operation.Parameters, &openapi3.ParameterRef{Value: param})
	}

	// 2. Process Request Body
	if method == "POST" || method == "PUT" || method == "PATCH" {
		reqBody, err := g.buildRequestBody(spec, reqDef.Body)
		if err != nil {
			return err
		}
		if reqBody != nil {
			operation.RequestBody = &openapi3.RequestBodyRef{Value: reqBody}
		}
	}

	// 3. Process Responses
	// Success Response
	respDef := config.EndpointDefinition.Response
	successCode := respDef.SuccessCode
	if successCode == 0 {
		successCode = 200
	}
	successResp, err := g.buildResponse(spec, respDef.Body, "Successful response")
	if err != nil {
		return err
	}
	operation.AddResponse(successCode, successResp)

	// Error Responses
	for codeStr := range config.ErrorMappings {
		// Use integer code for AddResponse if possible, or string for default/etc
		// Kin-openapi AddResponse takes int. For ranges/default use other methods or cast.
		// ErrorMappings keys are strings like "404", "500".
		var code int
		fmt.Sscanf(codeStr, "%d", &code)

		resp := &openapi3.Response{
			Description: utils.Ptr("Error response"),
			Content: openapi3.NewContentWithJSONSchema(&openapi3.Schema{
				Type: &openapi3.Types{"object"},
				Properties: map[string]*openapi3.SchemaRef{
					"code":    {Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}},
					"message": {Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}},
				},
			}),
		}

		if code > 0 {
			operation.AddResponse(code, resp)
		} else {
			// Fallback for non-integer codes (e.g. "default", "4xx") if supported
			operation.Responses.Set(codeStr, &openapi3.ResponseRef{Value: resp})
		}
	}

	// Default error if configured
	if respDef.ErrorStatusCode != 0 {
		operation.AddResponse(respDef.ErrorStatusCode, &openapi3.Response{
			Description: utils.Ptr("Default error response"),
		})
	}

	operation.AddResponse(500, &openapi3.Response{
		Description: utils.Ptr("Internal Server Error"),
	})

	// Assign Operation to PathItem
	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "HEAD":
		pathItem.Head = operation
	case "OPTIONS":
		pathItem.Options = operation
	}

	return nil
}

func (g *Generator) buildRequestBody(spec *openapi3.T, packet types.EndpointIOPacket) (*openapi3.RequestBody, error) {
	schemaRef, err := g.buildSchemaFromPacket(spec, packet)
	if err != nil {
		return nil, err
	}
	if schemaRef == nil {
		return nil, nil
	}

	return &openapi3.RequestBody{
		Content: openapi3.NewContentWithJSONSchemaRef(schemaRef),
	}, nil
}

func (g *Generator) buildResponse(spec *openapi3.T, packet types.EndpointIOPacket, description string) (*openapi3.Response, error) {
	schemaRef, err := g.buildSchemaFromPacket(spec, packet)
	if err != nil {
		return nil, err
	}

	resp := &openapi3.Response{
		Description: &description,
	}
	if schemaRef != nil {
		resp.Content = openapi3.NewContentWithJSONSchemaRef(schemaRef)
	}
	return resp, nil
}

func (g *Generator) buildSchemaFromPacket(spec *openapi3.T, packet types.EndpointIOPacket) (*openapi3.SchemaRef, error) {
	var mapAllSchemaRef *openapi3.SchemaRef

	// 1. Process MapAll (Reference Mode)
	if packet.MapAll != nil && *packet.MapAll != "" {
		uri := *packet.MapAll
		sid := extractSID(uri)
		if sid != "" {
			// Register schema component if needed
			if spec.Components.Schemas[sid] == nil {
				if coreObj, ok := g.registry.Get(sid); ok {
					spec.Components.Schemas[sid] = &openapi3.SchemaRef{
						Value: coreObj.OpenAPISchema(),
					}
				} else {
					// SID not found, maybe fallback to generic object
					mapAllSchemaRef = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}, Description: "Unknown Type: " + sid}}
				}
			}
			if mapAllSchemaRef == nil {
				mapAllSchemaRef = &openapi3.SchemaRef{Ref: "#/components/schemas/" + sid}
			}
		} else {
			// If MapAll is generic (no SID), treat as generic object
			mapAllSchemaRef = &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}}
		}
	}

	// 2. Process Fields (Inline/Overlay Mode)
	var fieldsSchema *openapi3.Schema

	if len(packet.Fields) > 0 {
		fieldsSchema = &openapi3.Schema{
			Type:       &openapi3.Types{"object"},
			Properties: make(map[string]*openapi3.SchemaRef),
			Required:   make([]string, 0),
		}

		for _, field := range packet.Fields {
			var fieldSchemaRef *openapi3.SchemaRef

			// Try to resolve SID from BindPath for detailed object schema
			if field.BindPath != "" {
				sid := extractSID(field.BindPath)
				if sid != "" {
					// Ensure component exists
					if spec.Components.Schemas[sid] == nil {
						if coreObj, ok := g.registry.Get(sid); ok {
							spec.Components.Schemas[sid] = &openapi3.SchemaRef{
								Value: coreObj.OpenAPISchema(),
							}
						}
					}
					// If component exists, use Ref
					if spec.Components.Schemas[sid] != nil {
						fieldSchemaRef = &openapi3.SchemaRef{Ref: "#/components/schemas/" + sid}
					}
				}
			}

			// Fallback to MType if no Ref
			if fieldSchemaRef == nil {
				fieldSchema := MTypeToOpenAPISchema(field.Type)
				fieldSchemaRef = &openapi3.SchemaRef{Value: fieldSchema}
			}

			// Apply Metadata (Description, Default)
			// If Ref is used, wrap in AllOf to comply with OAS 3.0 for description/default overrides
			if fieldSchemaRef.Ref != "" && (field.Description != "" || field.DefaultValue != nil) {
				wrapper := &openapi3.Schema{
					AllOf: []*openapi3.SchemaRef{
						{Ref: fieldSchemaRef.Ref},
					},
					Description: field.Description,
					Default:     field.DefaultValue,
				}
				fieldSchemaRef = &openapi3.SchemaRef{Value: wrapper}
			} else if fieldSchemaRef.Value != nil {
				// Inline schema, apply directly
				fieldSchemaRef.Value.Description = field.Description
				if field.DefaultValue != nil {
					fieldSchemaRef.Value.Default = field.DefaultValue
				}
			}

			fieldsSchema.Properties[field.Name] = fieldSchemaRef
			if field.Required {
				fieldsSchema.Required = append(fieldsSchema.Required, field.Name)
			}
		}
	}

	// 3. Merge Logic (MapAll + Fields)
	if mapAllSchemaRef != nil && fieldsSchema != nil {
		// Use AllOf to merge
		// Base: mapAllSchemaRef
		// Overlay: fieldsSchema (adds properties, overrides, and adds required fields)
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				AllOf: []*openapi3.SchemaRef{
					mapAllSchemaRef,
					{Value: fieldsSchema},
				},
			},
		}, nil
	}

	if mapAllSchemaRef != nil {
		return mapAllSchemaRef, nil
	}

	if fieldsSchema != nil {
		return &openapi3.SchemaRef{Value: fieldsSchema}, nil
	}

	// Empty body
	return nil, nil
}

func extractSID(uri string) string {
	// Simple parsing: rulemsg://dataT/...?sid=SID
	// Use asset package to parse
	assetObj, err := asset.ParseRuleMsg(uri)
	if err != nil {
		return ""
	}
	if assetObj.Host == cnst.DATAT {
		return assetObj.Query.Get("sid")
	}
	return ""
}

func convertPath(matrixPath string) string {
	// Convert /user/:id to /user/{id}
	// Split by /, check parts starting with :
	parts := strings.Split(matrixPath, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}
