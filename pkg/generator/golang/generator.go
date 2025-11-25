package golang

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/ir"
)

//go:embed templates/*
var templatesFS embed.FS

// GoGenerator implements the Generator interface for Go
type GoGenerator struct{}

// NewGoGenerator creates a new Go generator
func NewGoGenerator() *GoGenerator {
	return &GoGenerator{}
}

// GetType returns the generator type identifier
func (g *GoGenerator) GetType() string {
	return "go"
}

// Generate creates a Go SDK from the given configuration and IR
func (g *GoGenerator) Generate(client config.Client, in ir.IR) error {
	// Create directory structure
	if err := os.MkdirAll(client.OutDir, 0o755); err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"pascal":          toPascalCase,
		"camel":           toCamelCase,
		"snake":           toSnakeCase,
		"kebab":           toKebabCase,
		"serviceName":     func(tag string) string { return toPascalCase(tag) + "Service" },
		"serviceField":    func(tag string) string { return toPascalCase(tag) },
		"methodName":      func(op ir.IROperation) string { return ResolveMethodName(client, op) },
		"queryTypeName":   func(op ir.IROperation) string { return toPascalCase(op.Tag) + ResolveMethodName(client, op) + "Query" },
		"goType":          func(x any) string { return schemaToGoType(x) },
		"goStructTag":     func(name string) string { return fmt.Sprintf("`json:\"%s\"`", name) },
		"pathTemplate":    func(op ir.IROperation) string { return buildPathTemplate(op) },
		"pathParams":      func(op ir.IROperation) []ir.IRParam { return orderPathParams(op) },
		"queryParams":     func(op ir.IROperation) []ir.IRParam { return op.QueryParams },
		"hasPathParams":   func(op ir.IROperation) bool { return len(op.PathParams) > 0 },
		"hasQueryParams":  func(op ir.IROperation) bool { return len(op.QueryParams) > 0 },
		"hasRequestBody":  func(op ir.IROperation) bool { return op.RequestBody != nil },
		"methodSignature": func(op ir.IROperation) string { return buildMethodSignature(client, op, ResolveMethodName(client, op)) },
		"reMatch":         func(pattern, s string) bool { r := regexp.MustCompile(pattern); return r.MatchString(s) },
		"formatGoComment": formatGoComment,
		"replace":         strings.ReplaceAll,
		"printf":          fmt.Sprintf,
		"packageName":     func() string { return sanitizePackageName(client.PackageName) },
		"moduleName": func() string {
			if client.ModuleName != "" {
				return client.ModuleName
			}
			return sanitizePackageName(client.PackageName)
		},
		"clientName": func() string { return sanitizePackageName(strings.ToLower(client.Name)) },
		"hasPrefix":  func(s, prefix string) bool { return strings.HasPrefix(s, prefix) },
		// Namespace helper functions
		"groupByNamespace": func(services []ir.IRService) map[string][]ir.IRService {
			namespaces := make(map[string][]ir.IRService)
			for _, service := range services {
				parts := strings.Split(service.Tag, ".")
				if len(parts) == 1 {
					// Root level service
					if namespaces[""] == nil {
						namespaces[""] = []ir.IRService{}
					}
					namespaces[""] = append(namespaces[""], service)
				} else {
					// Namespaced service
					namespace := parts[0]
					if namespaces[namespace] == nil {
						namespaces[namespace] = []ir.IRService{}
					}
					namespaces[namespace] = append(namespaces[namespace], service)
				}
			}
			return namespaces
		},
		"getServiceName": func(tag string) string {
			parts := strings.Split(tag, ".")
			if len(parts) > 1 {
				return parts[1] // Return the part after the dot
			}
			return tag // Return the whole tag if no dot
		},
		// Method signature helpers for dual context pattern
		"methodSignatureWithContext": func(op ir.IROperation) string {
			return buildMethodSignature(client, op, ResolveMethodName(client, op)+"WithContext")
		},
		"methodSignatureNoContext": func(op ir.IROperation) string {
			return buildMethodSignatureNoContext(client, op, ResolveMethodName(client, op))
		},
	}

	// Merge sprig functions
	for k, v := range sprig.FuncMap() {
		funcMap[k] = v
	}

	// Generate client.go
	if err := renderFile(client, "client.go.gotmpl", filepath.Join(client.OutDir, "client.go"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// Generate models.go
	if err := renderFile(client, "models.go.gotmpl", filepath.Join(client.OutDir, "models.go"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// Generate services
	for _, service := range in.Services {
		// Skip services with no operations
		if len(service.Operations) == 0 {
			continue
		}
		fileName := fmt.Sprintf("%s.go", toSnakeCase(service.Tag))
		if err := renderFile(client, "service.go.gotmpl", filepath.Join(client.OutDir, fileName), funcMap, map[string]any{"Client": client, "Service": service}); err != nil {
			return err
		}
	}

	// Generate go.mod
	if err := renderFile(client, "go.mod.gotmpl", filepath.Join(client.OutDir, "go.mod"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}

	// Generate README.md
	if err := renderFile(client, "README.md.gotmpl", filepath.Join(client.OutDir, "README.md"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	return nil
}

// renderFile renders a template file to the target path
func renderFile(client config.Client, templateName, targetPath string, funcMap template.FuncMap, data map[string]any) error {
	// Check if file should be excluded
	if client.ShouldExcludeFile(targetPath) {
		return nil // Skip this file silently
	}

	tmplContent, err := templatesFS.ReadFile("templates/" + templateName)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templateName, err)
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return nil
}
