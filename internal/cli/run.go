package cli

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/viniciusdacal/sdk-gen/internal/config"
	ts "github.com/viniciusdacal/sdk-gen/internal/generators/typescript"
	"github.com/viniciusdacal/sdk-gen/internal/ir"
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
	doc, err := loader.LoadFromFile(input)
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
	doc, err := loader.LoadFromFile(cfg.Spec)
	if err != nil {
		return err
	}
	if err := doc.Validate(loader.Context); err != nil {
		return err
	}

	tags := collectTags(doc)
	sec := collectSecuritySchemes(doc)

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
		reqBody := extractRequestBodyTS(doc, op)
		resp := extractResponseTS(doc, op)
		servicesMap[tag].Operations = append(servicesMap[tag].Operations, ir.IROperation{
			OperationID: id,
			Method:      method,
			Path:        path,
			Tag:         tag,
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

func collectParamsTS(doc *openapi3.T, op *openapi3.Operation) (pathParams, queryParams []ir.IRParam) {
	for _, pr := range op.Parameters {
		if pr == nil || pr.Value == nil {
			continue
		}
		p := pr.Value
		param := ir.IRParam{
			Name:     p.Name,
			Required: p.Required,
			TypeTS:   schemaToTS(doc, p.Schema),
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

func extractRequestBodyTS(doc *openapi3.T, op *openapi3.Operation) *ir.IRRequestBody {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	rb := op.RequestBody.Value
	// Prefer application/json
	if media, ok := rb.Content["application/json"]; ok {
		return &ir.IRRequestBody{ContentType: "application/json", TypeTS: schemaToTS(doc, media.Schema), Required: rb.Required}
	}
	if media, ok := rb.Content["application/x-www-form-urlencoded"]; ok {
		// Simplified mapping
		t := schemaToTS(doc, media.Schema)
		if t == "unknown" {
			t = "Record<string, string>"
		}
		return &ir.IRRequestBody{ContentType: "application/x-www-form-urlencoded", TypeTS: t, Required: rb.Required}
	}
	if _, ok := rb.Content["multipart/form-data"]; ok {
		return &ir.IRRequestBody{ContentType: "multipart/form-data", TypeTS: "FormData | Record<string, Blob | string>", Required: rb.Required}
	}
	// Fallback to the first available media type
	for ct, media := range rb.Content {
		return &ir.IRRequestBody{ContentType: ct, TypeTS: schemaToTS(doc, media.Schema), Required: rb.Required}
	}
	return nil
}

func extractResponseTS(doc *openapi3.T, op *openapi3.Operation) ir.IRResponse {
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
				return ir.IRResponse{TypeTS: schemaToTS(doc, media.Schema)}
			}
			// Fallback to any content
			for _, media := range rr.Value.Content {
				return ir.IRResponse{TypeTS: schemaToTS(doc, media.Schema)}
			}
			return ir.IRResponse{TypeTS: "void"}
		}
	}
	// any 2xx
	if op.Responses != nil {
		for code, rr := range op.Responses.Map() {
			if len(code) == 3 && code[0] == '2' {
				if rr != nil && rr.Value != nil {
					if code == "204" {
						return ir.IRResponse{TypeTS: "void"}
					}
					if media, ok := rr.Value.Content["application/json"]; ok {
						return ir.IRResponse{TypeTS: schemaToTS(doc, media.Schema)}
					}
					for _, media := range rr.Value.Content {
						return ir.IRResponse{TypeTS: schemaToTS(doc, media.Schema)}
					}
				}
			}
		}
	}
	return ir.IRResponse{TypeTS: "unknown"}
}

func schemaToTS(doc *openapi3.T, sr *openapi3.SchemaRef) string {
	if sr == nil {
		return "unknown"
	}
	if sr.Ref != "" {
		// Prefer named component refs
		if strings.HasPrefix(sr.Ref, "#/components/schemas/") {
			name := strings.TrimPrefix(sr.Ref, "#/components/schemas/")
			return "Schema." + name
		}
		// External refs or non-components: best-effort name
		parts := strings.Split(sr.Ref, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			if name != "" {
				return "Schema." + name
			}
		}
		return "unknown"
	}
	// Note: skipping deep deref for non-$ref inline schemas
	if sr.Value == nil {
		return "unknown"
	}
	s := sr.Value

	// anyOf/oneOf/allOf
	if len(s.OneOf) > 0 {
		parts := make([]string, 0, len(s.OneOf))
		for _, sub := range s.OneOf {
			parts = append(parts, schemaToTS(doc, sub))
		}
		return strings.Join(parts, " | ")
	}
	if len(s.AnyOf) > 0 {
		parts := make([]string, 0, len(s.AnyOf))
		for _, sub := range s.AnyOf {
			parts = append(parts, schemaToTS(doc, sub))
		}
		return strings.Join(parts, " | ")
	}
	if len(s.AllOf) > 0 {
		parts := make([]string, 0, len(s.AllOf))
		for _, sub := range s.AllOf {
			parts = append(parts, schemaToTS(doc, sub))
		}
		return strings.Join(parts, " & ")
	}

	var t string
	if s.Type != nil && s.Type.Is(openapi3.TypeString) {
		if len(s.Enum) > 0 {
			lits := make([]string, 0, len(s.Enum))
			for _, v := range s.Enum {
				lits = append(lits, fmt.Sprintf("%q", fmt.Sprint(v)))
			}
			t = strings.Join(lits, " | ")
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
		t = "Array<" + schemaToTS(doc, s.Items) + ">"
	} else if s.Type != nil && s.Type.Is(openapi3.TypeObject) {
		// Object
		if len(s.Properties) > 0 {
			// Inline object
			parts := make([]string, 0, len(s.Properties))
			for name, prop := range s.Properties {
				req := false
				for _, r := range s.Required {
					if r == name {
						req = true
						break
					}
				}
				propTS := schemaToTS(doc, prop)
				if !req {
					parts = append(parts, fmt.Sprintf("%s?: %s", name, propTS))
				} else {
					parts = append(parts, fmt.Sprintf("%s: %s", name, propTS))
				}
			}
			t = "{" + strings.Join(parts, "; ") + "}"
		} else {
			t = "Record<string, unknown>"
		}
	} else {
		t = "unknown"
	}
	if s.Nullable {
		t = t + " | null"
	}
	return t
}

func firstAllowedTag(tags []string, allowed map[string]bool) string {
	for _, t := range tags {
		if allowed[t] {
			return t
		}
	}
	return ""
}
