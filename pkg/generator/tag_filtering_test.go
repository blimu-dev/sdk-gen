package generator

import (
	"regexp"
	"testing"
)

func TestShouldIncludeOperation(t *testing.T) {
	tests := []struct {
		name         string
		originalTags []string
		includeTags  []string
		excludeTags  []string
		expected     bool
		description  string
	}{
		{
			name:         "no filters - include all",
			originalTags: []string{"users", "internal"},
			includeTags:  []string{},
			excludeTags:  []string{},
			expected:     true,
			description:  "When no filters are specified, all operations should be included",
		},
		{
			name:         "include filter matches first tag",
			originalTags: []string{"users", "internal"},
			includeTags:  []string{"users"},
			excludeTags:  []string{},
			expected:     true,
			description:  "Operation should be included when first tag matches include filter",
		},
		{
			name:         "include filter matches second tag",
			originalTags: []string{"internal", "users"},
			includeTags:  []string{"users"},
			excludeTags:  []string{},
			expected:     true,
			description:  "Operation should be included when any tag matches include filter (this is the main fix)",
		},
		{
			name:         "include filter matches none",
			originalTags: []string{"internal", "admin"},
			includeTags:  []string{"users"},
			excludeTags:  []string{},
			expected:     false,
			description:  "Operation should be excluded when no tags match include filter",
		},
		{
			name:         "exclude filter matches first tag",
			originalTags: []string{"internal", "users"},
			includeTags:  []string{},
			excludeTags:  []string{"internal"},
			expected:     false,
			description:  "Operation should be excluded when any tag matches exclude filter",
		},
		{
			name:         "exclude filter matches second tag",
			originalTags: []string{"users", "internal"},
			includeTags:  []string{},
			excludeTags:  []string{"internal"},
			expected:     false,
			description:  "Operation should be excluded when any tag matches exclude filter",
		},
		{
			name:         "include and exclude both match different tags",
			originalTags: []string{"users", "internal"},
			includeTags:  []string{"users"},
			excludeTags:  []string{"internal"},
			expected:     false,
			description:  "Exclude should take precedence over include",
		},
		{
			name:         "include matches, exclude doesn't",
			originalTags: []string{"users", "public"},
			includeTags:  []string{"users"},
			excludeTags:  []string{"internal"},
			expected:     true,
			description:  "Operation should be included when include matches and exclude doesn't",
		},
		{
			name:         "regex patterns work",
			originalTags: []string{"users_v1", "internal_api"},
			includeTags:  []string{"^users_.*"},
			excludeTags:  []string{".*_api$"},
			expected:     false,
			description:  "Regex patterns should work for both include and exclude",
		},
		{
			name:         "regex include matches",
			originalTags: []string{"users_v1", "public"},
			includeTags:  []string{"^users_.*"},
			excludeTags:  []string{},
			expected:     true,
			description:  "Regex include patterns should work",
		},
		{
			name:         "multiple include patterns - any match",
			originalTags: []string{"orders", "billing"},
			includeTags:  []string{"users", "orders"},
			excludeTags:  []string{},
			expected:     true,
			description:  "Operation should be included if any tag matches any include pattern",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Compile regex patterns
			var includeRegexes []*regexp.Regexp
			for _, pattern := range test.includeTags {
				r, err := regexp.Compile(pattern)
				if err != nil {
					t.Fatalf("Invalid include regex pattern %q: %v", pattern, err)
				}
				includeRegexes = append(includeRegexes, r)
			}

			var excludeRegexes []*regexp.Regexp
			for _, pattern := range test.excludeTags {
				r, err := regexp.Compile(pattern)
				if err != nil {
					t.Fatalf("Invalid exclude regex pattern %q: %v", pattern, err)
				}
				excludeRegexes = append(excludeRegexes, r)
			}

			result := shouldIncludeOperation(test.originalTags, includeRegexes, excludeRegexes)
			if result != test.expected {
				t.Errorf("shouldIncludeOperation(%v, %v, %v) = %v, expected %v\nDescription: %s",
					test.originalTags, test.includeTags, test.excludeTags, result, test.expected, test.description)
			}
		})
	}
}
