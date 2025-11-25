package typescripttypes

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
	"github.com/blimu-dev/sdk-gen/pkg/utils"
)

//go:embed templates/*
var templatesFS embed.FS

// TypeScriptTypesGenerator implements the Generator interface for TypeScript type augmentation
type TypeScriptTypesGenerator struct{}

// NewTypeScriptTypesGenerator creates a new TypeScript types generator
func NewTypeScriptTypesGenerator() *TypeScriptTypesGenerator {
	return &TypeScriptTypesGenerator{}
}

// GetType returns the generator type identifier
func (g *TypeScriptTypesGenerator) GetType() string {
	return "typescript-types"
}

// Generate creates a TypeScript type augmentation file from the given configuration and IR
func (g *TypeScriptTypesGenerator) Generate(client config.Client, in ir.IR) error {
	// Ensure output directory exists
	if err := os.MkdirAll(client.OutDir, 0o755); err != nil {
		return err
	}

	// Get type augmentation options with defaults
	opts := client.TypeAugmentationOptions
	if opts.ModuleName == "" {
		opts.ModuleName = client.PackageName
	}
	if opts.Namespace == "" {
		opts.Namespace = "Schema"
	}
	if opts.OutputFileName == "" {
		opts.OutputFileName = client.PackageName + ".d.ts"
	}

	funcMap := template.FuncMap{
		"pascal":      toPascalCase,
		"camel":       toCamelCase,
		"kebab":       toKebabCase,
		"serviceName": func(tag string) string { return toPascalCase(tag) + "Service" },
		"serviceProp": func(tag string) string { return toCamelCase(tag) },
		"methodName":  func(op ir.IROperation) string { return resolveMethodName(client, op) },
		"pathTemplate":      func(op ir.IROperation) string { return buildPathTemplate(op) },
		"pathParamsInOrder": func(op ir.IROperation) []ir.IRParam { return orderPathParams(op) },
		"methodSignature":   func(op ir.IROperation) []string { return buildMethodSignature(op, resolveMethodName(client, op)) },
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

	// Generate the type augmentation file
	outputFile := filepath.Join(client.OutDir, opts.OutputFileName)
	if err := renderFile("types.d.ts.gotmpl", outputFile, funcMap, map[string]any{
		"Client":  client,
		"IR":      in,
		"Options": opts,
	}); err != nil {
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

// Alias functions to use centralized utilities
var toPascalCase = utils.ToPascalCase
var toCamelCase = utils.ToCamelCase
var toKebabCase = utils.ToKebabCase

// schemaToTSType converts an IR schema to TypeScript type string
func schemaToTSType(s ir.IRSchema) string {
	// Base type string without nullability; append null later
	var t string
	switch s.Kind {
	case ir.IRKindString:
		if s.Format == "binary" {
			t = "Blob"
		} else {
			t = "string"
		}
	case ir.IRKindNumber, ir.IRKindInteger:
		t = "number"
	case ir.IRKindBoolean:
		t = "boolean"
	case ir.IRKindNull:
		t = "null"
	case ir.IRKindRef:
		if s.Ref != "" {
			t = "Schema." + s.Ref
		} else {
			t = "unknown"
		}
	case ir.IRKindArray:
		if s.Items != nil {
			inner := schemaToTSType(*s.Items)
			// Wrap unions/intersections in parentheses inside Array<>
			if strings.Contains(inner, " | ") || strings.Contains(inner, " & ") {
				inner = "(" + inner + ")"
			}
			t = "Array<" + inner + ">"
		} else {
			t = "Array<unknown>"
		}
	case ir.IRKindOneOf:
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToTSType(*sub))
		}
		t = strings.Join(parts, " | ")
	case ir.IRKindAnyOf:
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToTSType(*sub))
		}
		t = strings.Join(parts, " | ")
	case ir.IRKindAllOf:
		parts := make([]string, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			parts = append(parts, schemaToTSType(*sub))
		}
		t = strings.Join(parts, " & ")
	case ir.IRKindEnum:
		// Prefer using name via Ref in properties; for safety, inline a union here
		if len(s.EnumValues) > 0 {
			vals := make([]string, 0, len(s.EnumValues))
			switch s.EnumBase {
			case ir.IRKindNumber, ir.IRKindInteger:
				for _, v := range s.EnumValues {
					vals = append(vals, v)
				}
			case ir.IRKindBoolean:
				for _, v := range s.EnumValues {
					if v == "true" || v == "false" {
						vals = append(vals, v)
					} else {
						vals = append(vals, "\""+v+"\"")
					}
				}
			default:
				for _, v := range s.EnumValues {
					vals = append(vals, "\""+v+"\"")
				}
			}
			t = strings.Join(vals, " | ")
		} else {
			t = "unknown"
		}
	case ir.IRKindObject:
		if len(s.Properties) == 0 {
			t = "Record<string, unknown>"
		} else {
			// Inline object shape for rare cases; nested ones should be refs
			parts := make([]string, 0, len(s.Properties))
			for _, f := range s.Properties {
				ft := schemaToTSType(*f.Type)
				if f.Required {
					parts = append(parts, f.Name+": "+ft)
				} else {
					parts = append(parts, f.Name+"?: "+ft)
				}
			}
			t = "{" + strings.Join(parts, "; ") + "}"
		}
	default:
		t = "unknown"
	}
	if s.Nullable && t != "null" {
		t += " | null"
	}
	return t
}

// deriveMethodName creates method names using basic REST-style heuristics
func deriveMethodName(op ir.IROperation) string {
	// Basic REST-style heuristics
	// Examples:
	// GET /brands -> list
	// POST /brands -> create
	// GET /brands/{id} -> retrieve
	// PATCH|PUT /brands/{id} -> update
	// DELETE /brands/{id} -> delete
	path := op.Path
	hasID := strings.Contains(path, "{") && strings.Contains(path, "}")
	switch op.Method {
	case "GET":
		if hasID {
			return "read"
		}
		return "list"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		if op.OperationID != "" {
			return toCamelCase(op.OperationID)
		}
		// fallback
		return strings.ToLower(op.Method)
	}
}

// resolveMethodName chooses final method name using optional parser, then operationId, then heuristic
func resolveMethodName(client config.Client, op ir.IROperation) string {
	// Default parse of operationId
	defaultParsed := defaultParseOperationID(op.OperationID)
	// try external parser (given original opId/method/path)
	if client.OperationIDParser != "" {
		// Note: We can't execute external commands here in type augmentation
		// Just use the default parsed name
	}
	if defaultParsed != "" {
		return toCamelCase(defaultParsed)
	}
	return deriveMethodName(op)
}

// defaultParseOperationID implements built-in parsing:
// - If opID contains "Controller_", return the substring after it
// - Otherwise return opID as-is
func defaultParseOperationID(opID string) string {
	if opID == "" {
		return ""
	}
	// Strip any prefix up to and including "Controller_"
	if idx := strings.Index(opID, "Controller_"); idx >= 0 {
		tail := opID[idx+len("Controller_"):]
		return tail
	}
	return opID
}

// buildPathTemplate converts OpenAPI path to TypeScript template literal
func buildPathTemplate(op ir.IROperation) string {
	// Convert /foo/{id}/bar/{slug} -> `/foo/${path.id}/bar/${path.slug}`
	path := op.Path
	// Find all {name} segments
	var b strings.Builder
	b.WriteString("`")
	for i := 0; i < len(path); i++ {
		if path[i] == '{' {
			// read name
			j := i + 1
			for j < len(path) && path[j] != '}' {
				j++
			}
			if j < len(path) {
				name := path[i+1 : j]
				b.WriteString("${encodeURIComponent(")
				b.WriteString(name)
				b.WriteString(")}")
				i = j
				continue
			}
		}
		b.WriteByte(path[i])
	}
	b.WriteString("`")
	return b.String()
}

// orderPathParams extracts path parameter order as they appear in the path
func orderPathParams(op ir.IROperation) []ir.IRParam {
	ordered := []ir.IRParam{}
	index := map[string]int{}
	for i, p := range op.PathParams {
		index[p.Name] = i
	}
	path := op.Path
	for i := 0; i < len(path); i++ {
		if path[i] == '{' {
			j := i + 1
			for j < len(path) && path[j] != '}' {
				j++
			}
			if j < len(path) {
				name := path[i+1 : j]
				if idx, ok := index[name]; ok {
					ordered = append(ordered, op.PathParams[idx])
				}
				i = j
				continue
			}
		}
	}
	return ordered
}

// buildMethodSignature constructs the TS parameter list, using the provided methodName for query type name
func buildMethodSignature(op ir.IROperation, methodName string) []string {
	parts := []string{}
	// path params as positional args
	for _, p := range orderPathParams(op) {
		parts = append(parts, fmt.Sprintf("%s: %s", p.Name, schemaToTSType(p.Schema)))
	}
	// query object
	if len(op.QueryParams) > 0 {
		// Reference named interface defined in schema.ts
		queryType := toPascalCase(op.Tag) + toPascalCase(methodName) + "Query"
		parts = append(parts, fmt.Sprintf("query?: Schema.%s", queryType))
	}
	// body
	if op.RequestBody != nil {
		opt := ""
		if !op.RequestBody.Required {
			opt = "?"
		}
		parts = append(parts, fmt.Sprintf("body%s: %s", opt, schemaToTSType(op.RequestBody.Schema)))
	}
	// init
	parts = append(parts, "init?: Omit<RequestInit, \"method\" | \"body\">")

	return parts
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
