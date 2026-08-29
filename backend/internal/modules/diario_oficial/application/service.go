// Package application implementa os casos de uso do módulo diario_oficial
// (§22/§34/§35): CreateDiarioOficialJob (o fluxo disparado via HTTP) e
// ProcessJob/HandleDeadLetter (a execução do lado do worker).
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/gazette"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/platform/audit"
	"github.com/yurythx/projeto-nova/internal/platform/configflags"
	"github.com/yurythx/projeto-nova/internal/platform/database"
	"github.com/yurythx/projeto-nova/internal/platform/jobs"
	"github.com/yurythx/projeto-nova/internal/platform/outbox"

	integrations "github.com/yurythx/projeto-nova/internal/modules/integrations/application"
)

const (
	JobType = "diario_oficial.test"

	EventJobCreated   = "diario_oficial.job.created"
	EventJobCompleted = "diario_oficial.job.completed"
	EventJobFailed    = "diario_oficial.job.failed"

	EventIntegrationStatusChanged = "integration.status.changed"

	integrationKey = "diario-oficial"

	FeatureFlagKey = "diario_oficial_scraping_enabled"
)

type jobPayload struct {
	JobID uuid.UUID `json:"job_id"`
}

type rondonopolisEditionPayload struct {
	ID          int64  `json:"id"`
	Number      string `json:"number"`
	PublishDate string `json:"publish_date"`
	DocURL      string `json:"doc_url"`
	Content     string `json:"content"`
}

// SearchKeyManager entrega uma API key do Typesense restrita a busca, para o
// navegador não receber a chave admin. Implementado em infrastructure.
type SearchKeyManager interface {
	EnsureFrontendSearchKey(ctx context.Context) (string, error)
}

type Service struct {
	db                 *pgxpool.Pool
	jobsRepo           *jobs.Repository
	outboxWriter       *outbox.Writer
	client             domain.Client
	rondonopolisClient domain.Client
	repo               domain.Repository
	editionRepo        domain.EditionRepository
	integrations       *integrations.Service
	audit              *audit.Writer
	flags              configflags.Store
	searchKeys         SearchKeyManager
	reindexer          FindingReindexer
	logger             *slog.Logger
}

// NewService constrói o Service. flags pode ser nil — nesse caso a
// checagem de feature flag em CreateTestJob/syncOnce é pulada e a ação é
// sempre permitida, o que mantém testes de aplicação que não se importam
// com feature flags simples de escrever (ver service_test.go).
func NewService(
	db *pgxpool.Pool,
	jobsRepo *jobs.Repository,
	outboxWriter *outbox.Writer,
	client domain.Client,
	repo domain.Repository,
	integrationsSvc *integrations.Service,
	auditWriter *audit.Writer,
	flags configflags.Store,
	logger *slog.Logger,
) *Service {
	return &Service{
		db:           db,
		jobsRepo:     jobsRepo,
		outboxWriter: outboxWriter,
		client:       client,
		repo:         repo,
		integrations: integrationsSvc,
		audit:        auditWriter,
		flags:        flags,
		logger:       logger,
	}
}

func (s *Service) SetEditionRepository(repo domain.EditionRepository) {
	s.editionRepo = repo
}

// WithSearchKeyManager injeta o provedor da API key search-only do Typesense.
func (s *Service) WithSearchKeyManager(m SearchKeyManager) *Service {
	s.searchKeys = m
	return s
}

// GetFrontendSearchKey devolve a API key do Typesense restrita a busca para o
// frontend usar direto (sem a chave admin no navegador).
func (s *Service) GetFrontendSearchKey(ctx context.Context) (string, error) {
	if s.searchKeys == nil {
		return "", apperrors.DependencyUnavailable("busca (Typesense) não configurada").WithCode("SEARCH_UNAVAILABLE")
	}
	return s.searchKeys.EnsureFrontendSearchKey(ctx)
}

// CreateTestJob implementa o fluxo do §34/§72 até o "Commit": cria o job e
// seu evento de outbox disparador atomicamente, e então retorna — quem
// chama (o transport) é responsável pela resposta 202. Nunca chama o
// sistema externo do Diário Oficial diretamente — essa é a
// responsabilidade do worker, acionado de forma assíncrona pelo evento
// que acabou de ser gravado no outbox.
func (s *Service) CreateTestJob(ctx context.Context, correlationID uuid.UUID, requestedBy *uuid.UUID) (*jobs.Job, error) {
	if s.flags != nil {
		enabled, err := s.flags.IsEnabled(ctx, FeatureFlagKey, true)
		if err != nil {
			return nil, fmt.Errorf("diario_oficial: check feature flag: %w", err)
		}
		if !enabled {
			return nil, apperrors.FeatureDisabled(fmt.Sprintf("the %q feature is currently disabled", FeatureFlagKey))
		}
	}

	job, err := jobs.New(JobType, correlationID, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("diario_oficial: build job: %w", err)
	}

	err = database.WithTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.jobsRepo.Create(ctx, tx, job); err != nil {
			return err
		}
		return s.outboxWriter.Write(ctx, tx, EventJobCreated, "job", job.ID.String(), correlationID, jobPayload{JobID: job.ID})
	})
	if err != nil {
		return nil, fmt.Errorf("diario_oficial: create test job: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{
			UserID:        requestedBy,
			Action:        audit.ActionJobCreated,
			ResourceType:  "job",
			ResourceID:    job.ID.String(),
			CorrelationID: &correlationID,
			Metadata:      map[string]any{"job_type": JobType},
		})
	}

	return job, nil
}

// ProcessJob implementa a execução do lado do worker (§35). Em caso de
// falha, registra o resultado da tentativa e retorna o erro para que quem
// chama (o consumer de mensageria) tente de novo conforme §12 —
// deliberadamente NÃO publica ainda uma notificação job.failed, já que o
// job ainda pode ter sucesso numa nova tentativa. Só HandleDeadLetter,
// chamado quando o RabbitMQ desiste, faz isso.
//
// Idempotência (§18): a entrega "pelo menos uma vez" (at-least-once) do
// RabbitMQ significa que o mesmo evento diario_oficial.job.created pode
// chegar mais de uma vez (uma redelivery competindo com um ack, um
// republish de retry cruzando com o original, ...). Se este job já
// alcançou um estado terminal, rodar a verificação de novo seria na
// melhor das hipóteses redundante e, como MarkProcessing/MarkCompleted
// impõem transições de status válidas, na pior das hipóteses daria erro —
// por isso jobs em estado terminal viram um no-op aqui, em vez de serem
// reprocessados.
func (s *Service) ProcessJob(ctx context.Context, jobID uuid.UUID, correlationID uuid.UUID) error {
	current, err := s.jobsRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("diario_oficial: load job %s: %w", jobID, err)
	}
	if current.Status == jobs.StatusCompleted || current.Status == jobs.StatusDeadLetter {
		s.logger.Info("diario_oficial: duplicate delivery of an already-finished job, skipping",
			slog.String("job_id", jobID.String()), slog.String("status", string(current.Status)))
		return nil
	}

	checkResult, checkErr := s.client.Check(ctx)

	txErr := database.WithTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.jobsRepo.MarkProcessing(ctx, tx, jobID); err != nil {
			return err
		}

		if checkErr != nil {
			return s.jobsRepo.MarkFailed(ctx, tx, jobID, checkErr.Error())
		}
		return s.jobsRepo.MarkCompleted(ctx, tx, jobID, checkResult)
	})
	if txErr != nil {
		return fmt.Errorf("diario_oficial: record processing outcome: %w", txErr)
	}

	if checkErr != nil {
		s.logger.Warn("diario_oficial: check failed, will retry", slog.String("job_id", jobID.String()), slog.Any("error", checkErr))
		return checkErr
	}

	// Sucesso: publica job.completed e atualiza o status da integração
	// numa única transação — a mesma garantia usada na criação do job
	// (§16), para que "o job terminou" e "o status da integração
	// mudou" nunca fiquem inconsistentes entre si.
	err = database.WithTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.outboxWriter.Write(ctx, tx, EventJobCompleted, "job", jobID.String(), correlationID, jobPayload{JobID: jobID}); err != nil {
			return err
		}
		updated, changed, err := s.integrations.RecordCheckResult(ctx, tx, integrationKey, true, "")
		if err != nil {
			return err
		}
		if changed {
			return s.outboxWriter.Write(ctx, tx, EventIntegrationStatusChanged, "integration", integrationKey, correlationID, integrationStatusPayload(updated.Key, string(updated.Status)))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("diario_oficial: publish completion: %w", err)
	}
	return nil
}

// HandleDeadLetter é chamado quando o RabbitMQ já esgotou
// RABBITMQ_MAX_RETRIES para a mensagem deste job: é o desfecho terminal,
// então é aqui que o job transiciona para dead_letter e a notificação
// job.failed voltada para o usuário é finalmente publicada — antes disso,
// toda falha era tratada como "ainda pode ter sucesso numa nova
// tentativa" (ver ProcessJob) e não gerava notificação nenhuma.
func (s *Service) HandleDeadLetter(ctx context.Context, jobID uuid.UUID, correlationID uuid.UUID, reason string) error {
	err := database.WithTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.jobsRepo.MarkDeadLetter(ctx, tx, jobID, reason); err != nil {
			return err
		}
		if err := s.outboxWriter.Write(ctx, tx, EventJobFailed, "job", jobID.String(), correlationID, jobPayload{JobID: jobID}); err != nil {
			return err
		}
		updated, changed, err := s.integrations.RecordCheckResult(ctx, tx, integrationKey, false, reason)
		if err != nil {
			return err
		}
		if changed {
			return s.outboxWriter.Write(ctx, tx, EventIntegrationStatusChanged, "integration", integrationKey, correlationID, integrationStatusPayload(updated.Key, string(updated.Status)))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("diario_oficial: handle dead letter: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{
			Action:        audit.ActionJobFailed,
			ResourceType:  "job",
			ResourceID:    jobID.String(),
			CorrelationID: &correlationID,
			Metadata:      map[string]any{"reason": reason},
		})
	}
	return nil
}

func integrationStatusPayload(key, status string) any {
	return struct {
		Key    string `json:"key"`
		Status string `json:"status"`
	}{Key: key, Status: status}
}

// WithRondonopolisClient injeta o adapter de integração com Rondonópolis.
func (s *Service) WithRondonopolisClient(client domain.Client) *Service {
	s.rondonopolisClient = client
	return s
}

// GetRondonopolisRawFeed retorna todas as edições/registros brutos retornados pela API de Rondonópolis.
func (s *Service) GetRondonopolisRawFeed(ctx context.Context, search string) ([]domain.SearchResultItem, error) {
	if s.rondonopolisClient == nil {
		return nil, apperrors.DependencyUnavailable("Rondonópolis integration is not configured").WithCode("INTEGRATION_UNAVAILABLE")
	}

	searchQuery := domain.SearchQuery{
		FreeText: search,
	}

	res, err := s.rondonopolisClient.Search(ctx, searchQuery)
	if err != nil || res == nil {
		return []domain.SearchResultItem{}, nil
	}

	return res.Items, nil
}

// GetRondonopolisHREvents busca atos de pessoal no PostgreSQL (findings já
// parseados) e, complementarmente, no scraping ao vivo da API de Rondonópolis.
//
// Garantias:
//   - eventType é normalizado para o vocabulário canônico antes de filtrar;
//   - PublicationDate é a data REAL da edição (edition_date), não a data de
//     ingestão do ETL — é por ela que a lista é ordenada cronologicamente;
//   - DocURL aponta para a página exata do PDF oficial (#page=N) quando a
//     página é conhecida;
//   - resultados das duas fontes são deduplicados por (servidor, tipo, edição).
func (s *Service) GetRondonopolisHREvents(ctx context.Context, eventType HREventType, search string, since *time.Time) ([]HREvent, error) {
	canonType := HREventType(gazette.NormalizeActType(string(eventType)))
	allEvents := make([]HREvent, 0)
	seen := make(map[string]bool)

	dedupKey := func(ev HREvent) string {
		return strings.ToLower(strings.TrimSpace(ev.Servidor)) + "|" + string(ev.Type) + "|" + strings.TrimSpace(ev.EditionNumber)
	}
	add := func(ev HREvent) {
		k := dedupKey(ev)
		if seen[k] {
			return
		}
		seen[k] = true
		allEvents = append(allEvents, ev)
	}

	// 1. Fonte primária: findings persistidos no PostgreSQL (Trigram + FTS).
	if s.editionRepo != nil {
		findings, _, fErr := s.editionRepo.SearchFindings(ctx, search, string(canonType), 200, 0)
		if fErr == nil {
			for _, f := range findings {
				evType := HREventType(gazette.NormalizeActType(f.ActType))
				if evType == "" || evType == HREventType(gazette.ActContrato) {
					continue // contrato não é ato de pessoal
				}

				pubDate := f.EditionDate
				if pubDate.IsZero() {
					pubDate = f.CreatedAt
				}

				add(HREvent{
					Type:              evType,
					Servidor:          derefStr(f.ServidorNome),
					ServidorCPF:       derefStr(f.CPF),
					ServidorMatricula: derefStr(f.Matricula),
					Secretaria:        firstNonEmpty(derefStr(f.Secretaria), "Prefeitura Municipal de Rondonópolis"),
					Cargo:             firstNonEmpty(derefStr(f.JobRole), extractCargo(f.RawContent)),
					DASLevel:          firstNonEmpty(derefStr(f.DASLevel), extractDASLevel(f.RawContent)),
					PortariaNumber:    firstNonEmpty(derefStr(f.PortariaNumber), firstSubmatchStr(f.RawContent, porRegex)),
					EditionNumber:     f.EditionNumber,
					ContextSnippet:    f.RawContent,
					PublicationDate:   pubDate,
					DocURL:            withPDFPage(f.PdfURL, f.PDFPageNumber),
					PDFPageNumber:     f.PDFPageNumber,
				})
			}
		}
	}

	// 2. Complemento: scraping ao vivo (edições ainda não ingeridas).
	if s.rondonopolisClient != nil {
		res, err := s.rondonopolisClient.Search(ctx, domain.SearchQuery{FreeText: search, Since: since})
		if err == nil && res != nil {
			for _, item := range res.Items {
				var ed rondonopolisEditionPayload
				if err := json.Unmarshal(item.RawPayload, &ed); err != nil {
					continue
				}
				for _, ev := range ExtractHREvents(ed.Content, ed.Number, ed.DocURL, item.AvailabilityDate) {
					if canonType != "" && ev.Type != canonType {
						continue
					}
					if search != "" {
						sLower := strings.ToLower(search)
						if !strings.Contains(strings.ToLower(ev.Servidor), sLower) &&
							!strings.Contains(strings.ToLower(ev.ServidorCPF), sLower) &&
							!strings.Contains(strings.ToLower(ev.ServidorMatricula), sLower) &&
							!strings.Contains(strings.ToLower(ev.ContextSnippet), sLower) {
							continue
						}
					}
					ev.DocURL = withPDFPage(ev.DocURL, ev.PDFPageNumber)
					add(ev)
				}
			}
		}
	}

	// Ordenação: busca vazia -> alfabética por servidor (listagem/navegação);
	// busca com termo -> cronológica pela data real da edição, mais recente
	// primeiro, com o nome como desempate estável.
	if search == "" {
		sort.Slice(allEvents, func(i, j int) bool {
			return strings.ToLower(allEvents[i].Servidor) < strings.ToLower(allEvents[j].Servidor)
		})
	} else {
		sort.SliceStable(allEvents, func(i, j int) bool {
			if !allEvents[i].PublicationDate.Equal(allEvents[j].PublicationDate) {
				return allEvents[i].PublicationDate.After(allEvents[j].PublicationDate)
			}
			return strings.ToLower(allEvents[i].Servidor) < strings.ToLower(allEvents[j].Servidor)
		})
	}

	return allEvents, nil
}

// withPDFPage acrescenta a âncora de página à URL da edição quando a página é
// conhecida (>1), para o link abrir direto na página do ato filtrado.
func withPDFPage(url string, page int) string {
	if url == "" || page <= 1 || strings.Contains(url, "#page=") {
		return url
	}
	return fmt.Sprintf("%s#page=%d", url, page)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstSubmatchStr(text string, re *regexp.Regexp) string {
	if m := re.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

type PublicContract struct {
	ID                string    `json:"id"`
	ContractNumber    string    `json:"contract_number"`
	ContractType      string    `json:"contract_type"`
	PortariaNumber    string    `json:"portaria_number"`
	NomeacaoDate      time.Time `json:"nomeacao_date"`
	Object            string    `json:"object"`
	Contractor        string    `json:"contractor"`
	ContractorCNPJ    string    `json:"contractor_cnpj"`
	Value             string    `json:"value"`
	FiscalNome        string    `json:"fiscal_nome"`
	FiscalCPF         string    `json:"fiscal_cpf"`
	FiscalMatricula   string    `json:"fiscal_matricula"`
	SuplenteNome      string    `json:"suplente_nome"`
	SuplenteCPF       string    `json:"suplente_cpf"`
	SuplenteMatricula string    `json:"suplente_matricula"`
	EditionNumber     string    `json:"edition_number"`
	PublicationDate   time.Time `json:"publication_date"`
	DocURL            string    `json:"doc_url"`
	Status            string    `json:"status"`
}

// GetPublicContracts retorna a listagem completa de contratos novos e existentes com fiscais e suplentes no Diário Oficial.
func (s *Service) GetPublicContracts(ctx context.Context, search string) ([]PublicContract, error) {
	contracts := make([]PublicContract, 0)

	// Integração com banco PostgreSQL para busca de contratos indexados
	if s.editionRepo != nil {
		findings, _, fErr := s.editionRepo.SearchFindings(ctx, search, "CONTRATO", 100, 0)
		if fErr == nil && len(findings) > 0 {
			for _, f := range findings {
				fName := ""
				if f.ServidorNome != nil {
					fName = *f.ServidorNome
				}
				cpf := ""
				if f.CPF != nil {
					cpf = *f.CPF
				}
				mat := ""
				if f.Matricula != nil {
					mat = *f.Matricula
				}
				empresa := ""
				if f.EmpresaNome != nil {
					empresa = *f.EmpresaNome
				}
				cnpj := ""
				if f.CNPJ != nil {
					cnpj = *f.CNPJ
				}
				valor := "R$ 0,00"
				if f.Valor != nil {
					valor = fmt.Sprintf("R$ %.2f", *f.Valor)
				}
				docURL := withPDFPage(f.PdfURL, f.PDFPageNumber)
				if docURL == "" {
					docURL = "https://www.rondonopolis.mt.gov.br/diario-oficial/"
				}
				edNum := f.EditionNumber
				if edNum == "" {
					edNum = "Edição Registrada"
				}
				// Data real de publicação da edição; a data de ingestão só
				// entra como último recurso.
				pubDate := f.EditionDate
				if pubDate.IsZero() {
					pubDate = f.CreatedAt
				}
				portaria := firstNonEmpty(derefStr(f.PortariaNumber), "Portaria Interna")

				contracts = append(contracts, PublicContract{
					ID:              f.ID.String(),
					ContractNumber:  "Contrato Indexado",
					ContractType:    "Prestação de Serviços",
					PortariaNumber:  portaria,
					NomeacaoDate:    pubDate,
					Object:          f.RawContent,
					Contractor:      empresa,
					ContractorCNPJ:  cnpj,
					Value:           valor,
					FiscalNome:      fName,
					FiscalCPF:       cpf,
					FiscalMatricula: mat,
					SuplenteNome:    "Fiscal Suplente",
					EditionNumber:   edNum,
					PublicationDate: pubDate,
					DocURL:          docURL,
					Status:          "ATIVO",
				})
			}
		}
	}

	if search == "" {
		return contracts, nil
	}

	filtered := make([]PublicContract, 0)
	sLower := strings.ToLower(search)
	sDigits := regexp.MustCompile(`\D`).ReplaceAllString(sLower, "")

	for _, c := range contracts {
		fCpfDigits := regexp.MustCompile(`\D`).ReplaceAllString(c.FiscalCPF, "")
		fMatDigits := regexp.MustCompile(`\D`).ReplaceAllString(c.FiscalMatricula, "")
		sCpfDigits := regexp.MustCompile(`\D`).ReplaceAllString(c.SuplenteCPF, "")
		sMatDigits := regexp.MustCompile(`\D`).ReplaceAllString(c.SuplenteMatricula, "")
		cnpjDigits := regexp.MustCompile(`\D`).ReplaceAllString(c.ContractorCNPJ, "")

		matchesFiscal := strings.Contains(strings.ToLower(c.FiscalNome), sLower) ||
			strings.Contains(strings.ToLower(c.FiscalCPF), sLower) ||
			(len(sDigits) >= 3 && strings.Contains(fCpfDigits, sDigits)) ||
			strings.Contains(strings.ToLower(c.FiscalMatricula), sLower) ||
			(len(sDigits) >= 3 && strings.Contains(fMatDigits, sDigits))

		matchesSuplente := strings.Contains(strings.ToLower(c.SuplenteNome), sLower) ||
			strings.Contains(strings.ToLower(c.SuplenteCPF), sLower) ||
			(len(sDigits) >= 3 && strings.Contains(sCpfDigits, sDigits)) ||
			strings.Contains(strings.ToLower(c.SuplenteMatricula), sLower) ||
			(len(sDigits) >= 3 && strings.Contains(sMatDigits, sDigits))

		matchesContractor := strings.Contains(strings.ToLower(c.Contractor), sLower) ||
			strings.Contains(strings.ToLower(c.ContractorCNPJ), sLower) ||
			(len(sDigits) >= 3 && strings.Contains(cnpjDigits, sDigits))

		matchesMeta := strings.Contains(strings.ToLower(c.ContractNumber), sLower) ||
			strings.Contains(strings.ToLower(c.ContractType), sLower) ||
			strings.Contains(strings.ToLower(c.PortariaNumber), sLower) ||
			strings.Contains(strings.ToLower(c.Object), sLower)

		if matchesFiscal || matchesSuplente || matchesContractor || matchesMeta {
			filtered = append(filtered, c)
		}
	}

	return filtered, nil
}

// GetEditions retorna o status de processamento ETL de todas as edições no banco.
func (s *Service) GetEditions(ctx context.Context, limit int) ([]domain.Edition, error) {
	if s.editionRepo == nil {
		return []domain.Edition{}, nil
	}
	return s.editionRepo.ListAllEditions(ctx, limit)
}

// GetReviewQueue retorna os findings de baixa confiança — o resíduo do parser
// que ficou fora da busca do usuário e precisa de revisão manual.
func (s *Service) GetReviewQueue(ctx context.Context, limit int) ([]domain.Finding, error) {
	if s.editionRepo == nil {
		return []domain.Finding{}, nil
	}
	return s.editionRepo.ListFindingsForReview(ctx, limit)
}

// FindingReindexer é o caminho para (re)sincronizar uma edição inteira com o
// Typesense depois que um finding dela muda. Implementado por worker.Reindexer;
// injetado com WithReindexer. Sem ele, promover/descartar ainda atualiza o
// PostgreSQL, só não reflete no índice até o próximo reindex de boot.
type FindingReindexer interface {
	ReindexEdition(ctx context.Context, editionID int64) (int, error)
}

// WithReindexer injeta o reindexador do Typesense usado pela fila de revisão.
func (s *Service) WithReindexer(r FindingReindexer) *Service {
	s.reindexer = r
	return s
}

// reindexEditionBestEffort ressincroniza a edição do finding com o Typesense.
// Nunca falha a operação de revisão por causa do índice — só loga.
func (s *Service) reindexEditionBestEffort(ctx context.Context, editionID int64, action string) {
	if s.reindexer == nil {
		return
	}
	if n, err := s.reindexer.ReindexEdition(ctx, editionID); err != nil {
		s.logger.Warn("revisão: reindex da edição falhou (best-effort)",
			slog.String("acao", action), slog.Int64("edition_id", editionID), slog.Any("erro", err))
	} else {
		s.logger.Info("revisão: edição reindexada",
			slog.String("acao", action), slog.Int64("edition_id", editionID), slog.Int("docs", n))
	}
}

var allowedReviewConfidence = map[string]bool{
	gazette.ConfidenceHigh:   true,
	gazette.ConfidenceMedium: true,
}

// allowedReviewActType: os tipos canônicos de pessoal que a UI filtra, mais
// CONTRATO e OUTROS — tudo que um revisor pode legitimamente atribuir.
var allowedReviewActType = func() map[string]bool {
	m := map[string]bool{gazette.ActContrato: true, gazette.ActOutros: true}
	for _, t := range gazette.CanonicalActTypes {
		m[t] = true
	}
	return m
}()

// PromoteFinding aplica a correção manual do revisor a um finding da fila e o
// sobe para a busca (confidence high|medium). Reindexa a edição no Typesense.
func (s *Service) PromoteFinding(ctx context.Context, id uuid.UUID, in domain.FindingReviewInput, reviewedBy *uuid.UUID) (*domain.Finding, error) {
	if s.editionRepo == nil {
		return nil, apperrors.BadRequest("repositório de edições indisponível")
	}
	in.ActType = strings.ToUpper(strings.TrimSpace(in.ActType))
	if !allowedReviewActType[in.ActType] {
		return nil, apperrors.BadRequest("act_type inválido: " + in.ActType)
	}
	in.Confidence = strings.ToLower(strings.TrimSpace(in.Confidence))
	if !allowedReviewConfidence[in.Confidence] {
		return nil, apperrors.BadRequest("confidence deve ser 'high' ou 'medium' ao promover")
	}
	in.ServidorNome = trimPtr(in.ServidorNome)
	in.EmpresaNome = trimPtr(in.EmpresaNome)
	if in.ServidorNome == nil && in.EmpresaNome == nil {
		return nil, apperrors.BadRequest("informe ao menos servidor_nome ou empresa_nome")
	}
	in.CPF = trimPtr(in.CPF)
	in.Matricula = trimPtr(in.Matricula)
	in.CNPJ = trimPtr(in.CNPJ)
	in.Secretaria = trimPtr(in.Secretaria)
	in.JobRole = trimPtr(in.JobRole)
	in.DASLevel = trimPtr(in.DASLevel)
	in.PortariaNumber = trimPtr(in.PortariaNumber)

	updated, err := s.editionRepo.UpdateFindingReview(ctx, id, in, reviewedBy)
	if err != nil {
		return nil, err
	}

	s.reindexEditionBestEffort(ctx, updated.EditionID, "promover")

	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{
			UserID:       reviewedBy,
			Action:       "diario_oficial.finding.promoted",
			ResourceType: "diario_oficial_finding",
			ResourceID:   id.String(),
			Metadata: map[string]any{
				"act_type":   updated.ActType,
				"confidence": updated.Confidence,
				"edition_id": updated.EditionID,
			},
		})
	}
	return updated, nil
}

// AcknowledgeFinding tira o finding da fila sem promovê-lo — o revisor
// confirmou que é low mas legítimo (fica só no PostgreSQL).
func (s *Service) AcknowledgeFinding(ctx context.Context, id uuid.UUID, note string, reviewedBy *uuid.UUID) error {
	if s.editionRepo == nil {
		return apperrors.BadRequest("repositório de edições indisponível")
	}
	if err := s.editionRepo.AcknowledgeFinding(ctx, id, reviewedBy, strings.TrimSpace(note)); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{
			UserID:       reviewedBy,
			Action:       "diario_oficial.finding.acknowledged",
			ResourceType: "diario_oficial_finding",
			ResourceID:   id.String(),
		})
	}
	return nil
}

// DiscardFinding remove um finding da fila de revisão em definitivo (ruído do
// parser). Reindexa a edição para tirar o doc do Typesense caso já estivesse lá.
func (s *Service) DiscardFinding(ctx context.Context, id uuid.UUID, reviewedBy *uuid.UUID) error {
	if s.editionRepo == nil {
		return apperrors.BadRequest("repositório de edições indisponível")
	}
	existing, err := s.editionRepo.GetFindingByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.editionRepo.DeleteFinding(ctx, id); err != nil {
		return err
	}
	s.reindexEditionBestEffort(ctx, existing.EditionID, "descartar")
	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{
			UserID:       reviewedBy,
			Action:       "diario_oficial.finding.discarded",
			ResourceType: "diario_oficial_finding",
			ResourceID:   id.String(),
			Metadata:     map[string]any{"edition_id": existing.EditionID, "act_type": existing.ActType},
		})
	}
	return nil
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
