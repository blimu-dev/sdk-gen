package cli

import (
	"github.com/blimu-dev/sdk-gen/pkg/generator"
	"github.com/blimu-dev/sdk-gen/pkg/openapi"
)

// RunGenerateParams contains parameters for the generate command
type RunGenerateParams struct {
	ConfigPath   string
	SingleClient string
	Fallback     FallbackParams
}

// FallbackParams contains fallback parameters when no config is provided
type FallbackParams struct {
	Spec        string
	Type        string
	OutDir      string
	PackageName string
	Name        string
	IncludeTags []string
	ExcludeTags []string
}

// RunGenerate runs the generate command using the public API
func RunGenerate(p RunGenerateParams) error {
	opts := generator.GenerateSDKOptions{
		ConfigPath:   p.ConfigPath,
		SingleClient: p.SingleClient,
		Spec:         p.Fallback.Spec,
		Type:         p.Fallback.Type,
		OutDir:       p.Fallback.OutDir,
		PackageName:  p.Fallback.PackageName,
		Name:         p.Fallback.Name,
		IncludeTags:  p.Fallback.IncludeTags,
		ExcludeTags:  p.Fallback.ExcludeTags,
	}

	return generator.GenerateSDK(opts)
}

// RunValidate runs the validate command using the public API
func RunValidate(input string) error {
	return openapi.ValidateDocument(input)
}
