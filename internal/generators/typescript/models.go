package typescript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/viniciusdacal/sdk-gen/internal/ir"
)

// CollectModels builds TypeScript model declarations for components.schemas,
// generating named interfaces for nested inline objects using the naming scheme:
// Parent + _ + Pascal(Property) and suffixing array items with _Item.
func CollectModels(doc *openapi3.T) []ir.IRModel {
	out := []ir.IRModel{}
	seen := map[string]struct{}{}
	if doc.Components == nil || doc.Components.Schemas == nil {
		return out
	}
	// Deterministic order
	names := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sr := doc.Components.Schemas[name]
		seen[name] = struct{}{}
		// If top-level component is a string enum, emit const + type alias instead of interface
		if sr != nil && sr.Value != nil && sr.Value.Type != nil && sr.Value.Type.Is(openapi3.TypeString) && len(sr.Value.Enum) > 0 {
			_ = ensureEnumDecl(name, "", false, sr.Value.Enum, &out, seen)
			continue
		}
		tsBody := schemaToTSForSchemaFile(doc, sr, name, "", false, &out, seen)
		decl := fmt.Sprintf("export interface %s %s", name, toInterfaceShape(tsBody))
		out = append(out, ir.IRModel{Name: name, Decl: decl})
	}
	return out
}

func toInterfaceShape(ts string) string {
	trimmed := strings.TrimSpace(ts)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return trimmed
	}
	return fmt.Sprintf("{ [key: string]: %s }", trimmed)
}

// schemaToTSForSchemaFile returns a TS type string suitable for use inside schema.ts (no Schema. prefix),
// and appends additional IRModel entries to out for any inline object schemas encountered.
func schemaToTSForSchemaFile(doc *openapi3.T, sr *openapi3.SchemaRef, parentName, propName string, isArrayItem bool, out *[]ir.IRModel, seen map[string]struct{}) string {
	if sr == nil {
		return "unknown"
	}
	if sr.Ref != "" {
		if strings.HasPrefix(sr.Ref, "#/components/schemas/") {
			name := strings.TrimPrefix(sr.Ref, "#/components/schemas/")
			return name
		}
		parts := strings.Split(sr.Ref, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			if name != "" {
				return name
			}
		}
		return "unknown"
	}
	if sr.Value == nil {
		return "unknown"
	}
	s := sr.Value

	if len(s.OneOf) > 0 {
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToTSForSchemaFile(doc, sub, parentName, propName, isArrayItem, out, seen))
		}
		t := strings.Join(parts, " | ")
		if s.Nullable {
			t += " | null"
		}
		return t
	}
	if len(s.AnyOf) > 0 {
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToTSForSchemaFile(doc, sub, parentName, propName, isArrayItem, out, seen))
		}
		t := strings.Join(parts, " | ")
		if s.Nullable {
			t += " | null"
		}
		return t
	}
	if len(s.AllOf) > 0 {
		parts := make([]string, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			parts = append(parts, schemaToTSForSchemaFile(doc, sub, parentName, propName, isArrayItem, out, seen))
		}
		t := strings.Join(parts, " & ")
		if s.Nullable {
			t += " | null"
		}
		return t
	}

	var t string
	if s.Type != nil && s.Type.Is(openapi3.TypeString) {
		if len(s.Enum) > 0 {
			enumName := ensureEnumDecl(parentName, propName, isArrayItem, s.Enum, out, seen)
			t = enumName
		} else if s.Format == "binary" {
			t = "Blob"
		} else {
			t = "string"
		}
	} else if s.Type != nil && (s.Type.Is(openapi3.TypeInteger) || s.Type.Is(openapi3.TypeNumber)) {
		t = "number"
	} else if s.Type != nil && s.Type.Is(openapi3.TypeBoolean) {
		t = "boolean"
	} else if s.Type != nil && s.Type.Is(openapi3.TypeArray) {
		if s.Items != nil && s.Items.Value != nil && s.Items.Value.Type != nil && s.Items.Value.Type.Is(openapi3.TypeObject) && len(s.Items.Value.Properties) > 0 {
			base := parentName
			if propName != "" {
				base = base + "_" + toPascalCase(propName)
			}
			itemName := base + "_Item"
			if _, ok := seen[itemName]; !ok {
				body := buildObjectBodyForSchemaFile(doc, s.Items.Value, itemName, out, seen)
				decl := fmt.Sprintf("export interface %s %s", itemName, toInterfaceShape(body))
				*out = append(*out, ir.IRModel{Name: itemName, Decl: decl})
				seen[itemName] = struct{}{}
			}
			t = "Array<" + itemName + ">"
		} else {
			inner := schemaToTSForSchemaFile(doc, s.Items, parentName, propName, true, out, seen)
			t = "Array<" + inner + ">"
		}
	} else if s.Type != nil && s.Type.Is(openapi3.TypeObject) {
		if propName == "" && !isArrayItem {
			t = buildObjectBodyForSchemaFile(doc, s, parentName, out, seen)
		} else {
			base := parentName
			if propName != "" {
				base = base + "_" + toPascalCase(propName)
			}
			nestedName := base
			if _, ok := seen[nestedName]; !ok {
				body := buildObjectBodyForSchemaFile(doc, s, nestedName, out, seen)
				decl := fmt.Sprintf("export interface %s %s", nestedName, toInterfaceShape(body))
				*out = append(*out, ir.IRModel{Name: nestedName, Decl: decl})
				seen[nestedName] = struct{}{}
			}
			t = nestedName
		}
	} else {
		t = "unknown"
	}
	if s.Nullable {
		t = t + " | null"
	}
	return t
}

func buildObjectBodyForSchemaFile(doc *openapi3.T, s *openapi3.Schema, parentName string, out *[]ir.IRModel, seen map[string]struct{}) string {
	if len(s.Properties) == 0 {
		return "Record<string, unknown>"
	}
	propNames := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)
	parts := make([]string, 0, len(propNames))
	for _, name := range propNames {
		prop := s.Properties[name]
		required := false
		for _, r := range s.Required {
			if r == name {
				required = true
				break
			}
		}
		propTS := schemaToTSForSchemaFile(doc, prop, parentName, name, false, out, seen)
		if !required {
			parts = append(parts, fmt.Sprintf("%s?: %s", name, propTS))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", name, propTS))
		}
	}
	return "{" + strings.Join(parts, "; ") + "}"
}

// ensureEnumDecl emits a const object and type alias for a string enum if not already added.
// Returns the type alias name to use.
func ensureEnumDecl(parentName, propName string, isArrayItem bool, enumVals []interface{}, out *[]ir.IRModel, seen map[string]struct{}) string {
	base := parentName
	if propName != "" {
		base = base + "_" + toPascalCase(propName)
	}
	if isArrayItem {
		base = base + "_Item"
	}
	enumName := base
	if _, ok := seen[enumName]; ok {
		return enumName
	}
	// Build const object
	// use quoted keys to be safe, values as quoted strings
	var b strings.Builder
	b.WriteString("export const ")
	b.WriteString(enumName)
	b.WriteString(" = {\n")
	for i, ev := range enumVals {
		if i > 0 {
			b.WriteString("\n")
		}
		v := fmt.Sprint(ev)
		b.WriteString("  \"")
		b.WriteString(v)
		b.WriteString("\": \"")
		b.WriteString(v)
		b.WriteString("\",")
	}
	b.WriteString("\n} as const\n")
	*out = append(*out, ir.IRModel{Name: enumName, Decl: b.String()})

	// Type alias using typeof const object
	typeDecl := fmt.Sprintf("export type %s = (typeof %s)[keyof typeof %s]; \n", enumName, enumName, enumName)
	*out = append(*out, ir.IRModel{Name: enumName + "_type", Decl: typeDecl})
	seen[enumName] = struct{}{}
	return enumName
}
