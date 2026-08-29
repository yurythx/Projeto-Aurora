package gazette

import "testing"

func TestIsPlausiblePersonName(t *testing.T) {
	valid := []string{
		"YURI SILVA SANTOS",
		"Maria de Souza Oliveira",
		"João Batista Ramos",
		"ANA PAULA DA CONCEIÇÃO",
	}
	for _, s := range valid {
		if !IsPlausiblePersonName(s) {
			t.Errorf("IsPlausiblePersonName(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",
		"Fulano",                            // 1 palavra
		"de Grosso",                         // 1 palavra real
		"das Quantidades",                   // 1 palavra real
		"DE AGOSTO DE",                      // 0 palavras reais
		"CARLOS 123",                        // dígito
		"AGROBEN COMÉRCIO DE PRODUTOS LTDA", // razão social
		"DIÁRIO OFICIAL ELETRÔNICO DE RONDONÓPOLIS", // cabeçalho
		"PROCESSO DE DISPENSA DE LICITAÇÃO",         // ato de compras
		"Secretaria Municipal de Saúde",             // órgão
		"DIVULGAÇÃO DOS LOCAIS DE PROVA",            // fragmento de procedimento
		"DA MANIFESTAÇÃO DE INTERESSE",              // fragmento de procedimento
	}
	for _, s := range invalid {
		if IsPlausiblePersonName(s) {
			t.Errorf("IsPlausiblePersonName(%q) = true, want false", s)
		}
	}
}

func TestLooksLikeOrganization(t *testing.T) {
	orgs := []string{
		"AGROBEN COMÉRCIO DE PRODUTOS AGROPECUÁRIOS LTDA",
		"CONSTRUTORA XYZ EIRELI",
		"12.345.678/0001-90",
		"Instituto Municipal de Previdência",
		"PROCESSO DE INEXIGIBILIDADE DE LICITAÇÃO",
	}
	for _, s := range orgs {
		if !LooksLikeOrganization(s) {
			t.Errorf("LooksLikeOrganization(%q) = false, want true", s)
		}
	}
	if LooksLikeOrganization("Maria de Souza Oliveira") {
		t.Error("LooksLikeOrganization(pessoa física) = true, want false")
	}
}
