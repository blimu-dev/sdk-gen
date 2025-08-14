package generator

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/blimu-dev/sdk-gen/pkg/ir"
	"github.com/getkin/kin-openapi/openapi3"
)

// schemaRefToIR converts an OpenAPI schema reference to IR schema
func schemaRefToIR(doc *openapi3.T, sr *openapi3.SchemaRef) ir.IRSchema {
	if sr == nil {
		return ir.IRSchema{Kind: ir.IRKindUnknown}
	}
	if sr.Ref != "" {
		if strings.HasPrefix(sr.Ref, "#/components/schemas/") {
			name := strings.TrimPrefix(sr.Ref, "#/components/schemas/")
			return ir.IRSchema{Kind: ir.IRKindRef, Ref: name}
		}
		parts := strings.Split(sr.Ref, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			if name != "" {
				return ir.IRSchema{Kind: ir.IRKindRef, Ref: name}
			}
		}
		return ir.IRSchema{Kind: ir.IRKindUnknown}
	}
	if sr.Value == nil {
		return ir.IRSchema{Kind: ir.IRKindUnknown}
	}
	s := sr.Value

	// Polymorphism discriminator
	var disc *ir.IRDiscriminator
	if s.Discriminator != nil {
		disc = &ir.IRDiscriminator{PropertyName: s.Discriminator.PropertyName, Mapping: s.Discriminator.Mapping}
	}

	// Compositions
	if len(s.OneOf) > 0 {
		subs := make([]*ir.IRSchema, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			sc := schemaRefToIR(doc, sub)
			subs = append(subs, &sc)
		}
		return ir.IRSchema{Kind: ir.IRKindOneOf, OneOf: subs, Nullable: s.Nullable, Discriminator: disc}
	}
	if len(s.AnyOf) > 0 {
		subs := make([]*ir.IRSchema, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			sc := schemaRefToIR(doc, sub)
			subs = append(subs, &sc)
		}
		return ir.IRSchema{Kind: ir.IRKindAnyOf, AnyOf: subs, Nullable: s.Nullable, Discriminator: disc}
	}
	if len(s.AllOf) > 0 {
		subs := make([]*ir.IRSchema, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			sc := schemaRefToIR(doc, sub)
			subs = append(subs, &sc)
		}
		return ir.IRSchema{Kind: ir.IRKindAllOf, AllOf: subs, Nullable: s.Nullable, Discriminator: disc}
	}
	if s.Not != nil {
		not := schemaRefToIR(doc, s.Not)
		return ir.IRSchema{Kind: ir.IRKindNot, Not: &not, Nullable: s.Nullable, Discriminator: disc}
	}

	// Enum (support non-string by coercing to string representation)
	if len(s.Enum) > 0 {
		vals := make([]string, 0, len(s.Enum))
		for _, v := range s.Enum {
			vals = append(vals, fmt.Sprint(v))
		}
		base := inferEnumBaseKind(s)
		return ir.IRSchema{Kind: ir.IRKindEnum, EnumValues: vals, EnumRaw: s.Enum, EnumBase: base, Nullable: s.Nullable, Discriminator: disc}
	}

	// Primitive kinds and object/array
	if s.Type != nil {
		switch {
		case s.Type.Is(openapi3.TypeString):
			return ir.IRSchema{Kind: ir.IRKindString, Nullable: s.Nullable, Format: s.Format, Discriminator: disc}
		case s.Type.Is(openapi3.TypeInteger):
			return ir.IRSchema{Kind: ir.IRKindInteger, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeNumber):
			return ir.IRSchema{Kind: ir.IRKindNumber, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeBoolean):
			return ir.IRSchema{Kind: ir.IRKindBoolean, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeArray):
			item := schemaRefToIR(doc, s.Items)
			return ir.IRSchema{Kind: ir.IRKindArray, Items: &item, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeObject):
			// Properties
			fields := make([]ir.IRField, 0, len(s.Properties))
			// deterministic order
			names := make([]string, 0, len(s.Properties))
			for n := range s.Properties {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				pr := s.Properties[n]
				fieldType := schemaRefToIR(doc, pr)
				required := false
				for _, r := range s.Required {
					if r == n {
						required = true
						break
					}
				}
				fields = append(fields, ir.IRField{Name: n, Type: &fieldType, Required: required, Annotations: extractAnnotations(pr)})
			}
			var addl *ir.IRSchema
			if s.AdditionalProperties.Schema != nil {
				ap := schemaRefToIR(doc, s.AdditionalProperties.Schema)
				addl = &ap
			}
			return ir.IRSchema{Kind: ir.IRKindObject, Properties: fields, AdditionalProperties: addl, Nullable: s.Nullable, Discriminator: disc}
		}
	}
	return ir.IRSchema{Kind: ir.IRKindUnknown, Nullable: s.Nullable, Discriminator: disc}
}

// schemaRefToIRWithNaming converts schema with naming for nested types
func schemaRefToIRWithNaming(doc *openapi3.T, sr *openapi3.SchemaRef, parentName, propName string, isArrayItem bool, out *[]ir.IRModelDef, seen map[string]struct{}) ir.IRSchema {
	if sr == nil {
		return ir.IRSchema{Kind: ir.IRKindUnknown}
	}
	if sr.Ref != "" {
		if strings.HasPrefix(sr.Ref, "#/components/schemas/") {
			name := strings.TrimPrefix(sr.Ref, "#/components/schemas/")
			return ir.IRSchema{Kind: ir.IRKindRef, Ref: name}
		}
		parts := strings.Split(sr.Ref, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			if name != "" {
				return ir.IRSchema{Kind: ir.IRKindRef, Ref: name}
			}
		}
		return ir.IRSchema{Kind: ir.IRKindUnknown}
	}
	if sr.Value == nil {
		return ir.IRSchema{Kind: ir.IRKindUnknown}
	}
	s := sr.Value

	// Discriminator
	var disc *ir.IRDiscriminator
	if s.Discriminator != nil {
		disc = &ir.IRDiscriminator{PropertyName: s.Discriminator.PropertyName, Mapping: s.Discriminator.Mapping}
	}

	// Compositions (no naming for subs; inline)
	if len(s.OneOf) > 0 {
		subs := make([]*ir.IRSchema, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			sc := schemaRefToIRWithNaming(doc, sub, parentName, propName, isArrayItem, out, seen)
			subs = append(subs, &sc)
		}
		return ir.IRSchema{Kind: ir.IRKindOneOf, OneOf: subs, Nullable: s.Nullable, Discriminator: disc}
	}
	if len(s.AnyOf) > 0 {
		subs := make([]*ir.IRSchema, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			sc := schemaRefToIRWithNaming(doc, sub, parentName, propName, isArrayItem, out, seen)
			subs = append(subs, &sc)
		}
		return ir.IRSchema{Kind: ir.IRKindAnyOf, AnyOf: subs, Nullable: s.Nullable, Discriminator: disc}
	}
	if len(s.AllOf) > 0 {
		subs := make([]*ir.IRSchema, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			sc := schemaRefToIRWithNaming(doc, sub, parentName, propName, isArrayItem, out, seen)
			subs = append(subs, &sc)
		}
		return ir.IRSchema{Kind: ir.IRKindAllOf, AllOf: subs, Nullable: s.Nullable, Discriminator: disc}
	}
	if s.Not != nil {
		not := schemaRefToIRWithNaming(doc, s.Not, parentName, propName, isArrayItem, out, seen)
		return ir.IRSchema{Kind: ir.IRKindNot, Not: &not, Nullable: s.Nullable, Discriminator: disc}
	}

	// Enum: create named model when in a nested context
	if len(s.Enum) > 0 {
		baseName := parentName
		if propName != "" {
			baseName = baseName + "_" + toPascal(propName)
		}
		if isArrayItem {
			baseName = baseName + "_Item"
		}
		if _, ok := seen[baseName]; !ok {
			vals := make([]string, 0, len(s.Enum))
			for _, v := range s.Enum {
				vals = append(vals, fmt.Sprint(v))
			}
			md := ir.IRModelDef{
				Name:        baseName,
				Schema:      ir.IRSchema{Kind: ir.IRKindEnum, EnumValues: vals, EnumRaw: s.Enum, EnumBase: inferEnumBaseKind(s), Nullable: s.Nullable, Discriminator: disc},
				Annotations: extractAnnotations(sr),
			}
			*out = append(*out, md)
			seen[baseName] = struct{}{}
		}
		return ir.IRSchema{Kind: ir.IRKindRef, Ref: baseName, Nullable: s.Nullable}
	}

	if s.Type != nil {
		switch {
		case s.Type.Is(openapi3.TypeString):
			return ir.IRSchema{Kind: ir.IRKindString, Nullable: s.Nullable, Format: s.Format, Discriminator: disc}
		case s.Type.Is(openapi3.TypeInteger):
			return ir.IRSchema{Kind: ir.IRKindInteger, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeNumber):
			return ir.IRSchema{Kind: ir.IRKindNumber, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeBoolean):
			return ir.IRSchema{Kind: ir.IRKindBoolean, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeArray):
			// Name array item if it is an inline object or enum
			itemSchema := s.Items
			if itemSchema != nil && itemSchema.Value != nil {
				itemVal := itemSchema.Value
				if len(itemVal.Enum) > 0 {
					// Use enum naming path
					ref := schemaRefToIRWithNaming(doc, itemSchema, parentName, propName, true, out, seen)
					return ir.IRSchema{Kind: ir.IRKindArray, Items: &ref, Nullable: s.Nullable, Discriminator: disc}
				}
				if itemVal.Type != nil && itemVal.Type.Is(openapi3.TypeObject) && len(itemVal.Properties) > 0 {
					base := parentName
					if propName != "" {
						base = base + "_" + toPascal(propName)
					}
					name := base + "_Item"
					if _, ok := seen[name]; !ok {
						def := buildNamedObjectDef(doc, itemVal, name, out, seen)
						*out = append(*out, def)
						seen[name] = struct{}{}
					}
					ref := ir.IRSchema{Kind: ir.IRKindRef, Ref: name}
					return ir.IRSchema{Kind: ir.IRKindArray, Items: &ref, Nullable: s.Nullable, Discriminator: disc}
				}
			}
			itm := schemaRefToIRWithNaming(doc, s.Items, parentName, propName, true, out, seen)
			return ir.IRSchema{Kind: ir.IRKindArray, Items: &itm, Nullable: s.Nullable, Discriminator: disc}
		case s.Type.Is(openapi3.TypeObject):
			// Build object and emit named model defs for nested inline object properties
			// Properties in deterministic order
			propNames := make([]string, 0, len(s.Properties))
			for n := range s.Properties {
				propNames = append(propNames, n)
			}
			sort.Strings(propNames)
			fields := make([]ir.IRField, 0, len(propNames))
			for _, n := range propNames {
				pr := s.Properties[n]
				val := pr.Value
				var fType ir.IRSchema
				if (propName != "" || isArrayItem) && val != nil && val.Type != nil && val.Type.Is(openapi3.TypeObject) && len(val.Properties) > 0 {
					// Nested inline object under a non-top-level object -> name it
					base := parentName
					if propName != "" {
						base = base + "_" + toPascal(propName)
					}
					name := base + "_" + toPascal(n)
					if _, ok := seen[name]; !ok {
						def := buildNamedObjectDef(doc, val, name, out, seen)
						*out = append(*out, def)
						seen[name] = struct{}{}
					}
					fType = ir.IRSchema{Kind: ir.IRKindRef, Ref: name}
				} else {
					fType = schemaRefToIRWithNaming(doc, pr, parentName, n, false, out, seen)
				}
				required := false
				for _, r := range s.Required {
					if r == n {
						required = true
						break
					}
				}
				fields = append(fields, ir.IRField{Name: n, Type: &fType, Required: required, Annotations: extractAnnotations(pr)})
			}
			var addl *ir.IRSchema
			if s.AdditionalProperties.Schema != nil {
				addlSchema := s.AdditionalProperties.Schema

				// If additionalProperties is an object with properties, merge them into the parent
				if addlSchema.Value != nil && addlSchema.Value.Type != nil &&
					addlSchema.Value.Type.Is(openapi3.TypeObject) && len(addlSchema.Value.Properties) > 0 {

					// Merge additionalProperties into the current object's fields
					addlPropNames := make([]string, 0, len(addlSchema.Value.Properties))
					for n := range addlSchema.Value.Properties {
						addlPropNames = append(addlPropNames, n)
					}
					sort.Strings(addlPropNames)

					for _, n := range addlPropNames {
						pr := addlSchema.Value.Properties[n]
						fType := schemaRefToIRWithNaming(doc, pr, parentName, n, false, out, seen)
						required := false
						for _, r := range addlSchema.Value.Required {
							if r == n {
								required = true
								break
							}
						}
						fields = append(fields, ir.IRField{Name: n, Type: &fType, Required: required, Annotations: extractAnnotations(pr)})
					}

					// Don't set addl since we merged the properties
					addl = nil
				} else {
					// For non-object additionalProperties, keep the current behavior
					addlParent := parentName
					if propName != "" {
						addlParent = addlParent + "_" + toPascal(propName)
					}
					if isArrayItem {
						addlParent = addlParent + "_Item"
					}
					aps := schemaRefToIRWithNaming(doc, s.AdditionalProperties.Schema, addlParent, "Properties", false, out, seen)
					addl = &aps
				}
			}
			// If this object itself is nested (not top-level), produce a named ref
			if propName != "" || isArrayItem {
				base := parentName
				if propName != "" {
					base = base + "_" + toPascal(propName)
				}
				if isArrayItem {
					base = base + "_Item"
				}
				if _, ok := seen[base]; !ok {
					def := ir.IRModelDef{
						Name:        base,
						Schema:      ir.IRSchema{Kind: ir.IRKindObject, Properties: fields, AdditionalProperties: addl, Nullable: s.Nullable, Discriminator: disc},
						Annotations: extractAnnotations(sr),
					}
					*out = append(*out, def)
					seen[base] = struct{}{}
				}
				return ir.IRSchema{Kind: ir.IRKindRef, Ref: base}
			}
			return ir.IRSchema{Kind: ir.IRKindObject, Properties: fields, AdditionalProperties: addl, Nullable: s.Nullable, Discriminator: disc}
		}
	}
	return ir.IRSchema{Kind: ir.IRKindUnknown, Nullable: s.Nullable, Discriminator: disc}
}

// extractAnnotations extracts annotations from a schema reference
func extractAnnotations(sr *openapi3.SchemaRef) ir.IRAnnotations {
	var a ir.IRAnnotations
	if sr == nil || sr.Value == nil {
		return a
	}
	s := sr.Value
	a.Title = s.Title
	a.Description = s.Description
	a.Deprecated = s.Deprecated
	a.ReadOnly = s.ReadOnly
	a.WriteOnly = s.WriteOnly
	a.Default = s.Default
	if s.Example != nil {
		a.Examples = []any{s.Example}
	}
	return a
}

// inferEnumBaseKind infers the base kind for an enum
func inferEnumBaseKind(s *openapi3.Schema) ir.IRSchemaKind {
	// Prefer explicit type when present
	if s.Type != nil {
		switch {
		case s.Type.Is(openapi3.TypeString):
			return ir.IRKindString
		case s.Type.Is(openapi3.TypeInteger):
			return ir.IRKindInteger
		case s.Type.Is(openapi3.TypeNumber):
			return ir.IRKindNumber
		case s.Type.Is(openapi3.TypeBoolean):
			return ir.IRKindBoolean
		}
	}
	// Fallback: inspect first enum value
	if len(s.Enum) > 0 {
		switch s.Enum[0].(type) {
		case string:
			return ir.IRKindString
		case int, int32, int64:
			return ir.IRKindInteger
		case float32, float64:
			return ir.IRKindNumber
		case bool:
			return ir.IRKindBoolean
		}
	}
	return ir.IRKindUnknown
}

// buildNamedObjectDef constructs a named object model def for an inline object schema
func buildNamedObjectDef(doc *openapi3.T, s *openapi3.Schema, name string, out *[]ir.IRModelDef, seen map[string]struct{}) ir.IRModelDef {
	// Properties in deterministic order
	propNames := make([]string, 0, len(s.Properties))
	for n := range s.Properties {
		propNames = append(propNames, n)
	}
	sort.Strings(propNames)
	fields := make([]ir.IRField, 0, len(propNames))
	for _, n := range propNames {
		pr := s.Properties[n]
		fType := schemaRefToIRWithNaming(doc, pr, name, n, false, out, seen)
		required := false
		for _, r := range s.Required {
			if r == n {
				required = true
				break
			}
		}
		fields = append(fields, ir.IRField{Name: n, Type: &fType, Required: required, Annotations: extractAnnotations(pr)})
	}
	var addl *ir.IRSchema
	if s.AdditionalProperties.Schema != nil {
		addlSchema := s.AdditionalProperties.Schema

		// If additionalProperties is an object with properties, merge them into the parent
		if addlSchema.Value != nil && addlSchema.Value.Type != nil &&
			addlSchema.Value.Type.Is(openapi3.TypeObject) && len(addlSchema.Value.Properties) > 0 {

			// Merge additionalProperties into the current object's fields
			addlPropNames := make([]string, 0, len(addlSchema.Value.Properties))
			for n := range addlSchema.Value.Properties {
				addlPropNames = append(addlPropNames, n)
			}
			sort.Strings(addlPropNames)

			for _, n := range addlPropNames {
				pr := addlSchema.Value.Properties[n]
				fType := schemaRefToIRWithNaming(doc, pr, name, n, false, out, seen)
				required := false
				for _, r := range addlSchema.Value.Required {
					if r == n {
						required = true
						break
					}
				}
				fields = append(fields, ir.IRField{Name: n, Type: &fType, Required: required, Annotations: extractAnnotations(pr)})
			}

			// Don't set addl since we merged the properties
			addl = nil
		} else {
			// For non-object additionalProperties, keep the current behavior
			aps := schemaRefToIRWithNaming(doc, s.AdditionalProperties.Schema, name, "Properties", false, out, seen)
			addl = &aps
		}
	}
	return ir.IRModelDef{
		Name:        name,
		Schema:      ir.IRSchema{Kind: ir.IRKindObject, Properties: fields, AdditionalProperties: addl, Nullable: s.Nullable},
		Annotations: ir.IRAnnotations{Title: s.Title, Description: s.Description, Deprecated: s.Deprecated, ReadOnly: s.ReadOnly, WriteOnly: s.WriteOnly, Default: s.Default},
	}
}

var nonAlnumSchema = regexp.MustCompile(`[^A-Za-z0-9]+`)

// toPascal converts a string to PascalCase
func toPascal(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// First split by non-alphanumeric characters
	parts := nonAlnumSchema.Split(s, -1)
	var allParts []string

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Further split camelCase/PascalCase words
		subParts := splitCamelCaseSchema(part)
		allParts = append(allParts, subParts...)
	}

	b := strings.Builder{}
	for _, p := range allParts {
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

// splitCamelCaseSchema splits a camelCase or PascalCase string into words
func splitCamelCaseSchema(s string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	var current strings.Builder

	runes := []rune(s)
	for i, r := range runes {
		// Check if this is the start of a new word
		isNewWord := false
		if i > 0 && isUppercaseSchema(r) {
			// Current char is uppercase
			if !isUppercaseSchema(runes[i-1]) {
				// Previous char was lowercase, so this starts a new word
				isNewWord = true
			} else if i < len(runes)-1 && !isUppercaseSchema(runes[i+1]) {
				// Previous char was uppercase, but next char is lowercase
				// This handles cases like "XMLHttp" -> "XML", "Http"
				isNewWord = true
			}
		}

		if isNewWord && current.Len() > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// isUppercaseSchema checks if a rune is uppercase
func isUppercaseSchema(r rune) bool {
	return r >= 'A' && r <= 'Z'
}
