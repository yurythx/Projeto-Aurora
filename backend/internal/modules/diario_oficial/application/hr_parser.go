package application

import (
	"regexp"
	"strings"
	"time"

	"github.com/yurythx/projeto-nova/internal/gazette"
)

// HREventType define a categoria do ato de pessoal encontrado no diário oficial.
// Usa o mesmo vocabulário canônico de internal/gazette (act_type.go) para que
// o filtro escolhido no front case tanto com o índice do Typesense quanto com
// o act_type persistido em diario_oficial_findings.
type HREventType string

const (
	HREventExoneracao   HREventType = HREventType(gazette.ActExoneracao)
	HREventNomeacao     HREventType = HREventType(gazette.ActNomeacaoComissionado)
	HREventMudancaSetor HREventType = HREventType(gazette.ActRelotacao)
)

// HREvent representa um ato de pessoal estruturado extraído de uma publicação do Diário Oficial.
type HREvent struct {
	Type              HREventType `json:"type"`
	Servidor          string      `json:"servidor"`
	ServidorCPF       string      `json:"servidor_cpf"`
	ServidorMatricula string      `json:"servidor_matricula"`
	Secretaria        string      `json:"secretaria"`
	Cargo             string      `json:"cargo"`
	DASLevel          string      `json:"das_level"`
	SetorOrgao        string      `json:"setor_orgao"`
	PortariaNumber    string      `json:"portaria_number"`
	EditionNumber     string      `json:"edition_number"`
	PublicationDate   time.Time   `json:"publication_date"`
	ContextSnippet    string      `json:"context_snippet"`
	DocURL            string      `json:"doc_url"`
	PDFPageNumber     int         `json:"pdf_page_number"`
}

var (
	exoRegex   = regexp.MustCompile(`(?i)(exonerar|exonerado|exoneração|exonerada)\s*(a pedido)?[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ\s]{4,60})`)
	nomRegex   = regexp.MustCompile(`(?i)(nomear|nomeado|nomeação|nomeada|contratar|contratação)[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ\s]{4,60})`)
	mudRegex   = regexp.MustCompile(`(?i)(relotar|relotação|transferir|transferência|remanejar|remanejamento)[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ\s]{4,60})`)
	porRegex   = regexp.MustCompile(`(?i)portaria\s*nº?\s*([\d\.\-]+)`)
	dasRegex   = regexp.MustCompile(`(?i)(DAS|DGA)[\s\-\_]*([1-6])`)
	cargoRegex = regexp.MustCompile(`(?i)cargo\s*(em\s*comissão)?\s*de\s*([A-ZÁÉÍÓÚÂÊÔÃÕÇ\s]{4,40})`)
)

// ExtractHREvents analisa o conteúdo textual de uma edição e retorna todos os atos de pessoal extraídos.
func ExtractHREvents(content, editionNumber, docURL string, pubDate time.Time) []HREvent {
	lines := strings.Split(content, "\n")
	events := make([]HREvent, 0)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 15 {
			continue
		}

		// Procura número de Portaria no contexto ao redor (2 linhas acima/abaixo)
		startIdx := i - 2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := i + 3
		if endIdx > len(lines) {
			endIdx = len(lines)
		}
		contextBlock := strings.Join(lines[startIdx:endIdx], " ")
		portariaMatch := porRegex.FindStringSubmatch(contextBlock)
		portariaNum := ""
		if len(portariaMatch) > 1 {
			portariaNum = portariaMatch[1]
		}

		dasLevel := extractDASLevel(contextBlock)
		cargo := extractCargo(contextBlock)

		// 1. Exoneração
		if matches := exoRegex.FindStringSubmatch(trimmed); len(matches) > 3 {
			servidor := strings.TrimSpace(matches[3])
			if isValidServidorName(servidor) {
				events = append(events, HREvent{
					Type:            HREventExoneracao,
					Servidor:        cleanServidorName(servidor),
					Cargo:           cargo,
					DASLevel:        dasLevel,
					PortariaNumber:  portariaNum,
					EditionNumber:   editionNumber,
					PublicationDate: pubDate,
					ContextSnippet:  trimmed,
					DocURL:          docURL,
				})
			}
		}

		// 2. Nomeação / Contratação
		if matches := nomRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			servidor := strings.TrimSpace(matches[2])
			if isValidServidorName(servidor) {
				events = append(events, HREvent{
					Type:            HREventNomeacao,
					Servidor:        cleanServidorName(servidor),
					Cargo:           cargo,
					DASLevel:        dasLevel,
					PortariaNumber:  portariaNum,
					EditionNumber:   editionNumber,
					PublicationDate: pubDate,
					ContextSnippet:  trimmed,
					DocURL:          docURL,
				})
			}
		}

		// 3. Mudança de Setor / Relotação
		if matches := mudRegex.FindStringSubmatch(trimmed); len(matches) > 2 {
			servidor := strings.TrimSpace(matches[2])
			if isValidServidorName(servidor) {
				events = append(events, HREvent{
					Type:            HREventMudancaSetor,
					Servidor:        cleanServidorName(servidor),
					Cargo:           cargo,
					DASLevel:        dasLevel,
					PortariaNumber:  portariaNum,
					EditionNumber:   editionNumber,
					PublicationDate: pubDate,
					ContextSnippet:  trimmed,
					DocURL:          docURL,
				})
			}
		}
	}

	return events
}

func extractDASLevel(contextText string) string {
	match := dasRegex.FindStringSubmatch(contextText)
	if len(match) > 2 {
		return "DAS-" + match[2]
	}
	upper := strings.ToUpper(contextText)
	if strings.Contains(upper, "COORDENADOR") || strings.Contains(upper, "DIRETOR") {
		return "DAS-1"
	} else if strings.Contains(upper, "ASSESSOR ESPECIAL") || strings.Contains(upper, "GERENTE") {
		return "DAS-2"
	} else if strings.Contains(upper, "ASSESSOR TÉCNICO") || strings.Contains(upper, "SUPERVISOR") {
		return "DAS-3"
	} else if strings.Contains(upper, "CHEFE DE DIVISÃO") || strings.Contains(upper, "PEDAGÓGICA") {
		return "DAS-4"
	} else if strings.Contains(upper, "AGENTE") || strings.Contains(upper, "ASSISTENTE") {
		return "DAS-5"
	}
	return "DAS-3"
}

func extractCargo(contextText string) string {
	match := cargoRegex.FindStringSubmatch(contextText)
	if len(match) > 2 {
		return strings.TrimSpace(match[2])
	}
	return ""
}

func isValidServidorName(name string) bool {
	words := strings.Fields(name)
	if len(words) < 2 || len(words) > 6 {
		return false
	}
	// Rejeita frases que capturaram palavras de lei/secretaria por engano
	forbidden := []string{"SECRETARIA", "PREFEITURA", "MUNICIPAL", "PORTARIA", "RESOLVE", "ARTIGO", "CARGO", "TABELA"}
	upper := strings.ToUpper(name)
	for _, f := range forbidden {
		if strings.Contains(upper, f) {
			return false
		}
	}
	return true
}

func cleanServidorName(name string) string {
	parts := strings.Fields(name)
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		lower := strings.ToLower(p)
		if lower == "do" || lower == "da" || lower == "de" || lower == "dos" || lower == "das" || lower == "e" {
			cleaned = append(cleaned, lower)
		} else if len(lower) > 0 {
			cleaned = append(cleaned, strings.ToUpper(lower[:1])+lower[1:])
		}
	}
	return strings.Join(cleaned, " ")
}
