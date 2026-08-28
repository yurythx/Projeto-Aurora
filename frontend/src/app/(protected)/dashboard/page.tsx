import { getServerSession } from "next-auth/next";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { StatusIndicator } from "@/components/ui/StatusIndicator";
import { FileText, AlertTriangle } from "lucide-react";
import { authOptions } from "@/lib/auth/options";
import { ApiError } from "@/lib/api/client";
import { serverApiGet } from "@/lib/api/server";
import type { Integration } from "@/types/api";

interface DashboardStats {
  total_vigentes: number;
  valor_total_vigentes: number;
  proximos_vencimento: number;
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
  let errorMessage: string | null = null;

  try {
    const [integrationsRes, statsRes] = await Promise.all([
      serverApiGet<Integration[]>("v1/integrations"),
      serverApiGet<DashboardStats>("v1/contratos/dashboard").catch(() => ({ data: null })),
    ]);
    integrations = integrationsRes.data;
    if (statsRes && statsRes.data) {
      stats = statsRes.data;
    }
  } catch (err) {
    errorMessage = err instanceof ApiError ? err.message : "Falha ao carregar";
  }

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
