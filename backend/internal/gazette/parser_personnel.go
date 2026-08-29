package gazette

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PersonnelAct representa um ato de pessoal extraído do Diário Oficial.
type PersonnelAct struct {
	ID              string  `json:"id"`
	EditionNumber   int32   `json:"edition_number"`
	EditionType     string  `json:"edition_type"`
	PublicationDate int64   `json:"publication_date"`
	ActType         string  `json:"act_type"` // NOMEACAO, EXONERACAO, DESIGNACAO_FUNCAO, RELOTACAO, CONTRATACAO
	PersonName      string  `json:"person_name"`
	PersonCPF       string  `json:"person_cpf,omitempty"`
	PersonMatricula string  `json:"person_matricula,omitempty"`
	JobRole         string  `json:"job_role"`
	Secretaria      string  `json:"secretaria"`
	DASLevel        string  `json:"das_level,omitempty"`
	SalaryValue     float64 `json:"salary_value,omitempty"`
	PortariaNumber  string  `json:"portaria_number,omitempty"`
	FullActText     string  `json:"full_act_text"`
	PDFPageNumber   int32   `json:"pdf_page_number"`
	PDFStorageURL   string  `json:"pdf_storage_url"`
}

// DASSalaryTable mapeia o nível de cargo comissionado (DAS) para seu salário constante.
var DASSalaryTable = map[string]float64{
	"DAS-1": 9800.00,
	"DAS-2": 7450.00,
	"DAS-3": 5800.00,
	"DAS-4": 4200.00,
	"DAS-5": 3100.00,
	"DAS-6": 2200.00,
}

// GetSalaryByDAS retorna o valor salarial mapeado para um nível DAS.
func GetSalaryByDAS(dasLevel string) float64 {
	cleaned := strings.ToUpper(strings.TrimSpace(dasLevel))
	if val, ok := DASSalaryTable[cleaned]; ok {
		return val
	}
	return 0.0
}

// Expressões regulares compiladas para isolamento de blocos e extração de entidades
var (
	// portariaHeaderRegex identifica o INÍCIO de cada Portaria — ancorado no
	// começo da linha e em CAIXA ALTA, para não fatiar em referências no meio
	// da frase ("...nomeado pela portaria nº 40.922...") que criavam blocos
	// espúrios contendo só a assinatura do Prefeito.
	portariaHeaderRegex = regexp.MustCompile(`(?m)^\s{0,20}(PORTARIA(?:\s+INTERNA)?\s+N[º°.\s]*\s*[\d.\-/]+)`)

	portariaNumberRegex = regexp.MustCompile(`(?i)PORTARIA\s+N[º°\.\s]*\s*([\d\.\-\/]+)`)

	dasLevelRegex  = regexp.MustCompile(`(?i)\b(DAS-[1-6]|DGA-[1-6]|FG-[1-6])\b`)
	cpfRegex       = regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b`)
	matriculaRegex = regexp.MustCompile(`(?i)matr[íi]cula\s*(?:n[º°\.\s]*)?\s*(\d+)`)

	// Relotação: captura unidade de destino limpa
	relotacaoDestinoRegex = regexp.MustCompile(`(?i)(?:relotar|relota-se).*?(?:para\s+a|para\s+o|na|no|junto\s+à|junto\s+ao|com\s+lotacao\s+na|com\s+lotação\s+na)\s+([A-ZÁÉÍÓÚÂÊÔÃÕÇ\s]{4,100}?)(?:\.|\,|\s+para\s+exercer|$)`)

	secretariaRegex = regexp.MustCompile(`(?i)(Secretaria\s+Municipal\s+(?:Adjunta\s+)?de\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ\s]+?)(?:\.|\,|$|\s+-\s+|\s+símbolo|\s+para\s+)`)

	// Órgãos que não começam com "Secretaria Municipal de": gabinete,
	// procuradoria, controladoria, fundações/institutos, e siglas usuais.
	orgaoRegex = regexp.MustCompile(`(?i)\b(Gabinete\s+do\s+Prefeito|Procuradoria(?:-Geral)?\s+do\s+Munic[íi]pio|Controladoria(?:-Geral)?\s+do\s+Munic[íi]pio|Funda[çc][ãa]o\s+[A-ZÁÉÍÓÚ][\wçãõáéíóú\s]{3,50}?|Instituto\s+Municipal\s+[A-ZÁÉÍÓÚ][\wçãõáéíóú\s]{3,50}?)(?:\.|\,|$|\s+-\s+|\s+s[íi]mbolo)`)

	siglaOrgaoRegex = regexp.MustCompile(`\b(SME|SMS|SMSA|SEMED|SEMUSA|SMA|SMAGI|SMF|SEFAZ|SMSU|SMDU|SMTU|SMOP|SMEL|SMCT|SMASH|SMDS|PGM|CGM|GABPREF)\b`)
)

// ParsePersonnelActs isola blocos de Portarias por índice de cabeçalho e extrai atos de pessoal formatados e sem truncamento.
func ParsePersonnelActs(rawText string, editionNumber int32, editionType string, pubDate time.Time, pdfPage int32, pdfURL string) []PersonnelAct {
	locs := portariaHeaderRegex.FindAllStringIndex(rawText, -1)
	var blocks []string

	if len(locs) == 0 {
		if strings.Contains(strings.ToUpper(rawText), "PORTARIA") {
			blocks = []string{rawText}
		}
	} else {
		for i := 0; i < len(locs); i++ {
			start := locs[i][0]
			end := len(rawText)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			blocks = append(blocks, rawText[start:end])
		}
	}

	var acts []PersonnelAct
	for idx, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		act := parseSingleBlock(block, editionNumber, editionType, pubDate, pdfPage, pdfURL, idx+1)
		if act.PersonName != "" {
			acts = append(acts, act)
		}
	}
	return acts
}

var leadingDigitsRegex = regexp.MustCompile(`\d+`)

// LeadingInt32 extrai o primeiro grupo de dígitos de uma string
// ("6265 (PDF)" -> 6265, "6263-E" -> 6263, "Edição Nº 6260" -> 6260).
// Zero quando não há dígitos.
func LeadingInt32(s string) int32 {
	n, _ := strconv.Atoi(leadingDigitsRegex.FindString(s))
	return int32(n)
}

// SplitPages divide a saída do pdftotext em páginas pelo form-feed (\f), que
// o poppler emite entre páginas por padrão. Preserva páginas vazias para que
// o índice do slice corresponda ao número real da página (1-based no chamador).
func SplitPages(rawText string) []string {
	return strings.Split(rawText, "\f")
}

func parseSingleBlock(block string, editionNumber int32, editionType string, pubDate time.Time, pdfPage int32, pdfURL string, index int) PersonnelAct {
	blockUpper := strings.ToUpper(block)

	// 1. Extração do número da portaria
	portariaNum := ""
	if match := portariaNumberRegex.FindStringSubmatch(block); len(match) > 1 {
		portariaNum = strings.TrimSpace(match[1])
	}

	// 2. Classificação precisa de act_type (Regra 2)
	actType := classifyActType(blockUpper)

	// 3. Extração de Nome Completo sem truncar sobrenomes (Regra 1)
	personName := extractFullName(block)

	// 4. Extração de CPF e Matrícula
	cpf := ""
	if match := cpfRegex.FindString(block); match != "" {
		cpf = match
	}
	matricula := ""
	if match := matriculaRegex.FindStringSubmatch(block); len(match) > 1 {
		matricula = match[1]
	}

	// 5. Nível DAS e Salário Vinculado por Tabela de Constantes (Regra 4)
	dasLevel := ""
	if match := dasLevelRegex.FindString(blockUpper); match != "" {
		dasLevel = strings.ToUpper(match)
	}
	salaryValue := GetSalaryByDAS(dasLevel)

	// 6. Tratamento de Secretaria (Regra 3) e Cargo
	secretaria := extractSecretaria(block, actType)
	jobRole := extractJobRole(block, dasLevel)

	id := fmt.Sprintf("act-%d-%d-%d", editionNumber, pdfPage, index)

	return PersonnelAct{
		ID:              id,
		EditionNumber:   editionNumber,
		EditionType:     editionType,
		PublicationDate: pubDate.Unix(),
		ActType:         actType,
		PersonName:      personName,
		PersonCPF:       cpf,
		PersonMatricula: matricula,
		JobRole:         jobRole,
		Secretaria:      secretaria,
		DASLevel:        dasLevel,
		SalaryValue:     salaryValue,
		PortariaNumber:  portariaNum,
		FullActText:     strings.TrimSpace(block),
		PDFPageNumber:   pdfPage,
		PDFStorageURL:   pdfURL,
	}
}

// classifyActType identifica o verbo do ato de pessoal e o mapeia para o
// vocabulário canônico (ver act_type.go). A ordem importa: "rescisão" e
// "exoneração" são checadas antes de "nomeação"/"contratação" porque o mesmo
// bloco de portaria costuma citar o cargo de origem ("exonerar do cargo de X
// e nomear para Y") — o primeiro verbo é o que classifica o ato.
func classifyActType(blockUpper string) ActType {
	contains := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(blockUpper, s) {
				return true
			}
		}
		return false
	}

	// Portaria de licitação/compras (RATIFICA/HOMOLOGA processo de dispensa,
	// inexigibilidade, pregão…) não é ato de pessoal, mesmo quando cita a
	// palavra "contratação". Sai como OUTROS e é descartada a montante.
	if contains("DISPENSA DE LICITAÇÃO", "INEXIGIBILIDADE DE LICITAÇÃO", "PROCESSO DE LICITAÇÃO",
		"RATIFICA O PROCESSO", "HOMOLOGA O PROCESSO", "PREGÃO ELETRÔNICO", "PREGÃO PRESENCIAL",
		"CHAMAMENTO PÚBLICO", "CREDENCIAMENTO") {
		return ActOutros
	}

	switch {
	case contains("DESIGNAR", "DESIGNA-SE", "DESIGNAÇÃO", "DESIGNACAO"):
		return ActDesignacaoFuncao
	case contains("RELOTAR", "RELOTA-SE", "RELOTAÇÃO", "RELOTACAO", "REMANEJAR", "REMANEJAMENTO", "TRANSFERIR", "TRANSFERÊNCIA", "TRANSFERENCIA"):
		return ActRelotacao
	case contains("RESCINDIR", "RESCISÃO", "RESCISAO"):
		return ActRescisao
	case contains("EXONERAR", "EXONERA-SE", "EXONERAÇÃO", "EXONERACAO"):
		return ActExoneracao
	case contains("DISPENSAR") && contains("FUNÇÃO", "FUNCAO", "GRATIFICAD"):
		// "dispensar da função gratificada" = saída de função (a guarda no
		// topo já removeu "dispensa DE LICITAÇÃO").
		return ActExoneracao
	case contains("NOMEAR", "NOMEA-SE", "NOMEAÇÃO", "NOMEACAO"):
		// Efetivo quando o texto ancora em concurso/cargo efetivo; caso
		// contrário é comissionado (o caso dominante nas portarias).
		if contains("EFETIVO", "EFETIVA", "CONCURSO PÚBLICO", "CONCURSO PUBLICO", "APROVAD") {
			return ActNomeacaoEfetivo
		}
		return ActNomeacaoComissionado
	case contains("CONTRATAR", "CONTRATAÇÃO", "CONTRATACAO"):
		return ActContratacaoTemporaria
	case contains("PRORROGAR", "PRORROGAÇÃO", "PRORROGACAO") && contains("CONTRAT"):
		// Prorrogação de contrato temporário de servidor.
		return ActContratacaoTemporaria
	default:
		return ActOutros
	}
}

// extractFullName extrai o nome completo preservando sobrenomes e removendo stop-words de cargos/portarias
func extractFullName(block string) string {
	keywords := []string{"NOMEAR", "EXONERAR", "DESIGNAR", "RELOTAR", "CONTRATAR", "SERVIDOR(A)", "SERVIDOR"}
	blockUpper := strings.ToUpper(block)

	for _, kw := range keywords {
		idx := strings.Index(blockUpper, kw)
		if idx != -1 {
			sub := block[idx+len(kw):]
			sub = strings.TrimLeft(sub, " :-–—,aoseu(a)")

			// Regex captura sequências de nomes maiúsculos / capitalizados incluindo conectivos (DE, DA, DO, DOS, DAS, E)
			nameRegex := regexp.MustCompile(`([A-ZÁÉÍÓÚÂÊÔÃÕÇ][a-zA-ZÁÉÍÓÚÂÊÔÃÕÇáéíóúâêôãõç]+(?:\s+(?:DE|DA|DO|DOS|DAS|E|[A-ZÁÉÍÓÚÂÊÔÃÕÇ][a-zA-ZÁÉÍÓÚÂÊÔÃÕÇáéíóúâêôãõç]+)){1,8})`)
			if match := nameRegex.FindString(sub); match != "" {
				cleaned := filterNameStopWords(strings.TrimSpace(match))
				if IsPlausiblePersonName(cleaned) {
					return cleaned
				}
			}
		}
	}

	// Fallback para padrões totalmente em maiúsculas (ex: YURI SILVA SANTOS)
	upperNameRegex := regexp.MustCompile(`\b([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+(?:DE|DA|DO|DOS|DAS|E|[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,})){1,8})\b`)
	for _, m := range upperNameRegex.FindAllString(block, -1) {
		cleaned := filterNameStopWords(strings.TrimSpace(m))
		if IsPlausiblePersonName(cleaned) {
			return cleaned
		}
	}

	return ""
}

var trailingConnectiveRegex = regexp.MustCompile(`(?i)\s+(de|da|do|dos|das|e)$`)

// collapseWS reduz qualquer sequência de espaços/quebras a um espaço único
// (o pdftotext quebra linha no meio de "Secretaria Municipal de ...").
func collapseWS(s string) string { return strings.Join(strings.Fields(s), " ") }

var nameLowerConnectives = map[string]bool{"de": true, "da": true, "do": true, "dos": true, "das": true, "e": true}

// titleCasePT normaliza o caixa de um nome de órgão/secretaria para Title Case
// pt-BR (conectivos minúsculos), para "SECRETARIA MUNICIPAL DE EDUCAÇÃO" e
// "Secretaria Municipal de Educação" não virarem duas facetas distintas.
// Siglas (só maiúsculas, sem espaço) são preservadas.
func titleCasePT(s string) string {
	s = collapseWS(s)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, " ") && s == strings.ToUpper(s) {
		return s // sigla: SEMED, PGM…
	}
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if i > 0 && nameLowerConnectives[w] {
			continue
		}
		r := []rune(w)
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

func filterNameStopWords(name string) string {
	stopWords := []string{"PARA", "EXERCER", "CARGO", "FUNCAO", "FUNÇÃO", "SIMBOLO", "SÍMBOLO", "PORTARIA", "MATRICULA", "MATRÍCULA", "CPF", "NIVEL", "NÍVEL", "COM", "LOTACAO", "LOTAÇÃO"}
	words := strings.Fields(name)
	var cleanWords []string
	for _, w := range words {
		wUpper := strings.ToUpper(w)
		isStop := false
		for _, stop := range stopWords {
			if wUpper == stop {
				isStop = true
				break
			}
		}
		if isStop {
			break
		}
		cleanWords = append(cleanWords, w)
	}
	out := strings.Join(cleanWords, " ")
	// Apara conectivo pendurado no fim ("JAILTON DE LUCENA DA" -> "... LUCENA"),
	// artefato do regex de nome ao cortar cedo antes do último sobrenome.
	for trailingConnectiveRegex.MatchString(out) {
		out = trailingConnectiveRegex.ReplaceAllString(out, "")
	}
	return strings.TrimSpace(out)
}

// extractSecretaria trata secretaria na relotação extraindo a unidade de destino limpa
func extractSecretaria(block string, actType string) string {
	if actType == "RELOTACAO" {
		if match := relotacaoDestinoRegex.FindStringSubmatch(block); len(match) > 1 {
			dest := filterSecretariaPrefixes(strings.TrimSpace(match[1]))
			if len(dest) > 3 {
				return titleCasePT(dest)
			}
		}
	}

	if match := secretariaRegex.FindStringSubmatch(block); len(match) > 1 {
		return titleCasePT(match[1])
	}
	if match := orgaoRegex.FindStringSubmatch(block); len(match) > 1 {
		return titleCasePT(match[1])
	}
	if match := siglaOrgaoRegex.FindString(block); match != "" {
		return strings.ToUpper(match)
	}

	// Sem match: devolve vazio. Um default fabricado ("Secretaria Municipal
	// de Governo") poluía a faceta de secretaria no Typesense e induzia o
	// usuário a erro. A UI mostra "não informado".
	return ""
}

func filterSecretariaPrefixes(dest string) string {
	destUpper := strings.ToUpper(dest)
	prefixes := []string{"PARA A ", "PARA O ", "NA ", "NO ", "JUNTO À ", "JUNTO AO "}
	for _, p := range prefixes {
		if strings.HasPrefix(destUpper, p) {
			dest = dest[len(p):]
			break
		}
	}
	return strings.TrimSpace(dest)
}

func extractJobRole(block string, dasLevel string) string {
	roleRegex := regexp.MustCompile(`(?i)(?:para\s+exercer\s+o\s+cargo\s+de|cargo\s+de|função\s+de|funcao\s+de)\s+([A-ZÁÉÍÓÚÂÊÔÃÕÇ\s\-]{4,60}?)(?:\,|\.|\s+símbolo|\s+nivel|\s+das|$)`)
	if match := roleRegex.FindStringSubmatch(block); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	if dasLevel != "" {
		// Valor DERIVADO do símbolo (não um chute): "Cargo Comissionado DAS-3".
		return fmt.Sprintf("Cargo Comissionado %s", dasLevel)
	}
	// Sem cargo identificável: vazio, não "Assessor Especial" fabricado.
	return ""
}
