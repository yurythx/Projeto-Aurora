package application

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/yurythx/projeto-nova/internal/gazette"
)

// Suíte de QUALIDADE do parser: fixa o comportamento esperado contra trechos
// no formato real do DIORONDON, para que regressões de precisão (nome de
// empresa virando "servidor", texto de edital virando nomeação, act_type
// errado) sejam pegas por teste, não em produção.

func findingsByType(fs []finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.ActType]++
	}
	return m
}

type finding = struct {
	ActType string
	Nome    string
	Page    int
}

func run(t *testing.T, text string) []finding {
	t.Helper()
	raw := NewPDFParser().ParseTextFindings(text, "6300", 1)
	out := make([]finding, 0, len(raw))
	for _, f := range raw {
		nome := ""
		if f.ServidorNome != nil {
			nome = *f.ServidorNome
		} else if f.EmpresaNome != nil {
			nome = *f.EmpresaNome
		}
		out = append(out, finding{ActType: f.ActType, Nome: nome, Page: f.PDFPageNumber})
	}
	return out
}

func TestParserQuality_RealActsClassifiedCorrectly(t *testing.T) {
	text := `SECRETARIA MUNICIPAL DE ADMINISTRAÇÃO

PORTARIA Nº 42.100, DE 12 DE AGOSTO DE 2026.
O PREFEITO MUNICIPAL DE RONDONÓPOLIS, no uso de suas atribuições legais,
RESOLVE:
Art. 1º Exonerar, a pedido, o servidor CARLOS EDUARDO ALMEIDA DA SILVA, matrícula nº 44102,
do cargo em comissão de Assessor Técnico, símbolo DAS-3, da Secretaria Municipal de Saúde.

PORTARIA Nº 42.101, DE 12 DE AGOSTO DE 2026.
RESOLVE:
Art. 1º Nomear a servidora MARIANA COSTA PEREIRA, CPF 123.456.789-00, para exercer o cargo de
Gerente de Projetos, símbolo DAS-2, da Secretaria Municipal de Educação.

PORTARIA Nº 42.102, DE 12 DE AGOSTO DE 2026.
RESOLVE:
Art. 1º Designar o servidor efetivo JOÃO BATISTA RAMOS, matrícula 10233, para responder pela
função gratificada de Chefe de Divisão de Compras.
`
	got := findingsByType(run(t, text))
	if got[gazette.ActExoneracao] < 1 {
		t.Errorf("esperava >=1 EXONERACAO, got %v", got)
	}
	if got[gazette.ActNomeacaoComissionado] < 1 {
		t.Errorf("esperava >=1 NOMEACAO_COMISSIONADO, got %v", got)
	}
	if got[gazette.ActDesignacaoFuncao] < 1 {
		t.Errorf("esperava >=1 DESIGNACAO_FUNCAO, got %v", got)
	}
	// Nenhum ato de pessoal deveria classificar como OUTROS aqui.
	if got[gazette.ActOutros] != 0 {
		t.Errorf("não esperava OUTROS, got %v", got)
	}
}

func TestParserQuality_ContractAndBoilerplateNeverBecomePersonnel(t *testing.T) {
	// Texto de extrato de contrato + aviso de licitação — nada disso é ato de
	// pessoal. Antes o scanner por linha capturava "AGROBEN COMÉRCIO..." e
	// "SOLICITADA EM ATÉ NOVENTA DIAS" como servidores.
	text := `EXTRATO DE CONTRATO Nº 140/2026
CONTRATANTE: Município de Rondonópolis. CONTRATADA: AGROBEN COMÉRCIO DE PRODUTOS
AGROPECUÁRIOS LTDA, CNPJ 41.987.654/0001-22. OBJETO: aquisição de insumos. VALOR: R$ 950.000,00.

PORTARIA Nº 36.470, DE 02 DE JANEIRO DE 2025.
O SECRETÁRIO MUNICIPAL DE FAZENDA, no uso de suas atribuições legais, RATIFICA O PROCESSO DE
DISPENSA DE LICITAÇÃO nº 12/2025, para a contratação de fornecedor de material de escritório.

AVISO DE LICITAÇÃO - PREGÃO ELETRÔNICO Nº 88/2026
A Prefeitura torna público que fará realizar licitação para a contratação de empresa
especializada na prestação de serviços de engenharia. A proposta poderá ser solicitada em até
noventa dias e deverá ser entregue conforme o edital.

DIÁRIO OFICIAL ELETRÔNICO DE RONDONÓPOLIS
Instituto Municipal de Previdência Social dos Servidores
`
	for _, f := range run(t, text) {
		switch f.ActType {
		case gazette.ActExoneracao, gazette.ActNomeacaoComissionado, gazette.ActNomeacaoEfetivo,
			gazette.ActRelotacao, gazette.ActDesignacaoFuncao, gazette.ActRescisao, gazette.ActContratacaoTemporaria:
			t.Errorf("texto de contrato/licitação/cabeçalho virou ato de pessoal: %+v", f)
			// Ato de pessoal nunca deve ter nome de organização como servidor.
			if gazette.LooksLikeOrganization(f.Nome) {
				t.Errorf("organização como servidor em ato de pessoal: %q", f.Nome)
			}
		}
	}
}

func TestParserQuality_PageNumbersPropagateAcrossFormFeeds(t *testing.T) {
	page1 := "SUMÁRIO\n\n"
	page2 := "PORTARIA Nº 42.200\nRESOLVE:\nArt. 1º Exonerar o servidor PEDRO HENRIQUE LIMA, matrícula 55.\n"
	page3 := "PORTARIA Nº 42.201\nRESOLVE:\nArt. 1º Nomear a servidora ANA PAULA SOUZA, CPF 111.222.333-44.\n"
	text := page1 + "\f" + page2 + "\f" + page3

	fs := run(t, text)
	if len(fs) == 0 {
		t.Fatal("nenhum finding extraído")
	}
	seenP2, seenP3 := false, false
	for _, f := range fs {
		if f.Page < 1 {
			t.Errorf("página inválida: %+v", f)
		}
		up := strings.ToUpper(f.Nome)
		if strings.Contains(up, "PEDRO") && f.Page == 2 {
			seenP2 = true
		}
		if strings.Contains(up, "ANA PAULA") && f.Page == 3 {
			seenP3 = true
		}
	}
	if !seenP2 || !seenP3 {
		t.Errorf("números de página não propagaram pelos form-feeds: %+v", fs)
	}
}

func TestParserQuality_ConfidenceAssignment(t *testing.T) {
	// Ato completo (verbo + nome + matrícula + DAS) -> high.
	high := `PORTARIA Nº 42.400
RESOLVE: Art. 1º Nomear a servidora MARIANA COSTA PEREIRA, matrícula nº 9911, para o cargo
em comissão de Gerente, símbolo DAS-2.`
	// Ato com nome plausível mas sem CPF/matrícula/DAS -> medium.
	medium := `PORTARIA Nº 42.401
RESOLVE: Art. 1º Designar o servidor JOÃO BATISTA RAMOS para responder pela função de Chefe.`
	// Fragmento de órgão capturado por engano -> low (fica só no PostgreSQL).
	low := `PORTARIA Nº 42.402
RESOLVE: Art. 1º Designar AGRICULTURA E PECUÁRIA para exercer atividades.`

	for _, tc := range []struct {
		name, text, want string
	}{
		{"high", high, "high"},
		{"medium", medium, "medium"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewPDFParser().ParseTextFindings(tc.text, "6300", 1)
			found := false
			for _, f := range fs {
				if f.ActType != gazette.ActContrato && f.ServidorNome != nil {
					found = true
					if f.Confidence != tc.want {
						t.Errorf("%q: confidence = %q, want %q (finding %+v)", tc.name, f.Confidence, tc.want, *f.ServidorNome)
					}
				}
			}
			if !found {
				t.Errorf("%q: nenhum ato de pessoal extraído", tc.name)
			}
		})
	}

	t.Run("low", func(t *testing.T) {
		for _, f := range NewPDFParser().ParseTextFindings(low, "6300", 1) {
			if f.ServidorNome != nil && *f.ServidorNome != "" && f.Confidence != gazette.ConfidenceLow {
				t.Errorf("fragmento de órgão %q deveria ser confidence=low, got %q", *f.ServidorNome, f.Confidence)
			}
		}
	})
}

func TestParserQuality_InvalidUTF8DoesNotLoseTheEdition(t *testing.T) {
	// Bytes que derrubaram as edições 6256/6264 (0xA2, 0xB3) no meio do texto.
	text := "PORTARIA N\xba 42.300\nRESOLVE:\nArt. 1\xba Exonerar o servidor RA\xb3UL FERREIRA GOMES, matr\xedcula 77.\n" +
		"Valor de refer\xeancia: 30m\xb3.\n"

	fs := NewPDFParser().ParseTextFindings(text, "6300", 1)
	for _, f := range fs {
		if f.ServidorNome != nil && !utf8.ValidString(*f.ServidorNome) {
			t.Errorf("servidor_nome não é UTF-8 válido: %q", *f.ServidorNome)
		}
		if !utf8.ValidString(f.RawContent) {
			t.Errorf("raw_content não é UTF-8 válido: %q", f.RawContent)
		}
	}
	if len(fs) == 0 {
		t.Error("edição com bytes inválidos não produziu nenhum finding (deveria, após RepairEncoding)")
	}
}
