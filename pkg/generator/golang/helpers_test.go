package golang

import (
	"testing"
)

func TestDefaultParseOperationID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"simpleMethod", "simpleMethod"},
		{"AuthorizationController_ListUserResources", "ListUserResources"},
		{"UserController_CreateUser", "CreateUser"},
		{"SomeController_GetById", "GetById"},
		{"NoController", "NoController"},
	}

	for _, test := range tests {
		result := defaultParseOperationID(test.input)
		if result != test.expected {
			t.Errorf("defaultParseOperationID(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "Hello"},
		{"helloWorld", "HelloWorld"},
		{"getUserById", "GetUserById"},
		{"XMLHttpRequest", "XmlHttpRequest"},
		{"listUserResources", "ListUserResources"},
		{"createUsersWithListInput", "CreateUsersWithListInput"},
		{"hello-world", "HelloWorld"},
		{"hello_world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"HELLO_WORLD", "HelloWorld"},
	}

	for _, test := range tests {
		result := toPascalCase(test.input)
		if result != test.expected {
			t.Errorf("toPascalCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"hello", []string{"hello"}},
		{"helloWorld", []string{"hello", "World"}},
		{"getUserById", []string{"get", "User", "By", "Id"}},
		{"listUserResources", []string{"list", "User", "Resources"}},
	}

	for _, test := range tests {
		result := splitCamelCase(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("splitCamelCase(%q) = %v, expected %v", test.input, result, test.expected)
			continue
		}
		for i, part := range result {
			if part != test.expected[i] {
				t.Errorf("splitCamelCase(%q) = %v, expected %v", test.input, result, test.expected)
				break
			}
		}
	}
}
