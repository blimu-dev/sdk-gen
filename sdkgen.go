// Package sdkgen provides a powerful Go library for generating type-safe SDKs from OpenAPI specifications.
//
// This package offers both a simple API for common use cases and a flexible API for advanced scenarios.
// It currently supports TypeScript SDK generation with plans for additional languages.
//
// Quick Start:
//
//	import "github.com/blimu-dev/sdk-gen"
//
//	// Generate a TypeScript SDK
//	err := sdkgen.GenerateTypeScriptSDK(
//		"https://petstore3.swagger.io/api/v3/openapi.json",
//		"./generated-sdk",
//		"petstore-client",
//		"PetStoreClient",
//	)
//
// For more advanced usage, see the generator package.
package sdkgen

import (
	"github.com/blimu-dev/sdk-gen/pkg/generator"
)

// GenerateTypeScriptSDK is a convenience function for generating a TypeScript SDK with minimal configuration.
// It generates a complete TypeScript SDK from an OpenAPI specification.
//
// Parameters:
//   - spec: Path to OpenAPI specification file or HTTP(S) URL
//   - outDir: Output directory for the generated SDK
//   - packageName: NPM package name for the generated SDK
//   - clientName: Name of the main client class
//
// Example:
//
//	err := sdkgen.GenerateTypeScriptSDK(
//		"./openapi.yaml",
//		"./my-sdk",
//		"my-api-client",
//		"MyAPIClient",
//	)
func GenerateTypeScriptSDK(spec, outDir, packageName, clientName string) error {
	return generator.GenerateTypeScriptSDK(spec, outDir, packageName, clientName)
}

// GenerateSDK generates an SDK with full configuration options.
// This function provides more control over the generation process.
//
// Example:
//
//	err := sdkgen.GenerateSDK(sdkgen.GenerateSDKOptions{
//		Spec:        "./openapi.yaml",
//		Type:        "typescript",
//		OutDir:      "./my-sdk",
//		PackageName: "my-api-client",
//		Name:        "MyAPIClient",
//		IncludeTags: []string{"users", "orders"},
//		ExcludeTags: []string{"internal"},
//	})
func GenerateSDK(opts GenerateSDKOptions) error {
	genOpts := generator.GenerateSDKOptions{
		ConfigPath:   opts.ConfigPath,
		SingleClient: opts.SingleClient,
		Spec:         opts.Spec,
		Type:         opts.Type,
		OutDir:       opts.OutDir,
		PackageName:  opts.PackageName,
		Name:         opts.Name,
		IncludeTags:  opts.IncludeTags,
		ExcludeTags:  opts.ExcludeTags,
	}
	return generator.GenerateSDK(genOpts)
}

// GenerateFromConfig generates SDKs from a YAML configuration file.
// Optionally, you can specify a single client name to generate only that client.
//
// Example:
//
//	// Generate all clients from config
//	err := sdkgen.GenerateFromConfig("./sdkgen.yaml")
//
//	// Generate only a specific client
//	err := sdkgen.GenerateFromConfig("./sdkgen.yaml", "my-client")
func GenerateFromConfig(configPath string, singleClient ...string) error {
	return generator.GenerateFromConfig(configPath, singleClient...)
}

// ValidateSpec validates an OpenAPI specification file.
// This is useful for checking if a spec is valid before attempting to generate an SDK.
//
// Example:
//
//	err := sdkgen.ValidateSpec("./openapi.yaml")
//	if err != nil {
//		log.Fatalf("Invalid OpenAPI spec: %v", err)
//	}
func ValidateSpec(specPath string) error {
	return generator.ValidateSpec(specPath)
}

// GenerateSDKOptions contains options for SDK generation
type GenerateSDKOptions struct {
	// ConfigPath is the path to the configuration file (optional)
	ConfigPath string

	// SingleClient generates only the named client from config (optional)
	SingleClient string

	// Fallback options when no config file is provided
	Spec        string   // OpenAPI spec file or URL
	Type        string   // Generator type (e.g., "typescript")
	OutDir      string   // Output directory
	PackageName string   // Package name for the generated SDK
	Name        string   // Client class name
	IncludeTags []string // Regex patterns for tags to include
	ExcludeTags []string // Regex patterns for tags to exclude
}
