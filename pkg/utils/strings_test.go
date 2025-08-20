package utils

import (
	"testing"
)

func TestRemoveAccents(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"cobrança", "cobranca"},
		{"negociação", "negociacao"},
		{"transferências", "transferencias"},
		{"antecipações", "antecipacoes"},
		{"informações", "informacoes"},
		{"configurações", "configuracoes"},
		{"notificações", "notificacoes"},
		{"café", "cafe"},
		{"açúcar", "acucar"},
		{"pão", "pao"},
		{"José", "Jose"},
		{"São Paulo", "Sao Paulo"},
		{"résumé", "resume"},
		{"naïve", "naive"},
		{"piñata", "pinata"},
	}

	for _, test := range tests {
		result := RemoveAccents(test.input)
		if result != test.expected {
			t.Errorf("RemoveAccents(%q) = %q, expected %q", test.input, result, test.expected)
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
		{"XMLHttpRequest", "XmlhttpRequest"},
		{"listUserResources", "ListUserResources"},
		{"createUsersWithListInput", "CreateUsersWithListInput"},
		{"hello-world", "HelloWorld"},
		{"hello_world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"HELLO_WORLD", "HelloWorld"},
		// Test accent removal
		{"cobrança", "Cobranca"},
		{"negociação", "Negociacao"},
		{"pagamentos", "Pagamentos"},
		{"transferências", "Transferencias"},
		{"antecipações", "Antecipacoes"},
		{"informações", "Informacoes"},
		{"configurações", "Configuracoes"},
		{"notificações", "Notificacoes"},
	}

	for _, test := range tests {
		result := ToPascalCase(test.input)
		if result != test.expected {
			t.Errorf("ToPascalCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToPascalCaseAdvanced(t *testing.T) {
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
		// Test accent removal
		{"cobrança", "Cobranca"},
		{"negociação", "Negociacao"},
		{"pagamentos", "Pagamentos"},
		{"transferências", "Transferencias"},
		{"antecipações", "Antecipacoes"},
		{"informações", "Informacoes"},
		{"configurações", "Configuracoes"},
		{"notificações", "Notificacoes"},
	}

	for _, test := range tests {
		result := ToPascalCaseAdvanced(test.input)
		if result != test.expected {
			t.Errorf("ToPascalCaseAdvanced(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"Hello", "hello"},
		{"helloWorld", "helloWorld"},
		{"getUserById", "getUserById"},
		{"hello-world", "helloWorld"},
		{"hello_world", "helloWorld"},
		{"hello world", "helloWorld"},
		{"HELLO_WORLD", "helloWorld"},
		// Test accent removal
		{"cobrança", "cobranca"},
		{"Negociação", "negociacao"},
	}

	for _, test := range tests {
		result := ToCamelCase(test.input)
		if result != test.expected {
			t.Errorf("ToCamelCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"helloWorld", "hello_world"},
		{"getUserById", "get_user_by_id"},
		{"XMLHttpRequest", "xmlhttp_request"},
		{"hello-world", "hello_world"},
		{"hello_world", "hello_world"},
		{"hello world", "hello_world"},
		{"HELLO_WORLD", "hello_world"},
		// Test accent removal
		{"cobrança", "cobranca"},
		{"negociação", "negociacao"},
		{"transferências", "transferencias"},
		{"antecipações", "antecipacoes"},
		{"informações", "informacoes"},
		{"configurações", "configuracoes"},
		{"notificações", "notificacoes"},
	}

	for _, test := range tests {
		result := ToSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("ToSnakeCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToSnakeCaseAdvanced(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"helloWorld", "hello_world"},
		{"getUserById", "get_user_by_id"},
		{"XMLHttpRequest", "xml_http_request"},
		{"hello-world", "hello_world"},
		{"hello_world", "hello_world"},
		{"hello world", "hello_world"},
		{"HELLO_WORLD", "hello_world"},
		// Test accent removal
		{"cobrança", "cobranca"},
		{"negociação", "negociacao"},
		{"transferências", "transferencias"},
		{"antecipações", "antecipacoes"},
		{"informações", "informacoes"},
		{"configurações", "configuracoes"},
		{"notificações", "notificacoes"},
	}

	for _, test := range tests {
		result := ToSnakeCaseAdvanced(test.input)
		if result != test.expected {
			t.Errorf("ToSnakeCaseAdvanced(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"helloWorld", "hello-world"},
		{"getUserById", "get-user-by-id"},
		{"XMLHttpRequest", "xmlhttp-request"},
		{"hello-world", "hello-world"},
		{"hello_world", "hello-world"},
		{"hello world", "hello-world"},
		{"HELLO_WORLD", "hello-world"},
		// Test accent removal
		{"cobrança", "cobranca"},
		{"negociação", "negociacao"},
		{"transferências", "transferencias"},
	}

	for _, test := range tests {
		result := ToKebabCase(test.input)
		if result != test.expected {
			t.Errorf("ToKebabCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"hello", []string{"hello"}},
		{"helloWorld", []string{"hello", "World"}},
		{"getUserById", []string{"get", "User", "By", "Id"}},
		{"XMLHttpRequest", []string{"XMLHttp", "Request"}},
		{"hello-world", []string{"hello", "world"}},
		{"hello_world", []string{"hello", "world"}},
		{"hello world", []string{"hello", "world"}},
		// Test accent removal
		{"cobrança", []string{"cobranca"}},
		{"negociação", []string{"negociacao"}},
	}

	for _, test := range tests {
		result := SplitWords(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("SplitWords(%q) = %v, expected %v", test.input, result, test.expected)
			continue
		}
		for i, part := range result {
			if part != test.expected[i] {
				t.Errorf("SplitWords(%q) = %v, expected %v", test.input, result, test.expected)
				break
			}
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
		{"XMLHttpRequest", []string{"XML", "Http", "Request"}},
	}

	for _, test := range tests {
		result := SplitCamelCase(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("SplitCamelCase(%q) = %v, expected %v", test.input, result, test.expected)
			continue
		}
		for i, part := range result {
			if part != test.expected[i] {
				t.Errorf("SplitCamelCase(%q) = %v, expected %v", test.input, result, test.expected)
				break
			}
		}
	}
}
