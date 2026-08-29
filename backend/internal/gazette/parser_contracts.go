package gazette

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ContractExtract é um "EXTRATO DE CONTRATO / ADITIVO" isolado e estruturado a
// partir do texto do Diário Oficial. Diferente da varredura por linha (que
// pegava "especializada na prestação de serviços…" como razão social), aqui o
// bloco é delimitado pelo cabeçalho e cada campo tem âncora própria.
type ContractExtract struct {
	ContractNumber  string
	Contratada      string
	CNPJ            string
	Objeto          string
	Valor           float64
	Vigencia        string
	FiscalNome      string
	FiscalMatricula string
	FullText        string
	PDFPageNumber   int32
}

var (
	contractHeaderRegex = regexp.MustCompile(`(?i)EXTRATO\s+(?:D[EO]\s+)?(?:\d[ºª°.\s-]*\s*)?(?:TERMO\s+ADITIVO|ADITIVO|CONTRATO|CONV[ÊE]NIO|ATA\s+DE\s+REGISTRO\s+DE\s+PRE[ÇC]OS)`)

	contractNumRegex   = regexp.MustCompile(`(?i)(?:CONTRATO|ADITIVO|CONV[ÊE]NIO|ATA)\s*(?:ADMINISTRATIVO\s*)?N[º°.\s]*\s*(\d{1,4}\s*/\s*\d{4})`)
	contratadaRegex    = regexp.MustCompile(`(?i)CONTRATAD[AO]\s*:?\s*([A-ZÀ-Ú][^\n;]{3,120}?)(?:\s*,?\s*(?:CNPJ|inscrita|com\s+sede|pessoa\s+jur)|\.\s|;|\n)`)
	objetoRegex        = regexp.MustCompile(`(?i)OBJETO\s*:?\s*([^\n]{8,400}?)(?:\.\s|\n|VALOR|VIG[ÊE]NCIA|DATA|FUNDAMENT)`)
	contractValorRegex = regexp.MustCompile(`(?i)VALOR(?:\s+(?:TOTAL|GLOBAL|MENSAL|ESTIMADO|ANUAL))?\s*:?\s*R\$\s*([\d.]+,\d{2})`)
	vigenciaRegex      = regexp.MustCompile(`(?i)VIG[ÊE]NCIA\s*:?\s*([^\n]{4,140}?)(?:\.\s|\n|VALOR|OBJETO|FUNDAMENT|DATA\s+DA)`)
	fiscalRegex        = regexp.MustCompile(`(?i)FISCAL(?:\s+(?:TITULAR|DO\s+CONTRATO|DE\s+CONTRATO))?\s*:?\s*([A-ZÀ-Ú][A-Za-zÀ-ú.\s]{4,70}?)(?:\s*,|\s*\(|\s*CPF|\s*matr[íi]cula|\n)`)
	fiscalMatRegex     = regexp.MustCompile(`(?i)fiscal[^.\n]{0,80}?matr[íi]cula\s*(?:n[º°.\s]*)?\s*([\d.]+)`)
)

// ParseContractExtracts isola blocos de extrato de contrato pelo cabeçalho e
// extrai os campos estruturados. Retorna só blocos com pelo menos CNPJ,
// contratada ou nº de contrato — texto solto não vira contrato.
func ParseContractExtracts(rawText string, pdfPage int32) []ContractExtract {
	locs := contractHeaderRegex.FindAllStringIndex(rawText, -1)
	if len(locs) == 0 {
		return nil
	}

	out := make([]ContractExtract, 0, len(locs))
	for i := range locs {
		start := locs[i][0]
		end := len(rawText)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		block := rawText[start:end]
		if len(block) > 3000 { // um extrato não passa disso; corta ruído
			block = block[:3000]
		}

		ce := ContractExtract{FullText: collapseWS(strings.TrimSpace(block)), PDFPageNumber: pdfPage}
		if m := contractNumRegex.FindStringSubmatch(block); len(m) > 1 {
			ce.ContractNumber = strings.Join(strings.Fields(m[1]), "")
		}
		if m := contratadaRegex.FindStringSubmatch(block); len(m) > 1 {
			ce.Contratada = collapseWS(m[1])
		}
		if m := cnpjInStringRegex.FindString(block); m != "" {
			ce.CNPJ = m
		}
		if m := objetoRegex.FindStringSubmatch(block); len(m) > 1 {
			ce.Objeto = collapseWS(m[1])
		}
		if m := contractValorRegex.FindStringSubmatch(block); len(m) > 1 {
			ce.Valor = parseBRL(m[1])
		}
		if m := vigenciaRegex.FindStringSubmatch(block); len(m) > 1 {
			ce.Vigencia = collapseWS(m[1])
		}
		if m := fiscalRegex.FindStringSubmatch(block); len(m) > 1 {
			name := collapseWS(m[1])
			if IsPlausiblePersonName(name) {
				ce.FiscalNome = name
			}
		}
		if m := fiscalMatRegex.FindStringSubmatch(block); len(m) > 1 {
			ce.FiscalMatricula = strings.ReplaceAll(m[1], ".", "")
		}

		// Descarta cabeçalho de tabela capturado como contratada
		// ("OBJETO TIPO PRAZO MODALIDADE"): exige marca de razão social.
		if ce.Contratada != "" && !orgTokenRegex.MatchString(ce.Contratada) {
			ce.Contratada = ""
		}
		// Só emite um extrato ESTRUTURADO (confidence=high) quando tem CNPJ +
		// razão social reconhecível. Caso contrário deixa a varredura por
		// linha tratar como confidence=medium — sem isso o parser gerava
		// "high" fraco a partir de fragmentos de tabela.
		if ce.CNPJ == "" || ce.Contratada == "" {
			continue
		}
		out = append(out, ce)
	}
	return out
}

var (
	// "EXTRATO DO CONTRATO INDIVIDUAL DE TRABALHO POR TEMPO DETERMINADO" —
	// contratação temporária de servidor (formato real do DIORONDON, com
	// campos rotulados Contratado(a)/Cargo/Remuneração Mensal/Vigência).
	tempHireHeaderRegex = regexp.MustCompile(`(?i)EXTRATO\s+D[OE]\s+CONTRATO\s+INDIVIDUAL\s+DE\s+TRABALHO(?:\s+POR\s+TEMPO\s+DETERMINADO)?`)
	tempHireNumRegex    = regexp.MustCompile(`(?i)DETERMINADO\s*N[°º:.\s]+\s*(\d{1,5}\s*/\s*\d{4})`)
	tempHireNameRegex   = regexp.MustCompile(`(?i)Contratad[oa]\s*\(?a?\)?\s*:?\s*([^\n]{4,80})`)
	tempHireCargoRegex  = regexp.MustCompile(`(?i)Cargo\s*:?\s*([^\n]{2,70})`)
	tempHireRemunRegex  = regexp.MustCompile(`(?i)Remunera[çc][ãa]o\s+Mensal\s*:?\s*R?\$?\s*([\d.]+,\d{2})`)
	tempHireVigRegex    = regexp.MustCompile(`(?i)Vig[êe]ncia\s*:?\s*([^\n]{4,70})`)
)

// ParseTemporaryHires isola os blocos de "EXTRATO DO CONTRATO INDIVIDUAL DE
// TRABALHO" e devolve atos de CONTRATACAO_TEMPORARIA já estruturados — antes
// caíam na varredura por linha como "CONTRATO" e iam parar na coleção de
// artigos, não em atos de pessoal.
func ParseTemporaryHires(rawText string, editionNumber int32, editionType string, pubDate time.Time, pdfPage int32, pdfURL string) []PersonnelAct {
	locs := tempHireHeaderRegex.FindAllStringIndex(rawText, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([]PersonnelAct, 0, len(locs))
	for i := range locs {
		start := locs[i][0]
		end := len(rawText)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		block := rawText[start:end]
		if len(block) > 2000 {
			block = block[:2000]
		}

		name := ""
		if m := tempHireNameRegex.FindStringSubmatch(block); len(m) > 1 {
			name = collapseWS(strings.Trim(m[1], " .,;:"))
		}
		if !IsPlausiblePersonName(name) {
			continue
		}
		act := PersonnelAct{
			ID:              "",
			EditionNumber:   editionNumber,
			EditionType:     editionType,
			PublicationDate: pubDate.Unix(),
			ActType:         ActContratacaoTemporaria,
			PersonName:      name,
			FullActText:     collapseWS(strings.TrimSpace(block)),
			PDFPageNumber:   pdfPage,
			PDFStorageURL:   pdfURL,
		}
		if m := tempHireNumRegex.FindStringSubmatch(block); len(m) > 1 {
			act.PortariaNumber = strings.Join(strings.Fields(m[1]), "")
		}
		if m := tempHireCargoRegex.FindStringSubmatch(block); len(m) > 1 {
			act.JobRole = collapseWS(strings.Trim(m[1], " .,;:"))
		}
		if m := tempHireRemunRegex.FindStringSubmatch(block); len(m) > 1 {
			act.SalaryValue = parseBRL(m[1])
		}
		out = append(out, act)
	}
	return out
}

var (
	// "Designar o servidor X, matrícula M, como Fiscal de Contrato (Titular),
	//  e a servidora Y, matrícula N, como Fiscal de Contrato (Suplente)".
	fiscalDesigRegex  = regexp.MustCompile(`(?i)Designar\s+[oa]\s+servidor[a]?\s+([A-ZÀ-Ú][^,]{4,70}?)\s*,\s*matr[íi]cula\s*(?:n[º°.\s]*)?\s*([\d.\- ]+?)\s*,?\s*como\s+Fiscal[^,]*\(\s*Titular\s*\)\s*,\s*e\s+[oa]\s+servidor[a]?\s*,?\s+([A-ZÀ-Ú][^,]{4,70}?)\s*,?\s*matr[íi]cula\s*(?:n[º°.\s]*)?\s*([\d.\- ]+?)\s*,?\s*como\s+Fiscal[^,]*\(\s*Suplente\s*\)`)
	fiscalHeaderRegex = regexp.MustCompile(`(?i)como\s+Fiscal\s+de\s+Contrato\s*\(\s*Titular\s*\)`)
)

// FiscalPair é uma designação de fiscal titular + suplente de contrato,
// extraída de um mesmo Art. 1º. Antes o parser de blocos pegava só o titular.
type FiscalPair struct {
	TitularNome       string
	TitularMatricula  string
	SuplenteNome      string
	SuplenteMatricula string
	FullText          string
	PDFPageNumber     int32
}

// ParseFiscalDesignations extrai os pares titular/suplente de fiscal de
// contrato. Valor de fiscalização central para a plataforma.
func ParseFiscalDesignations(rawText string, pdfPage int32) []FiscalPair {
	if !fiscalHeaderRegex.MatchString(rawText) {
		return nil
	}
	// Normaliza quebras de linha (matrícula vem quebrada: "612097-\n3").
	norm := collapseWS(rawText)
	out := make([]FiscalPair, 0)
	for _, m := range fiscalDesigRegex.FindAllStringSubmatch(norm, -1) {
		tn := collapseWS(strings.Trim(m[1], " .,;"))
		sn := collapseWS(strings.Trim(m[3], " .,;"))
		if !IsPlausiblePersonName(tn) || !IsPlausiblePersonName(sn) {
			continue
		}
		out = append(out, FiscalPair{
			TitularNome:       tn,
			TitularMatricula:  cleanMatricula(m[2]),
			SuplenteNome:      sn,
			SuplenteMatricula: cleanMatricula(m[4]),
			PDFPageNumber:     pdfPage,
		})
	}
	return out
}

func cleanMatricula(s string) string {
	return strings.NewReplacer(" ", "", ".", "", "-", "").Replace(strings.TrimSpace(s))
}

// parseBRL converte "1.234.567,89" -> 1234567.89.
func parseBRL(s string) float64 {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
