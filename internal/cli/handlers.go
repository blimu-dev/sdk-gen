package cli

import (
	"path/filepath"
)

type Client = struct {
	Type        string
	OutDir      string
	PackageName string
	Name        string
	IncludeTags []string
	ExcludeTags []string
}

// utility
func absPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	abs, _ := filepath.Abs(p)
	return abs
}
