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
} from "lucide-react";
import type { Metadata } from "next";
import { connection } from "next/server";
import Link from "next/link";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { PublicShell } from "@/components/layout/PublicShell";

const description =
  "Projeto Aurora — Plataforma Enterprise Base Genérica, pronta para acoplamento de múltiplos módulos de domínio com Clean Architecture e DevSecOps.";

export const metadata: Metadata = {
  title: "Projeto Aurora — Plataforma Enterprise Base",
  description,
  openGraph: { title: "Projeto Aurora", description, type: "website" },
};

const pillars = [
  { n: 1, label: "Autenticação OIDC Gov.br / Keycloak" },
  { n: 2, label: "e-MAG Acessibilidade & VLibras" },
  { n: 3, label: "Identidade Visual DSGov / White-Label" },
  { n: 4, label: "LGPD & Mascaramento PII em Logs" },
  { n: 5, label: "Headers Defensivos OWASP & CSP" },
  { n: 6, label: "Auditoria Imutável & Interoperabilidade" },
];

const govModules = [
  {
    code: "M01",
    title: "Segurança HTTP (OWASP / e-PING)",
    description: "Headers defensivos (HSTS, CSP com nonce, X-Frame DENY) e timeouts de servidor para prevenir DoS/Slowloris.",
    reason: "Protege a infraestrutura governamental contra ataques web clássicos e garante interoperabilidade segura.",
  },
  {
    code: "M02",
    title: "LGPD & Auditoria (Lei 13.709/2018)",
    description: "Mascaramento PII nativo (CPF, email, telefone), logs estruturados log/slog e correlação X-Request-ID.",
    reason: "Evita o vazamento acidental de dados sensíveis de cidadãos e servidores em logs e garante rastreabilidade.",
  },
  {
    code: "M03",
    title: "Autenticação OIDC Gov.br (Login Único)",
    description: "Validação local JWKS do Keycloak/Gov.br com mapeamento dos Níveis de Confiabilidade Bronze, Prata e Ouro.",
    reason: "Atende à Portaria SGD/SEDGG Nº 2.154 garantindo que apenas contas verificadas acessem serviços críticos.",
  },
  {
    code: "M04",
    title: "Acessibilidade e-MAG / WCAG 2.1 AA",
    description: "Barra de atalhos (Alt+1..4), VLibras nativo, alto contraste e-MAG, escala de fonte A+/A-/A e linting jsx-a11y.",
    reason: "Garante inclusão digital irrestrita para pessoas com deficiência visual, auditiva e motora no serviço público.",
  },
  {
    code: "M05",
    title: "Identidade Visual DSGov (Padrão Digital)",
    description: "Design system oficial com cores institucionais, componentes unificados (GovHeader, GovFooter) e tipografia oficial.",
    reason: "Promove padronização visual e transparência institucional em conformidade com o Guia de Identidade da SECOM/MGI.",
  },
];

const services = [
  {
    icon: LinkIcon,
    title: "Arquitetura Modular Clean Code",
    description:
      "Estrutura isolada em Go 1.25. Adicione novos módulos de negócio com total desacoplamento e governança.",
  },
  {
    icon: Bell,
    title: "Notificações em Tempo Real",
    description:
      "Hub WebSocket integrado para retransmissão instantânea de eventos de plataforma aos clientes.",
  },
  {
    icon: ScrollText,
    title: "Trilha de Auditoria LGPD",
    description:
      "Toda ação de escrita é registrada em logs de auditoria imutáveis no Postgres com proveniência e contexto.",
  },
  {
    icon: ShieldCheck,
    title: "Resiliência & Outbox Transacional",
    description:
      "Escrita atômica no banco de dados e publicação em background no RabbitMQ com entrega Exactly-Once.",
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
    <PublicShell>
      <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-20 px-6 py-12">
        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-5">
            <p className="dateline">Prefeitura Municipal de Rondonópolis · Plataforma Base</p>
            <h1 className="max-w-2xl text-4xl font-extrabold leading-[1.15] text-foreground sm:text-5xl">
              Sua fundação enterprise em conformidade com o Governo Federal.
            </h1>
            <p className="max-w-2xl text-muted text-base">
              O <strong className="font-semibold text-foreground">Projeto Aurora</strong> oferece infraestrutura de alta performance pré-configurada em estrita conformidade com as normas federais: <strong className="text-foreground">e-MAG, DSGov, LGPD, Gov.br (OIDC) e OWASP Top 10</strong>.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <Link href="/login">
                <Button size="md">Acessar Plataforma</Button>
              </Link>
              <Link href="/sobre">
                <Button size="md" variant="secondary">
                  Documentação & Conformidade
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

        {/* Seção Principal de Conformidade com Padrões do Governo Federal */}
        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-2">
            <p className="dateline">Conformidade com Padrões Governamentais (SGD/MGI)</p>
            <h2 className="text-2xl font-bold text-foreground">Os 5 Módulos de Conformidade do Governo Federal</h2>
            <p className="max-w-3xl text-sm text-muted">
              Desenvolvido respeitando rigorosamente as diretrizes da Secretaria de Governo Digital (SGD/MGI), garantindo segurança, privacidade, acessibilidade e interoperabilidade.
            </p>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {govModules.map((item) => (
              <div key={item.code} className="flex flex-col gap-3 rounded-xl border border-surface-border bg-surface p-5 shadow-xs">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-xs font-bold px-2 py-0.5 rounded bg-primary/10 text-primary">{item.code}</span>
                  <span className="text-[11px] font-semibold text-success">100% Conforme</span>
                </div>
                <h3 className="text-sm font-bold text-foreground">{item.title}</h3>
                <p className="text-xs text-muted leading-relaxed">{item.description}</p>
                <div className="mt-auto pt-3 border-t border-surface-border">
                  <p className="text-[11px] font-medium text-foreground/80">
                    <strong className="text-seal">Justificativa:</strong> {item.reason}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-2">
            <p className="dateline">Segurança & AppSec</p>
            <h2 className="text-2xl font-bold text-foreground">OWASP Top 10 Enterprise</h2>
            <p className="max-w-2xl text-sm text-muted">
              Padrões defensivos rigorosamente aplicados na fundação do sistema.
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
      </div>
    </PublicShell>
  );
}
