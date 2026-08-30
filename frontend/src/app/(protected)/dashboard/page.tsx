import { getServerSession } from "next-auth/next";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { StatusIndicator } from "@/components/ui/StatusIndicator";
import { authOptions } from "@/lib/auth/options";
import { ApiError } from "@/lib/api/client";
import { serverApiGet } from "@/lib/api/server";
import type { Integration } from "@/types/api";

interface DashboardStats {
  total_vigentes: number;
  valor_total_vigentes: number;
  proximos_vencimento: number;
}

interface DemandsDashboard {
  funnel: { etapa: number; label: string; total: number }[];
  total_em_andamento: number;
  total_arquivadas: number;
  sla_breached: number;
  sla_breached_items: {
    demanda_id: string;
    contrato_numero: string;
    etapa: number;
    days_in_stage: number;
    sla_days: number;
  }[];
  open_occurrences: number;
  certidoes_expired: number;
  certidoes_expiring_30d: number;
  certidoes_items: {
    demanda_id: string;
    contrato_numero: string;
    doc_type: string;
    label: string;
    validade_ate: string;
    expired: boolean;
  }[];
}

// Uma célula da "régua" de números do topo — deliberadamente NÃO é um card
// (§ redesenho 2026-08): a tela antiga eram seis cards idênticos, o padrão
// mais genérico possível. Aqui é uma linha contável com fios finos, no
// espírito de um livro-razão / extrato.
function LedgerCell({
  label,
  value,
  hint,
  alert = false,
}: {
  label: string;
  value: string | number;
  hint?: string;
  alert?: boolean;
}) {
  return (
    <div className="flex min-w-[8.25rem] shrink-0 flex-col gap-1 px-3.5 py-4 first:pl-0">
      <span className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted">{label}</span>
      <span
        className={`font-mono text-2xl font-semibold tabular-nums ${alert ? "text-seal" : "text-foreground"}`}
      >
        {value}
      </span>
      {hint && <span className="text-xs text-muted">{hint}</span>}
    </div>
  );
}

// Visão geral — dashboard de fiscalização de contratos do Projeto Nova.
// Server Component: busca sessão, estatísticas e funil de demandas no
// servidor antes de qualquer HTML sair.
export default async function DashboardOverviewPage() {
  const session = await getServerSession(authOptions);
  const firstName = (session?.user?.name ?? session?.user?.email ?? "").split(/[@\s]/)[0];

  const competencia = new Date()
    .toLocaleDateString("pt-BR", { month: "long", year: "numeric" })
    .toUpperCase();

  let integrations: Integration[] | null = null;
  let stats: DashboardStats | null = null;
  let demandsDash: DemandsDashboard | null = null;
  let errorMessage: string | null = null;

  try {
    const [integrationsRes, statsRes, demandsRes] = await Promise.all([
      serverApiGet<Integration[]>("v1/integrations"),
      serverApiGet<DashboardStats>("v1/contratos/dashboard").catch(() => ({ data: null })),
      serverApiGet<DemandsDashboard>("v1/demands/dashboard").catch(() => ({ data: null })),
    ]);
    integrations = integrationsRes.data;
    if (statsRes && statsRes.data) {
      stats = statsRes.data;
    }
    if (demandsRes && demandsRes.data) {
      demandsDash = demandsRes.data;
    }
  } catch (err) {
    errorMessage = err instanceof ApiError ? err.message : "Falha ao carregar";
  }

  const funnelMax = demandsDash
    ? Math.max(1, ...(demandsDash.funnel ?? []).map((f) => f.total))
    : 1;

  return (
    <div className="flex flex-col gap-10">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="flex flex-col gap-2">
          <p className="dateline">Competência · {competencia}</p>
          <h1 className="text-2xl font-semibold">{firstName ? `Olá, ${firstName}` : "Visão geral"}</h1>
          <p className="text-sm text-muted">
            Onde as demandas mensais estão paradas e o que precisa de você agora.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 text-sm">
          <Link href="/contratos">
            <Button size="sm">Abrir quadro de demandas</Button>
          </Link>
          <Link href="/diario">
            <Button size="sm" variant="secondary">
              Buscar no Diário
            </Button>
          </Link>
        </div>
      </header>

      {/* Régua de números — extrato, não grade de cards. */}
      {(stats || demandsDash) && (
        <section className="flex flex-col gap-3">
          <p className="dateline">Situação atual</p>
          <div className="flex divide-x divide-surface-border overflow-x-auto rounded-xl border border-surface-border bg-surface px-4 py-1">
            {stats && (
              <>
                <LedgerCell
                  label="Contratos vigentes"
                  value={stats.total_vigentes}
                  hint={`R$ ${stats.valor_total_vigentes.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`}
                />
                <LedgerCell
                  label="Vencem em 30 dias"
                  value={stats.proximos_vencimento}
                  alert={stats.proximos_vencimento > 0}
                />
              </>
            )}
            {demandsDash && (
              <>
                <LedgerCell
                  label="Demandas em andamento"
                  value={demandsDash.total_em_andamento}
                  hint={`${demandsDash.total_arquivadas} arquivada(s)`}
                />
                <LedgerCell
                  label="SLA estourado"
                  value={demandsDash.sla_breached}
                  hint="fora do prazo da etapa"
                  alert={demandsDash.sla_breached > 0}
                />
                <LedgerCell
                  label="Pendências abertas"
                  value={demandsDash.open_occurrences}
                  hint="ocorrências não resolvidas"
                  alert={demandsDash.open_occurrences > 0}
                />
                <LedgerCell
                  label="Certidões vencidas"
                  value={demandsDash.certidoes_expired}
                  hint={`+ ${demandsDash.certidoes_expiring_30d} vencem em 30 dias`}
                  alert={demandsDash.certidoes_expired > 0}
                />
              </>
            )}
          </div>
        </section>
      )}

      {/* Funil por etapa — o centro da tela: é o processo. */}
      {demandsDash && (
        <section className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <p className="dateline">IN SCL 01/2019 · Funil por etapa</p>
            <Link href="/contratos" className="text-xs text-primary hover:underline">
              Abrir quadro →
            </Link>
          </div>
          <Card>
            <CardContent className="flex flex-col divide-y divide-surface-border pt-2">
              {(demandsDash.funnel ?? []).map((f) => (
                <div key={f.etapa} className="flex items-center gap-4 py-3">
                  <span className="w-6 shrink-0 font-mono text-sm font-semibold text-seal">
                    {String(f.etapa).padStart(2, "0")}
                  </span>
                  <span className="w-56 shrink-0 truncate text-sm text-foreground">{f.label}</span>
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-black/[0.06] dark:bg-white/[0.06]">
                    <div
                      className="h-full rounded-full bg-primary"
                      style={{ width: `${Math.round((f.total / funnelMax) * 100)}%` }}
                    />
                  </div>
                  <span className="w-8 shrink-0 text-right font-mono text-sm font-semibold tabular-nums">
                    {f.total}
                  </span>
                </div>
              ))}
            </CardContent>
          </Card>
        </section>
      )}

      {/* Atenção imediata */}
      {demandsDash && (
        <section className="flex flex-col gap-4">
          <p className="dateline">Atenção imediata</p>
          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">SLA estourado</CardTitle>
              </CardHeader>
              <CardContent className="text-sm">
                {(demandsDash.sla_breached_items ?? []).length === 0 ? (
                  <p className="text-muted">Nenhuma demanda fora do prazo.</p>
                ) : (
                  <ul className="flex flex-col divide-y divide-surface-border">
                    {(demandsDash.sla_breached_items ?? []).map((it) => (
                      <li key={it.demanda_id} className="flex items-center justify-between gap-2 py-2">
                        <span className="truncate">
                          Contrato {it.contrato_numero || "—"} · Etapa {it.etapa}
                        </span>
                        <span className="shrink-0 font-mono text-seal tabular-nums">
                          {it.days_in_stage}d / {it.sla_days}d
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Certidões vencidas ou a vencer</CardTitle>
              </CardHeader>
              <CardContent className="text-sm">
                {(demandsDash.certidoes_items ?? []).length === 0 ? (
                  <p className="text-muted">Nenhuma certidão vencida ou a vencer em 30 dias.</p>
                ) : (
                  <ul className="flex flex-col divide-y divide-surface-border">
                    {(demandsDash.certidoes_items ?? []).slice(0, 12).map((it) => (
                      <li
                        key={`${it.demanda_id}-${it.doc_type}`}
                        className="flex items-center justify-between gap-2 py-2"
                      >
                        <span className="truncate">
                          {it.contrato_numero || "—"} · {it.label}
                        </span>
                        <span
                          className={`shrink-0 font-mono tabular-nums ${it.expired ? "text-seal" : "text-warning"}`}
                        >
                          {new Date(it.validade_ate).toLocaleDateString("pt-BR")}
                          {it.expired ? " · vencida" : ""}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>
          </div>
        </section>
      )}

      {/* Integrações — faixa compacta no rodapé. */}
      <section className="flex flex-col gap-4">
        <p className="dateline">Integrações</p>
        <Card>
          <CardContent className="pt-4">
            {errorMessage && <ErrorState message={errorMessage} />}
            {integrations && (
              <ul className="flex flex-col divide-y divide-surface-border">
                {integrations.map((integration) => (
                  <li key={integration.id} className="flex items-center justify-between py-2.5">
                    <span className="text-sm">{integration.name}</span>
                    <StatusIndicator status={integration.status} />
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
