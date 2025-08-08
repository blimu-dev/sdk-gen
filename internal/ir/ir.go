package ir

type IROperation struct {
	OperationID string
	Method      string
	Path        string
	Tag         string
	PathParams  []IRParam
	QueryParams []IRParam
	RequestBody *IRRequestBody
	Response    IRResponse
}

type IRService struct {
	Tag        string
	Operations []IROperation
}

type IR struct {
	Services        []IRService
	Models          []IRModel
	SecuritySchemes []IRSecurityScheme
}

type IRParam struct {
	Name     string
	Required bool
	TypeTS   string
}

type IRRequestBody struct {
	ContentType string
	TypeTS      string
	Required    bool
}

type IRResponse struct {
	TypeTS string
}

type IRModel struct {
	Name string
	Decl string
}

// IRSecurityScheme captures a simplified view of OpenAPI security schemes
// sufficient for SDK generation.
type IRSecurityScheme struct {
	// Key is the name of the security scheme in components.securitySchemes
	Key string
	// Type is one of: http, apiKey, oauth2, openIdConnect
	Type string
	// Scheme is used when Type is http (e.g., "basic", "bearer")
	Scheme string
	// In is used when Type is apiKey (e.g., "header", "query", "cookie")
	In string
	// Name is used when Type is apiKey; it is the header/query/cookie name
	Name string
	// BearerFormat may be provided for bearer tokens
	BearerFormat string
}
