package cli

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/blimu-dev/sdk-gen/internal/config"
	ts "github.com/blimu-dev/sdk-gen/internal/generators/typescript"
	"github.com/blimu-dev/sdk-gen/internal/ir"
	"github.com/getkin/kin-openapi/openapi3"
)

type FallbackParams struct {
	Spec        string
	Type        string
	OutDir      string
	PackageName string
	Name        string
	IncludeTags []string
	ExcludeTags []string
}

type RunGenerateParams struct {
	ConfigPath   string
	SingleClient string
	Fallback     FallbackParams
}

func RunValidate(input string) error {
	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	doc, err := loadOpenAPIDoc(loader, input)
	if err != nil {
		return err
	}
	return doc.Validate(loader.Context)
}

func RunGenerate(p RunGenerateParams) error {
	if p.ConfigPath == "" {
		if p.Fallback.Spec == "" || p.Fallback.Type == "" || p.Fallback.OutDir == "" || p.Fallback.PackageName == "" || p.Fallback.Name == "" {
			return errors.New("either --config or all of --input, --type, --out, --package-name, --client-name must be provided")
		}
		cfg := &config.Config{
			Spec: p.Fallback.Spec,
			Clients: []config.Client{
				{
					Type:        p.Fallback.Type,
					OutDir:      absPath(p.Fallback.OutDir),
					PackageName: p.Fallback.PackageName,
					Name:        p.Fallback.Name,
					IncludeTags: p.Fallback.IncludeTags,
					ExcludeTags: p.Fallback.ExcludeTags,
				},
			},
		}
		return generateFromConfig(cfg, "")
	}

	cfg, err := config.Load(p.ConfigPath)
	if err != nil {
		return err
	}
	return generateFromConfig(cfg, p.SingleClient)
}

func generateFromConfig(cfg *config.Config, onlyClient string) error {
	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	doc, err := loadOpenAPIDoc(loader, cfg.Spec)
	if err != nil {
		return err
	}
	if err := doc.Validate(loader.Context); err != nil {
		return err
	}

	tags := collectTags(doc)
	sec := collectSecuritySchemes(doc)
	// Build language-agnostic model defs from components.schemas
	modelDefs := buildStructuredModels(doc)

	for _, c := range cfg.Clients {
		if onlyClient != "" && c.Name != onlyClient {
			continue
		}
		include, exclude, err := compileTagFilters(c.IncludeTags, c.ExcludeTags)
		if err != nil {
			return err
		}
		filtered := filterTags(tags, include, exclude)
		// Build a minimal IR of operations grouped by tag
		ir := buildIR(doc, filtered)
		// Attach language-agnostic models
		ir.ModelDefs = modelDefs
		// Language-specific model collection happens in generators
		ir.SecuritySchemes = sec

		switch c.Type {
		case "typescript":
			// Generate TS models (including nested named interfaces)
			ir.Models = ts.CollectModels(doc)
			if err := ts.Generate(c, ir); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported client type: %s", c.Type)
		}
	}
	return nil
}

func collectTags(doc *openapi3.T) []string {
	uniq := map[string]struct{}{}
	// consider untagged as "misc"
	uniq["misc"] = struct{}{}
	for path, item := range doc.Paths.Map() {
		_ = path
		for _, op := range []*openapi3.Operation{item.Get, item.Post, item.Put, item.Patch, item.Delete, item.Options, item.Head, item.Trace} {
			if op == nil {
				continue
			}
			if len(op.Tags) == 0 {
				continue
			}
			for _, t := range op.Tags {
				uniq[t] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(uniq))
	for t := range uniq {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func compileTagFilters(include, exclude []string) ([]*regexp.Regexp, []*regexp.Regexp, error) {
	inc := make([]*regexp.Regexp, 0, len(include))
	for _, p := range include {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid includeTags pattern %q: %w", p, err)
		}
		inc = append(inc, r)
	}
	exc := make([]*regexp.Regexp, 0, len(exclude))
	for _, p := range exclude {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid excludeTags pattern %q: %w", p, err)
		}
		exc = append(exc, r)
	}
	return inc, exc, nil
}

func filterTags(all []string, include, exclude []*regexp.Regexp) map[string]bool {
	allowed := map[string]bool{}
	for _, t := range all {
		allowed[t] = true
	}
	if len(include) > 0 {
		for t := range allowed {
			matched := false
			for _, r := range include {
				if r.MatchString(t) {
					matched = true
					break
				}
			}
			if !matched {
				allowed[t] = false
			}
		}
	}
	if len(exclude) > 0 {
		for t := range allowed {
			if !allowed[t] {
				continue
			}
			for _, r := range exclude {
				if r.MatchString(t) {
					allowed[t] = false
					break
				}
			}
		}
	}
	return allowed
}

// Minimal IR structures
func buildIR(doc *openapi3.T, allowed map[string]bool) ir.IR {
	servicesMap := map[string]*ir.IRService{}
	// Always prepare misc
	servicesMap["misc"] = &ir.IRService{Tag: "misc"}

	addOp := func(tag string, op *openapi3.Operation, method, path string) {
		if _, ok := servicesMap[tag]; !ok {
			servicesMap[tag] = &ir.IRService{Tag: tag}
		}
		id := op.OperationID
		pathParams, queryParams := collectParamsTS(doc, op)
		reqBody := extractRequestBody(doc, op)
		resp := extractResponse(doc, op)
		servicesMap[tag].Operations = append(servicesMap[tag].Operations, ir.IROperation{
			OperationID: id,
			Method:      method,
			Path:        path,
			Tag:         tag,
			Summary:     op.Summary,
			Description: op.Description,
			Deprecated:  op.Deprecated,
			PathParams:  pathParams,
			QueryParams: queryParams,
			RequestBody: reqBody,
			Response:    resp,
		})
	}

	for path, item := range doc.Paths.Map() {
		if item.Get != nil {
			t := firstAllowedTag(item.Get.Tags, allowed)
			if t == "" {
				if len(item.Get.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Get, "GET", path)
			}
		}
		if item.Post != nil {
			t := firstAllowedTag(item.Post.Tags, allowed)
			if t == "" {
				if len(item.Post.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Post, "POST", path)
			}
		}
		if item.Put != nil {
			t := firstAllowedTag(item.Put.Tags, allowed)
			if t == "" {
				if len(item.Put.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Put, "PUT", path)
			}
		}
		if item.Patch != nil {
			t := firstAllowedTag(item.Patch.Tags, allowed)
			if t == "" {
				if len(item.Patch.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Patch, "PATCH", path)
			}
		}
		if item.Delete != nil {
			t := firstAllowedTag(item.Delete.Tags, allowed)
			if t == "" {
				if len(item.Delete.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Delete, "DELETE", path)
			}
		}
		if item.Options != nil {
			t := firstAllowedTag(item.Options.Tags, allowed)
			if t == "" {
				if len(item.Options.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Options, "OPTIONS", path)
			}
		}
		if item.Head != nil {
			t := firstAllowedTag(item.Head.Tags, allowed)
			if t == "" {
				if len(item.Head.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Head, "HEAD", path)
			}
		}
		if item.Trace != nil {
			t := firstAllowedTag(item.Trace.Tags, allowed)
			if t == "" {
				if len(item.Trace.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, item.Trace, "TRACE", path)
			}
		}
	}

	// Sort services and operations for determinism
	services := make([]ir.IRService, 0, len(servicesMap))
	for _, s := range servicesMap {
		sort.Slice(s.Operations, func(i, j int) bool {
			if s.Operations[i].Path == s.Operations[j].Path {
				return s.Operations[i].Method < s.Operations[j].Method
			}
			return s.Operations[i].Path < s.Operations[j].Path
		})
		services = append(services, *s)
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Tag < services[j].Tag })
	return ir.IR{Services: services}
}

// remove generic collectModelsTS: per-language collectors live in generator packages

// collectSecuritySchemes extracts simplified security scheme information from the document
func collectSecuritySchemes(doc *openapi3.T) []ir.IRSecurityScheme {
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return nil
	}
	// Deterministic order
	names := make([]string, 0, len(doc.Components.SecuritySchemes))
	for name := range doc.Components.SecuritySchemes {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ir.IRSecurityScheme, 0, len(names))
	for _, name := range names {
		sr := doc.Components.SecuritySchemes[name]
		if sr == nil || sr.Value == nil {
			continue
		}
		s := sr.Value
		sc := ir.IRSecurityScheme{Key: name, Type: s.Type}
		switch s.Type {
		case "http":
			sc.Scheme = s.Scheme
			sc.BearerFormat = s.BearerFormat
		case "apiKey":
			sc.In = string(s.In)
			sc.Name = s.Name
		case "oauth2":
			// Keep minimal; flows are not modeled yet
		case "openIdConnect":
			// Keep minimal
		}
		out = append(out, sc)
	}
	return out
}

// buildStructuredModels converts components.schemas into a language-agnostic IR
func buildStructuredModels(doc *openapi3.T) []ir.IRModelDef {
	out := []ir.IRModelDef{}
	if doc.Components == nil || doc.Components.Schemas == nil {
		return out
	}
	names := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	seen := map[string]struct{}{}
	for _, name := range names {
		sr := doc.Components.Schemas[name]
		schema := schemaRefToIRWithNaming(doc, sr, name, "", false, &out, seen)
		out = append(out, ir.IRModelDef{
			Name:        name,
			Schema:      schema,
			Annotations: extractAnnotations(sr),
		})
		seen[name] = struct{}{}
	}
	return out
}

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
				aps := schemaRefToIRWithNaming(doc, s.AdditionalProperties.Schema, parentName, "AdditionalProperties", false, out, seen)
				addl = &aps
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

func toPascal(s string) string {
	// Simple PascalCase from run.go nonAlnum regex would be overkill here; reuse basic logic
	parts := regexp.MustCompile(`[^A-Za-z0-9]+`).Split(strings.TrimSpace(s), -1)
	var b strings.Builder
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
		aps := schemaRefToIRWithNaming(doc, s.AdditionalProperties.Schema, name, "AdditionalProperties", false, out, seen)
		addl = &aps
	}
	return ir.IRModelDef{
		Name:        name,
		Schema:      ir.IRSchema{Kind: ir.IRKindObject, Properties: fields, AdditionalProperties: addl, Nullable: s.Nullable},
		Annotations: ir.IRAnnotations{Title: s.Title, Description: s.Description, Deprecated: s.Deprecated, ReadOnly: s.ReadOnly, WriteOnly: s.WriteOnly, Default: s.Default},
	}
}
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
func collectParamsTS(doc *openapi3.T, op *openapi3.Operation) (pathParams, queryParams []ir.IRParam) {
	for _, pr := range op.Parameters {
		if pr == nil || pr.Value == nil {
			continue
		}
		p := pr.Value
		param := ir.IRParam{
			Name:     p.Name,
			Required: p.Required,
			// TypeTS removed from services; keep empty to avoid TS coupling
			Schema:      schemaRefToIR(doc, p.Schema),
			Description: p.Description,
		}
		switch p.In {
		case openapi3.ParameterInPath:
			pathParams = append(pathParams, param)
		case openapi3.ParameterInQuery:
			queryParams = append(queryParams, param)
		}
	}
	// deterministic order
	sort.Slice(pathParams, func(i, j int) bool { return pathParams[i].Name < pathParams[j].Name })
	sort.Slice(queryParams, func(i, j int) bool { return queryParams[i].Name < queryParams[j].Name })
	return
}

func extractRequestBody(doc *openapi3.T, op *openapi3.Operation) *ir.IRRequestBody {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	rb := op.RequestBody.Value
	// Prefer application/json
	if media, ok := rb.Content["application/json"]; ok {
		return &ir.IRRequestBody{ContentType: "application/json", TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Required: rb.Required}
	}
	if media, ok := rb.Content["application/x-www-form-urlencoded"]; ok {
		// Simplified mapping
		return &ir.IRRequestBody{ContentType: "application/x-www-form-urlencoded", TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Required: rb.Required}
	}
	if _, ok := rb.Content["multipart/form-data"]; ok {
		return &ir.IRRequestBody{ContentType: "multipart/form-data", TypeTS: "", Schema: ir.IRSchema{Kind: ir.IRKindUnknown}, Required: rb.Required}
	}
	// Fallback to the first available media type
	for ct, media := range rb.Content {
		return &ir.IRRequestBody{ContentType: ct, TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Required: rb.Required}
	}
	return nil
}

func extractResponse(doc *openapi3.T, op *openapi3.Operation) ir.IRResponse {
	// Choose 200, 201, or any 2xx; 204 => void
	pick := func(code string) (*openapi3.ResponseRef, bool) {
		if op.Responses == nil {
			return nil, false
		}
		m := op.Responses.Map()
		rr, ok := m[code]
		return rr, ok
	}
	try := []string{"200", "201"}
	for _, code := range try {
		if rr, ok := pick(code); ok && rr != nil && rr.Value != nil {
			if media, ok := rr.Value.Content["application/json"]; ok {
				desc := ""
				if rr.Value.Description != nil {
					desc = *rr.Value.Description
				}
				return ir.IRResponse{TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Description: desc}
			}
			// Fallback to any content
			for _, media := range rr.Value.Content {
				desc := ""
				if rr.Value.Description != nil {
					desc = *rr.Value.Description
				}
				return ir.IRResponse{TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Description: desc}
			}
			desc := ""
			if rr.Value.Description != nil {
				desc = *rr.Value.Description
			}
			return ir.IRResponse{TypeTS: "void", Description: desc}
		}
	}
	// any 2xx
	if op.Responses != nil {
		for code, rr := range op.Responses.Map() {
			if len(code) == 3 && code[0] == '2' {
				if rr != nil && rr.Value != nil {
					if code == "204" {
						desc := ""
						if rr.Value.Description != nil {
							desc = *rr.Value.Description
						}
						return ir.IRResponse{TypeTS: "void", Description: desc}
					}
					if media, ok := rr.Value.Content["application/json"]; ok {
						desc := ""
						if rr.Value.Description != nil {
							desc = *rr.Value.Description
						}
						return ir.IRResponse{TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Description: desc}
					}
					for _, media := range rr.Value.Content {
						desc := ""
						if rr.Value.Description != nil {
							desc = *rr.Value.Description
						}
						return ir.IRResponse{TypeTS: "", Schema: schemaRefToIR(doc, media.Schema), Description: desc}
					}
				}
			}
		}
	}
	return ir.IRResponse{TypeTS: "unknown"}
}

// schemaToTS is used only to keep backwards-compatible TypeScript method signatures
// for request/response and params while we migrate services to templates; once services
// are migrated it can be removed.
// schemaToTS removed: services now use IR.Schema + TS templates for typing

func firstAllowedTag(tags []string, allowed map[string]bool) string {
	for _, t := range tags {
		if allowed[t] {
			return t
		}
	}
	return ""
}

// loadOpenAPIDoc loads the OpenAPI document from a local file path or an HTTP(S) URL.
func loadOpenAPIDoc(loader *openapi3.Loader, input string) (*openapi3.T, error) {
	// Try to parse as URL; if it looks like http(s), fetch via URL
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return loader.LoadFromURI(u)
	}
	// Fallback to reading from filesystem path
	return loader.LoadFromFile(input)
}
