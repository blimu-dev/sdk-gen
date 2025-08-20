package python

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/ir"
	"github.com/blimu-dev/sdk-gen/pkg/utils"
)

// schemaToPyTypeForService converts an IR schema to Python type string without quoting (for service files)
func schemaToPyTypeForService(s ir.IRSchema) string {
	// Base type string without nullability; append Optional later
	var t string
	switch s.Kind {
	case "string":
		if s.Format == "binary" {
			t = "bytes"
		} else {
			t = "str"
		}
	case "number":
		t = "float"
	case "integer":
		t = "int"
	case "boolean":
		t = "bool"
	case "null":
		t = "None"
	case "ref":
		if s.Ref != "" {
			// Don't quote model references in service files - they need direct imports
			t = "models." + s.Ref
		} else {
			t = "Any"
		}
	case "array":
		if s.Items != nil {
			inner := schemaToPyTypeForService(*s.Items)
			t = "List[" + inner + "]"
		} else {
			t = "List[Any]"
		}
	case "oneOf":
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToPyTypeForService(*sub))
		}
		t = "Union[" + strings.Join(parts, ", ") + "]"
	case "anyOf":
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToPyTypeForService(*sub))
		}
		t = "Union[" + strings.Join(parts, ", ") + "]"
	case "allOf":
		// Python doesn't have intersection types like TypeScript
		// We'll use the first type or Any as fallback
		if len(s.AllOf) > 0 {
			t = schemaToPyTypeForService(*s.AllOf[0])
		} else {
			t = "Any"
		}
	case "enum":
		// Use Literal for string enums, or the base type for others
		if s.EnumBase == "string" && len(s.EnumValues) > 0 {
			vals := make([]string, 0, len(s.EnumValues))
			for _, v := range s.EnumValues {
				vals = append(vals, "\""+v+"\"")
			}
			t = "Literal[" + strings.Join(vals, ", ") + "]"
		} else if len(s.EnumValues) > 0 {
			switch s.EnumBase {
			case "number":
				t = "float"
			case "integer":
				t = "int"
			case "boolean":
				t = "bool"
			default:
				t = "str"
			}
		} else {
			t = "Any"
		}
	case "object":
		if len(s.Properties) == 0 {
			t = "Dict[str, Any]"
		} else {
			// For inline objects, we'll use Dict[str, Any] as a fallback
			// In practice, these should be refs to proper models
			t = "Dict[str, Any]"
		}
	default:
		t = "Any"
	}

	// Handle nullable types with Optional
	if s.Nullable && t != "None" {
		t = "Optional[" + t + "]"
	}

	return t
}

// schemaToPyType converts an IR schema to Python type string (for models file with quoting)
func schemaToPyType(s ir.IRSchema) string {
	// Base type string without nullability; append Optional later
	var t string
	switch s.Kind {
	case "string":
		if s.Format == "binary" {
			t = "bytes"
		} else {
			t = "str"
		}
	case "number":
		t = "float"
	case "integer":
		t = "int"
	case "boolean":
		t = "bool"
	case "null":
		t = "None"
	case "ref":
		if s.Ref != "" {
			// Quote model references to handle forward references in Python
			t = "\"" + s.Ref + "\""
		} else {
			t = "Any"
		}
	case "array":
		if s.Items != nil {
			inner := schemaToPyType(*s.Items)
			t = "List[" + inner + "]"
		} else {
			t = "List[Any]"
		}
	case "oneOf":
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToPyType(*sub))
		}
		t = "Union[" + strings.Join(parts, ", ") + "]"
	case "anyOf":
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToPyType(*sub))
		}
		t = "Union[" + strings.Join(parts, ", ") + "]"
	case "allOf":
		// Python doesn't have intersection types like TypeScript
		// We'll use the first type or Any as fallback
		if len(s.AllOf) > 0 {
			t = schemaToPyType(*s.AllOf[0])
		} else {
			t = "Any"
		}
	case "enum":
		// Use Literal for string enums, or the base type for others
		if s.EnumBase == "string" && len(s.EnumValues) > 0 {
			vals := make([]string, 0, len(s.EnumValues))
			for _, v := range s.EnumValues {
				vals = append(vals, "\""+v+"\"")
			}
			t = "Literal[" + strings.Join(vals, ", ") + "]"
		} else if len(s.EnumValues) > 0 {
			switch s.EnumBase {
			case "number":
				t = "float"
			case "integer":
				t = "int"
			case "boolean":
				t = "bool"
			default:
				t = "str"
			}
		} else {
			t = "Any"
		}
	case "object":
		if len(s.Properties) == 0 {
			t = "Dict[str, Any]"
		} else {
			// For inline objects, we'll use Dict[str, Any] as a fallback
			// In practice, these should be refs to proper models
			t = "Dict[str, Any]"
		}
	default:
		t = "Any"
	}

	// Handle nullable types with Optional
	if s.Nullable && t != "None" {
		t = "Optional[" + t + "]"
	}

	return t
}

// fieldToPyType converts an IR field to Python type string with proper Optional handling
func fieldToPyType(field ir.IRField) string {
	baseType := schemaToPyType(*field.Type)
	if !field.Required && !strings.HasPrefix(baseType, "Optional[") {
		return "Optional[" + baseType + "]"
	}
	return baseType
}

// getPyDefault returns the default value for a Python field
func getPyDefault(field ir.IRField) string {
	if !field.Required {
		return "None"
	}
	// For required fields, we don't provide defaults in Pydantic models
	return ""
}

// deriveMethodName creates method names using basic REST-style heuristics
// This should only be used as a last resort when no OperationID is available
func deriveMethodName(op ir.IROperation) string {
	// If we have an OperationID, always use it (this shouldn't happen if resolveMethodName is working correctly)
	if op.OperationID != "" {
		return toSnakeCase(op.OperationID)
	}

	// Basic REST-style heuristics as fallback
	// Examples:
	// GET /brands -> list
	// POST /brands -> create
	// GET /brands/{id} -> get
	// PATCH|PUT /brands/{id} -> update
	// DELETE /brands/{id} -> delete
	path := op.Path
	hasID := strings.Contains(path, "{") && strings.Contains(path, "}")
	switch op.Method {
	case "GET":
		if hasID {
			return "get"
		}
		return "list"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
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
		out, err := exec.Command(client.OperationIDParser, op.OperationID, op.Method, op.Path).CombinedOutput()
		if err == nil {
			name := strings.TrimSpace(string(out))
			if name != "" {
				return toSnakeCase(name)
			}
		}
	}

	// Prioritize OperationID if available
	if defaultParsed != "" {
		return toSnakeCase(defaultParsed)
	}

	// Only fallback to HTTP method-based naming if no OperationID
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

// Alias functions to use centralized utilities
var toPascalCase = utils.ToPascalCase
var toCamelCase = utils.ToCamelCase
var toSnakeCase = utils.ToSnakeCase
var toKebabCase = utils.ToKebabCase

// buildPathTemplate converts OpenAPI path to Python f-string
func buildPathTemplate(op ir.IROperation) string {
	// Convert /foo/{id}/bar/{slug} -> f"/foo/{id}/bar/{slug}"
	path := op.Path
	// Find all {name} segments and replace with f-string format
	var b strings.Builder
	b.WriteString("f\"")
	for i := 0; i < len(path); i++ {
		if path[i] == '{' {
			// read name
			j := i + 1
			for j < len(path) && path[j] != '}' {
				j++
			}
			if j < len(path) {
				name := path[i+1 : j]
				b.WriteString("{")
				b.WriteString(name)
				b.WriteString("}")
				i = j
				continue
			}
		}
		b.WriteByte(path[i])
	}
	b.WriteString("\"")
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

// buildMethodSignature constructs the Python parameter list for service methods
func buildMethodSignature(op ir.IROperation, methodName string) []string {
	parts := []string{}

	// path params as positional args
	for _, p := range orderPathParams(op) {
		pyType := schemaToPyTypeForService(p.Schema)
		parts = append(parts, fmt.Sprintf("%s: %s", toSnakeCase(p.Name), pyType))
	}

	// query params as a single optional dict or individual params
	if len(op.QueryParams) > 0 {
		// For simplicity, we'll use individual optional parameters for each query param
		for _, p := range op.QueryParams {
			pyType := schemaToPyTypeForService(p.Schema)
			if !p.Required && !strings.HasPrefix(pyType, "Optional[") {
				pyType = "Optional[" + pyType + "]"
			}
			defaultVal := "None"
			if p.Required {
				// Required query params don't get default values
				parts = append(parts, fmt.Sprintf("%s: %s", toSnakeCase(p.Name), pyType))
			} else {
				parts = append(parts, fmt.Sprintf("%s: %s = %s", toSnakeCase(p.Name), pyType, defaultVal))
			}
		}
	}

	// body
	if op.RequestBody != nil {
		pyType := schemaToPyTypeForService(op.RequestBody.Schema)
		if !op.RequestBody.Required && !strings.HasPrefix(pyType, "Optional[") {
			pyType = "Optional[" + pyType + "]"
		}
		if op.RequestBody.Required {
			parts = append(parts, fmt.Sprintf("body: %s", pyType))
		} else {
			parts = append(parts, fmt.Sprintf("body: %s = None", pyType))
		}
	}

	return parts
}

// formatDocstring formats a string for use in Python docstrings
func formatDocstring(s string) string {
	if s == "" {
		return ""
	}
	// Replace any */ with *\/ to avoid breaking docstrings
	s = strings.ReplaceAll(s, "*/", "*\\/")
	// Ensure proper indentation
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, "    "+strings.TrimSpace(line))
	}
	return strings.Join(result, "\n")
}

// formatPythonComment formats a string as a Python raw string docstring for property descriptions
func formatPythonComment(s string) string {
	if s == "" {
		return ""
	}

	// Use raw string docstring format for better type hint integration
	// Escape any existing triple quotes to prevent breaking the docstring
	escaped := strings.ReplaceAll(s, `"""`, `\"\"\"`)

	// Build the raw string docstring with proper indentation (no extra spaces needed)
	return "r\"\"\"" + escaped + "\"\"\""
}
