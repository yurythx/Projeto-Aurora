package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-nova/internal/gazette"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

var (
	// Regexes de Atos de Pessoal (Atos administrativos)
	reExoneracao = regexp.MustCompile(`(?i)(exonerar|exonerado|exoneração|exonerada)\s*(a pedido)?[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}){1,4})`)
	// "contratar/contratação" foi removido daqui de propósito: no DIORONDON
	// essas palavras quase sempre introduzem a contratação de uma EMPRESA
	// (licitação), não a admissão de uma pessoa — capturavam texto de edital
	// ("contratação de Empresa Especializada..."). Contratação temporária de
	// servidor é pega pelo parser de blocos de Portaria (gazette).
	reNomeacao = regexp.MustCompile(`(?i)(nomear|nomeado|nomeação|nomeada)[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}){1,4})`)
	reMudanca  = regexp.MustCompile(`(?i)(relotar|relotação|transferir|transferência|remanejar|remanejamento)[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}){1,4})`)

	// Regexes de Entidades e Documentos
	reCPF       = regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b`)
	reCNPJ      = regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}\/\d{4}-\d{2}\b`)
	reMatricula = regexp.MustCompile(`(?i)(MAT-?\d+|\bmatrícula\s*:?\s*\d+)`)
	rePortaria  = regexp.MustCompile(`(?i)portaria\s*nº?\s*([\d\.\-]+)`)
	reValor     = regexp.MustCompile(`(?i)R\$\s*([\d\.\,]+)`)

	// Regexes de Contratos e Licitações
	reContrato = regexp.MustCompile(`(?i)contrato\s*nº?\s*([\d\/\-]+)`)
	reEmpresa  = regexp.MustCompile(`(?i)(empresa|contratada|razão\s*social)[\s,:-]+([A-Z0-9ÁÉÍÓÚÂÊÔÃÕÇ\s\.\-&]{4,60})`)
)

// PDFParser efetua a leitura em stream do PDF oficial de Rondonópolis via pdftotext -layout.
type PDFParser struct{}

func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

// ExtractTextStream repassa o stream HTTP diretamente via stdin para o comando pdftotext sem arquivos temporários em disco.
func (p *PDFParser) ExtractTextStream(ctx context.Context, bodyStream io.Reader) (string, error) {
	if bodyStream == nil {
		return "", fmt.Errorf("pdf stream is nil")
	}

	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-", "-")
	cmd.Stdin = bodyStream
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext execution failed: %w (stderr: %s)", err, errBuf.String())
	}

	// Alguns PDFs oficiais saem com bytes Latin-1/Windows-1252 soltos (0xA2
	// "¢", 0xB3 "³", 0xBA "º") que o pdftotext não converte; o PostgreSQL
	// (banco UTF-8) rejeita isso no INSERT (SQLSTATE 22021) e a edição
	// inteira ia para FAILED. gazette.RepairEncoding transcodifica esses
	// bytes (preservando o caractere) em vez de deletá-los, na fronteira,
	// para todo o resto do pipeline receber UTF-8 válido.
	return gazette.RepairEncoding(strings.TrimSpace(outBuf.String())), nil
}

// ParseTextFindings analisa o texto extraído do PDF e o fatia em findings
// estruturados de atos de pessoal e contratos.
//
// O texto é processado PÁGINA A PÁGINA (dividido pelo form-feed \f que o
// pdftotext emite): cada finding carrega o número real da página onde foi
// encontrado, que o front transforma num link direto para a página exata do
// PDF oficial (…#page=N). Todo act_type é emitido já no vocabulário canônico
// (internal/gazette/act_type.go).
func (p *PDFParser) ParseTextFindings(content, editionNumber string, editionID int64) []domain.Finding {
	// Defesa em profundidade: mesmo que o chamador não tenha passado pela
	// fronteira de extração, garante UTF-8 válido (bytes Latin-1 soltos do
	// pdftotext) antes de qualquer regex / persistência.
	content = gazette.RepairEncoding(content)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	now := time.Now()
	edNumInt := gazette.LeadingInt32(editionNumber)
	findings := make([]domain.Finding, 0)

	for pageIdx, pageText := range gazette.SplitPages(content) {
		pageNum := pageIdx + 1
		findings = append(findings, p.parsePage(pageText, pageNum, editionNumber, edNumInt, editionID, now)...)
	}
	return findings
}

// parsePage roda os dois extratores (parser de blocos de Portaria do
// internal/gazette + varredura linha-a-linha por regex) sobre UMA página.
func (p *PDFParser) parsePage(pageText string, pageNum int, editionNumber string, edNumInt int32, editionID int64, now time.Time) []domain.Finding {
	findings := make([]domain.Finding, 0)

	// 1. Extração atomizada por bloco de Portaria (metadados ricos: DAS,
	//    salário, secretaria, cargo). Blocos que o classificador não
	//    reconhece (ActOutros) são descartados: sem verbo de ato identificado
	//    o "nome" extraído é quase sempre cabeçalho ("DIÁRIO OFICIAL…").
	blockActs := gazette.ParsePersonnelActs(pageText, edNumInt, gazette.EditionTypeFromNumber(editionNumber), now, int32(pageNum), "")
	// Contratações temporárias ("EXTRATO DO CONTRATO INDIVIDUAL DE TRABALHO"):
	// formato rotulado próprio, antes iam parar na coleção de artigos.
	blockActs = append(blockActs, gazette.ParseTemporaryHires(pageText, edNumInt, gazette.EditionTypeFromNumber(editionNumber), now, int32(pageNum), "")...)

	// Designação de fiscal titular + SUPLENTE de contrato num mesmo Art. 1º —
	// o parser de blocos só pegava o titular. Emite os dois e marca o nome do
	// titular para não duplicar via o parser de blocos abaixo.
	fiscalTitularSeen := map[string]bool{}
	for _, fp := range gazette.ParseFiscalDesignations(pageText, int32(pageNum)) {
		fiscalTitularSeen[strings.ToUpper(fp.TitularNome)] = true
		for _, role := range []struct{ nome, mat, papel string }{
			{fp.TitularNome, fp.TitularMatricula, "Fiscal de Contrato (Titular)"},
			{fp.SuplenteNome, fp.SuplenteMatricula, "Fiscal de Contrato (Suplente)"},
		} {
			m := role.mat
			jr := role.papel
			findings = append(findings, domain.Finding{
				ID:            uuid.New(),
				EditionID:     editionID,
				ExternalID:    generateExternalHash(editionNumber, gazette.ActDesignacaoFuncao, role.nome, "", role.mat, jr),
				ActType:       gazette.ActDesignacaoFuncao,
				ServidorNome:  strPtr(role.nome),
				Matricula:     strPtr(m),
				JobRole:       &jr,
				PDFPageNumber: pageNum,
				Confidence:    gazette.ConfidenceHigh,
				RawContent:    jr + " — " + role.nome,
				CreatedAt:     now,
			})
		}
	}

	for _, act := range blockActs {
		if act.ActType == gazette.ActOutros {
			continue
		}
		if act.ActType == gazette.ActDesignacaoFuncao && fiscalTitularSeen[strings.ToUpper(act.PersonName)] {
			continue // já emitido pelo ParseFiscalDesignations (com titular+suplente)
		}
		extID := generateExternalHash(editionNumber, act.ActType, act.PersonName, act.PersonCPF, act.PersonMatricula, act.FullActText)
		var val *float64
		if act.SalaryValue > 0 {
			v := act.SalaryValue
			val = &v
		}
		hasID := act.PersonCPF != "" || act.PersonMatricula != "" || act.DASLevel != "" ||
			(act.ActType == gazette.ActContratacaoTemporaria && act.JobRole != "" && act.PortariaNumber != "")
		findings = append(findings, domain.Finding{
			ID:             uuid.New(),
			EditionID:      editionID,
			ExternalID:     extID,
			ActType:        act.ActType,
			ServidorNome:   strPtr(act.PersonName),
			CPF:            strPtr(act.PersonCPF),
			Matricula:      strPtr(act.PersonMatricula),
			Secretaria:     strPtr(act.Secretaria),
			JobRole:        strPtr(act.JobRole),
			DASLevel:       strPtr(act.DASLevel),
			PortariaNumber: strPtr(act.PortariaNumber),
			PDFPageNumber:  pageNum,
			Confidence:     personnelConfidence(act.PersonName, hasID),
			Valor:          val,
			RawContent:     act.FullActText,
			CreatedAt:      now,
		})
	}

	// 2. Extratos de contrato estruturados (bloco com cabeçalho "EXTRATO DE
	//    CONTRATO/ADITIVO"): contratada, CNPJ, objeto, valor, vigência, fiscal.
	contractCNPJsSeen := map[string]bool{}
	for _, ce := range gazette.ParseContractExtracts(pageText, int32(pageNum)) {
		var valorNum *float64
		if ce.Valor > 0 {
			v := ce.Valor
			valorNum = &v
		}
		if ce.CNPJ != "" {
			contractCNPJsSeen[ce.CNPJ] = true
		}
		findings = append(findings, domain.Finding{
			ID:             uuid.New(),
			EditionID:      editionID,
			ExternalID:     generateExternalHash(editionNumber, gazette.ActContrato, ce.Contratada, ce.CNPJ, ce.ContractNumber, ce.FullText),
			ActType:        gazette.ActContrato,
			EmpresaNome:    strPtr(ce.Contratada),
			CNPJ:           strPtr(ce.CNPJ),
			ServidorNome:   strPtr(ce.FiscalNome),
			Matricula:      strPtr(ce.FiscalMatricula),
			PortariaNumber: strPtr(ce.ContractNumber),
			PDFPageNumber:  pageNum,
			Confidence:     gazette.ConfidenceHigh, // bloco com cabeçalho + campos ancorados
			Valor:          valorNum,
			RawContent:     ce.FullText,
			CreatedAt:      now,
		})
	}

	// 3. Varredura linha-a-linha (pega o que o parser de blocos não isola).
	lines := strings.Split(pageText, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 15 {
			continue
		}

		startIdx := i - 3
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := i + 4
		if endIdx > len(lines) {
			endIdx = len(lines)
		}
		contextBlock := strings.Join(lines[startIdx:endIdx], " ")
		portariaVal := firstSubmatch(contextBlock, rePortaria)
		cpfVal := extractMatch(contextBlock, reCPF)
		matVal := extractMatch(contextBlock, reMatricula)

		// A varredura linha-a-linha é ruidosa (pega fragmentos como
		// "de Grosso", "DE AGOSTO DE"). Só emite um ato de pessoal quando há
		// CPF ou matrícula no contexto imediato — corroboração forte que o
		// texto de tabela/rodapé não tem. Atos estruturados sem identificador
		// são cobertos pelo parser de blocos de Portaria (internal/gazette).
		corroborated := cpfVal != "" || matVal != ""

		hr := func(actType, cleanName string) domain.Finding {
			return domain.Finding{
				ID:             uuid.New(),
				EditionID:      editionID,
				ExternalID:     generateExternalHash(editionNumber, actType, cleanName, cpfVal, matVal, trimmed),
				ActType:        actType,
				ServidorNome:   strPtr(cleanName),
				CPF:            strPtr(cpfVal),
				Matricula:      strPtr(matVal),
				PortariaNumber: strPtr(portariaVal),
				PDFPageNumber:  pageNum,
				// Varredura por linha: já exige CPF/matrícula no contexto
				// (corroborated) e nome plausível — confiança média.
				Confidence: gazette.ConfidenceMedium,
				RawContent: trimmed,
				CreatedAt:  now,
			}
		}

		isHRLine := false

		if m := reExoneracao.FindStringSubmatch(trimmed); len(m) > 3 && isValidPersonName(strings.TrimSpace(m[3])) {
			isHRLine = true
			if corroborated {
				findings = append(findings, hr(gazette.ActExoneracao, cleanPersonName(strings.TrimSpace(m[3]))))
			}
		}
		if m := reNomeacao.FindStringSubmatch(trimmed); len(m) > 2 && isValidPersonName(strings.TrimSpace(m[2])) {
			isHRLine = true
			if corroborated {
				findings = append(findings, hr(gazette.ActNomeacaoComissionado, cleanPersonName(strings.TrimSpace(m[2]))))
			}
		}
		if m := reMudanca.FindStringSubmatch(trimmed); len(m) > 2 && isValidPersonName(strings.TrimSpace(m[2])) {
			isHRLine = true
			if corroborated {
				findings = append(findings, hr(gazette.ActRelotacao, cleanPersonName(strings.TrimSpace(m[2]))))
			}
		}

		if !isHRLine {
			upperLine := strings.ToUpper(trimmed)
			// "CONTRATO INDIVIDUAL DE TRABALHO" já é tratado como contratação
			// temporária (ato de pessoal) — não repetir como contrato/artigo.
			if strings.Contains(upperLine, "CONTRATO INDIVIDUAL DE TRABALHO") {
				continue
			}
			if strings.Contains(upperLine, "CONTRATO") || strings.Contains(upperLine, "EMPRESA") || reCNPJ.MatchString(trimmed) {
				cnpjVal := extractMatch(contextBlock, reCNPJ)
				if cnpjVal != "" && contractCNPJsSeen[cnpjVal] {
					continue // já coberto por um extrato estruturado
				}
				valorStr := extractMatch(contextBlock, reValor)
				var valorNum *float64
				if valorStr != "" {
					v := parseValorMonetario(valorStr)
					valorNum = &v
				}

				empresaNome := ""
				if em := reEmpresa.FindStringSubmatch(contextBlock); len(em) > 2 {
					empresaNome = strings.TrimSpace(em[2])
				}

				// Nº real do contrato ("Contrato nº 140/2026"); se ausente,
				// cai para o nº da portaria de designação do fiscal.
				refNum := firstSubmatch(contextBlock, reContrato)
				if refNum == "" {
					refNum = portariaVal
				}

				if cnpjVal != "" || empresaNome != "" || valorNum != nil {
					findings = append(findings, domain.Finding{
						ID:             uuid.New(),
						EditionID:      editionID,
						ExternalID:     generateExternalHash(editionNumber, gazette.ActContrato, empresaNome, cnpjVal, valorStr, trimmed),
						ActType:        gazette.ActContrato,
						EmpresaNome:    strPtr(empresaNome),
						CNPJ:           strPtr(cnpjVal),
						PortariaNumber: strPtr(refNum),
						PDFPageNumber:  pageNum,
						Confidence:     gazette.ConfidenceMedium,
						Valor:          valorNum,
						RawContent:     trimmed,
						CreatedAt:      now,
					})
				}
			}
		}
	}

	return findings
}

// personnelConfidence classifica a confiança de um ato de pessoal do parser
// de blocos: nome implausível -> low (fica só no PostgreSQL, para auditoria);
// nome plausível com CPF/matrícula/DAS -> high; só nome plausível -> medium.
func personnelConfidence(name string, hasIdentifier bool) string {
	if !gazette.IsPlausiblePersonName(name) {
		return gazette.ConfidenceLow
	}
	if hasIdentifier {
		return gazette.ConfidenceHigh
	}
	return gazette.ConfidenceMedium
}

func firstSubmatch(text string, re *regexp.Regexp) string {
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func generateExternalHash(editionNumber, actType, mainSubject, docID1, docID2, rawText string) string {
	rawSnippet := rawText
	if len(rawSnippet) > 50 {
		rawSnippet = rawSnippet[:50]
	}
	input := fmt.Sprintf("%s_%s_%s_%s_%s_%s", editionNumber, actType, mainSubject, docID1, docID2, rawSnippet)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

func extractMatch(text string, re *regexp.Regexp) string {
	m := re.FindString(text)
	return strings.TrimSpace(m)
}

func parseValorMonetario(valStr string) float64 {
	cleaned := strings.ReplaceAll(valStr, "R$", "")
	cleaned = strings.ReplaceAll(cleaned, "r$", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	v, err := strconv.ParseFloat(strings.TrimSpace(cleaned), 64)
	if err != nil {
		return 0.0
	}
	return v
}

func isValidPersonName(name string) bool {
	if !gazette.IsPlausiblePersonName(name) {
		return false
	}
	// Boilerplate de edital/portaria que a regex gulosa captura
	// ("Solicitada Em Até Noventa Dias", "do Seguinte Objeto").
	forbidden := []string{
		"MUNICIPAL", "PORTARIA", "RESOLVE", "ARTIGO", "OBJETO", "PROFISSIONAIS",
		"TEMPOR", "AQUISI", "SOLICITAD", "PODER", "DEVER", "REFERENTE", "CONFORME", "SEGUINTE",
		"AGOSTO", "JANEIRO", "FEVEREIRO", "MARÇO", "MARCO", "ABRIL", "MAIO", "JUNHO",
		"JULHO", "SETEMBRO", "OUTUBRO", "NOVEMBRO", "DEZEMBRO",
	}
	upper := strings.ToUpper(name)
	for _, f := range forbidden {
		if strings.Contains(upper, f) {
			return false
		}
	}
	return true
}

func cleanPersonName(name string) string {
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

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
