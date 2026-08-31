import {
  Bell,
  Blocks,
  Bug,
  FileCheck,
  Globe,
  KeyRound,
  Link as LinkIcon,
  Lock,
  Package,
  ScrollText,
  Settings,
  ShieldCheck,
  UserCheck,
  Zap,
} from "lucide-react";
import type { Metadata } from "next";
import { connection } from "next/server";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Logo } from "@/components/ui/Logo";
import { Seal } from "@/components/ui/Seal";

const description =
  "Projeto Aurora — Plataforma Enterprise Base Genérica, pronta para acoplamento de múltiplos módulos de domínio com Clean Architecture e DevSecOps.";

export const metadata: Metadata = {
  title: "Projeto Aurora — Plataforma Enterprise Base",
  description,
  openGraph: { title: "Projeto Aurora", description, type: "website" },
};

const pillars = [
  { n: 1, label: "Autenticação OIDC / SSO + Local JWT" },
  { n: 2, label: "Transactional Outbox Pattern" },
  { n: 3, label: "RabbitMQ Mensageria & DLQ" },
  { n: 4, label: "Design System & e-MAG Acessibilidade" },
  { n: 5, label: "Rate Limiting & Idempotência" },
  { n: 6, label: "Auditoria Imutável & Telemetria" },
];

const services = [
  {
    icon: LinkIcon,
    title: "Arquitetura Modular",
    description:
      "Estrutura Clean Architecture isolada. Adicione novos módulos de negócio sem alterar o núcleo da plataforma.",
  },
  {
    icon: Bell,
    title: "Notificações em Tempo Real",
    description:
      "Hub WebSocket integrado para retransmissão instantânea de eventos de plataforma aos clientes.",
  },
  {
    icon: ScrollText,
    title: "Trilha de Auditoria",
    description:
      "Toda ação de escrita é registrada em logs de auditoria imutáveis com proveniência e contexto.",
  },
  {
    icon: ShieldCheck,
    title: "Resiliência & Outbox",
    description:
      "Escrita atômica no banco de dados e publicação em background no RabbitMQ sem perda de eventos.",
  },
];

const owaspPractices = [
  { code: "A01", icon: Lock, title: "Broken Access Control", description: "RBAC por permissão em cada rota protegida." },
  { code: "A02", icon: KeyRound, title: "Cryptographic Failures", description: "Argon2id / bcrypt, JWT assinado RS256 e TLS." },
  { code: "A03", icon: Bug, title: "Injection", description: "Queries estritamente parametrizadas com pgxpool." },
  { code: "A04", icon: Blocks, title: "Insecure Design", description: "Arquitetura limpa com pacotes desacoplados." },
  { code: "A05", icon: Settings, title: "Security Misconfiguration", description: "CSP com nonce por requisição e headers OWASP." },
  { code: "A06", icon: Package, title: "Vulnerable Components", description: "Verificação contínua de vulnerabilidades e deps." },
  { code: "A07", icon: UserCheck, title: "Auth Failures", description: "Lockout de conta, rate limiting por IP/user." },
  { code: "A08", icon: FileCheck, title: "Data Integrity Failures", description: "Chaves de idempotência e outbox transacional." },
  { code: "A09", icon: ScrollText, title: "Logging & Monitoring", description: "Prometheus metrics e suporte OpenTelemetry." },
  { code: "A10", icon: Globe, title: "SSRF", description: "Validação rigorosa de endpoints e requisições de saída." },
];

export default async function LandingPage() {
  await connection();

  return (
    <div className="flex min-h-screen flex-col">
      <header className="mx-auto flex w-full max-w-5xl items-center justify-between px-6 py-5">
        <span className="flex items-center gap-2.5 text-lg font-bold tracking-tight">
          <Logo size={30} />
          Projeto Aurora
        </span>
        <nav className="flex items-center gap-4 text-sm">
          <Link href="/sobre" className="text-muted hover:text-foreground">
            Sobre
          </Link>
          <Link href="/login">
            <Button size="sm">Entrar</Button>
          </Link>
        </nav>
      </header>

      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-20 px-6 py-12">
        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-5">
            <p className="dateline">Prefeitura Municipal de Rondonópolis · Plataforma Base</p>
            <h1 className="max-w-2xl text-4xl font-extrabold leading-[1.15] text-foreground sm:text-5xl">
              Sua fundação enterprise para novas aplicações.
            </h1>
            <p className="max-w-xl text-muted text-base">
              O <strong className="font-semibold text-foreground">Projeto Aurora</strong> oferece toda a
              infraestrutura fundamental pré-configurada: autenticação SSO/Local, outbox transacional,
              mensageria RabbitMQ, auditoria, idempotência e Design System oficial.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <Link href="/login">
                <Button size="md">Acessar Plataforma</Button>
              </Link>
              <Link href="/sobre">
                <Button size="md" variant="secondary">
                  Documentação Base
                </Button>
              </Link>
            </div>
          </div>

          <ol className="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-surface-border bg-surface-border sm:grid-cols-3 lg:grid-cols-6">
            {pillars.map((item) => (
              <li key={item.n} className="flex flex-col gap-2 bg-surface p-4">
                <span className="font-mono text-xs font-bold text-primary">
                  {String(item.n).padStart(2, "0")}
                </span>
                <span className="text-xs font-medium leading-snug text-foreground">{item.label}</span>
              </li>
            ))}
          </ol>
        </section>

        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-2">
            <p className="dateline">Segurança & Conformidade</p>
            <h2 className="text-2xl font-bold text-foreground">OWASP Top 10 Enterprise</h2>
            <p className="max-w-2xl text-sm text-muted">
              Padrões de segurança rigorosamente aplicados na fundação do sistema.
            </p>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
            {owaspPractices.map((item) => {
              const Icon = item.icon;
              return (
                <div
                  key={item.code}
                  className="flex flex-col gap-2 rounded-xl border border-surface-border bg-surface p-4"
                >
                  <div className="flex items-center gap-2">
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Icon size={16} aria-hidden="true" />
                    </span>
                    <div>
                      <p className="font-mono text-xs font-bold text-primary">{item.code}</p>
                      <p className="text-xs font-semibold text-foreground">{item.title}</p>
                    </div>
                  </div>
                  <p className="text-xs text-muted leading-relaxed">{item.description}</p>
                </div>
              );
            })}
          </div>
        </section>

        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-2">
            <p className="dateline">Capacidades Prontas</p>
            <h2 className="text-2xl font-bold text-foreground">Serviços de Plataforma</h2>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {services.map((service) => {
              const Icon = service.icon;
              return (
                <Card key={service.title}>
                  <CardHeader className="flex flex-row items-center gap-3">
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Icon size={18} aria-hidden="true" />
                    </span>
                    <CardTitle className="text-sm font-bold">{service.title}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-xs text-muted leading-relaxed">{service.description}</p>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </section>
      </main>

      <footer className="mx-auto flex w-full max-w-5xl items-center gap-4 border-t border-surface-border px-6 py-6 text-xs text-muted">
        <Seal size={40} decorative className="text-surface-border" />
        <p>
          © {new Date().getFullYear()} Prefeitura Municipal de Rondonópolis · Projeto Aurora Base
        </p>
      </footer>
    </div>
  );
}
