package openapi2jsonschema

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// transformSchema converts a libopenapi Schema to a JSONSchema Draft 2020-12 schema.
func transformSchema(proxy *base.SchemaProxy, cfg *config) *JSONSchema {
	if proxy == nil {
		return nil
	}

	// Handle $ref - if this proxy is a reference, emit a $ref node
	ref := proxy.GetReference()
	if ref != "" {
		return &JSONSchema{Ref: rewriteRef(ref)}
	}

	schema, err := proxy.BuildSchema()
	if err != nil || schema == nil {
		return nil
	}

	js := &JSONSchema{}

	// Type and nullable
	if len(schema.Type) > 0 {
		types := make([]string, len(schema.Type))
		copy(types, schema.Type)
		if schema.Nullable != nil && *schema.Nullable {
			types = append(types, "null")
		}
		js.Type = &SchemaType{Types: types}
	} else if schema.Nullable != nil && *schema.Nullable {
		js.Type = SingleType("null")
	}

	// Metadata
	js.Title = schema.Title
	js.Description = schema.Description
	js.Default = extractNodeValue(schema.Default)
	js.Format = schema.Format
	js.Pattern = schema.Pattern

	// Enum
	if len(schema.Enum) > 0 {
		js.Enum = make([]any, len(schema.Enum))
		for i, v := range schema.Enum {
			js.Enum[i] = extractNodeValue(v)
		}
	}

	// Numeric constraints
	if schema.MultipleOf != nil {
		v := float64(*schema.MultipleOf)
		js.MultipleOf = &v
	}
	if schema.Maximum != nil {
		v := float64(*schema.Maximum)
		js.Maximum = &v
	}
	if schema.Minimum != nil {
		v := float64(*schema.Minimum)
		js.Minimum = &v
	}
	// In OpenAPI 3.0, exclusiveMaximum/Minimum are booleans that modify maximum/minimum.
	// In OpenAPI 3.1 / Draft 2020-12, they are numeric values.
	// libopenapi models this as DynamicValue[bool, float64].
	if schema.ExclusiveMaximum != nil {
		if schema.ExclusiveMaximum.IsA() {
			// Boolean form (OpenAPI 3.0): if true, move maximum to exclusiveMaximum
			if schema.ExclusiveMaximum.A && js.Maximum != nil {
				js.ExclusiveMaximum = js.Maximum
				js.Maximum = nil
			}
		} else {
			// Numeric form (OpenAPI 3.1)
			v := float64(schema.ExclusiveMaximum.B)
			js.ExclusiveMaximum = &v
		}
	}
	if schema.ExclusiveMinimum != nil {
		if schema.ExclusiveMinimum.IsA() {
			if schema.ExclusiveMinimum.A && js.Minimum != nil {
				js.ExclusiveMinimum = js.Minimum
				js.Minimum = nil
			}
		} else {
			v := float64(schema.ExclusiveMinimum.B)
			js.ExclusiveMinimum = &v
		}
	}

	// String constraints
	if schema.MaxLength != nil {
		v := int64(*schema.MaxLength)
		js.MaxLength = &v
	}
	if schema.MinLength != nil {
		v := int64(*schema.MinLength)
		js.MinLength = &v
	}

	// Array constraints
	if schema.Items != nil && schema.Items.IsA() {
		js.Items = transformSchema(schema.Items.A, cfg)
	}
	if schema.MaxItems != nil {
		v := int64(*schema.MaxItems)
		js.MaxItems = &v
	}
	if schema.MinItems != nil {
		v := int64(*schema.MinItems)
		js.MinItems = &v
	}
	if schema.UniqueItems != nil {
		js.UniqueItems = schema.UniqueItems
	}

	// Object constraints
	if schema.MaxProperties != nil {
		v := int64(*schema.MaxProperties)
		js.MaxProperties = &v
	}
	if schema.MinProperties != nil {
		v := int64(*schema.MinProperties)
		js.MinProperties = &v
	}
	js.Required = schema.Required

	// Properties
	if schema.Properties != nil {
		js.Properties = transformOrderedSchemaMap(schema.Properties, cfg)
	}

	// AdditionalProperties
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() {
			// It's a schema
			js.AdditionalProperties = SchemaValue(transformSchema(schema.AdditionalProperties.A, cfg))
		} else {
			// It's a boolean
			js.AdditionalProperties = BoolSchema(schema.AdditionalProperties.B)
		}
	}

	// Composition
	if len(schema.AllOf) > 0 {
		js.AllOf = transformSchemaSlice(schema.AllOf, cfg)
	}
	if len(schema.AnyOf) > 0 {
		js.AnyOf = transformSchemaSlice(schema.AnyOf, cfg)
	}
	if len(schema.OneOf) > 0 {
		js.OneOf = transformSchemaSlice(schema.OneOf, cfg)
	}
	if schema.Not != nil {
		js.Not = transformSchema(schema.Not, cfg)
	}

	// ReadOnly / WriteOnly
	if schema.ReadOnly != nil && *schema.ReadOnly {
		t := true
		js.ReadOnly = &t
	}
	if schema.WriteOnly != nil && *schema.WriteOnly {
		t := true
		js.WriteOnly = &t
	}

	// Extensions
	if schema.Extensions != nil {
		js.Extensions = transformYAMLExtensions(schema.Extensions, cfg)
	}

	return js
}

// transformSchemaSlice converts a slice of SchemaProxy to JSONSchema slices.
func transformSchemaSlice(proxies []*base.SchemaProxy, cfg *config) []*JSONSchema {
	result := make([]*JSONSchema, 0, len(proxies))
	for _, p := range proxies {
		if s := transformSchema(p, cfg); s != nil {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// transformOrderedSchemaMap converts an ordered map of schema proxies to a Go map.
func transformOrderedSchemaMap(om *orderedmap.Map[string, *base.SchemaProxy], cfg *config) map[string]*JSONSchema {
	if om == nil {
		return nil
	}
	result := make(map[string]*JSONSchema)
	for pair := om.First(); pair != nil; pair = pair.Next() {
		if s := transformSchema(pair.Value(), cfg); s != nil {
			result[pair.Key()] = s
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// transformYAMLExtensions filters and converts extension fields from YAML nodes based on config.
func transformYAMLExtensions(extensions *orderedmap.Map[string, *yaml.Node], cfg *config) map[string]any {
	if extensions == nil {
		return nil
	}
	result := make(map[string]any)
	for pair := extensions.First(); pair != nil; pair = pair.Next() {
		key := pair.Key()
		if !isExtension(key) {
			continue
		}
		if !shouldIncludeExtension(key, cfg) {
			continue
		}
		var val any
		if err := pair.Value().Decode(&val); err == nil {
			result[key] = val
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// extractNodeValue extracts a Go value from a YAML node.
// If the value is a *yaml.Node, it decodes it to a native Go type.
// Returns nil if the node is nil or represents a YAML null.
func extractNodeValue(node any) any {
	if node == nil {
		return nil
	}
	if yn, ok := node.(*yaml.Node); ok {
		if yn == nil || yn.Tag == "!!null" {
			return nil
		}
		var val any
		if err := yn.Decode(&val); err != nil {
			return nil
		}
		return val
	}
	return node
}
