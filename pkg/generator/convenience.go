package generator

import (
	"path/filepath"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/openapi"
)

// GenerateSDK is a convenience function for generating SDKs with minimal configuration
func GenerateSDK(opts GenerateSDKOptions) error {
	service := NewService()

	genOpts := GenerateOptions{
		ConfigPath:   opts.ConfigPath,
		SingleClient: opts.SingleClient,
		Fallback: FallbackOptions{
			Spec:        opts.Spec,
			Type:        opts.Type,
			OutDir:      opts.OutDir,
			PackageName: opts.PackageName,
			Name:        opts.Name,
			IncludeTags: opts.IncludeTags,
			ExcludeTags: opts.ExcludeTags,
		},
	}

	return service.Generate(genOpts)
}

// GenerateSDKOptions contains options for the convenience GenerateSDK function
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

// GenerateTypeScriptSDK is a convenience function specifically for TypeScript SDK generation
func GenerateTypeScriptSDK(spec, outDir, packageName, clientName string) error {
	// Ensure absolute path for outDir
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	return GenerateSDK(GenerateSDKOptions{
		Spec:        spec,
		Type:        "typescript",
		OutDir:      absOutDir,
		PackageName: packageName,
		Name:        clientName,
	})
}

// GenerateFromConfig is a convenience function for generating from a config file
func GenerateFromConfig(configPath string, singleClient ...string) error {
	service := NewService()
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	onlyClient := ""
	if len(singleClient) > 0 {
		onlyClient = singleClient[0]
	}

	return service.GenerateFromConfig(cfg, onlyClient)
}

// ValidateSpec validates an OpenAPI specification
func ValidateSpec(specPath string) error {
	return openapi.ValidateDocument(specPath)
}
