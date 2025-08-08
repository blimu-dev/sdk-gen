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

	"github.com/viniciusdacal/sdk-gen/internal/config"
	"github.com/viniciusdacal/sdk-gen/internal/ir"
)

//go:embed templates/*
var templatesFS embed.FS

type IR = ir.IR
type IRService = ir.IRService
type IROperation = ir.IROperation
type IRParam = ir.IRParam

func Generate(client config.Client, ir IR) error {
	// Ensure directories
	srcDir := filepath.Join(client.OutDir, "src")
	servicesDir := filepath.Join(srcDir, "services")
	if err := os.MkdirAll(servicesDir, 0o755); err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"pascal":            toPascalCase,
		"camel":             toCamelCase,
		"kebab":             toKebabCase,
		"serviceName":       func(tag string) string { return toPascalCase(tag) + "Service" },
		"serviceProp":       func(tag string) string { return toCamelCase(tag) },
		"fileBase":          func(tag string) string { return strings.ToLower(toSnakeCase(tag)) },
		"methodName":        func(op IROperation) string { return deriveMethodName(op) },
		"pathTemplate":      func(op IROperation) string { return buildPathTemplate(op) },
		"pathParamsInOrder": func(op IROperation) []IRParam { return orderPathParams(op) },
		"methodSignature":   func(op IROperation) string { return methodSignature(op) },
	}

	// client.ts
	if err := renderFile("client.ts.gotmpl", filepath.Join(srcDir, "client.ts"), funcMap, map[string]any{"Client": client, "IR": ir}); err != nil {
		return err
	}
	// index.ts
	if err := renderFile("index.ts.gotmpl", filepath.Join(srcDir, "index.ts"), funcMap, map[string]any{"Client": client, "IR": ir}); err != nil {
		return err
	}
	// services per tag
	for _, s := range ir.Services {
		target := filepath.Join(servicesDir, fmt.Sprintf("%s.ts", strings.ToLower(toSnakeCase(s.Tag))))
		if err := renderFile("service.ts.gotmpl", target, funcMap, map[string]any{"Client": client, "Service": s}); err != nil {
			return err
		}
	}
	// schemas
	if len(ir.Models) > 0 {
		if err := renderFile("schema.ts.gotmpl", filepath.Join(srcDir, "schema.ts"), funcMap, map[string]any{"IR": ir}); err != nil {
			return err
		}
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

func methodSignature(op IROperation) string {
	parts := []string{}
	// path params as positional args
	for _, p := range orderPathParams(op) {
		parts = append(parts, fmt.Sprintf("%s: %s", p.Name, p.TypeTS))
	}
	// query object
	if len(op.QueryParams) > 0 {
		var b strings.Builder
		b.WriteString("query?: {")
		for i, qp := range op.QueryParams {
			if i > 0 {
				b.WriteString(", ")
			}
			if qp.Required {
				b.WriteString(fmt.Sprintf("%s: %s", qp.Name, qp.TypeTS))
			} else {
				b.WriteString(fmt.Sprintf("%s?: %s", qp.Name, qp.TypeTS))
			}
		}
		b.WriteString("}")
		parts = append(parts, b.String())
	}
	// body
	if op.RequestBody != nil {
		opt := ""
		if !op.RequestBody.Required {
			opt = "?"
		}
		parts = append(parts, fmt.Sprintf("body%s: %s", opt, op.RequestBody.TypeTS))
	}
	// init
	parts = append(parts, "init?: Omit<RequestInit, \"method\" | \"body\">")
	return strings.Join(parts, ", ")
}
