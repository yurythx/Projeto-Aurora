import { getServerSession } from "next-auth/next";
import Link from "next/link";
import { 
  Shield, 
  Users, 
  Settings, 
  Activity, 
  Layers, 
  Cpu, 
  Zap, 
  CheckCircle2, 
  Plug, 
  FileCode,
  ArrowRight
} from "lucide-react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { StatusIndicator } from "@/components/ui/StatusIndicator";
import { authOptions } from "@/lib/auth/options";
import { ApiError } from "@/lib/api/client";
import { serverApiGet } from "@/lib/api/server";
import { getSystemHealth } from "@/lib/health/getSystemHealth";
import type { Integration } from "@/types/api";

const statCellToneClass = {
  primary: "bg-primary/10 text-primary",
  success: "bg-success/10 text-success",
  warning: "bg-warning/10 text-warning",
  danger: "bg-danger/10 text-danger",
} as const;

function PlatformStatCell({
  label,
  value,
  hint,
  icon: Icon,
  tone = "primary",
}: {
  label: string;
  value: string | number;
  hint?: string;
  icon: typeof Shield;
  /** "primary" pros indicadores puramente descritivos (arquitetura,
   * autenticação, mensageria — sempre verdadeiros, não dependem de
   * checagem nenhuma); success/warning/danger pra um indicador que
   * carrega um status DE VERDADE (ex.: "Estado do Core", abaixo). */
  tone?: keyof typeof statCellToneClass;
}) {
  return (
    <div className="flex min-w-[10rem] flex-1 items-center gap-4 rounded-xl border border-surface-border bg-surface p-4 shadow-sm">
      <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${statCellToneClass[tone]}`}>
        <Icon size={20} />
      </div>
      <div className="flex flex-col gap-0.5">
        <span className="text-xs font-semibold uppercase tracking-wider text-muted">{label}</span>
        <span className="font-mono text-xl font-bold text-foreground">{value}</span>
        {hint && <span className="text-[11px] text-muted">{hint}</span>}
      </div>
    </div>
  );
}

export default async function DashboardOverviewPage() {
  const session = await getServerSession(authOptions);
  const firstName = (session?.user?.name ?? session?.user?.email ?? "").split(/[@\s]/)[0];

  const competencia = new Date()
    .toLocaleDateString("pt-BR", { month: "long", year: "numeric" })
    .toUpperCase();

  let integrations: Integration[] | null = null;
  let errorMessage: string | null = null;

  try {
    const integrationsRes = await serverApiGet<Integration[]>("v1/integrations");
    integrations = integrationsRes.data;
  } catch (err) {
    errorMessage = err instanceof ApiError ? err.message : "Falha ao carregar integracoes";
  }

  // "Estado do Core" abaixo era "Saudável" fixo no código-fonte, sempre —
  // a mesma classe de achado já corrigida no painel de Monitoramento
  // (PlatformMonitoringDashboard), só que esta tela tinha ficado de fora
  // daquela correção. getSystemHealth() é a mesma checagem real
  // (/ready do backend: postgres/rabbitmq/minio), chamada direto aqui
  // (Server Component) em vez de via HTTP em GET /api/health.
  const systemHealth = await getSystemHealth();

  const modules = [
    {
      id: "example",
      title: "Módulo Exemplo (Template)",
      description: "Estrutura Clean Architecture genérica de modelo para criação de novos módulos de negócio.",
      icon: FileCode,
      tag: "Template / Boilerplate",
      status: "PRONTO PARA USO",
      href: "/configuracao",
    },
    {
      id: "users",
      title: "Gestão de Usuários & Perfis",
      description: "Gerenciamento de contas locais, sincronização Keycloak SSO e atribuição de permissões (Roles).",
      icon: Users,
      tag: "Segurança & RBAC",
      status: "ATIVO",
      href: "/configuracao",
    },
    {
      id: "integrations",
      title: "Barramento de Integrações",
      description: "Monitoramento de saúde de provedores externos e registros de conectividade.",
      icon: Plug,
      tag: "Conectividade",
      status: "ONLINE",
      href: "/integracoes",
    },
    {
      id: "monitoring",
      title: "Monitoramento & Telemetria",
      description: "Métricas Prometheus, OpenTelemetry e gerenciamento do Transactional Outbox / RabbitMQ.",
      icon: Activity,
      tag: "DevSecOps",
      status: "MONITORADO",
      href: "/monitoramento",
    },
  ];

  return (
    <div className="flex flex-col gap-8 pb-10">
      {/* Header Principal */}
      <header className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-surface-border pb-6">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <span className="dateline">Projeto Aurora Base · {competencia}</span>
            <span className="inline-flex items-center rounded-full bg-success/10 px-2 py-0.5 text-[10px] font-semibold text-success">
              Versão Base Enterprise
            </span>
          </div>
          <h1 className="text-3xl font-bold tracking-tight">
            {firstName ? `Bem-vindo(a), ${firstName}` : "Visão Geral do Projeto Aurora"}
          </h1>
          <p className="text-sm text-muted">
            Estrutura base pronta para recebimento e acoplamento de novos módulos de domínio.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link href="/configuracao">
            <Button size="sm" className="gap-2">
              <Settings size={16} />
              Configurações
            </Button>
          </Link>
          <Link href="/monitoramento">
            <Button size="sm" variant="secondary" className="gap-2">
              <Activity size={16} />
              Métricas do Sistema
            </Button>
          </Link>
        </div>
      </header>

      {/* Estatísticas de Infraestrutura */}
      <section className="flex flex-col gap-3">
        <h2 className="text-xs font-semibold uppercase tracking-widest text-muted">Indicadores da Plataforma</h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <PlatformStatCell
            label="Arquitetura"
            value="Clean Arch"
            hint="Go 1.25 + Next.js App Router"
            icon={Layers}
          />
          <PlatformStatCell
            label="Autenticação"
            value="OIDC + SSO"
            hint="Keycloak & Local JWT Signer"
            icon={Shield}
          />
          <PlatformStatCell
            label="Mensageria"
            value="Outbox EventBus"
            hint="PostgreSQL + RabbitMQ DLQ"
            icon={Zap}
          />
          <PlatformStatCell
            label="Estado do Core"
            value={
              systemHealth.status === "ok"
                ? "Saudável"
                : systemHealth.status === "degraded"
                  ? "Degradado"
                  : "Indisponível"
            }
            hint={
              systemHealth.status === "ok"
                ? "Todos os serviços verificados online"
                : systemHealth.status === "degraded"
                  ? Object.entries(systemHealth.services)
                      .filter(([, s]) => s.status !== "ok")
                      .map(([name]) => name)
                      .join(", ") || "Algum serviço indisponível"
                  : "Backend não respondeu"
            }
            tone={
              systemHealth.status === "ok"
                ? "success"
                : systemHealth.status === "degraded"
                  ? "warning"
                  : "danger"
            }
            icon={Cpu}
          />
        </div>
      </section>

      {/* Grid de Módulos Prontos para Uso */}
      <section className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">Módulos da Aplicação</h2>
            <p className="text-xs text-muted">Serviços e componentes de infraestrutura disponíveis no Projeto Aurora.</p>
          </div>
        </div>

        <div className="grid gap-5 sm:grid-cols-2">
          {modules.map((mod) => {
            const Icon = mod.icon;
            return (
              <Card key={mod.id} className="relative overflow-hidden transition-all hover:border-primary/50 hover:shadow-md">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Icon size={22} />
                    </div>
                    <span className="rounded-md bg-surface-border/60 px-2 py-1 text-[11px] font-medium text-muted">
                      {mod.tag}
                    </span>
                  </div>
                  <CardTitle className="text-base font-bold pt-2">{mod.title}</CardTitle>
                  <CardDescription className="text-xs text-muted leading-relaxed">
                    {mod.description}
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-0 flex items-center justify-between border-t border-surface-border/50 mt-2 py-3 text-xs">
                  <span className="flex items-center gap-1.5 font-medium text-success">
                    <CheckCircle2 size={14} />
                    {mod.status}
                  </span>
                  <Link href={mod.href} className="flex items-center gap-1 text-primary hover:underline font-semibold">
                    Acessar <ArrowRight size={14} />
                  </Link>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </section>

      {/* Status de Conectividade e Integrações */}
      <section className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Status de Serviços & Integrações</h2>
          <Link href="/integracoes" className="text-xs text-primary hover:underline">
            Ver todas →
          </Link>
        </div>
        <Card>
          <CardContent className="pt-4">
            {errorMessage && <ErrorState message={errorMessage} />}
            {integrations && (
              <ul className="flex flex-col divide-y divide-surface-border">
                {integrations.map((integration) => (
                  <li key={integration.id} className="flex items-center justify-between py-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-md bg-surface-border/40 text-foreground">
                        <Plug size={16} />
                      </div>
                      <div>
                        <p className="text-sm font-semibold">{integration.name}</p>
                        <p className="text-xs text-muted">Serviço externo de apoio</p>
                      </div>
                    </div>
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
