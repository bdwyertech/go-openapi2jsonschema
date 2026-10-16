package openapi2jsonschema

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

// convertValueSpec converts a minimal inline OpenAPI 3.1 doc whose single
// component schema "S" has the given properties JSON, and returns "S".
func convertValueSpec(t *testing.T, props string) *JSONSchema {
	t.Helper()
	spec := `{"openapi":"3.1.0","info":{"title":"t","version":"1"},"paths":{},` +
		`"components":{"schemas":{"S":{"type":"object","properties":` + props + `}}}}`
	schema, err := Convert([]byte(spec), WithPreserveKubernetesExtensions())
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}
	s := schema.Defs["S"]
	if s == nil || s.Properties == nil {
		t.Fatal("schema S or its properties is nil")
	}
	return s
}

// assertPlainValue fails if v (recursively) contains a non-plain Go type,
// e.g. a *yaml.Node leaking through from a mismatched yaml package.
func assertPlainValue(t *testing.T, path string, v any) {
	t.Helper()
	switch x := v.(type) {
	case nil, string, bool, int, int64, uint64, float64:
	case []any:
		for i, e := range x {
			assertPlainValue(t, path+"["+itoa(i)+"]", e)
		}
	case map[string]any:
		for k, e := range x {
			assertPlainValue(t, path+"."+k, e)
		}
	default:
		t.Errorf("%s has non-plain type %s (%#v)", path, reflect.TypeOf(v), v)
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func TestConvert_EnumDecodedValues(t *testing.T) {
	s := convertValueSpec(t, `{
		"color":{"type":"string","enum":["red","green","blue"]},
		"level":{"type":"integer","enum":[1,2,3]}}`)

	cases := map[string][]any{
		"color": {"red", "green", "blue"},
		"level": {1, 2, 3},
	}
	for name, want := range cases {
		got := s.Properties[name].Enum
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s enum = %#v, want %#v", name, got, want)
		}
		for i, v := range got {
			if reflect.TypeOf(v) != reflect.TypeOf(want[i]) {
				t.Errorf("%s enum[%d] type = %T, want %T", name, i, v, want[i])
			}
		}
		assertPlainValue(t, name+".enum", got)
	}
}

func TestConvert_DefaultDecodedValues(t *testing.T) {
	s := convertValueSpec(t, `{
		"str":{"type":"string","default":"hello"},
		"num":{"type":"number","default":1.5},
		"int":{"type":"integer","default":42},
		"flag":{"type":"boolean","default":true},
		"obj":{"type":"object","default":{"a":1,"b":"x"}},
		"arr":{"type":"array","default":["p","q"]}}`)

	cases := map[string]any{
		"str":  "hello",
		"num":  1.5,
		"int":  42,
		"flag": true,
		"obj":  map[string]any{"a": 1, "b": "x"},
		"arr":  []any{"p", "q"},
	}
	for name, want := range cases {
		got := s.Properties[name].Default
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s default = %#v (%T), want %#v (%T)", name, got, got, want, want)
		}
		assertPlainValue(t, name+".default", got)
	}
}

func TestConvert_NullDefault(t *testing.T) {
	s := convertValueSpec(t, `{"n":{"type":["string","null"],"default":null}}`)
	if got := s.Properties["n"].Default; got != nil {
		t.Fatalf("null default = %#v (%T), want nil", got, got)
	}
	out, err := json.Marshal(s.Properties["n"])
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// Default is `any` with omitempty, so a nil default is omitted from JSON.
	if contains(string(out), `"default"`) {
		t.Errorf("null default should be omitted from JSON, got %s", out)
	}
}

func TestConvert_ExtensionDecodedValues(t *testing.T) {
	s := convertValueSpec(t, `{"e":{"type":"object",
		"x-kubernetes-list-type":"map",
		"x-kubernetes-int-or-string":true,
		"x-kubernetes-meta":{"group":"apps","n":2,"tags":["a"]}}}`)
	ext := s.Properties["e"].Extensions
	want := map[string]any{
		"x-kubernetes-list-type":     "map",
		"x-kubernetes-int-or-string": true,
		"x-kubernetes-meta":          map[string]any{"group": "apps", "n": 2, "tags": []any{"a"}},
	}
	if !reflect.DeepEqual(ext, want) {
		t.Errorf("extensions = %#v, want %#v", ext, want)
	}
	assertPlainValue(t, "extensions", map[string]any(ext))
}
