package python

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

// PythonGenerator implements the Generator interface for Python
type PythonGenerator struct{}

// NewPythonGenerator creates a new Python generator
func NewPythonGenerator() *PythonGenerator {
	return &PythonGenerator{}
}

// GetType returns the generator type identifier
func (g *PythonGenerator) GetType() string {
	return "python"
}

// Generate creates a Python SDK from the given configuration and IR
func (g *PythonGenerator) Generate(client config.Client, in ir.IR) error {
	// Ensure directories
	srcDir := filepath.Join(client.OutDir, client.PackageName)
	servicesDir := filepath.Join(srcDir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"snake":             toSnakeCase,
		"pascal":            toPascalCase,
		"camel":             toCamelCase,
		"kebab":             toKebabCase,
		"serviceName":       func(tag string) string { return toPascalCase(tag) + "Service" },
		"serviceVar":        func(tag string) string { return toSnakeCase(tag) },
		"fileBase":          func(tag string) string { return strings.ToLower(toSnakeCase(tag)) },
		"methodName":        func(op ir.IROperation) string { return resolveMethodName(client, op) },
		"pathTemplate":      func(op ir.IROperation) string { return buildPathTemplate(op) },
		"pathParamsInOrder": func(op ir.IROperation) []ir.IRParam { return orderPathParams(op) },
		"methodSignature":   func(op ir.IROperation) []string { return buildMethodSignature(op, resolveMethodName(client, op)) },
		"pyType": func(x any) string {
			switch v := x.(type) {
			case ir.IRSchema:
				return schemaToPyType(v)
			case *ir.IRSchema:
				if v != nil {
					return schemaToPyType(*v)
				}
				return "Any"
			default:
				return "Any"
			}
		},
		"pyTypeForService": func(x any) string {
			switch v := x.(type) {
			case ir.IRSchema:
				return schemaToPyTypeForService(v)
			case *ir.IRSchema:
				if v != nil {
					return schemaToPyTypeForService(*v)
				}
				return "Any"
			default:
				return "Any"
			}
		},
		"pyFieldType":    func(field ir.IRField) string { return fieldToPyType(field) },
		"isOptional":     func(field ir.IRField) bool { return !field.Required },
		"hasPathParams":  func(op ir.IROperation) bool { return len(op.PathParams) > 0 },
		"hasQueryParams": func(op ir.IROperation) bool { return len(op.QueryParams) > 0 },
		"hasRequestBody": func(op ir.IROperation) bool { return op.RequestBody != nil },
		"requestBodyRequired": func(op ir.IROperation) bool {
			return op.RequestBody != nil && op.RequestBody.Required
		},
		"stripSchemaNs":       func(s string) string { return strings.ReplaceAll(s, "Schema.", "") },
		"reMatch":             func(pattern, s string) bool { r := regexp.MustCompile(pattern); return r.MatchString(s) },
		"docstring":           func(s string) string { return formatDocstring(s) },
		"pyDefault":           func(field ir.IRField) string { return getPyDefault(field) },
		"httpMethodUpper":     func(method string) string { return strings.ToUpper(method) },
		"isStringEnum":        func(schema ir.IRSchema) bool { return schema.Kind == "enum" && schema.EnumBase == "string" },
		"enumValues":          func(schema ir.IRSchema) []string { return schema.EnumValues },
		"formatPythonComment": func(s string) string { return formatPythonComment(s) },
	}

	// Merge sprig functions
	for k, v := range sprig.FuncMap() {
		funcMap[k] = v
	}

	// client.py
	if err := renderFile("client.py.gotmpl", filepath.Join(srcDir, "client.py"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// __init__.py
	if err := renderFile("__init__.py.gotmpl", filepath.Join(srcDir, "__init__.py"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// models.py
	if err := renderFile("models.py.gotmpl", filepath.Join(srcDir, "models.py"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// services per tag
	for _, s := range in.Services {
		target := filepath.Join(servicesDir, fmt.Sprintf("%s.py", strings.ToLower(toSnakeCase(s.Tag))))
		if err := renderFile("service.py.gotmpl", target, funcMap, map[string]any{"Client": client, "Service": s}); err != nil {
			return err
		}
	}

	// services/__init__.py
	if err := renderFile("services_init.py.gotmpl", filepath.Join(servicesDir, "__init__.py"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// pyproject.toml
	if err := renderFile("pyproject.toml.gotmpl", filepath.Join(client.OutDir, "pyproject.toml"), funcMap, map[string]any{"Client": client}); err != nil {
		return err
	}

	// README.md
	if err := renderFile("README.md.gotmpl", filepath.Join(client.OutDir, "README.md"), funcMap, map[string]any{"Client": client, "IR": in}); err != nil {
		return err
	}

	// py.typed (for type hints)
	if err := renderFile("py.typed.gotmpl", filepath.Join(srcDir, "py.typed"), funcMap, map[string]any{}); err != nil {
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
