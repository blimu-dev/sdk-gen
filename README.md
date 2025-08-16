# SDK Generator

A powerful Go library and CLI tool for generating type-safe SDKs from OpenAPI specifications. Currently supports TypeScript with plans for additional languages.

## Features

- 🚀 **Multiple Language Support**: Currently TypeScript, with more languages planned
- 📝 **OpenAPI 3.x Support**: Full support for modern OpenAPI specifications
- 🎯 **Tag Filtering**: Include/exclude specific API endpoints by tags
- 🔧 **Highly Configurable**: Flexible configuration via YAML files or programmatic API
- 📦 **Library & CLI**: Use as a Go library or standalone CLI tool
- 🎨 **Beautiful Generated Code**: Clean, idiomatic code with excellent TypeScript types

## Installation

### As a CLI Tool

```bash
go install github.com/blimu-dev/sdk-gen/cmd/sdk-gen@latest
```

### As a Go Library

```bash
go get github.com/blimu-dev/sdk-gen
```

## Quick Start

### CLI Usage

```bash
# Generate TypeScript SDK from OpenAPI spec
sdk-gen generate \
  --input https://petstore3.swagger.io/api/v3/openapi.json \
  --type typescript \
  --out ./petstore-sdk \
  --package-name petstore-client \
  --client-name PetStoreClient

# Using a configuration file
sdk-gen generate --config sdkgen.yaml
```

### Library Usage

#### Simple TypeScript SDK Generation

```go
package main

import (
    "log"
    "github.com/blimu-dev/sdk-gen/pkg/generator"
)

func main() {
    err := generator.GenerateTypeScriptSDK(
        "https://petstore3.swagger.io/api/v3/openapi.json", // OpenAPI spec
        "./generated-sdk",                                   // Output directory
        "petstore-client",                                   // Package name
        "PetStoreClient",                                    // Client class name
    )
    if err != nil {
        log.Fatalf("Failed to generate SDK: %v", err)
    }

    log.Println("SDK generated successfully!")
}
```

#### Advanced Configuration

```go
package main

import (
    "log"
    "github.com/blimu-dev/sdk-gen/pkg/generator"
)

func main() {
    err := generator.GenerateSDK(generator.GenerateSDKOptions{
        Spec:        "./openapi.yaml",
        Type:        "typescript",
        OutDir:      "./my-sdk",
        PackageName: "my-api-client",
        Name:        "MyAPIClient",
        IncludeTags: []string{"users", "orders"}, // Only include these tags
        ExcludeTags: []string{"internal"},        // Exclude these tags
    })
    if err != nil {
        log.Fatalf("Failed to generate SDK: %v", err)
    }
}
```

#### Using Configuration Files

```go
package main

import (
    "log"
    "github.com/blimu-dev/sdk-gen/pkg/generator"
)

func main() {
    // Generate from config file
    err := generator.GenerateFromConfig("./sdkgen.yaml")
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }

    // Generate only specific client from config
    err = generator.GenerateFromConfig("./sdkgen.yaml", "my-client")
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }
}
```

## Configuration

### YAML Configuration

Create a `sdkgen.yaml` file:

```yaml
# OpenAPI specification (file path or URL)
spec: "./openapi.yaml"

# Global name for the API
name: "MyAPI"

# Client configurations
clients:
  - type: "typescript"
    outDir: "./typescript-sdk"
    packageName: "my-api-client"
    name: "MyAPIClient"
    includeTags: ["public", "users"]
    excludeTags: ["internal"]
    includeQueryKeys: true
    operationIdParser: "./scripts/parse-operation-id.sh"
    preCommand:
      - "rm -rf src/ && mkdir -p src/"
      - "test -f package.json || npm init -y"
    postCommand:
      - "npx prettier --write . && echo 'Formatted code'"
      - "npm run type-check || echo 'Type check failed, continuing...'"
      - "npm audit fix --audit-level moderate"

  - type: "go"
    outDir: "./go-sdk"
    packageName: "my-go-client"
    name: "MyGoClient"
    preCommand:
      - "rm -rf *.go && echo 'Cleaned old Go files'"
      - "test -f go.mod || go mod init my-go-client"
    postCommand:
      - "goimports -w . && go mod tidy"
      - "go test ./... || echo 'Tests failed, but SDK generated'"
```

### Configuration Options

- **`spec`**: Path to OpenAPI specification file or HTTP(S) URL
- **`name`**: Global name for the API
- **`clients`**: Array of client configurations
  - **`type`**: Generator type (`"typescript"`)
  - **`outDir`**: Output directory for generated code
  - **`packageName`**: Package name for the generated SDK
  - **`name`**: Client class name
  - **`includeTags`**: Array of regex patterns for tags to include
  - **`excludeTags`**: Array of regex patterns for tags to exclude
  - **`includeQueryKeys`**: Generate query key helpers for React Query (TypeScript only)
  - **`operationIdParser`**: Optional script to transform operation IDs
  - **`preCommand`**: Single command to run before SDK generation (Docker Compose array format)
  - **`postCommand`**: Single command to run after SDK generation (Docker Compose array format)

### Command Format

Commands use Docker Compose array format for safe and explicit argument parsing:

```yaml
# Simple command
postCommand: ["goimports", "-w", "."]

# Command with multiple arguments
postCommand: ["npx", "prettier", "--write", "."]

# Complex shell operations (use bash -c)
postCommand: ["bash", "-c", "npm run lint && npm run test || echo 'Tests failed'"]
```

**Benefits:**

- **Safe argument parsing** - No shell escaping issues
- **Explicit** - Clear separation of command and arguments
- **Cross-platform** - Works consistently across operating systems
- **No injection risks** - Arguments are properly escaped

**For complex shell operations**, wrap them in `bash -c`:

```yaml
preCommand: ["bash", "-c", "rm -rf dist/ && mkdir -p dist/"]
postCommand:
  ["bash", "-c", "go mod tidy && go test ./... || echo 'Tests failed'"]
```

## Generated TypeScript SDK

The generated TypeScript SDK includes:

- **Type-safe client class** with methods for each API endpoint
- **Complete TypeScript interfaces** for all request/response types
- **Service classes** organized by OpenAPI tags
- **React Query integration** (optional) with query keys and hooks
- **Comprehensive JSDoc comments** from OpenAPI descriptions

### Example Generated Usage

```typescript
import { MyAPIClient } from "./generated-sdk";

const client = new MyAPIClient({
  baseURL: "https://api.example.com",
  apiKey: "your-api-key",
});

// Type-safe API calls
const user = await client.users.retrieve("user-123");
const users = await client.users.list({ page: 1, limit: 10 });
const newUser = await client.users.create({
  name: "John Doe",
  email: "john@example.com",
});
```

## Library API Reference

### Core Functions

#### `generator.GenerateTypeScriptSDK(spec, outDir, packageName, clientName string) error`

Convenience function for generating a TypeScript SDK with minimal configuration.

#### `generator.GenerateSDK(opts GenerateSDKOptions) error`

Generate SDK with full configuration options.

#### `generator.GenerateFromConfig(configPath string, singleClient ...string) error`

Generate SDK from a YAML configuration file.

#### `generator.ValidateSpec(specPath string) error`

Validate an OpenAPI specification.

### Advanced Usage

#### Custom Generator Registry

```go
package main

import (
    "github.com/blimu-dev/sdk-gen/pkg/generator"
    "github.com/blimu-dev/sdk-gen/pkg/generator/typescript"
)

func main() {
    // Create custom registry
    registry := generator.NewRegistry()
    registry.Register(typescript.NewTypeScriptGenerator())
    // Add your custom generators here

    // Create service with custom registry
    service := generator.NewServiceWithRegistry(registry)

    // Use the service...
}
```

#### Implementing Custom Generators

```go
package mygenerator

import (
    "github.com/blimu-dev/sdk-gen/pkg/config"
    "github.com/blimu-dev/sdk-gen/pkg/ir"
)

type MyGenerator struct{}

func (g *MyGenerator) GetType() string {
    return "mylang"
}

func (g *MyGenerator) Generate(client config.Client, ir ir.IR) error {
    // Implement your generator logic here
    return nil
}
```

## Examples

See the [`examples/`](./examples/) directory for complete working examples:

- [`examples/basic/`](./examples/basic/) - Simple SDK generation
- [`examples/advanced/`](./examples/advanced/) - Advanced usage with custom configurations

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
