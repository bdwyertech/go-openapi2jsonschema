package openapi2jsonschema

import "strings"

const (
	openAPISchemaPrefix = "#/components/schemas/"
	jsonSchemaDefPrefix = "#/$defs/"
)

// rewriteRef converts an OpenAPI $ref path to a JSON Schema $ref path.
// e.g. "#/components/schemas/io.k8s.api.core.v1.Pod" → "#/definitions/io.k8s.api.core.v1.Pod"
func rewriteRef(ref string) string {
	if strings.HasPrefix(ref, openAPISchemaPrefix) {
		return jsonSchemaDefPrefix + ref[len(openAPISchemaPrefix):]
	}
	return ref
}
