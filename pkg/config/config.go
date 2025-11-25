package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for SDK generation
type Config struct {
	Spec    string   `yaml:"spec"`
	Name    string   `yaml:"name"`
	Clients []Client `yaml:"clients"`
}

// Client represents configuration for a single client SDK
type Client struct {
	Type        string   `yaml:"type"`
	OutDir      string   `yaml:"outDir"`
	PackageName string   `yaml:"packageName"`
	ModuleName  string   `yaml:"moduleName"`
	Name        string   `yaml:"name"`
	IncludeTags []string `yaml:"includeTags"`
	ExcludeTags []string `yaml:"excludeTags"`
	// IncludeQueryKeys toggles generation of __queryKeys helper methods in services
	IncludeQueryKeys bool `yaml:"includeQueryKeys"`
	// OperationIDParser is an optional executable script to transform operationId to a method name.
	// It will be executed as: <parser> <operationId> <method> <path>
	OperationIDParser string `yaml:"operationIdParser"`
	// PreCommand is an optional command to run before SDK generation starts.
	// Uses Docker Compose array format: ["goimports", "-w", "."]
	// The command will be executed in the output directory.
	PreCommand []string `yaml:"preCommand"`
	// PostCommand is an optional command to run after SDK generation completes.
	// Uses Docker Compose array format: ["goimports", "-w", "."]
	// The command will be executed in the output directory.
	PostCommand []string `yaml:"postCommand"`
	// DefaultBaseURL is the default base URL that will be used if no base URL is provided when creating a client
	DefaultBaseURL string `yaml:"defaultBaseURL"`
	// ExcludeFiles is a list of file paths (relative to outDir) that should not be generated
	// Example: ["package.json", "src/client.ts"]
	ExcludeFiles []string `yaml:"exclude"`
	// TypeAugmentationOptions are options specific to type augmentation generators
	TypeAugmentationOptions TypeAugmentationOptions `yaml:"typeAugmentation"`
}

// TypeAugmentationOptions contains options for type augmentation generators
type TypeAugmentationOptions struct {
	// ModuleName is the module name to augment (e.g., "@blimu/backend")
	ModuleName string `yaml:"moduleName"`
	// Namespace is the namespace within the module to augment (e.g., "Schema")
	Namespace string `yaml:"namespace"`
	// TypeNames is a list of type names to include. If empty, all enum types are included.
	TypeNames []string `yaml:"typeNames"`
	// OutputFileName is the name of the output file (defaults to packageName + ".d.ts")
	OutputFileName string `yaml:"outputFileName"`
}

// GetPreCommand returns the pre-generation command to execute.
func (c *Client) GetPreCommand() []string {
	return c.PreCommand
}

// GetPostCommand returns the post-generation command to execute.
func (c *Client) GetPostCommand() []string {
	return c.PostCommand
}

// ShouldExcludeFile checks if a file path should be excluded based on the ExcludeFiles list.
// targetPath should be an absolute path, and the comparison is done relative to OutDir.
func (c *Client) ShouldExcludeFile(targetPath string) bool {
	if len(c.ExcludeFiles) == 0 {
		return false
	}

	// Get relative path from OutDir to targetPath
	relPath, err := filepath.Rel(c.OutDir, targetPath)
	if err != nil {
		// If we can't get a relative path, the file is not under OutDir, so don't exclude
		return false
	}

	// Normalize the path (use forward slashes for consistency, handle . and ..)
	relPath = filepath.ToSlash(relPath)
	if relPath == "." {
		relPath = ""
	}

	// Check if the relative path matches any exclude pattern
	for _, excludePattern := range c.ExcludeFiles {
		// Normalize exclude pattern
		normalizedExclude := filepath.ToSlash(excludePattern)

		// Exact match
		if relPath == normalizedExclude {
			return true
		}

		// Check if the file is in a directory that matches the exclude pattern
		// For example, if exclude is "src/", then "src/client.ts" should match
		if normalizedExclude != "" && strings.HasPrefix(relPath, normalizedExclude+"/") {
			return true
		}
	}

	return false
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Spec == "" {
		return nil, errors.New("config.spec is required")
	}
	for i := range cfg.Clients {
		c := &cfg.Clients[i]
		if c.Type == "" || c.OutDir == "" || c.PackageName == "" || c.Name == "" {
			return nil, fmt.Errorf("clients[%d] missing required fields (type, outDir, packageName, name)", i)
		}
		if !filepath.IsAbs(c.OutDir) {
			abs, _ := filepath.Abs(c.OutDir)
			c.OutDir = abs
		}
	}
	// Do not absolutize when spec is an HTTP(S) URL
	if u, err := url.Parse(cfg.Spec); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		// keep as-is
	} else if !filepath.IsAbs(cfg.Spec) {
		abs, _ := filepath.Abs(cfg.Spec)
		cfg.Spec = abs
	}
	return &cfg, nil
}
