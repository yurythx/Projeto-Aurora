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
import { Logo } from "@/components/ui/Logo";
import { Seal } from "@/components/ui/Seal";

// §auditoria 2026-08: só a página inicial e /sobre têm metadados públicos
// — as únicas rotas públicas com algo que vale compartilhar.
const description =
  "Plataforma de fiscalização de contratos da Administração Municipal de Rondonópolis — as seis etapas da IN SCL 01/2019 num quadro só, com o Diário Oficial lido e cruzado automaticamente.";

export const metadata: Metadata = {
  title: "Projeto Nova — Fiscalização de Contratos Municipais",
  description,
  openGraph: { title: "Projeto Nova", description, type: "website" },
};

// O fluxo de 6 etapas da IN SCL 01/2019 — o assunto do produto. Numeração
// aqui é informação real (a ordem importa), não enfeite.
const etapas = [
  { n: 1, label: "Elaborar OF / Pré-empenho" },
  { n: 2, label: "Tramitar no Planejamento" },
  { n: 3, label: "Emitir OS / Envio à empresa" },
  { n: 4, label: "Execução e recepção" },
  { n: 5, label: "Relatório de pagamento / Certidões" },
  { n: 6, label: "Contabilidade / Liquidação" },
];

// Serviços/módulos reais da plataforma — nada aspiracional, cada item
// corresponde a um módulo que já existe em backend/internal.
const services = [
  {
    icon: LinkIcon,
    title: "Integrações extensíveis",
    description:
      "Arquitetura modular. Lê o Diário Oficial de Rondonópolis hoje e o próximo sistema amanhã sem tocar no núcleo da plataforma.",
  },
  {
    icon: Bell,
    title: "Notificações em tempo real",
    description:
      "Alerta na hora (via WebSocket) quando um edital ou contrato do Diário Oficial entra no seu quadro, ou quando um SLA estoura.",
  },
  {
    icon: ScrollText,
    title: "Trilha de auditoria",
    description:
      "Cada aprovação e movimentação de card fica registrada numa tabela que o banco recusa alterar — o controle interno consegue reconstruir o processo.",
  },
  {
    icon: ShieldCheck,
    title: "Resiliência a falhas",
    description:
      "RabbitMQ com outbox transacional e retry automático. Uma indisponibilidade externa não perde evento nem escrita.",
  },
];

// Mapeamento OWASP Top 10 -> prática concreta já implementada. Cada linha
// foi conferida no código durante esta sessão, não copiada de um
// checklist genérico.
const owaspPractices = [
  {
    code: "A01",
    icon: Lock,
    title: "Broken Access Control",
    description: "RBAC por permissão em cada rota sensível — nunca só a presença de um token.",
  },
  {
    code: "A02",
    icon: KeyRound,
    title: "Cryptographic Failures",
    description: "RS256 com chave própria para o login local, bcrypt, segredos nunca em texto puro.",
  },
  {
    code: "A03",
    icon: Bug,
    title: "Injection",
    description: "Toda consulta é parametrizada — zero concatenação de string em SQL no backend inteiro.",
  },
  {
    code: "A04",
    icon: Blocks,
    title: "Insecure Design",
    description: "Monólito modular com fronteiras de módulo e decisões de arquitetura documentadas (ADRs).",
  },
  {
    code: "A05",
    icon: Settings,
    title: "Security Misconfiguration",
    description: "CSP com nonce por requisição, containers non-root, headers de segurança em toda resposta.",
  },
  {
    code: "A06",
    icon: Package,
    title: "Vulnerable Components",
    description: "govulncheck, npm audit, Trivy e Dependabot rodando a cada mudança de código.",
  },
  {
    code: "A07",
    icon: UserCheck,
    title: "Auth Failures",
    description: "Bloqueio de conta, rate limit distribuído, e nunca um erro que revele se um usuário existe.",
  },
  {
    code: "A08",
    icon: FileCheck,
    title: "Data Integrity Failures",
    description: "Idempotência e outbox transacional — nenhum evento duplicado, nenhuma escrita perdida.",
  },
  {
    code: "A09",
    icon: ScrollText,
    title: "Logging & Monitoring",
    description: "Auditoria imutável de toda ação sensível, logs correlacionados por request id.",
  },
  {
    code: "A10",
    icon: Globe,
    title: "SSRF",
    description: "Nenhum endpoint aceita uma URL arbitrária vinda de quem chama.",
  },
];

export default async function LandingPage() {
  // Força renderização dinâmica — necessário para o CSP com nonce gerado
  // por requisição em proxy.ts. Ver o comentário equivalente em
  // app/login/page.tsx.
  await connection();

  return (
    <div className="flex min-h-screen flex-col">
      <header className="mx-auto flex w-full max-w-5xl items-center justify-between px-6 py-5">
        <span className="flex items-center gap-2.5 text-lg font-semibold">
          <Logo size={30} />
          Projeto Nova
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

      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-24 px-6 py-14">
        {/* Hero — a tese é o fluxo, não um slogan. */}
        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-5">
            <p className="dateline">Rondonópolis-MT · Secretaria de Administração</p>
            <h1 className="max-w-2xl text-4xl font-semibold leading-[1.1] text-foreground sm:text-5xl">
              Da requisição ao arquivamento, uma demanda de cada vez.
            </h1>
            <p className="max-w-xl text-muted">
              O <strong className="font-semibold text-foreground">Projeto Nova</strong> acompanha cada
              demanda mensal de contrato pelas seis etapas da IN SCL 01/2019, gera os documentos
              oficiais prontos para assinatura e lê o Diário Oficial de Rondonópolis para cruzar
              publicações com os seus contratos.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <Link href="/login">
                <Button size="md">Entrar</Button>
              </Link>
              <Link href="/sobre">
                <Button size="md" variant="secondary">
                  Como funciona
                </Button>
              </Link>
            </div>
          </div>

          {/* O fluxo de 6 etapas como elemento característico. */}
          <ol className="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-surface-border bg-surface-border sm:grid-cols-3 lg:grid-cols-6">
            {etapas.map((etapa) => (
              <li key={etapa.n} className="flex flex-col gap-2 bg-surface p-4">
                <span className="font-mono text-xs font-semibold text-seal">
                  {String(etapa.n).padStart(2, "0")}
                </span>
                <span className="text-sm leading-snug text-foreground">{etapa.label}</span>
              </li>
            ))}
          </ol>
        </section>

        {/* Segurança por padrão — OWASP Top 10 */}
        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-2">
            <p className="dateline">Segurança por padrão</p>
            <h2 className="text-2xl font-semibold text-foreground">OWASP Top 10, uma prática por linha</h2>
            <p className="max-w-2xl text-sm text-muted">
              As dez categorias e o que já existe no código para cada uma — conferido nesta base,
              não copiado de um checklist genérico.
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
                      <p className="font-mono text-xs font-semibold text-seal">{item.code}</p>
                      <p className="text-xs font-medium text-foreground">{item.title}</p>
                    </div>
                  </div>
                  <p className="text-xs text-muted">{item.description}</p>
                </div>
              );
            })}
          </div>
          <p className="text-xs text-muted">
            Um item (assinatura de artefatos de build, parte de A08) ainda não existe — está
            documentado como pendência, não escondido. Veja{" "}
            <Link href="/sobre" className="text-primary hover:underline">
              a página Sobre
            </Link>{" "}
            para o detalhe completo e o roadmap de segurança.
          </p>
        </section>

        {/* Serviços */}
        <section className="flex flex-col gap-8">
          <div className="flex flex-col gap-2">
            <p className="dateline">Em produção hoje</p>
            <h2 className="text-2xl font-semibold text-foreground">Serviços</h2>
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
                    <CardTitle className="text-sm">{service.title}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted">{service.description}</p>
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
          © {new Date().getFullYear()} Prefeitura Municipal de Rondonópolis ·{" "}
          <Link href="/sobre" className="hover:text-foreground">
            Sobre o sistema e a API
          </Link>
        </p>
      </footer>
    </div>
  );
}
