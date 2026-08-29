import { getServerSession } from "next-auth/next";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { StatusIndicator } from "@/components/ui/StatusIndicator";
import { FileText, AlertTriangle, Clock, ShieldAlert } from "lucide-react";
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

// Visão geral — dashboard de administração municipal do Projeto-Nova.
// Exibe status das integrações ativas e atalhos rápidos para os módulos
// principais. Server Component: busca a sessão e integrações no servidor
// antes de qualquer HTML sair — sem useEffect/skeleton manual.
export default async function DashboardOverviewPage() {
  const session = await getServerSession(authOptions);
  const firstName = (session?.user?.name ?? session?.user?.email ?? "").split(/[@\s]/)[0];

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
    ? Math.max(1, ...demandsDash.funnel.map((f) => f.total))
    : 1;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold">
          {firstName ? `Olá, ${firstName}` : "Visão geral"}
        </h1>
        <p className="text-sm text-muted">Painel de administração municipal.</p>
      </div>

      <div className="flex flex-wrap gap-3">
        <Link href="/diario-oficial">
          <Button variant="secondary" size="sm">
            Diário Oficial
          </Button>
        </Link>
        <Link href="/monitoramento">
          <Button variant="secondary" size="sm">
            Monitoramento
          </Button>
        </Link>
        <Link href="/integracoes">
          <Button variant="secondary" size="sm">
            Ver integrações
          </Button>
        </Link>
        <Link href="/configuracao/usuarios">
          <Button variant="secondary" size="sm">
            Ver usuários
          </Button>
        </Link>
        <Link href="/configuracao">
          <Button variant="secondary" size="sm">
            Configurações
          </Button>
        </Link>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {stats && (
          <>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardTitle className="text-sm font-medium text-muted-foreground">Contratos Vigentes</CardTitle>
                <FileText className="w-4 h-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.total_vigentes}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  R$ {stats.valor_total_vigentes.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardTitle className="text-sm font-medium text-muted-foreground">Próximos do Vencimento</CardTitle>
                <AlertTriangle className={`w-4 h-4 ${stats.proximos_vencimento > 0 ? "text-amber-500" : "text-muted-foreground"}`} />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stats.proximos_vencimento}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  Vencem nos próximos 30 dias
                </p>
              </CardContent>
            </Card>
          </>
        )}
      </div>

      {demandsDash && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              Demandas Mensais (IN SCL 01/2019)
            </h2>
            <Link href="/contratos">
              <Button variant="secondary" size="sm">
                Abrir Kanban
              </Button>
            </Link>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardTitle className="text-sm font-medium text-muted-foreground">Em andamento</CardTitle>
                <FileText className="w-4 h-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{demandsDash.total_em_andamento}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  {demandsDash.total_arquivadas} arquivada(s)
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardTitle className="text-sm font-medium text-muted-foreground">SLA estourado</CardTitle>
                <Clock className={`w-4 h-4 ${demandsDash.sla_breached > 0 ? "text-red-500" : "text-muted-foreground"}`} />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{demandsDash.sla_breached}</div>
                <p className="text-xs text-muted-foreground mt-1">demanda(s) fora do prazo da etapa</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardTitle className="text-sm font-medium text-muted-foreground">Pendências abertas</CardTitle>
                <AlertTriangle className={`w-4 h-4 ${demandsDash.open_occurrences > 0 ? "text-amber-500" : "text-muted-foreground"}`} />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{demandsDash.open_occurrences}</div>
                <p className="text-xs text-muted-foreground mt-1">ocorrências não resolvidas</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardTitle className="text-sm font-medium text-muted-foreground">Certidões</CardTitle>
                <ShieldAlert
                  className={`w-4 h-4 ${demandsDash.certidoes_expired > 0 ? "text-red-500" : demandsDash.certidoes_expiring_30d > 0 ? "text-amber-500" : "text-muted-foreground"}`}
                />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {demandsDash.certidoes_expired}
                  <span className="text-sm font-normal text-muted-foreground"> vencida(s)</span>
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  + {demandsDash.certidoes_expiring_30d} vencem em 30 dias
                </p>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 lg:grid-cols-3">
            <Card className="lg:col-span-1">
              <CardHeader>
                <CardTitle className="text-sm">Funil por etapa</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-2">
                {demandsDash.funnel.map((f) => (
                  <div key={f.etapa} className="flex items-center gap-2 text-xs">
                    <span className="w-6 shrink-0 font-mono text-muted-foreground">{f.etapa}</span>
                    <div className="flex-1">
                      <div className="flex justify-between">
                        <span className="truncate">{f.label}</span>
                        <span className="font-semibold">{f.total}</span>
                      </div>
                      <div className="mt-0.5 h-1.5 rounded bg-surface-hover">
                        <div
                          className="h-1.5 rounded bg-primary"
                          style={{ width: `${(f.total / funnelMax) * 100}%` }}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>

            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle className="text-sm">Atenção imediata</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 text-xs">
                <div>
                  <p className="font-semibold text-muted-foreground mb-1">SLA estourado</p>
                  {demandsDash.sla_breached_items.length === 0 ? (
                    <p className="text-muted-foreground">Nenhuma demanda fora do prazo.</p>
                  ) : (
                    <ul className="flex flex-col gap-1">
                      {demandsDash.sla_breached_items.map((it) => (
                        <li key={it.demanda_id} className="flex justify-between gap-2">
                          <span>
                            Contrato {it.contrato_numero || "—"} · Etapa {it.etapa}
                          </span>
                          <span className="font-mono text-red-500">
                            {it.days_in_stage}d / {it.sla_days}d
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
                <div>
                  <p className="font-semibold text-muted-foreground mb-1">Certidões vencidas / a vencer</p>
                  {demandsDash.certidoes_items.length === 0 ? (
                    <p className="text-muted-foreground">Nenhuma certidão vencida ou a vencer em 30 dias.</p>
                  ) : (
                    <ul className="flex flex-col gap-1">
                      {demandsDash.certidoes_items.slice(0, 12).map((it) => (
                        <li key={`${it.demanda_id}-${it.doc_type}`} className="flex justify-between gap-2">
                          <span className="truncate">
                            {it.contrato_numero || "—"} · {it.label}
                          </span>
                          <span className={`font-mono ${it.expired ? "text-red-500" : "text-amber-500"}`}>
                            {new Date(it.validade_ate).toLocaleDateString("pt-BR")}
                            {it.expired ? " (vencida)" : ""}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Status das integrações</CardTitle>
        </CardHeader>
        <CardContent>
          {errorMessage && <ErrorState message={errorMessage} />}
          {integrations && (
            <ul className="flex flex-col gap-3">
              {integrations.map((integration) => (
                <li key={integration.id} className="flex items-center justify-between">
                  <span className="text-sm">{integration.name}</span>
                  <StatusIndicator status={integration.status} />
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
