package openapi2jsonschema

import (
	"encoding/json"
	"os"
	"testing"
)

func TestConvert_KubernetesOpenAPI(t *testing.T) {
	input, err := os.ReadFile("testdata/k8s-openapi-v3-sample.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	expectedSchemas := []string{
		"io.k8s.api.core.v1.Pod",
		"io.k8s.api.core.v1.PodSpec",
		"io.k8s.api.core.v1.Container",
		"io.k8s.api.core.v1.ContainerPort",
		"io.k8s.api.core.v1.EnvVar",
		"io.k8s.api.core.v1.PodStatus",
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
	}
	for _, name := range expectedSchemas {
		if _, ok := schema.Defs[name]; !ok {
			t.Errorf("missing K8s schema: %s", name)
		}
	}
}

func TestConvert_KubernetesRefRewriting(t *testing.T) {
	input, err := os.ReadFile("testdata/k8s-openapi-v3-sample.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	pod := schema.Defs["io.k8s.api.core.v1.Pod"]
	if pod == nil {
		t.Fatal("Pod schema is nil")
	}

	// metadata should be a $ref to ObjectMeta
	metaProp := pod.Properties["metadata"]
	if metaProp == nil {
		t.Fatal("Pod.metadata is nil")
	}
	expectedRef := "#/$defs/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
	if metaProp.Ref != expectedRef {
		t.Errorf("Pod.metadata.$ref = %q, want %q", metaProp.Ref, expectedRef)
	}

	// spec should be a $ref to PodSpec
	specProp := pod.Properties["spec"]
	if specProp == nil {
		t.Fatal("Pod.spec is nil")
	}
	expectedRef = "#/$defs/io.k8s.api.core.v1.PodSpec"
	if specProp.Ref != expectedRef {
		t.Errorf("Pod.spec.$ref = %q, want %q", specProp.Ref, expectedRef)
	}
}

func TestConvert_KubernetesExtensionsStripped(t *testing.T) {
	input, err := os.ReadFile("testdata/k8s-openapi-v3-sample.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	// Default: extensions should be stripped
	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	output, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Should not contain x-kubernetes in default mode
	outputStr := string(output)
	if contains(outputStr, "x-kubernetes") {
		t.Error("default mode should strip x-kubernetes-* extensions")
	}
}

func TestConvert_KubernetesExtensionsPreserved(t *testing.T) {
	input, err := os.ReadFile("testdata/k8s-openapi-v3-sample.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input, WithPreserveKubernetesExtensions())
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	pod := schema.Defs["io.k8s.api.core.v1.Pod"]
	if pod == nil {
		t.Fatal("Pod schema is nil")
	}

	// Should have x-kubernetes-group-version-kind extension
	if pod.Extensions == nil {
		t.Fatal("Pod extensions is nil, expected x-kubernetes-group-version-kind")
	}
	if _, ok := pod.Extensions["x-kubernetes-group-version-kind"]; !ok {
		t.Error("Pod should have x-kubernetes-group-version-kind extension")
	}
}

func TestConvert_KubernetesAdditionalProperties(t *testing.T) {
	input, err := os.ReadFile("testdata/k8s-openapi-v3-sample.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	meta := schema.Defs["io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"]
	if meta == nil {
		t.Fatal("ObjectMeta schema is nil")
	}

	labels := meta.Properties["labels"]
	if labels == nil {
		t.Fatal("ObjectMeta.labels is nil")
	}

	// labels should have additionalProperties with type: string
	if labels.AdditionalProperties == nil {
		t.Fatal("labels.additionalProperties is nil")
	}
	if labels.AdditionalProperties.Schema == nil {
		t.Fatal("labels.additionalProperties should be a schema")
	}
	if len(labels.AdditionalProperties.Schema.Type.Types) != 1 || labels.AdditionalProperties.Schema.Type.Types[0] != "string" {
		t.Errorf("labels.additionalProperties.type = %v, want [string]", labels.AdditionalProperties.Schema.Type.Types)
	}
}

func TestConvertSchema_KubernetesPod(t *testing.T) {
	input, err := os.ReadFile("testdata/k8s-openapi-v3-sample.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	schema, err := ConvertSchema(input, "io.k8s.api.core.v1.Pod")
	if err != nil {
		t.Fatalf("ConvertSchema() error: %v", err)
	}

	if schema.Ref != "#/$defs/io.k8s.api.core.v1.Pod" {
		t.Errorf("root $ref = %q, want %q", schema.Ref, "#/$defs/io.k8s.api.core.v1.Pod")
	}

	// Verify the output is valid JSON
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
