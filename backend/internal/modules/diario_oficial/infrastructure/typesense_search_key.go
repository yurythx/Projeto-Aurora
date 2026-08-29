package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova/pkg/typesense"
)

const frontendSearchKeyKV = "typesense_frontend_search_key"

var frontendSearchCollections = []string{"diorondon_articles", "diorondon_personnel_acts"}

// TypesenseSearchKeyManager entrega ao navegador uma API key do Typesense
// restrita a `documents:search` — em vez da chave admin, que hoje vaza no
// bundle do frontend e permite apagar coleções. O valor completo (o Typesense
// só o devolve na criação) é guardado em diario_oficial_kv.
type TypesenseSearchKeyManager struct {
	ts   *typesense.Client
	pool *pgxpool.Pool
}

func NewTypesenseSearchKeyManager(ts *typesense.Client, pool *pgxpool.Pool) *TypesenseSearchKeyManager {
	return &TypesenseSearchKeyManager{ts: ts, pool: pool}
}

// EnsureFrontendSearchKey devolve a key search-only, criando-a (e persistindo)
// se ainda não existir ou se a guardada não estiver mais no servidor.
func (m *TypesenseSearchKeyManager) EnsureFrontendSearchKey(ctx context.Context) (string, error) {
	if m == nil || m.ts == nil || m.pool == nil {
		return "", fmt.Errorf("typesense search key manager not configured")
	}

	var stored string
	_ = m.pool.QueryRow(ctx, `SELECT v FROM diario_oficial_kv WHERE k = $1`, frontendSearchKeyKV).Scan(&stored)

	if stored != "" && m.keyStillExists(ctx, stored) {
		return stored, nil
	}

	created, err := m.ts.CreateSearchOnlyKey(ctx, "diorondon-frontend-search (auto)", frontendSearchCollections)
	if err != nil {
		return "", fmt.Errorf("create typesense search key: %w", err)
	}
	if _, err := m.pool.Exec(ctx, `
		INSERT INTO diario_oficial_kv (k, v, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (k) DO UPDATE SET v = EXCLUDED.v, updated_at = now()`,
		frontendSearchKeyKV, created.Value); err != nil {
		return "", fmt.Errorf("persist typesense search key: %w", err)
	}
	return created.Value, nil
}

func (m *TypesenseSearchKeyManager) keyStillExists(ctx context.Context, value string) bool {
	keys, err := m.ts.ListKeys(ctx)
	if err != nil {
		return true // não dá pra confirmar; assume que segue válida (evita recriar toda hora)
	}
	for _, k := range keys {
		if k.ValuePrefix != "" && strings.HasPrefix(value, k.ValuePrefix) {
			return true
		}
	}
	return false
}
