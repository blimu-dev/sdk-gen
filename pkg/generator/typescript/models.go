package typescript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/blimu-dev/sdk-gen/pkg/ir"
	"github.com/getkin/kin-openapi/openapi3"
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
		// Skip top-level enums here. Enums are rendered via templates from IR.ModelDefs.
		if sr != nil && sr.Value != nil && len(sr.Value.Enum) > 0 {
			continue
		}
		tsBody := schemaToTSForSchemaFile(doc, sr, name, "", false, &out, seen)
		decl := fmt.Sprintf("export interface %s %s", name, toInterfaceShape(tsBody))
		out = append(out, ir.IRModel{Name: name, Decl: decl})
	}
	return out
}

// toInterfaceShape converts a TypeScript type to an interface shape
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
	if s.Not != nil {
		// TypeScript doesn't have a direct "not" type, so we'll use unknown
		t := "unknown"
		if s.Nullable {
			t += " | null"
		}
		return t
	}

	// Enum: create named type when in a nested context
	if len(s.Enum) > 0 {
		baseName := parentName
		if propName != "" {
			baseName = baseName + "_" + toPascalCase(propName)
		}
		if isArrayItem {
			baseName = baseName + "_Item"
		}
		if _, ok := seen[baseName]; !ok {
			vals := make([]string, 0, len(s.Enum))
			// Determine if this is a string enum or numeric enum
			isStringEnum := true
			if s.Type != nil && (s.Type.Is(openapi3.TypeInteger) || s.Type.Is(openapi3.TypeNumber)) {
				isStringEnum = false
			} else if len(s.Enum) > 0 {
				// Check the first value
				switch s.Enum[0].(type) {
				case int, int32, int64, float32, float64:
					isStringEnum = false
				}
			}

			if isStringEnum {
				for _, v := range s.Enum {
					vals = append(vals, fmt.Sprintf("\"%v\"", v))
				}
			} else {
				for _, v := range s.Enum {
					vals = append(vals, fmt.Sprint(v))
				}
			}
			enumDecl := fmt.Sprintf("export type %s = %s", baseName, strings.Join(vals, " | "))
			*out = append(*out, ir.IRModel{Name: baseName, Decl: enumDecl})
			seen[baseName] = struct{}{}
		}
		t := baseName
		if s.Nullable {
			t += " | null"
		}
		return t
	}

	// Primitive types
	if s.Type != nil {
		switch {
		case s.Type.Is(openapi3.TypeString):
			t := "string"
			if s.Format == "binary" {
				t = "Blob"
			}
			if s.Nullable {
				t += " | null"
			}
			return t
		case s.Type.Is(openapi3.TypeInteger), s.Type.Is(openapi3.TypeNumber):
			t := "number"
			if s.Nullable {
				t += " | null"
			}
			return t
		case s.Type.Is(openapi3.TypeBoolean):
			t := "boolean"
			if s.Nullable {
				t += " | null"
			}
			return t
		case s.Type.Is(openapi3.TypeArray):
			// Handle array items
			itemType := "unknown"
			if s.Items != nil {
				// Check if the item is an inline object that should be named
				if s.Items.Value != nil && s.Items.Value.Type != nil && s.Items.Value.Type.Is(openapi3.TypeObject) && len(s.Items.Value.Properties) > 0 {
					// Create a named type for the array item
					itemName := parentName
					if propName != "" {
						itemName = itemName + "_" + toPascalCase(propName)
					}
					itemName = itemName + "_Item"
					if _, ok := seen[itemName]; !ok {
						itemTS := schemaToTSForSchemaFile(doc, s.Items, itemName, "", true, out, seen)
						itemDecl := fmt.Sprintf("export interface %s %s", itemName, toInterfaceShape(itemTS))
						*out = append(*out, ir.IRModel{Name: itemName, Decl: itemDecl})
						seen[itemName] = struct{}{}
					}
					itemType = itemName
				} else {
					itemType = schemaToTSForSchemaFile(doc, s.Items, parentName, propName, true, out, seen)
				}
			}
			// Wrap complex types in parentheses
			if strings.Contains(itemType, " | ") || strings.Contains(itemType, " & ") {
				itemType = "(" + itemType + ")"
			}
			t := "Array<" + itemType + ">"
			if s.Nullable {
				t += " | null"
			}
			return t
		case s.Type.Is(openapi3.TypeObject):
			// Handle object properties
			if len(s.Properties) == 0 {
				t := "Record<string, unknown>"
				if s.Nullable {
					t += " | null"
				}
				return t
			}

			// If this is a nested object (not top-level), create a named interface
			if propName != "" || isArrayItem {
				objName := parentName
				if propName != "" {
					objName = objName + "_" + toPascalCase(propName)
				}
				if isArrayItem {
					objName = objName + "_Item"
				}
				if _, ok := seen[objName]; !ok {
					// Build the object interface
					propParts := make([]string, 0, len(s.Properties))
					propNames := make([]string, 0, len(s.Properties))
					for name := range s.Properties {
						propNames = append(propNames, name)
					}
					sort.Strings(propNames)

					for _, name := range propNames {
						prop := s.Properties[name]
						propType := schemaToTSForSchemaFile(doc, prop, objName, name, false, out, seen)
						required := false
						for _, req := range s.Required {
							if req == name {
								required = true
								break
							}
						}
						if required {
							propParts = append(propParts, fmt.Sprintf("  %s: %s", name, propType))
						} else {
							propParts = append(propParts, fmt.Sprintf("  %s?: %s", name, propType))
						}
					}
					objBody := "{\n" + strings.Join(propParts, ";\n") + ";\n}"
					objDecl := fmt.Sprintf("export interface %s %s", objName, objBody)
					*out = append(*out, ir.IRModel{Name: objName, Decl: objDecl})
					seen[objName] = struct{}{}
				}
				t := objName
				if s.Nullable {
					t += " | null"
				}
				return t
			}

			// Inline object for top-level schemas
			propParts := make([]string, 0, len(s.Properties))
			propNames := make([]string, 0, len(s.Properties))
			for name := range s.Properties {
				propNames = append(propNames, name)
			}
			sort.Strings(propNames)

			for _, name := range propNames {
				prop := s.Properties[name]
				propType := schemaToTSForSchemaFile(doc, prop, parentName, name, false, out, seen)
				required := false
				for _, req := range s.Required {
					if req == name {
						required = true
						break
					}
				}
				if required {
					propParts = append(propParts, fmt.Sprintf("  %s: %s", name, propType))
				} else {
					propParts = append(propParts, fmt.Sprintf("  %s?: %s", name, propType))
				}
			}
			t := "{\n" + strings.Join(propParts, ";\n") + ";\n}"
			if s.Nullable {
				t += " | null"
			}
			return t
		}
	}

	t := "unknown"
	if s.Nullable {
		t += " | null"
	}
	return t
}
