package lgpd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-aurora/internal/platform/audit"
	"github.com/yurythx/projeto-aurora/internal/platform/database"
)

// erasureDetail explica, na própria linha da solicitação e para o
// titular, por que a "eliminação" é uma anonimização e não um DELETE.
const erasureDetail = "Dados pessoais anonimizados (username, e-mail, nome, hash de senha, vínculo Keycloak e papéis). " +
	"A linha do usuário é mantida por integridade referencial; a trilha de auditoria e os registros de " +
	"consentimento são preservados como registro legal, inclusive desta própria operação (LGPD art. 16, I e III)."

// ErasureProcessor é um processor do worker (mesmo formato de
// ratelimit.Cleanup): a cada 30s varre data_subject_requests pendentes do
// tipo 'erasure' e anonimiza o titular. Roda uma vez no boot também, para
// drenar o backlog sem esperar o primeiro tick.
func ErasureProcessor(pool *pgxpool.Pool, logger *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		processErasureBatch(ctx, pool, logger)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				processErasureBatch(ctx, pool, logger)
			}
		}
	}
}

func processErasureBatch(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id FROM data_subject_requests
		WHERE kind = 'erasure' AND status = 'pending'
		ORDER BY created_at ASC LIMIT 20`)
	if err != nil {
		logger.Error("lgpd: varredura de solicitações de exclusão falhou", slog.Any("error", err))
		return
	}
	type job struct{ reqID, userID uuid.UUID }
	var batch []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.reqID, &j.userID); err != nil {
			continue
		}
		batch = append(batch, j)
	}
	rows.Close()

	for _, j := range batch {
		if err := processOneErasure(ctx, pool, j.reqID, j.userID); err != nil {
			logger.Error("lgpd: falha ao processar exclusão",
				slog.String("request_id", j.reqID.String()), slog.Any("error", err))
			// marca como failed com um detalhe seguro; o titular vê em
			// /minhas-solicitacoes e um admin pode reprocessar.
			_, _ = pool.Exec(ctx, `
				UPDATE data_subject_requests
				SET status = 'failed', detail = 'Falha no processamento; a equipe foi notificada.'
				WHERE id = $1 AND status IN ('pending','processing')`, j.reqID)
			continue
		}
		logger.Info("lgpd: exclusão (anonimização) concluída",
			slog.String("request_id", j.reqID.String()))
	}
}

func processOneErasure(ctx context.Context, pool *pgxpool.Pool, reqID, userID uuid.UUID) error {
	return database.WithTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		// Trava a solicitação e confirma que ainda está pendente (duas
		// réplicas de worker não podem processar a mesma).
		var status string
		if err := tx.QueryRow(ctx,
			`SELECT status FROM data_subject_requests WHERE id = $1 FOR UPDATE`, reqID).Scan(&status); err != nil {
			return fmt.Errorf("lock request: %w", err)
		}
		if status != "pending" {
			return nil // outra réplica pegou primeiro
		}
		if _, err := tx.Exec(ctx,
			`UPDATE data_subject_requests SET status = 'processing' WHERE id = $1`, reqID); err != nil {
			return fmt.Errorf("mark processing: %w", err)
		}

		// Anonimização — NUNCA DELETE. Mantém id (FK de audit_logs e
		// user_consents) e desativa a conta.
		//
		// password_hash recebe o tombstone 'ANONYMIZED' (não NULL): a
		// constraint users_has_auth_method exige keycloak_subject OU
		// password_hash preenchidos, e keycloak_subject (o "sub" do
		// Gov.br) É dado pessoal e precisa sair. 'ANONYMIZED' não é um
		// hash bcrypt válido, então bcrypt.CompareHashAndPassword sempre
		// falha; além disso active=false barra o login antes disso.
		short := userID.String()[:8]
		if _, err := tx.Exec(ctx, `
			UPDATE users SET
				username         = 'anon_' || $2,
				email            = 'anon_' || $2 || '@anonimizado.invalid',
				display_name     = '',
				password_hash    = 'ANONYMIZED',
				keycloak_subject = NULL,
				roles            = '{}',
				active           = false,
				updated_at       = now()
			WHERE id = $1`, userID, short); err != nil {
			return fmt.Errorf("anonymize user: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE data_subject_requests
			SET status = 'completed', completed_at = now(), detail = $2
			WHERE id = $1`, reqID, erasureDetail); err != nil {
			return fmt.Errorf("mark completed: %w", err)
		}

		// Trilha da conclusão, na mesma transação.
		uid := userID
		return audit.NewWriter(tx).Record(ctx, audit.Entry{
			UserID:       &uid,
			Action:       "lgpd.erasure.completed",
			ResourceType: "user",
			ResourceID:   userID.String(),
			Metadata:     map[string]any{"request_id": reqID.String(), "method": "anonymization"},
		})
	})
}
