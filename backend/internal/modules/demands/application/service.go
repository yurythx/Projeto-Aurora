package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
	"github.com/yurythx/projeto-nova/internal/platform/audit"
	"github.com/yurythx/projeto-nova/internal/platform/auth"
	"github.com/yurythx/projeto-nova/internal/platform/database"
	"github.com/yurythx/projeto-nova/internal/platform/outbox"
	"github.com/yurythx/projeto-nova/internal/platform/storage"
)

// ContratoRef é o mínimo que a geração automática de demandas precisa saber
// sobre um contrato vigente.
type ContratoRef struct {
	ID     uuid.UUID
	Numero string
}

// ContractLister é a fonte de contratos vigentes (implementada em app/ sobre
// o repositório de contratos) — evita o módulo demands depender do módulo
// contratos diretamente.
type ContractLister interface {
	ListVigentes(ctx context.Context) ([]ContratoRef, error)
}

// Service coordena os fluxos das Demandas Mensais.
type Service struct {
	db        *pgxpool.Pool
	repo      domain.Repository
	outbox    *outbox.Writer
	storage   storage.Provider
	audit     *audit.Writer
	auditLog  *audit.Reader
	contracts ContractLister
	bucket    string
	logger    *slog.Logger
}

// WithContractLister injeta a fonte de contratos vigentes (geração mensal).
func (s *Service) WithContractLister(cl ContractLister) *Service {
	s.contracts = cl
	return s
}

// WithAuditReader injeta o leitor da trilha de auditoria (painel de
// histórico da demanda).
func (s *Service) WithAuditReader(r *audit.Reader) *Service {
	s.auditLog = r
	return s
}

// DemandHistory devolve a trilha de auditoria de uma demanda (criação,
// mudanças de etapa) da mais recente para a mais antiga.
func (s *Service) DemandHistory(ctx context.Context, demandID uuid.UUID) ([]audit.LogRow, error) {
	if s.auditLog == nil {
		return []audit.LogRow{}, nil
	}
	return s.auditLog.ListByResource(ctx, "demand", demandID.String(), 200)
}

// NewService constrói o Service.
func NewService(db *pgxpool.Pool, repo domain.Repository, outboxWriter *outbox.Writer, storageProvider storage.Provider, auditWriter *audit.Writer, bucket string, logger *slog.Logger) *Service {
	return &Service{db: db, repo: repo, outbox: outboxWriter, storage: storageProvider, audit: auditWriter, bucket: bucket, logger: logger}
}

// MoveKanbanCard tenta mover o card da demanda de uma etapa para outra.
func (s *Service) MoveKanbanCard(ctx context.Context, demandID uuid.UUID, targetEtapa int) error {
	// 1. Busca a demanda com todos os documentos
	demand, err := s.repo.GetByID(ctx, demandID)
	if err != nil {
		return fmt.Errorf("demands service: get demand: %w", err)
	}

	nextStage := domain.EtapaKanban(targetEtapa)
	if nextStage < domain.Etapa1ElaborarOF || nextStage > domain.Etapa6Contabilidade {
		return fmt.Errorf("etapa inválida: %d", targetEtapa)
	}

	// 2. Aciona a Máquina de Estados para checar as travas
	if err := ValidateTransition(demand, nextStage, time.Now()); err != nil {
		s.logger.Warn("bloqueio de transicao no kanban",
			slog.String("demanda_id", demandID.String()),
			slog.Int("de_etapa", int(demand.Etapa)),
			slog.Int("para_etapa", int(nextStage)),
			slog.String("erro", err.Error()),
		)
		// Propaga o erro validado para a camada de HTTP formatar adequadamente (Bad Request)
		return err
	}

	// 3. Efetiva a transição
	oldStage := demand.Etapa
	demand.Etapa = nextStage
	demand.StatusEtapa = domain.StatusPendente
	demand.EtapaStartedAt = time.Now()

	// 4. Salva a transição e publica o evento atomica e confiavelmente (Outbox)
	correlationID := uuid.New()
	err = database.WithTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		// Precisamos converter s.repo para uma interface que aceita Tx,
		// ou fazer cast para PostgresRepository localmente.
		// Para simplificar no Go, se o repo suportar transação:
		type txUpdater interface {
			UpdateTx(context.Context, pgx.Tx, domain.MonthlyDemand) error
		}
		if updater, ok := s.repo.(txUpdater); ok {
			if err := updater.UpdateTx(ctx, tx, demand); err != nil {
				return err
			}
		} else {
			// Fallback (não atômico) se o mock não implementar UpdateTx
			if err := s.repo.Update(ctx, demand); err != nil {
				return err
			}
		}

		eventPayload := domain.DemandEtapaChangedEvent{
			DemandID:   demandID,
			ContratoID: demand.ContratoID,
			OldEtapa:   oldStage,
			NewEtapa:   nextStage,
			OccurredAt: demand.EtapaStartedAt,
		}

		return s.outbox.Write(ctx, tx, eventPayload.EventType(), eventPayload.AggregateType(), eventPayload.AggregateID(), correlationID, eventPayload)
	})

	if err != nil {
		return fmt.Errorf("demands service: move kanban atomico: %w", err)
	}

	if s.audit != nil {
		actorID := s.actorID(ctx)
		_ = s.audit.Record(ctx, audit.Entry{
			UserID:        actorID,
			Action:        "demand.etapa_changed",
			ResourceType:  "demand",
			ResourceID:    demandID.String(),
			CorrelationID: &correlationID,
			Metadata: map[string]any{
				"old_etapa": oldStage,
				"new_etapa": nextStage,
			},
		})
	}

	s.logger.Info("kanban card movido",
		slog.String("demanda_id", demandID.String()),
		slog.Int("de_etapa", int(oldStage)),
		slog.Int("para_etapa", int(nextStage)),
		slog.String("correlation_id", correlationID.String()),
	)
	return nil
}

// EnsureMonthlyDemands garante que todo contrato vigente tem uma demanda
// mensal (na Etapa 1) para o mês `anoMes` (formato "2006-01"). Idempotente —
// só cria as que faltam. Chamado por um loop do worker.
func (s *Service) EnsureMonthlyDemands(ctx context.Context, anoMes string) (int, error) {
	if s.contracts == nil {
		return 0, nil // sem fonte de contratos configurada
	}
	vigentes, err := s.contracts.ListVigentes(ctx)
	if err != nil {
		return 0, fmt.Errorf("demands service: list vigentes: %w", err)
	}
	created := 0
	for _, c := range vigentes {
		exists, err := s.repo.ExistsForContractMonth(ctx, c.ID, anoMes)
		if err != nil {
			s.logger.Warn("demands: checar demanda mensal existente falhou", slog.String("contrato", c.Numero), slog.Any("erro", err))
			continue
		}
		if exists {
			continue
		}
		now := time.Now()
		d := domain.MonthlyDemand{
			ID: uuid.New(), ContratoID: c.ID, AnoMes: anoMes,
			Etapa: domain.Etapa1ElaborarOF, StatusEtapa: domain.StatusPendente,
			Observacoes:    "Demanda mensal gerada automaticamente.",
			EtapaStartedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.repo.Create(ctx, d); err != nil {
			s.logger.Warn("demands: criar demanda mensal automática falhou", slog.String("contrato", c.Numero), slog.Any("erro", err))
			continue
		}
		created++
	}
	if created > 0 {
		s.logger.Info("demands: demandas mensais geradas automaticamente", slog.String("ano_mes", anoMes), slog.Int("criadas", created))
	}
	return created, nil
}

// SweepSLA varre as demandas paradas além do prazo da etapa e abre uma
// pendência (contract_occurrences) para cada — uma por demanda/tipo enquanto
// não resolvida. Chamado por um loop do worker.
func (s *Service) SweepSLA(ctx context.Context) (int, error) {
	now := time.Now()
	// Só precisa avaliar demandas cuja etapa começou há mais de 5 dias (o
	// menor SLA); o cálculo fino é por etapa em domain.ComputeSLA.
	stale, err := s.repo.ListStale(ctx, now.AddDate(0, 0, -5))
	if err != nil {
		return 0, fmt.Errorf("demands service: list stale: %w", err)
	}
	opened := 0
	for _, d := range stale {
		sla := domain.ComputeSLA(d.Etapa, d.EtapaStartedAt, now)
		if !sla.Breached {
			continue
		}
		occ := domain.Occurrence{
			ID: uuid.New(), ContratoID: d.ContratoID, DemandaID: &d.ID,
			Tipo: domain.OccurrenceSLABreach,
			Descricao: fmt.Sprintf("Demanda %s parada há %d dias na etapa %d (SLA %d dias).",
				d.AnoMes, sla.DaysInStage, d.Etapa, sla.SLADays),
			SLAVenceEm: sla.DueAt,
		}
		created, err := s.repo.EnsureSLAOccurrence(ctx, occ)
		if err != nil {
			s.logger.Warn("demands: abrir pendência de SLA falhou", slog.String("demanda", d.ID.String()), slog.Any("erro", err))
			continue
		}
		if created {
			opened++
		}
	}
	if opened > 0 {
		s.logger.Info("demands: pendências de SLA abertas", slog.Int("count", opened))
	}
	return opened, nil
}

// OpenOccurrences lista as pendências não resolvidas (dashboard/alertas).
func (s *Service) OpenOccurrences(ctx context.Context, limit int) ([]domain.Occurrence, error) {
	return s.repo.ListOpenOccurrences(ctx, limit)
}

// NextRequirements devolve o checklist da PRÓXIMA etapa de uma demanda — o
// que já está anexado, o que falta e o que está vencido — para o Kanban
// mostrar o cadeado e a lista antes de o fiscal tentar arrastar o card.
func (s *Service) NextRequirements(ctx context.Context, demandID uuid.UUID) (StageCheck, error) {
	demand, err := s.repo.GetByID(ctx, demandID)
	if err != nil {
		return StageCheck{}, fmt.Errorf("demands service: get demand: %w", err)
	}
	target := demand.Etapa + 1
	if target > domain.Etapa6Contabilidade {
		// Última etapa: nada a exigir para "avançar".
		return StageCheck{FromEtapa: int(demand.Etapa), ToEtapa: int(demand.Etapa), CanAdvance: false, Docs: []DocStatus{}}, nil
	}
	return CheckAdvance(demand, target, time.Now()), nil
}

// KanbanView retorna o quadro completo para exibição.
func (s *Service) KanbanView(ctx context.Context) (map[domain.EtapaKanban][]domain.MonthlyDemand, error) {
	cols, err := s.repo.ListAllKanban(ctx)
	if err != nil {
		return nil, fmt.Errorf("demands service: kanban view: %w", err)
	}
	return cols, nil
}

// GenerateUploadURL cria uma URL temporária do MinIO para o upload.
func (s *Service) GenerateUploadURL(ctx context.Context, demandID uuid.UUID, docType, fileName string) (string, string, error) {
	// Ex: /demands/{demandID}/{docType}-{timestamp}-{fileName}
	objectPath := fmt.Sprintf("%s/%s-%d-%s", demandID.String(), docType, time.Now().Unix(), fileName)

	// Gera URL válida por 15 minutos
	url, err := s.storage.PresignedPutURL(ctx, s.bucket, objectPath, 15*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("demands service: presigned url: %w", err)
	}

	return url, objectPath, nil
}

// ConfirmUpload salva a referência do documento no banco após o cliente terminar de subir no MinIO.
// validade é o prazo de validade da certidão (nil quando não se aplica) —
// usado pela Máquina de Estados para bloquear a demanda com certidão vencida.
func (s *Service) ConfirmUpload(ctx context.Context, demandID uuid.UUID, docType, filePath, fileName string, validade *time.Time) error {
	// Cria o documento de prova
	doc := domain.DemandDocument{
		ID:         uuid.New(),
		DemandaID:  demandID,
		DocType:    domain.DocumentType(docType),
		FilePath:   filePath,
		FileName:   fileName,
		UploadedAt: time.Now(),
		Validade:   validade,
	}

	if err := s.repo.AddDocument(ctx, doc); err != nil {
		return fmt.Errorf("demands service: confirm upload: %w", err)
	}

	s.logger.Info("documento de demanda anexado com sucesso",
		slog.String("demanda_id", demandID.String()),
		slog.String("doc_type", docType),
	)
	return nil
}

// GetDemandByID busca todos os detalhes da demanda (usado para gerar PDF de impressão).
func (s *Service) GetDemandByID(ctx context.Context, demandID uuid.UUID) (domain.MonthlyDemand, error) {
	return s.repo.GetByID(ctx, demandID)
}

type CreateDemandInput struct {
	ContratoID  uuid.UUID
	AnoMes      string
	Observacoes string
}

// actorID resolve o autor autenticado da chamada para colunas *_by /
// auditoria. Loga alto (uma vez por chamada) se o subject do token não for
// UUID — a autoria não pode se perder em silêncio numa ação de compliance.
func (s *Service) actorID(ctx context.Context) *uuid.UUID {
	id, subject, ok := auth.ActorUUID(ctx)
	if !ok && subject != "" {
		s.logger.Warn("demands: subject do token não é UUID — autoria não registrada", slog.String("subject", subject))
	}
	return id
}

// CreateDemand cria uma nova demanda mensal para o contrato indicado.
func (s *Service) CreateDemand(ctx context.Context, input CreateDemandInput) (domain.MonthlyDemand, error) {
	now := time.Now()
	createdBy := s.actorID(ctx)

	demand := domain.MonthlyDemand{
		ID:             uuid.New(),
		ContratoID:     input.ContratoID,
		AnoMes:         input.AnoMes,
		Etapa:          domain.Etapa1ElaborarOF,
		StatusEtapa:    domain.StatusPendente,
		Observacoes:    input.Observacoes,
		EtapaStartedAt: now,
		CreatedBy:      createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, demand); err != nil {
		return domain.MonthlyDemand{}, fmt.Errorf("demands service: create demand: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{
			UserID:       createdBy,
			Action:       "demand.created",
			ResourceType: "demand",
			ResourceID:   demand.ID.String(),
			Metadata: map[string]any{
				"contrato_id": input.ContratoID,
				"ano_mes":     input.AnoMes,
			},
		})
	}

	s.logger.Info("demanda mensal criada com sucesso",
		slog.String("demanda_id", demand.ID.String()),
		slog.String("contrato_id", input.ContratoID.String()),
		slog.String("ano_mes", input.AnoMes),
	)

	// Recarrega os dados completos da demanda com dados do contrato
	fullDemand, err := s.repo.GetByID(ctx, demand.ID)
	if err != nil {
		// Se GetByID falhar por algum motivo de junção imediata, retorna a demanda criada
		return demand, nil
	}

	return fullDemand, nil
}
