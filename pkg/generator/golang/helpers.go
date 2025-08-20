package golang

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/ir"
	"github.com/blimu-dev/sdk-gen/pkg/utils"
)

// schemaToGoType converts an IR schema to Go type string
func schemaToGoType(x any) string {
	switch v := x.(type) {
	case ir.IRSchema:
		return schemaToGoTypeImpl(v)
	case *ir.IRSchema:
		if v != nil {
			return schemaToGoTypeImpl(*v)
		}
		return "interface{}"
	default:
		return "interface{}"
	}
}

func schemaToGoTypeImpl(s ir.IRSchema) string {
	var t string
	switch s.Kind {
	case "string":
		if s.Format == "binary" {
			t = "[]byte"
		} else {
			t = "string"
		}
	case "number":
		t = "float64"
	case "integer":
		t = "int64"
	case "boolean":
		t = "bool"
	case "null":
		t = "interface{}"
	case "ref":
		if s.Ref != "" {
			t = toPascalCase(s.Ref)
		} else {
			t = "interface{}"
		}
	case "array":
		if s.Items != nil {
			inner := schemaToGoTypeImpl(*s.Items)
			t = "[]" + inner
		} else {
			t = "[]interface{}"
		}
	case "oneOf", "anyOf":
		// For Go, we'll use interface{} for union types
		// In a more sophisticated implementation, we could generate type-safe unions
		t = "interface{}"
	case "allOf":
		// For Go, we'll use interface{} for intersection types
		// In a more sophisticated implementation, we could generate embedded structs
		t = "interface{}"
	case "enum":
		// Use string for enums, could be enhanced to use custom types
		t = "string"
	case "object":
		if len(s.Properties) > 0 {
			// For inline objects, we'll use map[string]interface{}
			// In a more sophisticated implementation, we could generate inline structs
			t = "map[string]interface{}"
		} else {
			t = "map[string]interface{}"
		}
	default:
		t = "interface{}"
	}

	// Handle nullable types
	if s.Nullable {
		t = "*" + t
	}

	return t
}

// Alias functions to use centralized utilities (advanced versions for better camelCase handling)
var toPascalCase = utils.ToPascalCaseAdvanced
var toCamelCase = utils.ToCamelCaseAdvanced
var toSnakeCase = utils.ToSnakeCaseAdvanced
var toKebabCase = utils.ToKebabCaseAdvanced

// formatGoComment formats a string as a proper Go comment, handling multiline descriptions
func formatGoComment(s string) string {
	if s == "" {
		return ""
	}

	// Split into lines and prefix each with //
	lines := strings.Split(s, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			result = append(result, "//")
		} else {
			result = append(result, "// "+line)
		}
	}

	return strings.Join(result, "\n")
}

// ResolveMethodName chooses final method name using optional parser, then operationId, then heuristic
func ResolveMethodName(client config.Client, op ir.IROperation) string {
	// Default parse of operationId
	defaultParsed := defaultParseOperationID(op.OperationID)

	// Try external parser (given original opId/method/path)
	if client.OperationIDParser != "" {
		// Note: In a real implementation, you'd want to execute the external parser
		// For now, we'll skip this and use the default parsing
	}

	if defaultParsed != "" {
		return toPascalCase(defaultParsed)
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

// deriveMethodName creates method names using basic REST-style heuristics
func deriveMethodName(op ir.IROperation) string {
	// Basic REST-style heuristics
	// Examples:
	// GET /brands -> List
	// POST /brands -> Create
	// GET /brands/{id} -> Get
	// PATCH|PUT /brands/{id} -> Update
	// DELETE /brands/{id} -> Delete
	path := op.Path
	hasID := strings.Contains(path, "{") && strings.Contains(path, "}")
	switch strings.ToUpper(op.Method) {
	case "GET":
		if hasID {
			return "Get"
		}
		return "List"
	case "POST":
		return "Create"
	case "PUT", "PATCH":
		return "Update"
	case "DELETE":
		return "Delete"
	default:
		if op.OperationID != "" {
			return toPascalCase(op.OperationID)
		}
		// fallback
		return toPascalCase(op.Method)
	}
}

// buildPathTemplate builds a path template for Go string formatting
func buildPathTemplate(op ir.IROperation) string {
	path := op.Path

	// Replace OpenAPI path parameters {param} with Go format specifiers
	for _, param := range op.PathParams {
		placeholder := "{" + param.Name + "}"
		path = strings.ReplaceAll(path, placeholder, "%v")
	}

	return fmt.Sprintf(`"%s"`, path)
}

// orderPathParams returns path parameters in the order they appear in the path
func orderPathParams(op ir.IROperation) []ir.IRParam {
	if len(op.PathParams) == 0 {
		return nil
	}

	// Create a map for quick lookup
	paramMap := make(map[string]ir.IRParam)
	for _, param := range op.PathParams {
		paramMap[param.Name] = param
	}

	// Extract parameter names from path in order
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(op.Path, -1)

	var ordered []ir.IRParam
	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			if param, exists := paramMap[paramName]; exists {
				ordered = append(ordered, param)
			}
		}
	}

	return ordered
}

// buildMethodSignature builds the method signature for a Go method
func buildMethodSignature(client config.Client, op ir.IROperation, methodName string) string {
	var params []string

	// Context parameter (always first)
	params = append(params, "ctx context.Context")

	// Path parameters
	for _, param := range orderPathParams(op) {
		goType := schemaToGoType(param.Schema)
		params = append(params, fmt.Sprintf("%s %s", toCamelCase(param.Name), goType))
	}

	// Query parameters (as a struct)
	if len(op.QueryParams) > 0 {
		// Use proper naming that includes the operation tag/service name
		queryTypeName := toPascalCase(op.Tag) + strings.TrimSuffix(ResolveMethodName(client, op), "WithContext") + "Query"
		params = append(params, fmt.Sprintf("query *%s", queryTypeName))
	}

	// Request body
	if op.RequestBody != nil {
		goType := schemaToGoType(op.RequestBody.Schema)
		params = append(params, fmt.Sprintf("body %s", goType))
	}

	// Return type
	responseType := schemaToGoType(op.Response.Schema)

	signature := fmt.Sprintf("%s(%s) (%s, error)", methodName, strings.Join(params, ", "), responseType)
	return signature
}

// buildMethodSignatureNoContext builds the method signature without context parameter
func buildMethodSignatureNoContext(client config.Client, op ir.IROperation, methodName string) string {
	var params []string

	// Path parameters (no context parameter)
	for _, param := range orderPathParams(op) {
		goType := schemaToGoType(param.Schema)
		params = append(params, fmt.Sprintf("%s %s", toCamelCase(param.Name), goType))
	}

	// Query parameters (as a struct)
	if len(op.QueryParams) > 0 {
		// Use proper naming that includes the operation tag/service name
		queryTypeName := toPascalCase(op.Tag) + ResolveMethodName(client, op) + "Query"
		params = append(params, fmt.Sprintf("query *%s", queryTypeName))
	}

	// Request body
	if op.RequestBody != nil {
		goType := schemaToGoType(op.RequestBody.Schema)
		params = append(params, fmt.Sprintf("body %s", goType))
	}

	// Return type
	responseType := schemaToGoType(op.Response.Schema)

	signature := fmt.Sprintf("%s(%s) (%s, error)", methodName, strings.Join(params, ", "), responseType)
	return signature
}

// sanitizePackageName ensures the package name is valid for Go
func sanitizePackageName(name string) string {
	// Extract the last part of the package name if it looks like a module path
	parts := strings.Split(name, "/")
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	// Convert to lowercase and replace invalid characters
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(name, "")

	// Ensure it doesn't start with a number
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "pkg" + name
	}

	// Ensure it's not empty
	if name == "" {
		name = "client"
	}

	return name
}
