package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Spec    string   `yaml:"spec"`
	Name    string   `yaml:"name"`
	Clients []Client `yaml:"clients"`
}

type Client struct {
	Type        string   `yaml:"type"`
	OutDir      string   `yaml:"outDir"`
	PackageName string   `yaml:"packageName"`
	Name        string   `yaml:"name"`
	IncludeTags []string `yaml:"includeTags"`
	ExcludeTags []string `yaml:"excludeTags"`
}

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
	if !filepath.IsAbs(cfg.Spec) {
		abs, _ := filepath.Abs(cfg.Spec)
		cfg.Spec = abs
	}
	return &cfg, nil
}
