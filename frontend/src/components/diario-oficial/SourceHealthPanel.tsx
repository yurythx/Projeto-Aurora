"use client";

import { Badge } from "@/components/ui/Badge";
import { ApiError, useApiQuery } from "@/lib/api/swr";
import type { SourceHealth } from "@/types/api";

const SOURCE_LABEL: Record<string, string> = {
  rondonopolis: "Diário Oficial de Rondonópolis (DIORONDON-E)",
};

export function SourceHealthPanel() {
  const { data: health, error, isLoading, mutate } = useApiQuery<SourceHealth>("v1/diario-oficial/health");

  return (
    <div className="rounded-lg border border-surface-border bg-surface p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Saúde da Fonte do Diário Oficial</h2>
          <p className="text-xs text-muted">
            Monitoramento de status e tempo de resposta da API do Diário Oficial de Rondonópolis (DIORONDON-E).
          </p>
        </div>
        <button
          type="button"
          onClick={() => void mutate()}
          disabled={isLoading}
          className="shrink-0 rounded-md border border-surface-border px-2.5 py-1 text-xs text-foreground hover:bg-black/5 disabled:opacity-50 dark:hover:bg-white/5 font-medium transition-all"
        >
          Verificar de novo
        </button>
      </div>

      {error ? (
        <p className="text-sm text-danger">
          {error instanceof ApiError ? error.message : "Falha ao verificar a saúde da fonte"}
        </p>
      ) : !health ? (
        <p className="text-sm text-muted">Verificando…</p>
      ) : (() => {
        const isHealthy = health.healthy ?? (health as unknown as { Healthy?: boolean }).Healthy ?? true;
        const sourceName = SOURCE_LABEL[health.source || (health as unknown as { Source?: string }).Source || "rondonopolis"] ?? "Diário Oficial de Rondonópolis (DIORONDON-E)";
        const message = health.message || (health as unknown as { Message?: string }).Message || "Ativo e operando normalmente";
        return (
          <div
            className="flex items-center justify-between gap-2 rounded-md border border-surface-border px-3 py-2 bg-background/50"
            title={message}
          >
            <div className="flex items-center gap-2 truncate">
              <span className={`h-2 w-2 rounded-full ${isHealthy ? "bg-emerald-500 animate-pulse" : "bg-rose-500"}`} />
              <div className="flex flex-col truncate">
                <span className="truncate text-sm font-medium text-foreground">
                  {sourceName}
                </span>
                <span className="text-xs text-muted font-normal truncate">
                  {message}
                </span>
              </div>
            </div>
            <Badge tone={isHealthy ? "success" : "danger"}>{isHealthy ? "No ar" : "Fora do ar"}</Badge>
          </div>
        );
      })()}
    </div>
  );
}
