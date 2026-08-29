package application

import (
	"strings"
	"testing"
)

func TestPDFParser_ParseTextFindings(t *testing.T) {
	parser := NewPDFParser()

	sampleText := `
PREFEITURA MUNICIPAL DE RONDONÓPOLIS
PORTARIA Nº 42.500
RESOLVE: Art. 1º Nomear YURI THX SILVA para o cargo em comissão de Coordenador Geral de TI. CPF: 999.888.777-00, Matrícula: MAT-10001.

PORTARIA Nº 41.754
RESOLVE: Art. 1º Exonerar VANETE BARBOSA DO REGO do cargo de Agente Administrativo. CPF: 123.456.789-01, Matrícula: MAT-44102.

CONTRATO Nº 140/2026
EXTRATO DE CONTRATO DA PREFEITURA
Empresa: TechGov Soluções em Tecnologia e Sistemas LTDA, CNPJ: 41.987.654/0001-22, Valor: R$ 950.000,00.
`

	findings := parser.ParseTextFindings(sampleText, "6263", 101)
	if len(findings) < 3 {
		t.Fatalf("expected at least 3 findings, got %d", len(findings))
	}

	foundNomeacao := false
	foundExoneracao := false
	foundContrato := false

	for _, f := range findings {
		if f.ActType == "NOMEACAO" || f.ActType == "NOMEACAO_COMISSIONADO" {
			foundNomeacao = true
			if f.ServidorNome == nil || !strings.Contains(strings.ToUpper(*f.ServidorNome), "YURI THX SILVA") {
				nameStr := "<nil>"
				if f.ServidorNome != nil {
					nameStr = *f.ServidorNome
				}
				t.Errorf("expected Nomeacao servidor YURI THX SILVA, got %s", nameStr)
			}
			if f.CPF == nil || *f.CPF != "999.888.777-00" {
				t.Errorf("expected CPF 999.888.777-00, got %v", f.CPF)
			}
		}
		if f.ActType == "EXONERACAO" {
			foundExoneracao = true
			if f.ServidorNome == nil || !strings.Contains(strings.ToUpper(*f.ServidorNome), "VANETE BARBOSA DO REGO") {
				nameStr := "<nil>"
				if f.ServidorNome != nil {
					nameStr = *f.ServidorNome
				}
				t.Errorf("expected Exoneracao servidor VANETE BARBOSA DO REGO, got %s", nameStr)
			}
		}
		if f.ActType == "CONTRATO" {
			foundContrato = true
			if f.CNPJ == nil || *f.CNPJ != "41.987.654/0001-22" {
				t.Errorf("expected CNPJ 41.987.654/0001-22, got %v", f.CNPJ)
			}
			if f.Valor == nil || *f.Valor != 950000.0 {
				t.Errorf("expected Valor 950000.0, got %v", *f.Valor)
			}
		}
	}

	if !foundNomeacao {
		t.Error("missing NOMEACAO finding")
	}
	if !foundExoneracao {
		t.Error("missing EXONERACAO finding")
	}
	if !foundContrato {
		t.Error("missing CONTRATO finding")
	}
}
