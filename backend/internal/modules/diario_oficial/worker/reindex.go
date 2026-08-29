package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/gazette"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/platform/metrics"
	"github.com/yurythx/projeto-nova/pkg/typesense"
)

// Reindexer reconstrói as coleções do Typesense a partir da fonte da verdade:
// as linhas de diario_oficial_findings já parseadas e persistidas no
// PostgreSQL (com os metadados da edição via JOIN). Antes disso o Typesense só
// era escrito no exato momento da ingestão de uma edição nova — qualquer
// edição já COMPLETED antes do cliente Typesense existir, ou uma indexação
// best-effort que falhou em silêncio, ficava para sempre fora do índice. Era
// por isso que as duas coleções estavam com 0 documentos apesar de centenas de
// findings no banco.
//
// O act_type decide a coleção de destino: atos de pessoal vão para
// diorondon_personnel_acts; CONTRATO vai para diorondon_articles.
type Reindexer struct {
	repo     domain.EditionRepository
	tsClient *typesense.Client
	logger   *slog.Logger
}

func NewReindexer(repo domain.EditionRepository, tsClient *typesense.Client, logger *slog.Logger) *Reindexer {
	return &Reindexer{repo: repo, tsClient: tsClient, logger: logger}
}

// ReindexAll varre TODOS os findings por keyset e faz upsert em lote nas duas
// coleções. Idempotente: o id do documento é derivado do id (uuid) do finding,
// então rodar de novo sobrescreve em vez de duplicar.
//
// recreate=true apaga e recria as coleções antes de indexar — necessário para
// aplicar mudanças de schema (o Typesense não permite alterá-lo in-place) e
// para remover documentos órfãos de findings que deixaram de existir.
func (r *Reindexer) ReindexAll(ctx context.Context, recreate bool) (int, error) {
	if r.tsClient == nil {
		return 0, fmt.Errorf("reindex: typesense client not configured")
	}
	if recreate {
		if err := r.tsClient.RecreateCollections(ctx); err != nil {
			return 0, fmt.Errorf("reindex: recreate collections: %w", err)
		}
	} else if err := r.tsClient.EnsureCollections(ctx); err != nil {
		return 0, fmt.Errorf("reindex: ensure collections: %w", err)
	}

	const batch = 500
	after := uuid.Nil
	total := 0
	for {
		findings, err := r.repo.ListFindingsForIndex(ctx, after, batch)
		if err != nil {
			return total, fmt.Errorf("reindex: list findings: %w", err)
		}
		if len(findings) == 0 {
			break
		}
		n, err := r.indexBatch(ctx, findings)
		total += n
		if err != nil {
			return total, err
		}
		after = findings[len(findings)-1].ID
		if len(findings) < batch {
			break
		}
	}

	if r.logger != nil {
		r.logger.Info("Typesense: reindex completo", "documentos_indexados", total)
	}
	return total, nil
}

// ReindexEdition reindexa apenas os findings de uma edição — chamado logo após
// a ingestão, no lugar de re-rodar o parser de blocos (que praticamente não
// casa com o texto real do DIORONDON).
//
// Como o DELETE+reinsert em SaveFindingsTx dá aos findings UUIDs novos a cada
// reprocessamento, os documentos antigos da edição no Typesense ficariam
// órfãos. Por isso apaga primeiro tudo daquela edição nas duas coleções e só
// então reindexa o conjunto atual.
func (r *Reindexer) ReindexEdition(ctx context.Context, editionID int64) (int, error) {
	if r.tsClient == nil {
		return 0, nil
	}
	if err := r.tsClient.EnsureCollections(ctx); err != nil {
		return 0, fmt.Errorf("reindex edition: ensure collections: %w", err)
	}
	findings, err := r.repo.ListFindingsByEdition(ctx, editionID)
	if err != nil {
		return 0, fmt.Errorf("reindex edition: list findings: %w", err)
	}

	// Número da edição para o filtro do Typesense (int32). Se a edição não
	// tiver findings (nenhum metadado), pula a limpeza — nada a órfãozar.
	if len(findings) > 0 {
		edNum := gazette.LeadingInt32(findings[0].EditionNumber)
		for _, coll := range []string{"diorondon_personnel_acts", "diorondon_articles"} {
			if _, delErr := r.tsClient.DeleteDocumentsByFilter(ctx, coll, fmt.Sprintf("edition_number:=%d", edNum)); delErr != nil && r.logger != nil {
				r.logger.Warn("Typesense: falha ao limpar docs antigos da edição (best-effort)",
					"collection", coll, "edition_number", edNum, "error", delErr)
			}
		}
	}

	return r.indexBatch(ctx, findings)
}

func (r *Reindexer) indexBatch(ctx context.Context, findings []domain.Finding) (int, error) {
	personnel := make([]interface{}, 0, len(findings))
	articles := make([]interface{}, 0, len(findings))

	skippedLowQuality := 0
	for i := range findings {
		f := findings[i]
		switch gazette.NormalizeActType(f.ActType) {
		case gazette.ActContrato:
			articles = append(articles, articleDocFromFinding(f))
		case gazette.ActOutros, "":
			// Sem classificação útil — não indexa.
		default:
			// Barreira de qualidade na fronteira do índice: o PostgreSQL
			// guarda TODOS os findings (auditoria / reprocessamento), mas a
			// busca do usuário só recebe confiança >= media com nome que de
			// fato parece pessoa física. Filtra o resíduo do parser de regex
			// ("AGRICULTURA E PECUÁRIA", "GT CIB", "ESTADO DE MATO GROSSO").
			if f.Confidence == gazette.ConfidenceLow || !gazette.IsPlausiblePersonName(deref(f.ServidorNome)) {
				skippedLowQuality++
				continue
			}
			personnel = append(personnel, personnelDocFromFinding(f))
		}
	}
	if skippedLowQuality > 0 && r.logger != nil {
		r.logger.Info("Typesense: atos de pessoal de baixa qualidade não indexados (mantidos no PostgreSQL)",
			"count", skippedLowQuality)
	}

	if err := r.tsClient.ImportDocuments(ctx, "diorondon_personnel_acts", personnel); err != nil {
		return 0, fmt.Errorf("reindex: import personnel: %w", err)
	}
	metrics.DiarioTypesenseIndexedTotal.WithLabelValues("diorondon_personnel_acts").Add(float64(len(personnel)))
	if err := r.tsClient.ImportDocuments(ctx, "diorondon_articles", articles); err != nil {
		return len(personnel), fmt.Errorf("reindex: import articles: %w", err)
	}
	metrics.DiarioTypesenseIndexedTotal.WithLabelValues("diorondon_articles").Add(float64(len(articles)))
	return len(personnel) + len(articles), nil
}

// pdfURLForFinding devolve a URL da edição com âncora de página quando a página
// é conhecida (>1), para o link apontar para a página exata do ato.
func pdfURLForFinding(f domain.Finding) string {
	u := f.PdfURL
	if u == "" {
		return ""
	}
	if f.PDFPageNumber > 1 && !strings.Contains(u, "#page=") {
		return fmt.Sprintf("%s#page=%d", u, f.PDFPageNumber)
	}
	return u
}

func editionTypeFromNumber(editionNumber string) string {
	up := strings.ToUpper(strings.TrimSpace(editionNumber))
	if strings.HasSuffix(up, "S") || strings.HasSuffix(up, "E") || strings.Contains(up, "SUPLEMENT") || strings.Contains(up, "EXTRA") {
		return "SUPLEMENTAR"
	}
	return "ORDINARIA"
}

func pubDateUnix(f domain.Finding) int64 {
	if !f.EditionDate.IsZero() {
		return f.EditionDate.Unix()
	}
	// Sem data de edição confiável: usa a data de ingestão só para não
	// mandar um valor absurdamente negativo ao campo de ordenação.
	if !f.CreatedAt.IsZero() {
		return f.CreatedAt.Unix()
	}
	return time.Now().Unix()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func personnelDocFromFinding(f domain.Finding) typesense.DiorondonPersonnelAct {
	page := int32(f.PDFPageNumber)
	if page < 1 {
		page = 1
	}
	var salary float64
	if f.Valor != nil {
		salary = *f.Valor
	}
	return typesense.DiorondonPersonnelAct{
		ID:              "act-" + f.ID.String(),
		EditionNumber:   gazette.LeadingInt32(f.EditionNumber),
		EditionType:     editionTypeFromNumber(f.EditionNumber),
		PublicationDate: pubDateUnix(f),
		ActType:         gazette.NormalizeActType(f.ActType),
		PersonName:      deref(f.ServidorNome),
		PersonCPF:       deref(f.CPF),
		PersonMatricula: deref(f.Matricula),
		JobRole:         deref(f.JobRole),
		Secretaria:      deref(f.Secretaria),
		DASLevel:        deref(f.DASLevel),
		SalaryValue:     salary,
		PortariaNumber:  deref(f.PortariaNumber),
		FullActText:     f.RawContent,
		PDFPageNumber:   page,
		PDFStorageURL:   pdfURLForFinding(f),
		Confidence:      f.Confidence,
	}
}

func articleDocFromFinding(f domain.Finding) typesense.DiorondonArticle {
	page := int32(f.PDFPageNumber)
	if page < 1 {
		page = 1
	}
	contractNumbers := []string{}
	if f.PortariaNumber != nil && *f.PortariaNumber != "" {
		contractNumbers = append(contractNumbers, *f.PortariaNumber)
	}
	cnpjs := []string{}
	if f.CNPJ != nil && *f.CNPJ != "" {
		cnpjs = append(cnpjs, *f.CNPJ)
	}
	officials := []string{}
	if f.ServidorNome != nil && *f.ServidorNome != "" {
		officials = append(officials, *f.ServidorNome)
	}
	return typesense.DiorondonArticle{
		ID:              "art-" + f.ID.String(),
		EditionNumber:   gazette.LeadingInt32(f.EditionNumber),
		EditionType:     editionTypeFromNumber(f.EditionNumber),
		PublicationDate: pubDateUnix(f),
		PageNumber:      page,
		ContractNumbers: contractNumbers,
		CNPJs:           cnpjs,
		OfficialsNamed:  officials,
		Content:         f.RawContent,
		PDFStorageURL:   pdfURLForFinding(f),
	}
}
