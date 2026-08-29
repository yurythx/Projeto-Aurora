package gazette

import (
	"testing"
	"time"
)

func TestParseContractExtracts(t *testing.T) {
	text := `PREFEITURA MUNICIPAL DE RONDONÓPOLIS

EXTRATO DE CONTRATO Nº 140/2026
CONTRATANTE: Município de Rondonópolis.
CONTRATADA: AGROBEN COMÉRCIO DE PRODUTOS AGROPECUÁRIOS LTDA, CNPJ 41.987.654/0001-22.
OBJETO: aquisição de insumos agrícolas para a Secretaria Municipal de Agricultura.
VALOR: R$ 950.000,00.
VIGÊNCIA: 12 (doze) meses a contar da assinatura.
Fiscal Titular: MARIANA COSTA PEREIRA, matrícula nº 9911.

EXTRATO DE 1º TERMO ADITIVO AO CONTRATO Nº 88/2025
CONTRATADA: CONSTRUTORA XYZ EIRELI, CNPJ 12.345.678/0001-90.
OBJETO: prorrogação de prazo.
VALOR: R$ 120.000,50.

AVISO DE LICITAÇÃO - texto solto sem extrato nenhum, não deve virar contrato.`

	got := ParseContractExtracts(text, 5)
	if len(got) != 2 {
		t.Fatalf("esperava 2 extratos, got %d: %+v", len(got), got)
	}

	c1 := got[0]
	if c1.ContractNumber != "140/2026" {
		t.Errorf("c1 número = %q, want 140/2026", c1.ContractNumber)
	}
	if c1.CNPJ != "41.987.654/0001-22" {
		t.Errorf("c1 CNPJ = %q", c1.CNPJ)
	}
	if c1.Contratada == "" || c1.Contratada[:7] != "AGROBEN" {
		t.Errorf("c1 contratada = %q", c1.Contratada)
	}
	if c1.Valor != 950000.00 {
		t.Errorf("c1 valor = %.2f, want 950000.00", c1.Valor)
	}
	if c1.FiscalNome != "Mariana Costa Pereira" && c1.FiscalNome != "MARIANA COSTA PEREIRA" {
		t.Errorf("c1 fiscal = %q", c1.FiscalNome)
	}
	if c1.FiscalMatricula != "9911" {
		t.Errorf("c1 fiscal matrícula = %q", c1.FiscalMatricula)
	}
	if c1.Objeto == "" {
		t.Errorf("c1 objeto vazio")
	}

	c2 := got[1]
	if c2.ContractNumber != "88/2025" {
		t.Errorf("c2 número = %q, want 88/2025", c2.ContractNumber)
	}
	if c2.Valor != 120000.50 {
		t.Errorf("c2 valor = %.2f, want 120000.50", c2.Valor)
	}
}

func TestParseTemporaryHires(t *testing.T) {
	text := `DEPARTAMENTO DE RECURSOS HUMANOS EXTRATO DO CONTRATO INDIVIDUAL DE
TRABALHO POR TEMPO DETERMINADO N°: 3750/2026
Objeto: CONTRATAÇÃO DE GIOVANNA ROSA CAMPOS CALACA, PSICOLOGO, EM CARÁTER TEMPORÁRIO.
Contratado(a): GIOVANNA ROSA CAMPOS CALACA
Cargo: PSICOLOGO
Remuneração Mensal: 5.538,28
Vigência: 19/08/2026 até 18/08/2027
Signatários: MUNICIPIO DE RONDONOPOLIS e GIOVANNA ROSA CAMPOS CALACA.

DEPARTAMENTO DE RECURSOS HUMANOS EXTRATO DO CONTRATO INDIVIDUAL DE
TRABALHO POR TEMPO DETERMINADO N°: 3628/2026
Contratado(a): DOUGLAS GABRIEL SOUSA DOS SANTOS
Cargo: ESTAGIÁRIO
Remuneração Mensal: R$ 1.410,27
Vigência: 03/08/2026 até 31/12/2026`

	pubDate := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	got := ParseTemporaryHires(text, 6260, "ORDINARIA", pubDate, 4, "http://x/6260.pdf")
	if len(got) != 2 {
		t.Fatalf("esperava 2, got %d: %+v", len(got), got)
	}
	if got[0].ActType != ActContratacaoTemporaria {
		t.Errorf("act_type = %q", got[0].ActType)
	}
	if got[0].PersonName != "GIOVANNA ROSA CAMPOS CALACA" {
		t.Errorf("nome = %q", got[0].PersonName)
	}
	if got[0].JobRole != "PSICOLOGO" {
		t.Errorf("cargo = %q", got[0].JobRole)
	}
	if got[0].SalaryValue != 5538.28 {
		t.Errorf("salário = %.2f, want 5538.28", got[0].SalaryValue)
	}
	if got[0].PortariaNumber != "3750/2026" {
		t.Errorf("ref = %q, want 3750/2026", got[0].PortariaNumber)
	}
	if got[0].PDFPageNumber != 4 {
		t.Errorf("página = %d", got[0].PDFPageNumber)
	}
	if got[1].SalaryValue != 1410.27 {
		t.Errorf("c2 salário = %.2f, want 1410.27", got[1].SalaryValue)
	}
}

func TestParseFiscalDesignations(t *testing.T) {
	// Formato real (ed. 6260), com a matrícula quebrada em duas linhas.
	text := `Art. 1º Designar o servidor VERSIANE DE OLIVEIRA, matrícula nº 612097-
3, como Fiscal de Contrato (Titular), e a servidora, THAYANNE CORREA DE OLIVEIRA
matrícula nº 1560477002, como Fiscal de Contrato (Suplente), responsáveis pelo
acompanhamento e fiscalização dos contratos relacionados abaixo:`

	got := ParseFiscalDesignations(text, 7)
	if len(got) != 1 {
		t.Fatalf("esperava 1 par, got %d: %+v", len(got), got)
	}
	fp := got[0]
	if fp.TitularNome != "VERSIANE DE OLIVEIRA" {
		t.Errorf("titular = %q", fp.TitularNome)
	}
	if fp.TitularMatricula != "6120973" {
		t.Errorf("titular matrícula = %q, want 6120973 (linha quebrada reconstruída)", fp.TitularMatricula)
	}
	if fp.SuplenteNome != "THAYANNE CORREA DE OLIVEIRA" {
		t.Errorf("suplente = %q", fp.SuplenteNome)
	}
	if fp.SuplenteMatricula != "1560477002" {
		t.Errorf("suplente matrícula = %q", fp.SuplenteMatricula)
	}
}

func TestParseFiscalDesignations_NoHeader(t *testing.T) {
	if got := ParseFiscalDesignations("Art. 1º Nomear FULANO DE TAL para o cargo em comissão.", 1); len(got) != 0 {
		t.Errorf("sem 'Fiscal de Contrato (Titular)' não deveria retornar par, got %+v", got)
	}
}

func TestParseContractExtracts_NoHeaderNoResult(t *testing.T) {
	if got := ParseContractExtracts("texto qualquer com CNPJ 11.222.333/0001-44 mas sem cabeçalho de extrato", 1); len(got) != 0 {
		t.Errorf("sem cabeçalho de extrato não deveria retornar contrato, got %+v", got)
	}
}
