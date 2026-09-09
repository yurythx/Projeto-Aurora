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

type wormExporter struct {
	pool          *pgxpool.Pool
	store         storage.Provider
	worm          storage.WORMWriter // != nil quando store suporta object-lock
	bucket        string
	retentionDays int
	logger        *slog.Logger
	bucketReady   bool
}

// WORMExporter é um processor do worker (mesmo formato de
// ratelimit.Cleanup). Serializa cada dia COMPLETO de audit_logs ainda não
// exportado, encadeia o SHA-256 ao do dia anterior (evidência de
// adulteração) e sobe arquivo + digest para o object storage. F2.6.
//
// Se o provider implementa storage.WORMWriter, o bucket é criado com
// object-lock e cada objeto recebe retenção Compliance de retentionDays;
// caso contrário, cai para Put comum e registra o aviso (a cadeia de
// hash ainda dá evidência de adulteração).
func WORMExporter(pool *pgxpool.Pool, store storage.Provider, bucket string, retentionDays int, logger *slog.Logger) func(ctx context.Context) error {
	e := &wormExporter{pool: pool, store: store, bucket: bucket, retentionDays: retentionDays, logger: logger}
	if w, ok := store.(storage.WORMWriter); ok {
		e.worm = w
	}
	return func(ctx context.Context) error {
		e.run(ctx)
		ticker := time.NewTicker(wormInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				e.run(ctx)
			}
		}
	}
}

func (e *wormExporter) ensureBucket(ctx context.Context) bool {
	if e.bucketReady {
		return true
	}
	if e.worm != nil {
		if err := e.worm.EnsureImmutableBucket(ctx, e.bucket, e.retentionDays); err != nil {
			e.logger.Error("audit worm: não foi possível preparar o bucket imutável — object-lock indisponível; a cadeia de hash segue como evidência",
				slog.String("bucket", e.bucket), slog.Any("error", err))
			e.worm = nil // cai para Put comum daqui pra frente
		}
	}
	if e.worm == nil {
		if be, ok := e.store.(interface {
			EnsureBucket(context.Context, string) error
		}); ok {
			if err := be.EnsureBucket(ctx, e.bucket); err != nil {
				e.logger.Error("audit worm: não foi possível garantir o bucket", slog.Any("error", err))
				return false
			}
		}
		e.logger.Warn("audit worm: bucket SEM object-lock — configure um bucket dedicado com lock na infraestrutura para a garantia WORM completa",
			slog.String("bucket", e.bucket))
	}
	e.bucketReady = true
	return true
}

func (e *wormExporter) run(ctx context.Context) {
	if !e.ensureBucket(ctx) {
		return
	}
	var earliest *time.Time
	if err := e.pool.QueryRow(ctx, `SELECT min(created_at) FROM audit_logs`).Scan(&earliest); err != nil || earliest == nil {
		return
	}
	yesterday := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	day := earliest.UTC().Truncate(24 * time.Hour)

	for !day.After(yesterday) {
		var exists bool
		if err := e.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_worm_exports WHERE day = $1)`, day).Scan(&exists); err != nil {
			e.logger.Error("audit worm: consulta de watermark falhou", slog.Any("error", err))
			return
		}
		if !exists {
			if err := e.exportOneDay(ctx, day); err != nil {
				e.logger.Error("audit worm: falha ao exportar o dia",
					slog.String("day", day.Format("2006-01-02")), slog.Any("error", err))
				return // tenta de novo no próximo tick; a cadeia precisa ser sequencial
			}
			e.logger.Info("audit worm: dia exportado", slog.String("day", day.Format("2006-01-02")))
		}
		day = day.Add(24 * time.Hour)
	}
}

func (e *wormExporter) put(ctx context.Context, key string, body []byte, contentType string) error {
	if e.worm != nil {
		return e.worm.PutImmutable(ctx, e.bucket, key, bytes.NewReader(body), int64(len(body)), contentType, e.retentionDays)
	}
	return e.store.Put(ctx, e.bucket, key, bytes.NewReader(body), int64(len(body)), contentType)
}

func (e *wormExporter) exportOneDay(ctx context.Context, day time.Time) error {
	next := day.Add(24 * time.Hour)

	var prevSHA string
	_ = e.pool.QueryRow(ctx,
		`SELECT sha256 FROM audit_worm_exports WHERE day < $1 ORDER BY day DESC LIMIT 1`, day).Scan(&prevSHA)

	rows, err := e.pool.Query(ctx, `
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
	objectKey := fmt.Sprintf("%s/%s.jsonl", day.Format("2006/01"), day.Format("2006-01-02"))

	if err := e.put(ctx, objectKey, buf.Bytes(), "application/x-ndjson"); err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	digestBody := []byte(digest + "  " + day.Format("2006-01-02") + ".jsonl\n")
	if err := e.put(ctx, objectKey+".sha256", digestBody, "text/plain"); err != nil {
		return fmt.Errorf("put digest: %w", err)
	}

	if _, err := e.pool.Exec(ctx, `
		INSERT INTO audit_worm_exports (day, row_count, sha256, prev_sha256, object_key)
		VALUES ($1, $2, $3, $4, $5)`, day, count, digest, prevSHA, objectKey); err != nil {
		return fmt.Errorf("insert watermark: %w", err)
	}

	_ = NewWriter(e.pool).Record(ctx, Entry{
		Action:       "audit.worm.exported",
		ResourceType: "audit_worm_exports",
		ResourceID:   day.Format("2006-01-02"),
		Metadata: map[string]any{
			"rows": count, "sha256": digest, "object_key": objectKey,
			"bucket": e.bucket, "immutable": e.worm != nil,
		},
	})
	return nil
}
