package gazette

import (
	"testing"
	"time"
)

func TestParsePersonnelActs_BlockIsolationAndFullName(t *testing.T) {
	rawGazetteText := `
DIÁRIO OFICIAL DE RONDONÓPOLIS - EDICÃO N.º 5.432

PORTARIA N.º 1.234/2026
O Prefeito Municipal, no uso de suas atribuições, resolve:
Designar o servidor YURI SILVA SANTOS, CPF 123.456.789-00, matrícula 9876, para exercer a função de Coordenador de TI, símbolo DAS-1.
PUBLIQUE-SE.

PORTARIA N.º 1.235/2026
O Prefeito Municipal resolve:
Relotar a servidora MARIA DE SOUZA OLIVEIRA para a Secretaria Municipal de Saúde, símbolo DAS-2.
REGISTRE-SE E CUMPRA-SE.
`

	pubDate := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	acts := ParsePersonnelActs(rawGazetteText, 5432, "Ordinária", pubDate, 1, "http://localhost/pdf/5432.pdf")

	if len(acts) != 2 {
		t.Fatalf("Esperado 2 atos isolados por bloco de portaria, obteve %d", len(acts))
	}

	// Teste Ato 1: DESIGNACAO_FUNCAO
	act1 := acts[0]
	if act1.ActType != "DESIGNACAO_FUNCAO" {
		t.Errorf("Ato 1: ActType esperado DESIGNACAO_FUNCAO, obteve %s", act1.ActType)
	}
	if act1.PersonName != "YURI SILVA SANTOS" {
		t.Errorf("Ato 1: Nome completo sem truncar esperado 'YURI SILVA SANTOS', obteve '%s'", act1.PersonName)
	}
	if act1.PortariaNumber != "1.234/2026" {
		t.Errorf("Ato 1: Portaria esperada '1.234/2026', obteve '%s'", act1.PortariaNumber)
	}
	if act1.DASLevel != "DAS-1" {
		t.Errorf("Ato 1: DASLevel esperado DAS-1, obteve '%s'", act1.DASLevel)
	}
	if act1.SalaryValue != 9800.00 {
		t.Errorf("Ato 1: Salário vinculado por constante esperado 9800.00, obteve %.2f", act1.SalaryValue)
	}

	// Teste Ato 2: RELOTACAO
	act2 := acts[1]
	if act2.ActType != "RELOTACAO" {
		t.Errorf("Ato 2: ActType esperado RELOTACAO, obteve %s", act2.ActType)
	}
	if act2.PersonName != "MARIA DE SOUZA OLIVEIRA" {
		t.Errorf("Ato 2: Nome completo sem truncar esperado 'MARIA DE SOUZA OLIVEIRA', obteve '%s'", act2.PersonName)
	}
	if act2.Secretaria != "Secretaria Municipal de Saúde" {
		t.Errorf("Ato 2: Secretaria de destino limpa esperada 'Secretaria Municipal de Saúde', obteve '%s'", act2.Secretaria)
	}
	if act2.DASLevel != "DAS-2" {
		t.Errorf("Ato 2: DASLevel esperado DAS-2, obteve '%s'", act2.DASLevel)
	}
	if act2.SalaryValue != 7450.00 {
		t.Errorf("Ato 2: Salário vinculado por constante esperado 7450.00, obteve %.2f", act2.SalaryValue)
	}
}

func TestFilterNameStopWords_TrimsTrailingConnective(t *testing.T) {
	cases := map[string]string{
		"Jailton De Lucena Da":       "Jailton De Lucena",
		"Sandra Helena Do":           "Sandra Helena",
		"Maria De Souza Oliveira":    "Maria De Souza Oliveira",
		"Ana Paula Da Conceicao E":   "Ana Paula Da Conceicao",
		"Fulano De Tal Portaria 123": "Fulano De Tal", // corta em "Portaria" e depois o "De" pendurado
	}
	for in, want := range cases {
		if got := filterNameStopWords(in); got != want {
			t.Errorf("filterNameStopWords(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyActType_ExtraVerbs(t *testing.T) {
	cases := map[string]ActType{
		"RESOLVE DISPENSAR O SERVIDOR X DA FUNÇÃO GRATIFICADA DE CHEFE":     ActExoneracao,
		"RESOLVE PRORROGAR O CONTRATO TEMPORÁRIO DA SERVIDORA Y":            ActContratacaoTemporaria,
		"RATIFICA O PROCESSO DE DISPENSA DE LICITAÇÃO Nº 12/2025":           ActOutros,
		"RESOLVE NOMEAR FULANO EM VIRTUDE DE APROVAÇÃO EM CONCURSO PÚBLICO": ActNomeacaoEfetivo,
	}
	for block, want := range cases {
		if got := classifyActType(block); got != want {
			t.Errorf("classifyActType(%q) = %q, want %q", block, got, want)
		}
	}
}

func TestExtractSecretaria_Siglas(t *testing.T) {
	if got := extractSecretaria("lotação na SEMED para exercer função", "DESIGNACAO_FUNCAO"); got != "SEMED" {
		t.Errorf("sigla SEMED: got %q", got)
	}
	if got := extractSecretaria("junto ao Gabinete do Prefeito.", "DESIGNACAO_FUNCAO"); got != "Gabinete do Prefeito" {
		t.Errorf("Gabinete: got %q", got)
	}
	if got := extractSecretaria("texto sem orgao nenhum aqui", "EXONERACAO"); got != "" {
		t.Errorf("sem match deveria ser vazio, got %q", got)
	}
}

func TestGetSalaryByDAS(t *testing.T) {
	tests := []struct {
		dasLevel string
		expected float64
	}{
		{"DAS-1", 9800.00},
		{"das-2", 7450.00},
		{"DAS-3", 5800.00},
		{"DAS-4", 4200.00},
		{"DAS-5", 3100.00},
		{"DAS-6", 2200.00},
		{"INVALID", 0.0},
	}

	for _, tc := range tests {
		got := GetSalaryByDAS(tc.dasLevel)
		if got != tc.expected {
			t.Errorf("GetSalaryByDAS(%s): esperado %.2f, obteve %.2f", tc.dasLevel, tc.expected, got)
		}
	}
}
