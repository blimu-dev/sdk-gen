package generator

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/blimu-dev/sdk-gen/pkg/config"
	"github.com/blimu-dev/sdk-gen/pkg/ir"
	"github.com/getkin/kin-openapi/openapi3"
)

// buildIR creates an IR from an OpenAPI document
func (s *Service) buildIR(doc *openapi3.T) (ir.IR, error) {
	tags := collectTags(doc)
	sec := collectSecuritySchemes(doc)
	modelDefs := buildStructuredModels(doc)

	// For now, include all tags - filtering will be done per client
	allowed := make(map[string]bool)
	for _, tag := range tags {
		allowed[tag] = true
	}

	// Build IR with all operations
	result := buildIRFromDoc(doc, allowed)
	result.SecuritySchemes = sec
	result.ModelDefs = modelDefs

	return result, nil
}

// filterIR filters the IR based on client configuration
func (s *Service) filterIR(fullIR ir.IR, client config.Client) (ir.IR, error) {
	include, exclude, err := compileTagFilters(client.IncludeTags, client.ExcludeTags)
	if err != nil {
		return ir.IR{}, err
	}

	// Filter services and operations based on their original tags
	filteredServices := make([]ir.IRService, 0)
	for _, service := range fullIR.Services {
		filteredOps := make([]ir.IROperation, 0)
		for _, op := range service.Operations {
			if shouldIncludeOperation(op.OriginalTags, include, exclude) {
				filteredOps = append(filteredOps, op)
			}
		}
		// Only include the service if it has at least one operation after filtering
		if len(filteredOps) > 0 {
			filteredService := service
			filteredService.Operations = filteredOps
			filteredServices = append(filteredServices, filteredService)
		}
	}

	// Filter ModelDefs to only include those referenced by filtered operations
	filteredIR := ir.IR{
		Services:        filteredServices,
		Models:          fullIR.Models,
		SecuritySchemes: fullIR.SecuritySchemes,
		ModelDefs:       fullIR.ModelDefs,
	}
	filteredIR.ModelDefs = filterUnusedModelDefs(filteredIR, fullIR.ModelDefs)

	return filteredIR, nil
}

// collectTags extracts all tags from the OpenAPI document
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

// compileTagFilters compiles regex patterns for tag filtering
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

// shouldIncludeOperation determines if an operation should be included based on its original tags
func shouldIncludeOperation(originalTags []string, include, exclude []*regexp.Regexp) bool {
	// If no include patterns, assume all tags are initially included
	included := len(include) == 0

	// Check include patterns - operation is included if ANY of its tags match ANY include pattern
	if len(include) > 0 {
		for _, tag := range originalTags {
			for _, r := range include {
				if r.MatchString(tag) {
					included = true
					break
				}
			}
			if included {
				break
			}
		}
	}

	// If not included by include patterns, exclude it
	if !included {
		return false
	}

	// Check exclude patterns - operation is excluded if ANY of its tags match ANY exclude pattern
	if len(exclude) > 0 {
		for _, tag := range originalTags {
			for _, r := range exclude {
				if r.MatchString(tag) {
					return false
				}
			}
		}
	}

	return true
}

// filterTags filters tags based on include/exclude patterns
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

// buildIRFromDoc builds IR structures from OpenAPI document
func buildIRFromDoc(doc *openapi3.T, allowed map[string]bool) ir.IR {
	servicesMap := map[string]*ir.IRService{}
	// Always prepare misc
	servicesMap["misc"] = &ir.IRService{Tag: "misc"}

	addOp := func(tag string, op *openapi3.Operation, method, path string) {
		if _, ok := servicesMap[tag]; !ok {
			servicesMap[tag] = &ir.IRService{Tag: tag}
		}
		id := op.OperationID
		pathParams, queryParams := collectParams(doc, op)
		reqBody := extractRequestBody(doc, op)
		resp := extractResponse(doc, op)

		// Copy original tags, defaulting to ["misc"] if no tags
		originalTags := make([]string, len(op.Tags))
		copy(originalTags, op.Tags)
		if len(originalTags) == 0 {
			originalTags = []string{"misc"}
		}

		servicesMap[tag].Operations = append(servicesMap[tag].Operations, ir.IROperation{
			OperationID:  id,
			Method:       method,
			Path:         path,
			Tag:          tag,
			OriginalTags: originalTags,
			Summary:      op.Summary,
			Description:  op.Description,
			Deprecated:   op.Deprecated,
			PathParams:   pathParams,
			QueryParams:  queryParams,
			RequestBody:  reqBody,
			Response:     resp,
		})
	}

	for path, item := range doc.Paths.Map() {
		operations := []*openapi3.Operation{
			item.Get, item.Post, item.Put, item.Patch,
			item.Delete, item.Options, item.Head, item.Trace,
		}
		methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE"}

		for i, op := range operations {
			if op == nil {
				continue
			}
			t := firstAllowedTag(op.Tags, allowed)
			if t == "" {
				if len(op.Tags) == 0 && allowed["misc"] {
					t = "misc"
				}
			}
			if t != "" {
				addOp(t, op, methods[i], path)
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

// firstAllowedTag returns the first allowed tag from a list
func firstAllowedTag(tags []string, allowed map[string]bool) string {
	for _, t := range tags {
		if allowed[t] {
			return t
		}
	}
	return ""
}

// collectSecuritySchemes extracts security scheme information
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

// collectParams extracts parameters from an operation
func collectParams(doc *openapi3.T, op *openapi3.Operation) (pathParams, queryParams []ir.IRParam) {
	for _, pr := range op.Parameters {
		if pr == nil || pr.Value == nil {
			continue
		}
		p := pr.Value
		param := ir.IRParam{
			Name:        p.Name,
			Required:    p.Required,
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

// extractRequestBody extracts request body information
func extractRequestBody(doc *openapi3.T, op *openapi3.Operation) *ir.IRRequestBody {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	rb := op.RequestBody.Value
	// Prefer application/json
	if media, ok := rb.Content["application/json"]; ok {
		return &ir.IRRequestBody{
			ContentType: "application/json",
			TypeTS:      "",
			Schema:      schemaRefToIR(doc, media.Schema),
			Required:    rb.Required,
		}
	}
	if media, ok := rb.Content["application/x-www-form-urlencoded"]; ok {
		return &ir.IRRequestBody{
			ContentType: "application/x-www-form-urlencoded",
			TypeTS:      "",
			Schema:      schemaRefToIR(doc, media.Schema),
			Required:    rb.Required,
		}
	}
	if _, ok := rb.Content["multipart/form-data"]; ok {
		return &ir.IRRequestBody{
			ContentType: "multipart/form-data",
			TypeTS:      "",
			Schema:      ir.IRSchema{Kind: ir.IRKindUnknown},
			Required:    rb.Required,
		}
	}
	// Fallback to the first available media type
	for ct, media := range rb.Content {
		return &ir.IRRequestBody{
			ContentType: ct,
			TypeTS:      "",
			Schema:      schemaRefToIR(doc, media.Schema),
			Required:    rb.Required,
		}
	}
	return nil
}

// extractResponse extracts response information
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

	// Pre-populate seen with component names to prevent inline duplicates
	for _, name := range names {
		seen[name] = struct{}{}
	}

	for _, name := range names {
		sr := doc.Components.Schemas[name]
		// For component schemas, use schemaRefToIR to get the actual schema without creating inline models
		schema := schemaRefToIR(doc, sr)
		out = append(out, ir.IRModelDef{
			Name:        name,
			Schema:      schema,
			Annotations: extractAnnotations(sr),
		})
	}
	return out
}

// filterUnusedModelDefs removes ModelDefs that are not referenced by any operations
func filterUnusedModelDefs(filteredIR ir.IR, allModelDefs []ir.IRModelDef) []ir.IRModelDef {
	// Build a map of all ModelDefs for quick lookup
	modelDefMap := make(map[string]ir.IRModelDef)
	for _, md := range allModelDefs {
		modelDefMap[md.Name] = md
	}

	// Collect all schema references from filtered operations
	referenced := make(map[string]bool)
	visited := make(map[string]bool) // Track visited refs to avoid cycles

	// Helper function to collect references from a schema recursively
	var collectRefs func(schema ir.IRSchema)
	collectRefs = func(schema ir.IRSchema) {
		if schema.Kind == ir.IRKindRef && schema.Ref != "" {
			refName := schema.Ref
			referenced[refName] = true
			// If this ref points to a ModelDef and we haven't visited it, collect its transitive references
			if !visited[refName] {
				visited[refName] = true
				if md, ok := modelDefMap[refName]; ok {
					collectRefs(md.Schema)
				}
			}
		}
		if schema.Items != nil {
			collectRefs(*schema.Items)
		}
		if schema.AdditionalProperties != nil {
			collectRefs(*schema.AdditionalProperties)
		}
		for _, sub := range schema.OneOf {
			if sub != nil {
				collectRefs(*sub)
			}
		}
		for _, sub := range schema.AnyOf {
			if sub != nil {
				collectRefs(*sub)
			}
		}
		for _, sub := range schema.AllOf {
			if sub != nil {
				collectRefs(*sub)
			}
		}
		if schema.Not != nil {
			collectRefs(*schema.Not)
		}
		for _, field := range schema.Properties {
			if field.Type != nil {
				collectRefs(*field.Type)
			}
		}
	}

	// Collect references from all operations
	for _, service := range filteredIR.Services {
		for _, op := range service.Operations {
			// Collect from path params
			for _, param := range op.PathParams {
				collectRefs(param.Schema)
			}
			// Collect from query params
			for _, param := range op.QueryParams {
				collectRefs(param.Schema)
			}
			// Collect from request body
			if op.RequestBody != nil {
				collectRefs(op.RequestBody.Schema)
			}
			// Collect from response
			collectRefs(op.Response.Schema)
		}
	}

	// Filter ModelDefs to only include referenced ones
	filtered := make([]ir.IRModelDef, 0)
	for _, md := range allModelDefs {
		if referenced[md.Name] {
			filtered = append(filtered, md)
		}
	}

	return filtered
}

// This file is getting long - I'll need to continue with the schema conversion functions
// in the next part...
