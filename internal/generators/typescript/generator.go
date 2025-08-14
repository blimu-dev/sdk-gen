package typescript

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/blimu-dev/sdk-gen/internal/config"
	"github.com/blimu-dev/sdk-gen/internal/ir"
)

//go:embed templates/*
var templatesFS embed.FS

type IR = ir.IR
type IRService = ir.IRService
type IROperation = ir.IROperation
type IRParam = ir.IRParam

func Generate(client config.Client, in IR) error {
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
		"methodName":  func(op IROperation) string { return resolveMethodName(client, op) },
		"queryTypeName": func(op IROperation) string {
			return toPascalCase(op.Tag) + toPascalCase(resolveMethodName(client, op)) + "Query"
		},
		"pathTemplate":      func(op IROperation) string { return buildPathTemplate(op) },
		"queryKeyBase":      func(op IROperation) string { return buildQueryKeyBase(op) },
		"pathParamsInOrder": func(op IROperation) []IRParam { return orderPathParams(op) },
		"methodSignature":   func(op IROperation) []string { return buildMethodSignature(op, resolveMethodName(client, op)) },
		"methodSignatureNoInit": func(op IROperation) []string {
			parts := buildMethodSignature(op, resolveMethodName(client, op))
			if len(parts) > 0 {
				return parts[:len(parts)-1]
			}
			return parts
		},
		"queryKeyArgs": func(op IROperation) []string { return queryKeyArgs(op) },
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
	return nil
}

func schemaToTSType(s ir.IRSchema) string {
	// Base type string without nullability; append null later
	var t string
	switch s.Kind {
	case "string":
		if s.Format == "binary" {
			t = "Blob"
		} else {
			t = "string"
		}
	case "number", "integer":
		t = "number"
	case "boolean":
		t = "boolean"
	case "null":
		t = "null"
	case "ref":
		if s.Ref != "" {
			t = "Schema." + s.Ref
		} else {
			t = "unknown"
		}
	case "array":
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
	case "oneOf":
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToTSType(*sub))
		}
		t = strings.Join(parts, " | ")
	case "anyOf":
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToTSType(*sub))
		}
		t = strings.Join(parts, " | ")
	case "allOf":
		parts := make([]string, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			parts = append(parts, schemaToTSType(*sub))
		}
		t = strings.Join(parts, " & ")
	case "enum":
		// Prefer using name via Ref in properties; for safety, inline a union here
		if len(s.EnumValues) > 0 {
			vals := make([]string, 0, len(s.EnumValues))
			switch s.EnumBase {
			case "number", "integer":
				for _, v := range s.EnumValues {
					vals = append(vals, v)
				}
			case "boolean":
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
	case "object":
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

// schemaToLocalTSType renders types for use inside schema.ts (no Schema. prefix for refs)
func schemaToLocalTSType(s ir.IRSchema) string {
	var t string
	switch s.Kind {
	case "string":
		if s.Format == "binary" {
			t = "Blob"
		} else {
			t = "string"
		}
	case "number", "integer":
		t = "number"
	case "boolean":
		t = "boolean"
	case "null":
		t = "null"
	case "ref":
		if s.Ref != "" {
			t = s.Ref
		} else {
			t = "unknown"
		}
	case "array":
		if s.Items != nil {
			inner := schemaToLocalTSType(*s.Items)
			if strings.Contains(inner, " | ") || strings.Contains(inner, " & ") {
				inner = "(" + inner + ")"
			}
			t = "Array<" + inner + ">"
		} else {
			t = "Array<unknown>"
		}
	case "oneOf":
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToLocalTSType(*sub))
		}
		t = strings.Join(parts, " | ")
	case "anyOf":
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToLocalTSType(*sub))
		}
		t = strings.Join(parts, " | ")
	case "allOf":
		parts := make([]string, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			parts = append(parts, schemaToLocalTSType(*sub))
		}
		t = strings.Join(parts, " & ")
	case "enum":
		if len(s.EnumValues) > 0 {
			vals := make([]string, 0, len(s.EnumValues))
			switch s.EnumBase {
			case "number", "integer":
				for _, v := range s.EnumValues {
					vals = append(vals, v)
				}
			case "boolean":
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
	case "object":
		if len(s.Properties) == 0 {
			t = "Record<string, unknown>"
		} else {
			parts := make([]string, 0, len(s.Properties))
			for _, f := range s.Properties {
				ft := schemaToLocalTSType(*f.Type)
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
func renderFile(tmplName, outPath string, fm template.FuncMap, data any) error {
	t, err := template.New(tmplName).Funcs(sprig.TxtFuncMap()).Funcs(fm).ParseFS(templatesFS, "templates/"+tmplName)
	if err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}

func deriveMethodName(op IROperation) string {
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
			return "retrieve"
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
func resolveMethodName(client config.Client, op IROperation) string {
	// Default parse of operationId
	defaultParsed := defaultParseOperationID(op.OperationID)
	// try external parser (given original opId/method/path)
	if client.OperationIDParser != "" {
		out, err := exec.Command(client.OperationIDParser, op.OperationID, op.Method, op.Path).CombinedOutput()
		if err == nil {
			name := strings.TrimSpace(string(out))
			if name != "" {
				return toCamelCase(name)
			}
		}
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

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

func toPascalCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := nonAlnum.Split(s, -1)
	b := strings.Builder{}
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(strings.ToLower(p[1:]))
		}
	}
	return b.String()
}

func toCamelCase(s string) string {
	p := toPascalCase(s)
	if p == "" {
		return ""
	}
	return strings.ToLower(p[:1]) + p[1:]
}

func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	s = nonAlnum.ReplaceAllString(s, " ")
	fields := strings.Fields(s)
	for i := range fields {
		fields[i] = strings.ToLower(fields[i])
	}
	return strings.Join(fields, "_")
}

func toKebabCase(s string) string {
	s = strings.TrimSpace(s)
	s = nonAlnum.ReplaceAllString(s, " ")
	fields := strings.Fields(s)
	for i := range fields {
		fields[i] = strings.ToLower(fields[i])
	}
	return strings.Join(fields, "-")
}

func buildPathTemplate(op IROperation) string {
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

// buildQueryKeyBase returns a TS string literal for the base of a react-query key.
// Example: "/v1/organizations/{id}" -> "'v1/organizations'"
func buildQueryKeyBase(op IROperation) string {
	path := op.Path
	// Split by '/'; skip parameter placeholders like {id}
	parts := strings.Split(path, "/")
	baseParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" { // leading slash
			continue
		}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			continue
		}
		baseParts = append(baseParts, p)
	}
	base := strings.Join(baseParts, "/")
	return "'" + base + "'"
}

func orderPathParams(op IROperation) []IRParam {
	// Extract path parameter order as they appear in the path
	ordered := []IRParam{}
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
func buildMethodSignature(op IROperation, methodName string) []string {
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

// queryKeyArgs returns the parameter names (no types) in the same order as the method parameters,
// excluding the trailing init parameter. Includes:
// - path params in path order
// - 'query' when there are query params
// - 'body' when there's a request body
func queryKeyArgs(op IROperation) []string {
	out := []string{}
	for _, p := range orderPathParams(op) {
		out = append(out, p.Name)
	}
	if len(op.QueryParams) > 0 {
		out = append(out, "query")
	}
	if op.RequestBody != nil {
		out = append(out, "body")
	}
	return out
}
