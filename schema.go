package openapi2jsonschema

import "encoding/json"

const schemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

// JSONSchema represents a JSON Schema Draft 2020-12 document or sub-schema.
// Fields are pointers or omitempty to produce clean, minimal output.
type JSONSchema struct {
	Schema  string `json:"$schema,omitempty"`
	ID      string `json:"$id,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	Comment string `json:"$comment,omitempty"`

	// Core
	Type        *SchemaType            `json:"type,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Default     any                    `json:"default,omitempty"`
	Enum        []any                  `json:"enum,omitempty"`
	Const       any                    `json:"const,omitempty"`
	Examples    []any                  `json:"examples,omitempty"`
	Defs        map[string]*JSONSchema `json:"$defs,omitempty"`

	// Numeric
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`

	// String
	MaxLength *int64 `json:"maxLength,omitempty"`
	MinLength *int64 `json:"minLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Format    string `json:"format,omitempty"`

	// Array
	Items       *JSONSchema   `json:"items,omitempty"`
	PrefixItems []*JSONSchema `json:"prefixItems,omitempty"`
	MaxItems    *int64        `json:"maxItems,omitempty"`
	MinItems    *int64        `json:"minItems,omitempty"`
	UniqueItems *bool         `json:"uniqueItems,omitempty"`
	Contains    *JSONSchema   `json:"contains,omitempty"`

	// Object
	MaxProperties        *int64                 `json:"maxProperties,omitempty"`
	MinProperties        *int64                 `json:"minProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	PatternProperties    map[string]*JSONSchema `json:"patternProperties,omitempty"`
	AdditionalProperties *BoolOrSchema          `json:"additionalProperties,omitempty"`
	PropertyNames        *JSONSchema            `json:"propertyNames,omitempty"`

	// Composition
	AllOf []*JSONSchema `json:"allOf,omitempty"`
	AnyOf []*JSONSchema `json:"anyOf,omitempty"`
	OneOf []*JSONSchema `json:"oneOf,omitempty"`
	Not   *JSONSchema   `json:"not,omitempty"`

	// Conditional
	If   *JSONSchema `json:"if,omitempty"`
	Then *JSONSchema `json:"then,omitempty"`
	Else *JSONSchema `json:"else,omitempty"`

	// Metadata
	ReadOnly  *bool `json:"readOnly,omitempty"`
	WriteOnly *bool `json:"writeOnly,omitempty"`

	// Extensions (x-kubernetes-* etc.) stored as extra keys
	Extensions map[string]any `json:"-"`
}

// MarshalJSON implements custom marshaling to include extension fields
// alongside the standard schema fields.
func (s *JSONSchema) MarshalJSON() ([]byte, error) {
	type Alias JSONSchema
	base, err := json.Marshal((*Alias)(s))
	if err != nil {
		return nil, err
	}
	if len(s.Extensions) == 0 {
		return base, nil
	}
	// Merge extensions into the base object
	var m map[string]json.RawMessage
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, err
	}
	for k, v := range s.Extensions {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = raw
	}
	return json.Marshal(m)
}

// BoolOrSchema represents a value that can be either a boolean or a JSON Schema.
// In Draft 2020-12, additionalProperties can be true, false, or a schema object.
type BoolOrSchema struct {
	Bool   *bool
	Schema *JSONSchema
}

// BoolSchema creates a BoolOrSchema with a boolean value.
func BoolSchema(b bool) *BoolOrSchema {
	return &BoolOrSchema{Bool: &b}
}

// SchemaValue creates a BoolOrSchema with a schema value.
func SchemaValue(s *JSONSchema) *BoolOrSchema {
	return &BoolOrSchema{Schema: s}
}

// MarshalJSON outputs either a boolean or a schema object.
func (bs *BoolOrSchema) MarshalJSON() ([]byte, error) {
	if bs.Bool != nil {
		return json.Marshal(*bs.Bool)
	}
	if bs.Schema != nil {
		return json.Marshal(bs.Schema)
	}
	return []byte("true"), nil
}

// UnmarshalJSON handles both boolean and schema forms.
func (bs *BoolOrSchema) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		bs.Bool = &b
		return nil
	}
	bs.Schema = &JSONSchema{}
	return json.Unmarshal(data, bs.Schema)
}

// SchemaType handles JSON Schema's type field which can be a string or array of strings.
type SchemaType struct {
	Types []string
}

// MarshalJSON outputs a single string if there's one type, or an array if multiple.
func (t *SchemaType) MarshalJSON() ([]byte, error) {
	if t == nil || len(t.Types) == 0 {
		return []byte("null"), nil
	}
	if len(t.Types) == 1 {
		return json.Marshal(t.Types[0])
	}
	return json.Marshal(t.Types)
}

// UnmarshalJSON handles both string and array forms.
func (t *SchemaType) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		t.Types = []string{single}
		return nil
	}
	return json.Unmarshal(data, &t.Types)
}

// IsEmpty returns true if no types are set.
func (t *SchemaType) IsEmpty() bool {
	return t == nil || len(t.Types) == 0
}

// SingleType creates a SchemaType pointer with a single type string.
func SingleType(t string) *SchemaType {
	return &SchemaType{Types: []string{t}}
}

// NullableType creates a SchemaType pointer that includes "null" alongside the given type.
func NullableType(t string) *SchemaType {
	return &SchemaType{Types: []string{t, "null"}}
}
