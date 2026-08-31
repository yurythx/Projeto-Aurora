import { Layers, Lock, Radar, RefreshCw } from "lucide-react";
import type { Metadata } from "next";
import { connection } from "next/server";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Logo } from "@/components/ui/Logo";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";

const description =
  "Visão geral da arquitetura do Projeto Aurora, base genérica enterprise com monólito modular em Go, Next.js, OIDC/Keycloak, Outbox e RabbitMQ.";

export const metadata: Metadata = {
  title: "Sobre — Projeto Aurora",
  description,
  openGraph: { title: "Sobre o Projeto Aurora", description, type: "website" },
};

// Resumo público da matriz OWASP
const owaspMapping = [
  { code: "A01", risk: "Broken Access Control", today: "RBAC por permissão em cada rota sensível — nunca só a presença de um token." },
  { code: "A02", risk: "Cryptographic Failures", today: "RS256 com chave própria para o login local, bcrypt, segredos via arquivo, nunca em texto puro." },
  { code: "A03", risk: "Injection", today: "100% consultas parametrizadas — conferido: zero concatenação de string em SQL em todo o backend." },
  { code: "A04", risk: "Insecure Design", today: "Monólito modular com fronteiras de módulo e decisões de arquitetura documentadas em ADRs." },
  { code: "A05", risk: "Security Misconfiguration", today: "CSP com nonce por requisição, containers non-root, headers de segurança em toda resposta." },
  { code: "A06", risk: "Vulnerable Components", today: "govulncheck, npm audit, Trivy e Dependabot rodando a cada mudança de código, no CI." },
  { code: "A07", risk: "Identification & Auth Failures", today: "Bloqueio de conta, rate limiting distribuído, erro sempre genérico (nunca revela se um usuário existe)." },
  { code: "A08", risk: "Software & Data Integrity Failures", today: "Idempotência e outbox transacional com entrega Exactly-Once para novos módulos." },
  { code: "A09", risk: "Logging & Monitoring Failures", today: "Auditoria imutável (a tabela recusa UPDATE/DELETE), logs correlacionados por request id, métricas e tracing." },
  { code: "A10", risk: "SSRF", today: "Conferido: nenhum endpoint aceita uma URL arbitrária vinda de quem chama." },
];

const principles = [
  {
    icon: Layers,
    title: "Monólito modular",
    description:
      "Um único deployable, dividido em módulos com fronteiras claras — a simplicidade operacional de um monólito pronto para novos acoplamentos de negócio.",
  },
  {
    icon: RefreshCw,
    title: "Resiliência & Outbox",
    description:
      "Circuit breaker e retry com backoff em toda chamada externa, fila de mensagens mortas (DLQ) e um transactional outbox.",
  },
  {
    icon: Lock,
    title: "Segurança por padrão",
    description:
      "Autenticação via Keycloak (OIDC) ou login local com chave RSA própria, CSP com nonce, auditoria imutável e rate limiting.",
  },
  {
    icon: Radar,
    title: "Observabilidade",
    description:
      "Métricas Prometheus, tracing OpenTelemetry e logs estruturados correlacionados por request id em toda a pilha.",
  },
];

export default async function AboutPage() {
  await connection();

  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center justify-between px-6 py-5">
        <Link href="/" className="flex items-center gap-2 text-lg font-semibold">
          <Logo size={32} />
          Projeto Aurora
        </Link>
        <nav className="flex items-center gap-4 text-sm">
          <Link href="/" className="text-muted hover:text-foreground">
            Início
          </Link>
          <Link href="/login">
            <Button size="sm">Entrar</Button>
          </Link>
        </nav>
      </header>

      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-12 px-6 py-12">
        <section className="flex flex-col gap-4">
          <h1 className="text-3xl font-bold text-foreground">Sobre a Plataforma Aurora Base</h1>
          <p className="text-muted">
            O <strong>Projeto Aurora</strong> é uma plataforma base genérica enterprise de alta performance, estruturada para servir de fundação para novos módulos de aplicações públicas e corporativas. Com arquitetura limpa em Go 1.25 no backend e Next.js 16 no frontend, ele abstrai a complexidade de autenticação SSO, segurança AppSec, mensageria via RabbitMQ e auditoria.
          </p>
        </section>

        <section className="flex flex-col gap-4">
          <h2 className="text-xl font-semibold text-foreground">Princípios Arquiteturais</h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {principles.map((principle) => {
              const Icon = principle.icon;
              return (
                <Card key={principle.title}>
                  <CardHeader className="flex flex-row items-center gap-3 space-y-0">
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Icon size={18} aria-hidden="true" />
                    </span>
                    <CardTitle className="text-base">{principle.title}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted">{principle.description}</p>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <div>
            <h2 className="text-xl font-semibold text-foreground">OWASP Top 10 Enterprise</h2>
            <p className="mt-1 text-sm text-muted">
              Tratado como checklist de engenharia desde o primeiro commit. A coluna da direita descreve a prática hoje na base do Projeto Aurora.
            </p>
          </div>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell>Risco</TableHeaderCell>
                <TableHeaderCell>Prática hoje no Projeto Aurora</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {owaspMapping.map((item) => (
                <TableRow key={item.code}>
                  <TableCell className="whitespace-nowrap align-top font-medium text-foreground">
                    <span className="font-mono text-xs font-semibold text-seal">{item.code}</span>{" "}
                    {item.risk}
                  </TableCell>
                  <TableCell className="text-muted">{item.today}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </section>

        <section className="flex flex-col gap-3">
          <h2 className="text-xl font-semibold text-foreground">Stack de Tecnologias</h2>
          <p className="text-muted">
            <strong>Backend em Go (Clean Architecture):</strong> Monólito modular (Golang 1.25), PostgreSQL 16, RabbitMQ (AMQP), MinIO S3 Storage e motor Typesense.
          </p>
          <p className="text-muted">
            <strong>Frontend em Next.js (App Router):</strong> React 19 com TypeScript, Tailwind CSS, NextAuth.js com SSO Keycloak + JWT local e WebSockets para eventos em tempo real.
          </p>
        </section>
      </main>

      <footer className="border-t border-surface-border px-6 py-6 text-center text-xs text-muted">
        © {new Date().getFullYear()} Projeto Aurora — Plataforma Base Enterprise
      </footer>
    </div>
  );
}
