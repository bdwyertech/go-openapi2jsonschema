// Package openapi2jsonschema converts OpenAPI 3.x documents to JSON Schema Draft 2020-12.
package openapi2jsonschema

// config holds the internal configuration for a conversion.
type config struct {
	schemaID                     string
	preserveKubernetesExtensions bool
	stripAllExtensions           bool
}

// Option configures the conversion behavior.
type Option func(*config)

// WithSchemaID sets the $id field on the root JSON Schema output.
func WithSchemaID(id string) Option {
	return func(c *config) {
		c.schemaID = id
	}
}

// WithPreserveKubernetesExtensions keeps x-kubernetes-* extension fields
// in the output JSON Schema. By default they are stripped.
func WithPreserveKubernetesExtensions() Option {
	return func(c *config) {
		c.preserveKubernetesExtensions = true
	}
}

// WithStripExtensions removes all x-* extension fields from the output.
// This takes precedence over WithPreserveKubernetesExtensions.
func WithStripExtensions() Option {
	return func(c *config) {
		c.stripAllExtensions = true
	}
}

func newConfig(opts []Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
