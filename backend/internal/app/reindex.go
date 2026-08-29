package app

import (
	"context"
	"fmt"
	"slices"

	diarioInfra "github.com/yurythx/projeto-nova/internal/modules/diario_oficial/infrastructure"
	diarioWorker "github.com/yurythx/projeto-nova/internal/modules/diario_oficial/worker"
)

// ReindexDiarioOficial reconstrói as coleções do Typesense (diorondon_personnel_acts
// e diorondon_articles) a partir de diario_oficial_findings ⨝ diario_oficial_editions.
// Executado pelo subcomando `worker reindex-diario [--recreate]`.
//
// Com --recreate as coleções são apagadas e recriadas antes (aplica mudanças
// de schema e remove documentos órfãos).
func ReindexDiarioOficial(ctx context.Context, deps *Dependencies, args []string) error {
	if deps.Typesense == nil {
		return fmt.Errorf("reindex-diario: Typesense não configurado (defina TYPESENSE_URL/TYPESENSE_API_KEY)")
	}
	recreate := slices.Contains(args, "--recreate")

	repo := diarioInfra.NewPostgresEditionRepository(deps.DB)
	reindexer := diarioWorker.NewReindexer(repo, deps.Typesense, deps.Logger)

	n, err := reindexer.ReindexAll(ctx, recreate)
	if err != nil {
		return fmt.Errorf("reindex-diario: %w", err)
	}
	deps.Logger.Info("reindex-diario concluído", "documentos_indexados", n, "recreate", recreate)
	return nil
}
