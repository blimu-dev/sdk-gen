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
		"dict":          func() map[string]interface{} { return make(map[string]interface{}) },
		"hasKey":        func(dict map[string]interface{}, key string) bool { _, exists := dict[key]; return exists },
		"set":           func(dict map[string]interface{}, key string, value interface{}) string { dict[key] = value; return "" },
		"quotePropName": quoteTSPropertyName,
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
	// Deduplicate model definitions to prevent duplicate enum/type generation
	deduplicatedIR := deduplicateModelDefs(in)
	if err := renderFile("schema.ts.gotmpl", filepath.Join(srcDir, "schema.ts"), funcMap, map[string]any{"IR": deduplicatedIR}); err != nil {
		return err
	}
	// package.json
	if err := renderFile("package.json.gotmpl", filepath.Join(client.OutDir, "package.json"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}
	// eslint.config.mjs
	if err := renderFile("eslint.config.mjs.gotmpl", filepath.Join(client.OutDir, "eslint.config.mjs"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}
	// .prettierrc.json
	if err := renderFile(".prettierrc.json.gotmpl", filepath.Join(client.OutDir, ".prettierrc.json"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}
	// .prettierignore
	if err := renderFile(".prettierignore.gotmpl", filepath.Join(client.OutDir, ".prettierignore"), funcMap, map[string]any{"Client": client}); err != nil {
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

// deduplicateModelDefs removes duplicate model definitions, keeping the first occurrence
// Prioritizes enum definitions over ref definitions
func deduplicateModelDefs(in ir.IR) ir.IR {
	seen := make(map[string]bool)
	var deduplicatedModelDefs []ir.IRModelDef

	// First pass: add all enum definitions
	for _, modelDef := range in.ModelDefs {
		if modelDef.Schema.Kind == ir.IRKindEnum && !seen[modelDef.Name] {
			deduplicatedModelDefs = append(deduplicatedModelDefs, modelDef)
			seen[modelDef.Name] = true
		}
	}

	// Second pass: add non-enum definitions that aren't duplicates
	for _, modelDef := range in.ModelDefs {
		if modelDef.Schema.Kind != ir.IRKindEnum && !seen[modelDef.Name] {
			deduplicatedModelDefs = append(deduplicatedModelDefs, modelDef)
			seen[modelDef.Name] = true
		}
	}

	return ir.IR{
		Services:        in.Services,
		Models:          in.Models,
		SecuritySchemes: in.SecuritySchemes,
		ModelDefs:       deduplicatedModelDefs,
	}
}

// quoteTSPropertyName quotes TypeScript property names that contain special characters
func quoteTSPropertyName(name string) string {
	// Check if the name contains characters that require quoting
	needsQuoting := false
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '$') {
			needsQuoting = true
			break
		}
	}

	// Also quote if the name starts with a number
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		needsQuoting = true
	}

	if needsQuoting {
		return `"` + name + `"`
	}
	return name
}
