package openapi

import (
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadDocument loads an OpenAPI document from a local file path or an HTTP(S) URL
func LoadDocument(input string) (*openapi3.T, error) {
	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	return LoadDocumentWithLoader(loader, input)
}

// LoadDocumentWithLoader loads an OpenAPI document using a custom loader
func LoadDocumentWithLoader(loader *openapi3.Loader, input string) (*openapi3.T, error) {
	// Try to parse as URL; if it looks like http(s), fetch via URL
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return loader.LoadFromURI(u)
	}
	// Fallback to reading from filesystem path
	return loader.LoadFromFile(input)
}

// ValidateDocument validates an OpenAPI document
func ValidateDocument(input string) error {
	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	doc, err := LoadDocumentWithLoader(loader, input)
	if err != nil {
		return err
	}
	return doc.Validate(loader.Context)
}
