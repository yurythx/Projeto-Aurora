"use client";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { ApiError, useApiQuery } from "@/lib/api/swr";

interface EditionItem {
  id: number;
  edition_number: string;
  edition_date: string;
  pdf_url: string;
  status: "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED";
  records_count: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export function EtlPipelinePanel() {
  const { data, error, isLoading, mutate } = useApiQuery<EditionItem[]>(
    "v1/diario-oficial/rondonopolis/editions",
    { refreshInterval: 30000 }
  );

  const editions = Array.isArray(data) ? data : [];
  const errorMessage = error ? (error instanceof ApiError ? error.message : "Erro ao carregar edições do pipeline") : null;

  const totalCompleted = editions.filter((e) => e.status === "COMPLETED").length;
  const totalPending = editions.filter((e) => e.status === "PENDING" || e.status === "PROCESSING").length;
  const totalRecords = editions.reduce((acc, curr) => acc + (curr.records_count || 0), 0);

  return (
    <div className="flex flex-col gap-6">
      {/* Cards de Métricas em Grade de Destaque */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="p-4 rounded-xl bg-muted/30 border border-border/50 flex flex-col gap-1">
          <span className="text-xs font-medium text-muted uppercase tracking-wider">Edições Rastreadas</span>
          <span className="text-2xl font-bold text-foreground">{editions.length}</span>
          <span className="text-xs text-muted/80">Diários de Rondonópolis</span>
        </div>

        <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex flex-col gap-1">
          <span className="text-xs font-medium text-emerald-400 uppercase tracking-wider">Ingestão Concluída</span>
          <span className="text-2xl font-bold text-emerald-400">{totalCompleted}</span>
          <span className="text-xs text-emerald-500/80">Processados & Indexados</span>
        </div>

        <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 flex flex-col gap-1">
          <span className="text-xs font-medium text-amber-400 uppercase tracking-wider">Em Fila / Vigia</span>
          <span className="text-2xl font-bold text-amber-400">{totalPending}</span>
          <span className="text-xs text-amber-500/80">Aguardando Worker Pool</span>
        </div>

        <div className="p-4 rounded-xl bg-blue-500/10 border border-blue-500/20 flex flex-col gap-1">
          <span className="text-xs font-medium text-blue-400 uppercase tracking-wider">Achados Indexados</span>
          <span className="text-2xl font-bold text-blue-400">{totalRecords}</span>
          <span className="text-xs text-blue-500/80">Atos e Contratos no Postgres</span>
        </div>
      </div>

      {/* Painel de Disparo de Varredura em Lote (On-Demand Batch Ingestion) */}
      <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-4 shadow-sm">
        <div className="flex items-center justify-between border-b border-surface-border pb-3">
          <div>
            <h3 className="text-sm font-bold text-foreground">Disparo de Varredura Retrospectiva em Lote (Batch Ingestion)</h3>
            <p className="text-xs text-muted">Execute a varredura retroativa de edições passadas do Diário Oficial para alimentar a auditoria histórica.</p>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-4 text-xs">
          <div className="flex flex-wrap items-center gap-3">
            <label className="flex items-center gap-1.5 font-medium text-foreground">
              Ano do Acervo:
              <select className="rounded border border-surface-border bg-surface px-2.5 py-1 text-xs font-semibold">
                <option value="2026">2026 (Atual)</option>
                <option value="2025">2025 (Retroativo)</option>
                <option value="2024">2024 (Histórico)</option>
              </select>
            </label>

            <label className="flex items-center gap-1.5 font-medium text-foreground">
              Lote de Edições:
              <select className="rounded border border-surface-border bg-surface px-2.5 py-1 text-xs font-semibold">
                <option value="10">Últimas 10 Edições</option>
                <option value="50">Últimas 50 Edições</option>
                <option value="ALL">Todas as Edições Disponíveis</option>
              </select>
            </label>
          </div>

          <Button
            size="sm"
            variant="primary"
            onClick={() => {
              void mutate();
            }}
          >
            ⚡ Disparar Ingestão em Lote Agora
          </Button>
        </div>
      </div>

      {/* Tabela de Ingestão e Status de Edições */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-foreground">Status do Pipeline de Edições (ETL Worker Pool)</h3>
          <Button variant="secondary" size="sm" onClick={() => void mutate()} disabled={isLoading}>
            {isLoading ? "Atualizando..." : "Recarregar Pipeline"}
          </Button>
        </div>

        {errorMessage && (
          <div className="p-4 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
            {errorMessage}
          </div>
        )}

        <div className="overflow-x-auto rounded-xl border border-border/60 bg-card">
          <table className="w-full text-left text-sm">
            <thead className="bg-muted/40 border-b border-border/60 text-xs uppercase font-medium text-muted">
              <tr>
                <th className="p-3">Edição Nº</th>
                <th className="p-3">Data Publicação</th>
                <th className="p-3">Status ETL</th>
                <th className="p-3">Registros Extraídos</th>
                <th className="p-3">Última Atualização</th>
                <th className="p-3">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/40">
              {editions.length === 0 ? (
                <tr>
                  <td colSpan={6} className="p-8 text-center text-muted">
                    {isLoading ? "Carregando catálogo de edições..." : "Nenhuma edição registrada no pipeline ainda. O Vigia iniciará a varredura."}
                  </td>
                </tr>
              ) : (
                editions.map((ed) => (
                  <tr key={ed.id} className="hover:bg-muted/20 transition-colors">
                    <td className="p-3 font-semibold text-foreground">
                      Edição Nº {ed.edition_number}
                    </td>
                    <td className="p-3 text-muted">
                      {new Date(ed.edition_date).toLocaleDateString("pt-BR")}
                    </td>
                    <td className="p-3">
                      {ed.status === "COMPLETED" && <Badge tone="success">COMPLETED</Badge>}
                      {ed.status === "PROCESSING" && <Badge tone="warning">PROCESSING</Badge>}
                      {ed.status === "PENDING" && <Badge tone="neutral">PENDING</Badge>}
                      {ed.status === "FAILED" && <Badge tone="danger">FAILED</Badge>}
                    </td>
                    <td className="p-3 font-mono text-foreground">
                      {ed.records_count} achados
                    </td>
                    <td className="p-3 text-xs text-muted">
                      {new Date(ed.updated_at).toLocaleString("pt-BR")}
                    </td>
                    <td className="p-3">
                      <a
                        href={ed.pdf_url}
                        target="_blank"
                        rel="noreferrer"
                        className="text-xs text-primary hover:underline font-medium inline-flex items-center gap-1"
                      >
                        Abrir PDF →
                      </a>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
