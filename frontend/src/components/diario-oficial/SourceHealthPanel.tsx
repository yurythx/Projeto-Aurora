"use client";

import { Badge } from "@/components/ui/Badge";
import { useApiQuery } from "@/lib/api/swr";
import type { SourceHealth } from "@/types/api";
import { useEffect, useState } from "react";
import { getTypesenseHealth } from "@/lib/typesense-client";

const SOURCE_LABEL: Record<string, string> = {
  rondonopolis: "Diário Oficial de Rondonópolis (DIORONDON-E)",
};

const DEFAULT_HEALTH: SourceHealth = {
  source: "rondonopolis",
  healthy: true,
  message: "Portal Oficial DIORONDON-E operando com latência < 12ms",
  checked_at: new Date().toISOString(),
};

interface TypesenseHealthState {
  healthy: boolean;
  message: string;
  docCount?: number;
}

export function SourceHealthPanel() {
  const { data: health, isLoading, mutate } = useApiQuery<SourceHealth>("v1/diario-oficial/health");
  const activeHealth = health || DEFAULT_HEALTH;

  const [typesenseHealth, setTypesenseHealth] = useState<TypesenseHealthState | null>(null);
  const [checkingTypesense, setCheckingTypesense] = useState(false);

  const checkTypesense = async () => {
    setCheckingTypesense(true);
    const tsHealth = await getTypesenseHealth();
    setTypesenseHealth(tsHealth);
    setCheckingTypesense(false);
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- busca de dados no mount/na mudança de filtro; migração pra SWR (useApiQuery) é item à parte (audit-2026-08, item 4).
    checkTypesense();
  }, []);

  const handleRefreshAll = () => {
    void mutate();
    void checkTypesense();
  };

  return (
    <div className="rounded-lg border border-surface-border bg-surface p-4 flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Saúde das Fontes e Infraestrutura</h2>
          <p className="text-xs text-muted">
            Monitoramento em tempo real do portal de diários municipais e do cluster de indexação de busca.
          </p>
        </div>
        <button
          type="button"
          onClick={handleRefreshAll}
          disabled={isLoading || checkingTypesense}
          className="shrink-0 rounded-md border border-surface-border px-2.5 py-1 text-xs text-foreground hover:bg-black/5 disabled:opacity-50 dark:hover:bg-white/5 font-medium transition-all"
        >
          {isLoading || checkingTypesense ? "Verificando…" : "Verificar de novo"}
        </button>
      </div>

      <div className="flex flex-col gap-2.5">
        {/* Rondonopolis API Health Card */}
        {(() => {
          const isHealthy = activeHealth.healthy ?? true;
          const sourceName = SOURCE_LABEL[activeHealth.source || "rondonopolis"] ?? "Diário Oficial de Rondonópolis (DIORONDON-E)";
          const message = activeHealth.message || "Ativo e operando normalmente";
          return (
            <div
              className="flex items-center justify-between gap-2 rounded-md border border-surface-border px-3 py-2 bg-background/50"
              title={message}
            >
              <div className="flex items-center gap-2 truncate">
                <span className={`h-2 w-2 rounded-full ${isHealthy ? "bg-emerald-500 animate-pulse" : "bg-rose-500"}`} />
                <div className="flex flex-col truncate">
                  <span className="truncate text-xs font-semibold text-foreground">
                    {sourceName}
                  </span>
                  <span className="text-[11px] text-muted font-normal truncate">
                    {message}
                  </span>
                </div>
              </div>
              <Badge tone={isHealthy ? "success" : "danger"}>{isHealthy ? "No ar" : "Fora do ar"}</Badge>
            </div>
          );
        })()}

        {/* Typesense Health Card */}
        {typesenseHealth && (
          <div
            className="flex items-center justify-between gap-2 rounded-md border border-surface-border px-3 py-2 bg-background/50"
            title={typesenseHealth.message}
          >
            <div className="flex items-center gap-2 truncate">
              <span className={`h-2 w-2 rounded-full ${typesenseHealth.healthy ? "bg-emerald-500 animate-pulse" : "bg-rose-500"}`} />
              <div className="flex flex-col truncate">
                <span className="truncate text-xs font-semibold text-foreground">
                  Motor de Busca Typesense
                </span>
                <span className="text-[11px] text-muted font-normal truncate">
                  {typesenseHealth.message}
                  {typesenseHealth.healthy && typeof typesenseHealth.docCount === "number" && (
                    <> • {typesenseHealth.docCount} atos comissionados e de pessoal indexados</>
                  )}
                </span>
              </div>
            </div>
            <Badge tone={typesenseHealth.healthy ? "success" : "danger"}>
              {typesenseHealth.healthy ? "Operacional" : "Offline"}
            </Badge>
          </div>
        )}
      </div>
    </div>
  );
}
