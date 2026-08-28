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

// Service coordena os fluxos das Demandas Mensais.
type Service struct {
	db      *pgxpool.Pool
	repo    domain.Repository
	outbox  *outbox.Writer
	storage storage.Provider
	audit   *audit.Writer
	bucket  string
	logger  *slog.Logger
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
	if err := ValidateTransition(demand, nextStage); err != nil {
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
		var actorID *uuid.UUID
		if identity, ok := auth.IdentityFromContext(ctx); ok {
			if parsed, err := uuid.Parse(identity.Subject); err == nil {
				actorID = &parsed
			}
		}
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
func (s *Service) ConfirmUpload(ctx context.Context, demandID uuid.UUID, docType, filePath, fileName string) error {
	// Cria o documento de prova
	doc := domain.DemandDocument{
		ID:         uuid.New(),
		DemandaID:  demandID,
		DocType:    domain.DocumentType(docType),
		FilePath:   filePath,
		FileName:   fileName,
		UploadedAt: time.Now(),
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

// CreateDemand cria uma nova demanda mensal para o contrato indicado.
func (s *Service) CreateDemand(ctx context.Context, input CreateDemandInput) (domain.MonthlyDemand, error) {
	now := time.Now()
	var createdBy *uuid.UUID
	if identity, ok := auth.IdentityFromContext(ctx); ok {
		if parsed, err := uuid.Parse(identity.Subject); err == nil {
			createdBy = &parsed
		}
	}

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
