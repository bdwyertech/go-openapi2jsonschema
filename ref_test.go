package openapi2jsonschema

import "testing"

func TestRewriteRef(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#/components/schemas/Pet", "#/$defs/Pet"},
		{"#/components/schemas/io.k8s.api.core.v1.Pod", "#/$defs/io.k8s.api.core.v1.Pod"},
		{"#/components/schemas/Foo.Bar", "#/$defs/Foo.Bar"},
		{"#/some/other/path", "#/some/other/path"},
		{"", ""},
	}

	for _, tt := range tests {
		got := rewriteRef(tt.input)
		if got != tt.want {
			t.Errorf("rewriteRef(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
