package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

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
	Name        string   `yaml:"name"`
	IncludeTags []string `yaml:"includeTags"`
	ExcludeTags []string `yaml:"excludeTags"`
	// IncludeQueryKeys toggles generation of __queryKeys helper methods in services
	IncludeQueryKeys bool `yaml:"includeQueryKeys"`
	// OperationIDParser is an optional executable script to transform operationId to a method name.
	// It will be executed as: <parser> <operationId> <method> <path>
	OperationIDParser string `yaml:"operationIdParser"`
	// PostGenCommand is an optional command to run after SDK generation completes.
	// It will be executed in the output directory. Useful for formatting, linting, or cleanup.
	PostGenCommand string `yaml:"postGenCommand"`
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
