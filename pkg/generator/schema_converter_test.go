package generator

import (
	"testing"
)

func TestToPascal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "Hello"},
		{"helloWorld", "HelloWorld"},
		{"additionalProperties", "AdditionalProperties"},
		{"Properties", "Properties"},
		{"userResources", "UserResources"},
		{"listUserResources", "ListUserResources"},
		{"createUsersWithListInput", "CreateUsersWithListInput"},
		{"XMLHttpRequest", "XmlHttpRequest"},
		{"hello-world", "HelloWorld"},
		{"hello_world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"HELLO_WORLD", "HelloWorld"},
	}

	for _, test := range tests {
		result := toPascal(test.input)
		if result != test.expected {
			t.Errorf("toPascal(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSplitCamelCaseSchema(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"hello", []string{"hello"}},
		{"helloWorld", []string{"hello", "World"}},
		{"additionalProperties", []string{"additional", "Properties"}},
		{"getUserById", []string{"get", "User", "By", "Id"}},
		{"listUserResources", []string{"list", "User", "Resources"}},
	}

	for _, test := range tests {
		result := splitCamelCaseSchema(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("splitCamelCaseSchema(%q) = %v, expected %v", test.input, result, test.expected)
			continue
		}
		for i, part := range result {
			if part != test.expected[i] {
				t.Errorf("splitCamelCaseSchema(%q) = %v, expected %v", test.input, result, test.expected)
				break
			}
		}
	}
}
