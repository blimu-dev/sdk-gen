package ir

type IROperation struct {
	OperationID string
	Method      string
	Path        string
	Tag         string
	Summary     string
	Description string
	Deprecated  bool
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
	// ModelDefs holds a language-agnostic structured representation of components schemas
	ModelDefs []IRModelDef
}

type IRParam struct {
	Name     string
	Required bool
	Schema   IRSchema
	// Description from the OpenAPI parameter
	Description string
}

type IRRequestBody struct {
	ContentType string
	TypeTS      string
	Required    bool
	Schema      IRSchema
}

type IRResponse struct {
	TypeTS string
	Schema IRSchema
	// Description contains the response description chosen for this operation
	Description string
}

type IRModel struct {
	Name string
	Decl string
}

// IRModelDef represents a named model (typically a component or a generated inline type)
// with a structured schema that is language-agnostic.
type IRModelDef struct {
	Name        string
	Schema      IRSchema
	Annotations IRAnnotations
}

// IRAnnotations captures non-structural metadata that some generators may render.
type IRAnnotations struct {
	Title       string
	Description string
	Deprecated  bool
	ReadOnly    bool
	WriteOnly   bool
	Default     any
	Examples    []any
}

type IRSchemaKind string

const (
	IRKindUnknown IRSchemaKind = "unknown"
	IRKindString  IRSchemaKind = "string"
	IRKindNumber  IRSchemaKind = "number"
	IRKindInteger IRSchemaKind = "integer"
	IRKindBoolean IRSchemaKind = "boolean"
	IRKindNull    IRSchemaKind = "null"
	IRKindArray   IRSchemaKind = "array"
	IRKindObject  IRSchemaKind = "object"
	IRKindEnum    IRSchemaKind = "enum"
	IRKindRef     IRSchemaKind = "ref"
	IRKindOneOf   IRSchemaKind = "oneOf"
	IRKindAnyOf   IRSchemaKind = "anyOf"
	IRKindAllOf   IRSchemaKind = "allOf"
	IRKindNot     IRSchemaKind = "not"
)

// IRSchema models a JSON Schema (as used by OpenAPI 3.1) shape in a language-agnostic way
type IRSchema struct {
	Kind     IRSchemaKind
	Nullable bool
	Format   string

	// Object
	Properties           []IRField
	AdditionalProperties *IRSchema // typed maps; nil when absent

	// Array
	Items *IRSchema

	// Enum
	EnumValues []string     // stringified values for portability
	EnumRaw    []any        // original values preserving type where possible
	EnumBase   IRSchemaKind // underlying base kind: string, number, integer, boolean, unknown

	// Ref (component name or canonical name)
	Ref string

	// Compositions
	OneOf []*IRSchema
	AnyOf []*IRSchema
	AllOf []*IRSchema
	Not   *IRSchema

	// Polymorphism
	Discriminator *IRDiscriminator
}

type IRField struct {
	Name     string
	Type     *IRSchema
	Required bool
	// Pass-through annotations commonly used by generators
	Annotations IRAnnotations
}

type IRDiscriminator struct {
	PropertyName string
	Mapping      map[string]string
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
