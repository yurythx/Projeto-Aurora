package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Stats reporta a contagem de outbox_events por status, para o painel de
// Monitoramento (frontend) mostrar um número de verdade em vez de "0
// Pendentes" fixo no código-fonte (achado de auditoria — o painel nunca
// consultou o banco pra isso).
type Stats struct {
	pool *pgxpool.Pool
}

func NewStats(pool *pgxpool.Pool) *Stats {
	return &Stats{pool: pool}
}

// Counts é a contagem atual de outbox_events por status
// (pending/published/failed — ver o CHECK constraint da migration
// 000001). Uma chave ausente do mapa (nunca teve nenhuma linha naquele
// status) equivale a zero.
type Counts struct {
	Pending   int64 `json:"pending"`
	Published int64 `json:"published"`
	Failed    int64 `json:"failed"`
}

// Get consulta o Postgres diretamente a cada chamada — sem cache, mesmo
// espírito de configflags/ratelimit: esta tela é lida raramente (um
// painel de monitoramento aberto por um humano, não um hot path de
// requisição de negócio), então o custo de uma consulta direta é
// irrelevante frente a manter uma contagem em cache que poderia divergir
// do estado real do banco.
func (s *Stats) Get(ctx context.Context) (Counts, error) {
	rows, err := s.pool.Query(ctx, `SELECT status, count(*) FROM outbox_events GROUP BY status`)
	if err != nil {
		return Counts{}, fmt.Errorf("outbox: stats: %w", err)
	}
	defer rows.Close()

	var out Counts
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return Counts{}, fmt.Errorf("outbox: stats scan: %w", err)
		}
		switch status {
		case "pending":
			out.Pending = count
		case "published":
			out.Published = count
		case "failed":
			out.Failed = count
		}
	}
	if err := rows.Err(); err != nil {
		return Counts{}, fmt.Errorf("outbox: stats iterate: %w", err)
	}
	return out, nil
}
