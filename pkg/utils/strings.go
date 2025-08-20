package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	nonAlnum   = regexp.MustCompile(`[^A-Za-z0-9]+`)
	camelSplit = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// RemoveAccents removes accents from a string, converting accented characters to their base forms
func RemoveAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

// SplitWords splits a string into words, handling camelCase, PascalCase, snake_case, and kebab-case
func SplitWords(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Remove accents first
	s = RemoveAccents(s)

	// First, handle camelCase/PascalCase by inserting separators before capital letters
	s = camelSplit.ReplaceAllString(s, "$1 $2")

	// Then split on non-alphanumeric characters and spaces
	parts := regexp.MustCompile(`[^A-Za-z0-9]+`).Split(s, -1)

	// Filter out empty parts
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// SplitCamelCase splits a camelCase or PascalCase string into words
// This is used by the Go generator which has a more sophisticated approach
func SplitCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	var current strings.Builder

	runes := []rune(s)
	for i, r := range runes {
		// Check if this is the start of a new word
		isNewWord := false
		if i > 0 && isUppercase(r) {
			// Current char is uppercase
			if !isUppercase(runes[i-1]) {
				// Previous char was lowercase, so this starts a new word
				isNewWord = true
			} else if i < len(runes)-1 && !isUppercase(runes[i+1]) {
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

// isUppercase checks if a rune is uppercase
func isUppercase(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	parts := SplitWords(s)
	if len(parts) == 0 {
		return ""
	}

	b := strings.Builder{}
	for _, p := range parts {
		if p == "" {
			continue
		}
		// Capitalize first letter, keep rest of the word as lowercase
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(strings.ToLower(p[1:]))
		}
	}
	return b.String()
}

// ToPascalCaseAdvanced converts a string to PascalCase using the more sophisticated Go approach
func ToPascalCaseAdvanced(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Remove accents first
	s = RemoveAccents(s)

	// First split by non-alphanumeric characters
	parts := nonAlnum.Split(s, -1)
	var allParts []string

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Further split camelCase/PascalCase words
		subParts := SplitCamelCase(part)
		allParts = append(allParts, subParts...)
	}

	if len(allParts) == 0 {
		return ""
	}

	var result strings.Builder
	for _, part := range allParts {
		if part == "" {
			continue
		}
		// Capitalize first letter, lowercase the rest
		if len(part) == 1 {
			result.WriteString(strings.ToUpper(part))
		} else {
			result.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
		}
	}
	return result.String()
}

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	p := ToPascalCase(s)
	if p == "" {
		return ""
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// ToCamelCaseAdvanced converts a string to camelCase using the more sophisticated Go approach
func ToCamelCaseAdvanced(s string) string {
	p := ToPascalCaseAdvanced(s)
	if p == "" {
		return ""
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	parts := SplitWords(s)
	if len(parts) == 0 {
		return ""
	}

	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}

// ToSnakeCaseAdvanced converts a string to snake_case using the more sophisticated Go approach
func ToSnakeCaseAdvanced(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Remove accents first
	s = RemoveAccents(s)

	// First split by non-alphanumeric characters
	parts := nonAlnum.Split(s, -1)
	var allParts []string

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Further split camelCase/PascalCase words
		subParts := SplitCamelCase(part)
		allParts = append(allParts, subParts...)
	}

	if len(allParts) == 0 {
		return ""
	}

	// Convert all parts to lowercase and join with underscores
	for i := range allParts {
		allParts[i] = strings.ToLower(allParts[i])
	}
	return strings.Join(allParts, "_")
}

// ToKebabCase converts a string to kebab-case
func ToKebabCase(s string) string {
	parts := SplitWords(s)
	if len(parts) == 0 {
		return ""
	}

	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "-")
}

// ToKebabCaseAdvanced converts a string to kebab-case using the more sophisticated Go approach
func ToKebabCaseAdvanced(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Remove accents first
	s = RemoveAccents(s)

	// First split by non-alphanumeric characters
	parts := nonAlnum.Split(s, -1)
	var allParts []string

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Further split camelCase/PascalCase words
		subParts := SplitCamelCase(part)
		allParts = append(allParts, subParts...)
	}

	if len(allParts) == 0 {
		return ""
	}

	// Convert all parts to lowercase and join with hyphens
	for i := range allParts {
		allParts[i] = strings.ToLower(allParts[i])
	}
	return strings.Join(allParts, "-")
}
