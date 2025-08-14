package typescript

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

// TypeScriptGenerator implements the Generator interface for TypeScript
type TypeScriptGenerator struct{}

// NewTypeScriptGenerator creates a new TypeScript generator
func NewTypeScriptGenerator() *TypeScriptGenerator {
	return &TypeScriptGenerator{}
}

// GetType returns the generator type identifier
func (g *TypeScriptGenerator) GetType() string {
	return "typescript"
}

// Generate creates a TypeScript SDK from the given configuration and IR
func (g *TypeScriptGenerator) Generate(client config.Client, in ir.IR) error {
	// Ensure directories
	srcDir := filepath.Join(client.OutDir, "src")
	servicesDir := filepath.Join(srcDir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"pascal":      toPascalCase,
		"camel":       toCamelCase,
		"kebab":       toKebabCase,
		"serviceName": func(tag string) string { return toPascalCase(tag) + "Service" },
		"serviceProp": func(tag string) string { return toCamelCase(tag) },
		"fileBase":    func(tag string) string { return strings.ToLower(toSnakeCase(tag)) },
		"methodName":  func(op ir.IROperation) string { return resolveMethodName(client, op) },
		"queryTypeName": func(op ir.IROperation) string {
			return toPascalCase(op.Tag) + toPascalCase(resolveMethodName(client, op)) + "Query"
		},
		"pathTemplate":      func(op ir.IROperation) string { return buildPathTemplate(op) },
		"queryKeyBase":      func(op ir.IROperation) string { return buildQueryKeyBase(op) },
		"pathParamsInOrder": func(op ir.IROperation) []ir.IRParam { return orderPathParams(op) },
		"methodSignature":   func(op ir.IROperation) []string { return buildMethodSignature(op, resolveMethodName(client, op)) },
		"methodSignatureNoInit": func(op ir.IROperation) []string {
			parts := buildMethodSignature(op, resolveMethodName(client, op))
			if len(parts) > 0 {
				return parts[:len(parts)-1]
			}
			return parts
		},
		"queryKeyArgs": func(op ir.IROperation) []string { return queryKeyArgs(op) },
		"tsType": func(x any) string {
			switch v := x.(type) {
			case ir.IRSchema:
				return schemaToTSType(v)
			case *ir.IRSchema:
				if v != nil {
					return schemaToTSType(*v)
				}
				return "unknown"
			default:
				return "unknown"
			}
		},
		"stripSchemaNs": func(s string) string { return strings.ReplaceAll(s, "Schema.", "") },
		"reMatch":       func(pattern, s string) bool { r := regexp.MustCompile(pattern); return r.MatchString(s) },
	}

	// Merge sprig functions
	for k, v := range sprig.FuncMap() {
		funcMap[k] = v
	}

	// client.ts
	if err := renderFile("client.ts.gotmpl", filepath.Join(srcDir, "client.ts"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}
	// index.ts
	if err := renderFile("index.ts.gotmpl", filepath.Join(srcDir, "index.ts"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}
	// services per tag
	for _, s := range in.Services {
		target := filepath.Join(servicesDir, fmt.Sprintf("%s.ts", strings.ToLower(toSnakeCase(s.Tag))))
		if err := renderFile("service.ts.gotmpl", target, funcMap, map[string]any{"Client": client, "Service": s}); err != nil {
			return err
		}
	}
	// schemas (always render; may hold operation query interfaces even without models)
	if err := renderFile("schema.ts.gotmpl", filepath.Join(srcDir, "schema.ts"), funcMap, map[string]any{"IR": in}); err != nil {
		return err
	}
	// package.json
	if err := renderFile("package.json.gotmpl", filepath.Join(client.OutDir, "package.json"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}
	// tsconfig.json
	if err := renderFile("tsconfig.json.gotmpl", filepath.Join(client.OutDir, "tsconfig.json"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}
	// README.md
	if err := renderFile("README.md.gotmpl", filepath.Join(client.OutDir, "README.md"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}
	return nil
}

// renderFile renders a template file to the target path
func renderFile(templateName, targetPath string, funcMap template.FuncMap, data map[string]any) error {
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
