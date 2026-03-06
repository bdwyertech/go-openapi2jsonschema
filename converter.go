package openapi2jsonschema

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

// Convert parses an OpenAPI 3.x document from bytes and returns a JSON Schema
// Draft 2020-12 document containing all component schemas as $defs.
func Convert(input []byte, opts ...Option) (*JSONSchema, error) {
	cfg := newConfig(opts)

	doc, err := libopenapi.NewDocument(input)
	if err != nil {
		return nil, fmt.Errorf("parsing OpenAPI document: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("building OpenAPI 3.x model: %w", err)
	}
	if model == nil {
		return nil, fmt.Errorf("failed to build OpenAPI 3.x model")
	}

	return buildRootSchema(model.Model.Components.Schemas, cfg)
}

// ConvertReader reads an OpenAPI 3.x document from an io.Reader and returns
// a JSON Schema Draft 2020-12 document.
func ConvertReader(r io.Reader, opts ...Option) (*JSONSchema, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return Convert(data, opts...)
}

// ConvertSchema parses an OpenAPI 3.x document and returns a JSON Schema
// for a single named schema. The root schema will have a $ref pointing
// to the named schema in $defs, and only referenced schemas are included.
func ConvertSchema(input []byte, schemaName string, opts ...Option) (*JSONSchema, error) {
	cfg := newConfig(opts)

	doc, err := libopenapi.NewDocument(input)
	if err != nil {
		return nil, fmt.Errorf("parsing OpenAPI document: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("building OpenAPI 3.x model: %w", err)
	}
	if model == nil {
		return nil, fmt.Errorf("failed to build OpenAPI 3.x model")
	}

	schemas := model.Model.Components.Schemas
	if schemas == nil {
		return nil, fmt.Errorf("no schemas found in OpenAPI document")
	}

	// Verify the requested schema exists
	target, ok := schemas.Get(schemaName)
	if !ok || target == nil {
		return nil, fmt.Errorf("schema %q not found in components/schemas", schemaName)
	}

	// Build all defs (could optimize to only include referenced schemas later)
	defs, err := buildDefs(schemas, cfg)
	if err != nil {
		return nil, err
	}

	root := &JSONSchema{
		Schema: schemaDraft202012,
		Ref:    "#/$defs/" + schemaName,
		Defs:   defs,
	}
	if cfg.schemaID != "" {
		root.ID = cfg.schemaID
	}

	return root, nil
}

// ConvertSchemaByKindSuffix parses an OpenAPI 3.x document and returns a JSON Schema
// for the schema whose key ends with ".{kind}" (e.g. ".VerticalPodAutoscaler").
// This is useful for Kubernetes schemas where the full key is like
// "io.k8s.autoscaling.v1.VerticalPodAutoscaler" but callers only know the Kind.
func ConvertSchemaByKindSuffix(input []byte, kind string, opts ...Option) (*JSONSchema, error) {
	cfg := newConfig(opts)

	doc, err := libopenapi.NewDocument(input)
	if err != nil {
		return nil, fmt.Errorf("parsing OpenAPI document: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("building OpenAPI 3.x model: %w", err)
	}
	if model == nil {
		return nil, fmt.Errorf("failed to build OpenAPI 3.x model")
	}

	schemas := model.Model.Components.Schemas
	if schemas == nil {
		return nil, fmt.Errorf("no schemas found in OpenAPI document")
	}

	// Find the full schema name by suffix match
	suffix := "." + kind
	var schemaName string
	for pair := schemas.First(); pair != nil; pair = pair.Next() {
		if strings.HasSuffix(pair.Key(), suffix) {
			schemaName = pair.Key()
			break
		}
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema not found for kind %q (suffix %q)", kind, suffix)
	}

	defs, err := buildDefs(schemas, cfg)
	if err != nil {
		return nil, err
	}

	root := &JSONSchema{
		Schema: schemaDraft202012,
		Ref:    "#/$defs/" + schemaName,
		Defs:   defs,
	}
	if cfg.schemaID != "" {
		root.ID = cfg.schemaID
	}

	return root, nil
}

// ConvertToJSON is a convenience function that converts an OpenAPI 3.x document
// to a JSON Schema and returns it as indented JSON bytes.
func ConvertToJSON(input []byte, opts ...Option) ([]byte, error) {
	schema, err := Convert(input, opts...)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(schema, "", "  ")
}

// buildRootSchema constructs the root JSON Schema with all component schemas as $defs.
func buildRootSchema(schemas *orderedmap.Map[string, *base.SchemaProxy], cfg *config) (*JSONSchema, error) {
	defs, err := buildDefs(schemas, cfg)
	if err != nil {
		return nil, err
	}

	root := &JSONSchema{
		Schema: schemaDraft202012,
		Defs:   defs,
	}
	if cfg.schemaID != "" {
		root.ID = cfg.schemaID
	}

	return root, nil
}

// buildDefs converts all OpenAPI component schemas into JSON Schema $defs.
func buildDefs(schemas *orderedmap.Map[string, *base.SchemaProxy], cfg *config) (map[string]*JSONSchema, error) {
	if schemas == nil {
		return nil, nil
	}

	defs := make(map[string]*JSONSchema)
	for pair := schemas.First(); pair != nil; pair = pair.Next() {
		name := pair.Key()
		proxy := pair.Value()
		if proxy == nil {
			continue
		}
		js := transformSchema(proxy, cfg)
		if js != nil {
			defs[name] = js
		}
	}

	if len(defs) == 0 {
		return nil, nil
	}
	return defs, nil
}
