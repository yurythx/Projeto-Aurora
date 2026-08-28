package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/domain/events"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/application"
)

// publicationMatchedPayload reflete o evento disparado pelo módulo
// diario_oficial quando uma publicação (DJEN ou Rondonópolis) casa
// com um termo monitorado (ex: "CONTRATO", "ADITIVO").
type publicationMatchedPayload struct {
	MonitoredTermID    uuid.UUID `json:"monitored_term_id"`
	MonitoredTermLabel string    `json:"monitored_term_label"`
	PublicationID      uuid.UUID `json:"publication_id"`
	Tribunal           string    `json:"tribunal"`
	TipoComunicacao    string    `json:"tipo_comunicacao"`
	ProcessNumber      string    `json:"process_number"`
}

// PublicationMatchedHandler consome eventos "diario_oficial.publication.matched"
// e aciona o módulo de Contratos para cadastrar ou atualizar contratos
// originados no Diário Oficial.
func PublicationMatchedHandler(svc *application.Service, logger *slog.Logger) events.MessageHandler {
	return func(ctx context.Context, msg events.Event) error {
		var payload publicationMatchedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			logger.Error("contratos: failed to parse publication matched event",
				slog.String("message_id", msg.ID.String()),
				slog.Any("error", err),
			)
			// Retornamos nil para descartar mensagem malformada (não fará retry).
			return nil
		}

		logger.Info("contratos: processing publication match",
			slog.String("publication_id", payload.PublicationID.String()),
			slog.String("term", payload.MonitoredTermLabel),
		)

		now := time.Now()
		
		// Converte os dados do evento em um contrato.
		// Na vida real, poderíamos consultar os detalhes completos da 
		// publicação (PDF, partes envolvidas) via API ou shared DB, 
		// mas aqui usamos os dados contidos no evento.
		input := application.CreateFromDiarioInput{
			Numero:         payload.ProcessNumber,
			Objeto:         "Contrato gerado a partir de publicação (" + payload.MonitoredTermLabel + ")",
			Contratante:    payload.Tribunal,
			Contratado:     "A DEFINIR",
			CNPJ:           "00.000.000/0000-00", // A extração exata viria do ETL.
			TipoEvento:     payload.TipoComunicacao,
			EditionNumber  : payload.PublicationID.String()[:8], // Fallback para ID truncado.
			DataPublicacao : &now,
			PublicadoEm    : &now,
			Contexto       : "Publicação casou com termo monitorado: " + payload.MonitoredTermLabel,
			DocURL         : "/diario-oficial", // Placeholder.
		}

		// Se não veio número de processo, gera um pseudo-número baseado na publicação.
		if input.Numero == "" {
			input.Numero = "PUB-" + payload.PublicationID.String()[:8]
		}

		if err := svc.CreateFromDiario(ctx, input); err != nil {
			logger.Error("contratos: failed to create contract from diary",
				slog.String("publication_id", payload.PublicationID.String()),
				slog.Any("error", err),
			)
			// Retorna erro para causar o Nack e eventual retry (DLQ).
			return err
		}

		return nil
	}
}
