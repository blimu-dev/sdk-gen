package golang

import (
	"testing"

	"github.com/blimu-dev/sdk-gen/pkg/utils"
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
		result := toPascalCase(test.input)
		if result != test.expected {
			t.Errorf("toPascalCase(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

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
		result := utils.RemoveAccents(test.input)
		if result != test.expected {
			t.Errorf("removeAccents(%q) = %q, expected %q", test.input, result, test.expected)
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
		result := toSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("toSnakeCase(%q) = %q, expected %q", test.input, result, test.expected)
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
		result := utils.SplitCamelCase(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("utils.SplitCamelCase(%q) = %v, expected %v", test.input, result, test.expected)
			continue
		}
		for i, part := range result {
			if part != test.expected[i] {
				t.Errorf("utils.SplitCamelCase(%q) = %v, expected %v", test.input, result, test.expected)
				break
			}
		}
	}
}

func TestFormatGoComment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"Simple comment", "// Simple comment"},
		{"Line 1\nLine 2", "// Line 1\n// Line 2"},
		{"Line 1\n\nLine 3", "// Line 1\n//\n// Line 3"},
		{"Especifica quantos dias antes do vencimento a notificação deve se enviada.\n Para o evento  `PAYMENT_DUEDATE_WARNING` os valores aceitos são: `0`, `5`, `10`, `15` e `30`\n Para o evento `PAYMENT_OVERDUE` os valores aceitos são: `1`, `7`, `15` e `30`", "// Especifica quantos dias antes do vencimento a notificação deve se enviada.\n// Para o evento  `PAYMENT_DUEDATE_WARNING` os valores aceitos são: `0`, `5`, `10`, `15` e `30`\n// Para o evento `PAYMENT_OVERDUE` os valores aceitos são: `1`, `7`, `15` e `30`"},
	}

	for _, test := range tests {
		result := formatGoComment(test.input)
		if result != test.expected {
			t.Errorf("formatGoComment(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
