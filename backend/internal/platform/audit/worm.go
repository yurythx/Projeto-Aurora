package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-aurora/internal/platform/storage"
)

// wormInterval é de quanto em quanto tempo o worker verifica se há dias
// completos ainda não exportados. Diário seria suficiente; 6h dá margem
// para o worker ter ficado fora do ar por um tempo sem acumular atraso.
const wormInterval = 6 * time.Hour

// WORMExporter é um processor do worker (mesmo formato de
// ratelimit.Cleanup). Serializa cada dia COMPLETO de audit_logs ainda não
// exportado, encadeia o SHA-256 ao do dia anterior (evidência de
// adulteração) e sobe arquivo + digest para o object storage. F2.6.
func WORMExporter(pool *pgxpool.Pool, store storage.Provider, bucket string, logger *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		exportPendingDays(ctx, pool, store, bucket, logger)
		ticker := time.NewTicker(wormInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				exportPendingDays(ctx, pool, store, bucket, logger)
			}
		}
	}
}

func exportPendingDays(ctx context.Context, pool *pgxpool.Pool, store storage.Provider, bucket string, logger *slog.Logger) {
	// Só dias COMPLETOS: de min(created_at) até ontem (UTC).
	var earliest *time.Time
	if err := pool.QueryRow(ctx, `SELECT min(created_at) FROM audit_logs`).Scan(&earliest); err != nil || earliest == nil {
		return
	}
	yesterday := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	day := earliest.UTC().Truncate(24 * time.Hour)

	for !day.After(yesterday) {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_worm_exports WHERE day = $1)`, day).Scan(&exists); err != nil {
			logger.Error("audit worm: consulta de watermark falhou", slog.Any("error", err))
			return
		}
		if !exists {
			if err := exportOneDay(ctx, pool, store, bucket, day); err != nil {
				logger.Error("audit worm: falha ao exportar o dia",
					slog.String("day", day.Format("2006-01-02")), slog.Any("error", err))
				return // tenta de novo no próximo tick; a cadeia precisa ser sequencial
			}
			logger.Info("audit worm: dia exportado", slog.String("day", day.Format("2006-01-02")))
		}
		day = day.Add(24 * time.Hour)
	}
}

func exportOneDay(ctx context.Context, pool *pgxpool.Pool, store storage.Provider, bucket string, day time.Time) error {
	next := day.Add(24 * time.Hour)

	var prevSHA string
	_ = pool.QueryRow(ctx,
		`SELECT sha256 FROM audit_worm_exports WHERE day < $1 ORDER BY day DESC LIMIT 1`, day).Scan(&prevSHA)

	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(user_id::text,''), action, COALESCE(resource_type,''),
		       COALESCE(resource_id,''), COALESCE(metadata::text,'{}'),
		       COALESCE(correlation_id::text,''), COALESCE(ip_address::text,''), created_at
		FROM audit_logs
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY created_at ASC, id ASC`, day, next)
	if err != nil {
		return fmt.Errorf("query day: %w", err)
	}
	defer rows.Close()

	var buf bytes.Buffer
	header, _ := json.Marshal(map[string]any{
		"_worm_header": true,
		"day":          day.Format("2006-01-02"),
		"prev_sha256":  prevSHA,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
	buf.Write(header)
	buf.WriteByte('\n')

	count := 0
	for rows.Next() {
		var id, userID, action, rType, rID, meta, corr, ip string
		var createdAt time.Time
		if err := rows.Scan(&id, &userID, &action, &rType, &rID, &meta, &corr, &ip, &createdAt); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		line, _ := json.Marshal(map[string]any{
			"id": id, "user_id": userID, "action": action, "resource_type": rType,
			"resource_id": rID, "metadata": json.RawMessage(meta), "correlation_id": corr,
			"ip_address": ip, "created_at": createdAt.UTC().Format(time.RFC3339Nano),
		})
		buf.Write(line)
		buf.WriteByte('\n')
		count++
	}
	if rows.Err() != nil {
		return fmt.Errorf("rows: %w", rows.Err())
	}

	sum := sha256.Sum256(buf.Bytes())
	digest := hex.EncodeToString(sum[:])
	objectKey := fmt.Sprintf("audit-worm/%s/%s.jsonl", day.Format("2006/01"), day.Format("2006-01-02"))

	content := buf.Bytes()
	if err := store.Put(ctx, bucket, objectKey, bytes.NewReader(content), int64(len(content)), "application/x-ndjson"); err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	digestBody := []byte(digest + "  " + day.Format("2006-01-02") + ".jsonl\n")
	if err := store.Put(ctx, bucket, objectKey+".sha256", bytes.NewReader(digestBody), int64(len(digestBody)), "text/plain"); err != nil {
		return fmt.Errorf("put digest: %w", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_worm_exports (day, row_count, sha256, prev_sha256, object_key)
		VALUES ($1, $2, $3, $4, $5)`, day, count, digest, prevSHA, objectKey); err != nil {
		return fmt.Errorf("insert watermark: %w", err)
	}

	// A própria exportação vira um fato auditável.
	_ = NewWriter(pool).Record(ctx, Entry{
		Action:       "audit.worm.exported",
		ResourceType: "audit_worm_exports",
		ResourceID:   day.Format("2006-01-02"),
		Metadata:     map[string]any{"rows": count, "sha256": digest, "object_key": objectKey},
	})
	return nil
}
