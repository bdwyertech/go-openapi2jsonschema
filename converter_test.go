package openapi2jsonschema

import (
	"encoding/json"
	"os"
	"testing"
)

func TestConvert_SimpleOpenAPI(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	if schema.Schema != schemaDraft202012 {
		t.Errorf("$schema = %q, want %q", schema.Schema, schemaDraft202012)
	}

	if schema.Defs == nil {
		t.Fatal("$defs is nil, expected schema definitions")
	}

	// Check that all 4 schemas are present
	expectedSchemas := []string{"Pet", "PetList", "Error", "Dog"}
	for _, name := range expectedSchemas {
		if _, ok := schema.Defs[name]; !ok {
			t.Errorf("missing schema definition: %s", name)
		}
	}
}

func TestConvert_PetSchema(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	pet := schema.Defs["Pet"]
	if pet == nil {
		t.Fatal("Pet schema is nil")
	}

	// Type should be "object"
	if len(pet.Type.Types) != 1 || pet.Type.Types[0] != "object" {
		t.Errorf("Pet type = %v, want [object]", pet.Type.Types)
	}

	// Required should contain "name"
	if len(pet.Required) != 1 || pet.Required[0] != "name" {
		t.Errorf("Pet required = %v, want [name]", pet.Required)
	}

	// Properties should exist
	if pet.Properties == nil {
		t.Fatal("Pet properties is nil")
	}

	// Check id property
	idProp := pet.Properties["id"]
	if idProp == nil {
		t.Fatal("Pet.id property is nil")
	}
	if idProp.Format != "int64" {
		t.Errorf("Pet.id format = %q, want %q", idProp.Format, "int64")
	}
	if idProp.ReadOnly == nil || !*idProp.ReadOnly {
		t.Error("Pet.id readOnly should be true")
	}

	// Check name property constraints
	nameProp := pet.Properties["name"]
	if nameProp == nil {
		t.Fatal("Pet.name property is nil")
	}
	if nameProp.MinLength == nil || *nameProp.MinLength != 1 {
		t.Errorf("Pet.name minLength = %v, want 1", nameProp.MinLength)
	}
	if nameProp.MaxLength == nil || *nameProp.MaxLength != 100 {
		t.Errorf("Pet.name maxLength = %v, want 100", nameProp.MaxLength)
	}

	// Check additionalProperties: false
	if pet.AdditionalProperties == nil {
		t.Fatal("Pet additionalProperties is nil")
	}
	if pet.AdditionalProperties.Bool == nil || *pet.AdditionalProperties.Bool != false {
		t.Error("Pet additionalProperties should be false")
	}
}

func TestConvert_NullableField(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	pet := schema.Defs["Pet"]
	tagProp := pet.Properties["tag"]
	if tagProp == nil {
		t.Fatal("Pet.tag property is nil")
	}

	// nullable: true should produce type: ["string", "null"]
	if len(tagProp.Type.Types) != 2 {
		t.Fatalf("Pet.tag type = %v, want [string, null]", tagProp.Type.Types)
	}
	if tagProp.Type.Types[0] != "string" || tagProp.Type.Types[1] != "null" {
		t.Errorf("Pet.tag type = %v, want [string, null]", tagProp.Type.Types)
	}
}

func TestConvert_EnumField(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	pet := schema.Defs["Pet"]
	statusProp := pet.Properties["status"]
	if statusProp == nil {
		t.Fatal("Pet.status property is nil")
	}

	if len(statusProp.Enum) != 3 {
		t.Fatalf("Pet.status enum length = %d, want 3", len(statusProp.Enum))
	}
}

func TestConvert_RefRewriting(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	petList := schema.Defs["PetList"]
	if petList == nil {
		t.Fatal("PetList schema is nil")
	}

	itemsProp := petList.Properties["items"]
	if itemsProp == nil {
		t.Fatal("PetList.items property is nil")
	}

	// items.items should be a $ref to #/$defs/Pet
	if itemsProp.Items == nil {
		t.Fatal("PetList.items.items is nil")
	}
	if itemsProp.Items.Ref != "#/$defs/Pet" {
		t.Errorf("PetList.items.items.$ref = %q, want %q", itemsProp.Items.Ref, "#/$defs/Pet")
	}
}

func TestConvert_AllOfComposition(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	dog := schema.Defs["Dog"]
	if dog == nil {
		t.Fatal("Dog schema is nil")
	}

	if len(dog.AllOf) != 2 {
		t.Fatalf("Dog allOf length = %d, want 2", len(dog.AllOf))
	}

	// First allOf entry should be a $ref to Pet
	if dog.AllOf[0].Ref != "#/$defs/Pet" {
		t.Errorf("Dog allOf[0].$ref = %q, want %q", dog.AllOf[0].Ref, "#/$defs/Pet")
	}

	// Second allOf entry should have breed property
	if dog.AllOf[1].Properties == nil || dog.AllOf[1].Properties["breed"] == nil {
		t.Error("Dog allOf[1] should have breed property")
	}
}

func TestConvertSchema_SingleSchema(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := ConvertSchema(input, "Pet")
	if err != nil {
		t.Fatalf("ConvertSchema() error: %v", err)
	}

	if schema.Ref != "#/$defs/Pet" {
		t.Errorf("root $ref = %q, want %q", schema.Ref, "#/$defs/Pet")
	}

	if schema.Defs == nil || schema.Defs["Pet"] == nil {
		t.Error("Pet should be in $defs")
	}
}

func TestConvertSchema_NotFound(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	_, err = ConvertSchema(input, "NonExistent")
	if err == nil {
		t.Error("ConvertSchema() should return error for non-existent schema")
	}
}

func TestWithSchemaID(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input, WithSchemaID("https://example.com/schemas"))
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	if schema.ID != "https://example.com/schemas" {
		t.Errorf("$id = %q, want %q", schema.ID, "https://example.com/schemas")
	}
}

func TestConvertToJSON_ValidOutput(t *testing.T) {
	input, err := os.ReadFile("testdata/simple-openapi.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	output, err := ConvertToJSON(input)
	if err != nil {
		t.Fatalf("ConvertToJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify $schema is present
	if parsed["$schema"] != schemaDraft202012 {
		t.Errorf("$schema = %v, want %v", parsed["$schema"], schemaDraft202012)
	}

	// Verify $defs is present
	if _, ok := parsed["$defs"]; !ok {
		t.Error("$defs missing from JSON output")
	}
}
