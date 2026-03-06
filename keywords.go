package openapi2jsonschema

import "strings"

// openAPIOnlyKeywords are keywords present in OpenAPI Schema Objects
// that have no equivalent in JSON Schema Draft 2020-12 and should be stripped.
var openAPIOnlyKeywords = map[string]bool{
	"discriminator": true,
	"xml":           true,
	"externalDocs":  true,
	"example":       true,
}

// isOpenAPIOnlyKeyword returns true if the key is an OpenAPI-specific keyword
// that should be stripped from JSON Schema output.
func isOpenAPIOnlyKeyword(key string) bool {
	return openAPIOnlyKeywords[key]
}

// isExtension returns true if the key is an OpenAPI extension (x-* prefix).
func isExtension(key string) bool {
	return strings.HasPrefix(key, "x-")
}

// isKubernetesExtension returns true if the key is a Kubernetes-specific
// extension (x-kubernetes-* prefix).
func isKubernetesExtension(key string) bool {
	return strings.HasPrefix(key, "x-kubernetes-")
}

// shouldIncludeExtension determines whether an extension key should be
// included in the output based on the configuration.
func shouldIncludeExtension(key string, cfg *config) bool {
	if cfg.stripAllExtensions {
		return false
	}
	if isKubernetesExtension(key) {
		return cfg.preserveKubernetesExtensions
	}
	// Non-kubernetes extensions are stripped by default
	return false
}
