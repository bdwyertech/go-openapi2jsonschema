# openapi2jsonschema

A Go library that converts OpenAPI 3.x documents to JSON Schema Draft 2020-12.

## Install

```bash
go get github.com/bdwyertech/go-openapi2jsonschema
```

## Usage

### Convert all schemas

```go
package main

import (
	"fmt"
	"os"

	openapi2jsonschema "github.com/bdwyertech/go-openapi2jsonschema"
)

func main() {
	input, _ := os.ReadFile("k8s-openapi.json")

	// Convert all component schemas to JSON Schema
	schema, err := openapi2jsonschema.Convert(input)
	if err != nil {
		panic(err)
	}

	output, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Println(string(output))
}
```

### Convert a single schema

```go
// Get a JSON Schema rooted at a specific type
schema, err := openapi2jsonschema.ConvertSchema(input, "io.k8s.api.core.v1.Pod")
```

### Convert to JSON bytes directly

```go
jsonBytes, err := openapi2jsonschema.ConvertToJSON(input)
```

### Read from io.Reader

```go
schema, err := openapi2jsonschema.ConvertReader(resp.Body)
```

## Options

```go
// Set $id on the root schema
schema, _ := openapi2jsonschema.Convert(input,
	openapi2jsonschema.WithSchemaID("https://kubernetes.io/schemas"),
)

// Preserve x-kubernetes-* extensions in output
schema, _ := openapi2jsonschema.Convert(input,
	openapi2jsonschema.WithPreserveKubernetesExtensions(),
)

// Strip all x-* extensions
schema, _ := openapi2jsonschema.Convert(input,
	openapi2jsonschema.WithStripExtensions(),
)
```

## What it does

| OpenAPI 3.x | JSON Schema Draft 2020-12 |
|---|---|
| `#/components/schemas/X` | `#/$defs/X` |
| `nullable: true` | `type: ["string", "null"]` |
| `discriminator` | stripped |
| `xml` | stripped |
| `externalDocs` | stripped |
| `example` | stripped |
| `readOnly` / `writeOnly` | preserved |
| `allOf` / `oneOf` / `anyOf` / `not` | preserved |
| `additionalProperties` (bool or schema) | preserved |
| `x-kubernetes-*` | configurable (stripped by default) |

## Output format

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$defs": {
    "io.k8s.api.core.v1.Pod": { ... },
    "io.k8s.api.core.v1.PodSpec": { ... }
  }
}
```

When using `ConvertSchema()` for a single type:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$ref": "#/$defs/io.k8s.api.core.v1.Pod",
  "$defs": { ... }
}
```

## Dependencies

- [libopenapi](https://github.com/pb33f/libopenapi) — OpenAPI 3.x parsing

## License

MIT
