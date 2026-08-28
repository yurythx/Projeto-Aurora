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
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

var (
	// Regexes de Atos de Pessoal (Atos administrativos)
	reExoneracao = regexp.MustCompile(`(?i)(exonerar|exonerado|exoneração|exonerada)\s*(a pedido)?[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}){1,4})`)
	reNomeacao   = regexp.MustCompile(`(?i)(nomear|nomeado|nomeação|nomeada|contratar|contratação)[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}){1,4})`)
	reMudanca    = regexp.MustCompile(`(?i)(relotar|relotação|transferir|transferência|remanejar|remanejamento)[\s,:-]+([A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}(?:\s+[A-ZÁÉÍÓÚÂÊÔÃÕÇ]{2,}){1,4})`)

	// Regexes de Entidades e Documentos
	reCPF       = regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b`)
	reCNPJ      = regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}\/\d{4}-\d{2}\b`)
	reMatricula = regexp.MustCompile(`(?i)(MAT-?\d+|\bmatrícula\s*:?\s*\d+)`)
	rePortaria  = regexp.MustCompile(`(?i)portaria\s*nº?\s*([\d\.\-]+)`)
	reValor     = regexp.MustCompile(`(?i)R\$\s*([\d\.\,]+)`)

	// Regexes de Contratos e Licitações
	reContrato  = regexp.MustCompile(`(?i)contrato\s*nº?\s*([\d\/\-]+)`)
	reEmpresa   = regexp.MustCompile(`(?i)(empresa|contratada|razão\s*social)[\s,:-]+([A-Z0-9ÁÉÍÓÚÂÊÔÃÕÇ\s\.\-&]{4,60})`)
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

	return strings.TrimSpace(outBuf.String()), nil
}

// ParseTextFindings analisa o texto extraído do PDF e fatias em achados/findings estruturados de atos de pessoal e contratos.
func (p *PDFParser) ParseTextFindings(content, editionNumber string, editionID int64) []domain.Finding {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	findings := make([]domain.Finding, 0)
	now := time.Now()

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 15 {
			continue
		}

		// Bloco de contexto ao redor (3 linhas acima / abaixo)
		startIdx := i - 3
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := i + 4
		if endIdx > len(lines) {
			endIdx = len(lines)
		}
		contextBlock := strings.Join(lines[startIdx:endIdx], " ")

		isHRLine := false

		// 1. Ingestão de Exoneração
		if matches := reExoneracao.FindStringSubmatch(trimmed); len(matches) > 3 {
			servidor := strings.TrimSpace(matches[3])
			if isValidPersonName(servidor) {
				isHRLine = true
				cleanName := cleanPersonName(servidor)
				cpfVal := extractMatch(contextBlock, reCPF)
				matVal := extractMatch(contextBlock, reMatricula)

				extID := generateExternalHash(editionNumber, "EXONERACAO", cleanName, cpfVal, matVal, trimmed)
				findings = append(findings, domain.Finding{
					ID:           uuid.New(),
					EditionID:    editionID,
					ExternalID:   extID,
					ActType:      "EXONERACAO",
					ServidorNome: strPtr(cleanName),
					CPF:          strPtr(cpfVal),
					Matricula:    strPtr(matVal),
					RawContent:   trimmed,
					CreatedAt:    now,
				})
			}
		}

		// 2. Ingestão de Nomeação / Contratação
		if matches := reNomeacao.FindStringSubmatch(trimmed); len(matches) > 2 {
			servidor := strings.TrimSpace(matches[2])
			if isValidPersonName(servidor) {
				isHRLine = true
				cleanName := cleanPersonName(servidor)
				cpfVal := extractMatch(contextBlock, reCPF)
				matVal := extractMatch(contextBlock, reMatricula)

				extID := generateExternalHash(editionNumber, "NOMEACAO", cleanName, cpfVal, matVal, trimmed)
				findings = append(findings, domain.Finding{
					ID:           uuid.New(),
					EditionID:    editionID,
					ExternalID:   extID,
					ActType:      "NOMEACAO",
					ServidorNome: strPtr(cleanName),
					CPF:          strPtr(cpfVal),
					Matricula:    strPtr(matVal),
					RawContent:   trimmed,
					CreatedAt:    now,
				})
			}
		}

		// 3. Ingestão de Relotação / Mudança de Setor
		if matches := reMudanca.FindStringSubmatch(trimmed); len(matches) > 2 {
			servidor := strings.TrimSpace(matches[2])
			if isValidPersonName(servidor) {
				isHRLine = true
				cleanName := cleanPersonName(servidor)
				cpfVal := extractMatch(contextBlock, reCPF)
				matVal := extractMatch(contextBlock, reMatricula)

				extID := generateExternalHash(editionNumber, "MUDANCA_SETOR", cleanName, cpfVal, matVal, trimmed)
				findings = append(findings, domain.Finding{
					ID:           uuid.New(),
					EditionID:    editionID,
					ExternalID:   extID,
					ActType:      "MUDANCA_SETOR",
					ServidorNome: strPtr(cleanName),
					CPF:          strPtr(cpfVal),
					Matricula:    strPtr(matVal),
					RawContent:   trimmed,
					CreatedAt:    now,
				})
			}
		}

		// 4. Ingestão de Contratos e Licitações
		if !isHRLine {
			upperLine := strings.ToUpper(trimmed)
			if strings.Contains(upperLine, "CONTRATO") || strings.Contains(upperLine, "EMPRESA") || reCNPJ.MatchString(trimmed) {
				cnpjVal := extractMatch(contextBlock, reCNPJ)
				valorStr := extractMatch(contextBlock, reValor)
				var valorNum *float64
				if valorStr != "" {
					v := parseValorMonetario(valorStr)
					valorNum = &v
				}

				empresaMatch := reEmpresa.FindStringSubmatch(contextBlock)
				empresaNome := ""
				if len(empresaMatch) > 2 {
					empresaNome = strings.TrimSpace(empresaMatch[2])
				}

				if cnpjVal != "" || empresaNome != "" || valorNum != nil {
					extID := generateExternalHash(editionNumber, "CONTRATO", empresaNome, cnpjVal, valorStr, trimmed)
					findings = append(findings, domain.Finding{
						ID:          uuid.New(),
						EditionID:   editionID,
						ExternalID:  extID,
						ActType:     "CONTRATO",
						EmpresaNome: strPtr(empresaNome),
						CNPJ:        strPtr(cnpjVal),
						Valor:       valorNum,
						RawContent:  trimmed,
						CreatedAt:   now,
					})
				}
			}
		}
	}

	return findings
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
	words := strings.Fields(name)
	if len(words) < 2 || len(words) > 6 {
		return false
	}
	forbidden := []string{"SECRETARIA", "PREFEITURA", "MUNICIPAL", "PORTARIA", "RESOLVE", "ARTIGO", "CARGO", "TABELA"}
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
