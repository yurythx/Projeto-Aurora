import {
  Accessibility,
  FileCheck2,
  Gauge,
  KeyRound,
  Landmark,
  ListChecks,
  Lock,
  Network,
  ScrollText,
  ShieldCheck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { connection } from "next/server";

import { PublicShell } from "@/components/layout/PublicShell";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";

const description =
  "Catálogo de referência do Projeto Aurora: normas seguidas, parceiros oficiais (Gov.br, VLibras, DSGov), stack, ferramentas de CI e cada parâmetro de configuração — com a norma que atende e como é aplicado.";

export const metadata: Metadata = {
  title: "Padrões & Parâmetros — Projeto Aurora",
  description,
  openGraph: { title: "Padrões & Parâmetros — Projeto Aurora", description, type: "website" },
};

// ---------------------------------------------------------------------------
// 1. Normas
// ---------------------------------------------------------------------------
const normas = [
  {
    norma: "e-MAG 2.0 / WCAG 2.1 AA",
    base: "Lei 13.146/2015",
    exige: "Acessibilidade digital: navegação por teclado, contraste, rótulos, Libras, tipografia flexível.",
    onde: "Barra e-MAG (skip links Alt+1..4, alto contraste, escala de fonte), VLibras nativo, lang=\"pt-BR\", tokens em rem, eslint-plugin-jsx-a11y.",
  },
  {
    norma: "LGPD",
    base: "Lei 13.709/2018",
    exige: "Minimização de PII, base legal, consentimento registrável, direitos do titular (art. 18), mascaramento em logs.",
    onde: "user_consents + modal + trigger de auditoria; logging.PIICPF/PIIEmail/PIIPhone; UserListItem sem e-mail; data_subject_requests.",
  },
  {
    norma: "LAI",
    base: "Lei 12.527/2011 · LC 131/2009",
    exige: "Transparência ativa, trilha de auditoria consultável, exportação em formato aberto.",
    onde: "audit_logs append-only; GET /api/v1/audit/export (CSV, restrito a audit:read). Transparência ativa e JSON/XML: em andamento (Fase 3).",
  },
  {
    norma: "Portaria SGD/SEDGG Nº 2.154",
    base: "Login Único Gov.br",
    exige: "Níveis de confiabilidade Bronze / Prata / Ouro no fluxo de autenticação federada.",
    onde: "Validador OIDC/JWKS (go-oidc) mapeia o nível da conta Gov.br para autorização; identity.HasGovBRLevelAtLeast(...).",
  },
  {
    norma: "DSGov / GovBR-DS",
    base: "Padrão Digital de Governo",
    exige: "Identidade visual oficial, rodapé institucional com canais e LGPD/LAI, suporte white-label.",
    onde: "Design System em components/ui + BrandingProvider server-side; Footer com LAI/LGPD e canais de atendimento.",
  },
  {
    norma: "e-PING",
    base: "Interoperabilidade",
    exige: "API REST documentada em contrato aberto (OpenAPI 3.0).",
    onde: "GET /openapi.json + Swagger UI em /docs; docs/openapi.yaml.",
  },
  {
    norma: "OWASP ASVS / Top 10",
    base: "Segurança de aplicação",
    exige: "Controle de acesso, criptografia, injeção, cabeçalhos, rate limiting, componentes vulneráveis.",
    onde: "RBAC por permissão em cada rota; RS256 + bcrypt; SQL 100% parametrizado; CSP+HSTS; rate limit distribuído; govulncheck no CI.",
  },
];

// ---------------------------------------------------------------------------
// 2. Parceiros e órgãos oficiais
// ---------------------------------------------------------------------------
const parceiros = [
  {
    nome: "Login Único Gov.br (OIDC)",
    mantenedor: "SGD/MGI — Serpro",
    origem: "acesso.gov.br",
    integracao:
      "Autenticação federada via Keycloak como broker OIDC. O backend valida o JWT localmente por JWKS, sem chamada por requisição, e lê o nível Bronze/Prata/Ouro das claims.",
  },
  {
    nome: "VLibras",
    mantenedor: "SPB / gov.br — UFPB / Serpro",
    origem: "vlibras.gov.br/app",
    integracao:
      "VLibrasWidget.tsx carrega o player (WebAssembly) do domínio oficial; a CSP libera vlibras.gov.br, *.vlibras.gov.br e cdn.jsdelivr.net (script via strict-dynamic).",
  },
  {
    nome: "e-MAG — Modelo de Acessibilidade em Governo Eletrônico",
    mantenedor: "SGD/MGI",
    origem: "gov.br/governodigital",
    integracao:
      "Barra de atalhos por teclado (Alt+1..4), alto contraste preto/amarelo, redimensionamento de fonte, página /acessibilidade.",
  },
  {
    nome: "GovBR-DS / DSGov (Design System)",
    mantenedor: "SGD/MGI",
    origem: "gov.br/ds",
    integracao:
      "Vocabulário de tokens (cor, tipografia, espaçamento) no globals.css e nos componentes de components/ui; rodapé institucional unificado.",
  },
  {
    nome: "e-PING (Padrões de Interoperabilidade)",
    mantenedor: "SGD/MGI",
    origem: "gov.br/governodigital",
    integracao:
      "Contrato OpenAPI 3.0 público (/openapi.json, /docs) — RESTful, JSON, versionado em /api/v1.",
  },
];

// ---------------------------------------------------------------------------
// 3. Stack
// ---------------------------------------------------------------------------
const stackBackend = [
  ["go-chi/chi", "v5.3.2", "Router HTTP e pilha de middlewares"],
  ["go-chi/cors", "v1.2.2", "CORS travado à origem do frontend, AllowCredentials"],
  ["jackc/pgx", "v5.10.0", "Driver + pool PostgreSQL; consultas sempre parametrizadas"],
  ["coreos/go-oidc + go-jose", "v3.20.0 / v4.1.4", "Discovery OIDC e verificação JWKS do Keycloak / Gov.br"],
  ["golang-jwt/jwt", "v5.3.1", "Emissão/verificação do token do login local (RS256)"],
  ["go-playground/validator", "v10.30.3", "Validação de entrada por struct-tag (httputil.Validate)"],
  ["rabbitmq/amqp091-go", "v1.14.0", "Outbox transacional → RabbitMQ, filas + DLQ"],
  ["minio-go", "v7.3.0", "Object storage S3 via URLs pré-assinadas"],
  ["gorilla/websocket", "v1.5.3", "Broker WebSocket de notificações (autenticado por ticket)"],
  ["prometheus/client_golang", "v1.24.1", "Métricas /metrics (rotuladas pelo padrão de rota do chi)"],
  ["go.opentelemetry.io/otel", "v1.45.0", "Tracing distribuído (span por requisição, propagado ao worker)"],
  ["sony/gobreaker", "v2.4.0", "Circuit breaker para chamadas a provedores externos"],
  ["golang.org/x/crypto", "v0.55.0", "bcrypt (senha do login local)"],
];

const stackFrontend = [
  ["next (App Router)", "16.3.2", "SSR/SSG, proxy.ts (CSP com nonce + gate de sessão), React Compiler"],
  ["next-auth", "4.24", "Sessão JWT em cookie HttpOnly; Keycloak (PKCE) + login local"],
  ["react / react-dom", "19.2.8", "UI; memoização automática via React Compiler"],
  ["swr", "2.5", "Data fetching client-side pelo proxy BFF (sem token no navegador)"],
  ["zod", "4.4", "Schemas de validação (formulários e respostas de API)"],
  ["tailwindcss", "4.x", "Tokens DSGov via @theme inline; alto contraste e escala de fonte por data-* em <html>"],
  ["lucide-react", "1.33", "Ícones (decorativos com aria-hidden)"],
  ["Newsreader · Inter · Geist Mono", "Google Fonts", "Serifada editorial nos títulos, Inter na UI, mono em dados oficiais"],
];

const stackInfra = [
  ["PostgreSQL", "16-bookworm", "Dados, outbox, audit_logs append-only, rate_limit_buckets, busca full-text (tsvector + pg_trgm)"],
  ["RabbitMQ", "3.13-management", "Mensageria assíncrona, filas Dead-Letter"],
  ["MinIO", "latest", "Object storage S3-compatível"],
  ["Keycloak", "externo", "Broker OIDC Gov.br (consumido, nunca administrado — §29)"],
  ["Goose", "v3", "Migrations SQL versionadas (backend/migrations)"],
];

// ---------------------------------------------------------------------------
// 4. Ferramentas
// ---------------------------------------------------------------------------
const ferramentas = [
  ["govulncheck", "Vulnerabilidades conhecidas nas dependências Go (chamadas de fato alcançáveis)", "backend-ci", "sim"],
  ["staticcheck", "Análise estática: código morto, bugs sutis (SA*, U1000)", "backend-ci", "sim"],
  ["gosec", "Lint de segurança (path traversal, overflow, XSS via taint)", "backend-ci", "report-only até zerar o baseline"],
  ["go vet + go test -race -p 1", "Correção + corridas de dados; -p 1 pelos testes de integração que compartilham o banco", "backend-ci", "sim"],
  ["goose up → down-to 0 → up", "Prova que toda migration tem Down válido", "job migrations-reversibility", "sim"],
  ["eslint + eslint-plugin-jsx-a11y", "Lint + regras de acessibilidade e-MAG (via eslint-config-next)", "frontend-ci", "sim"],
  ["tsc --noEmit", "Type check estrito", "frontend-ci", "sim"],
  ["vitest / @testing-library", "Testes unitários de componentes e libs", "frontend-ci", "sim"],
  ["playwright", "Testes end-to-end (frontend/e2e)", "local / pipeline dedicada", "—"],
  ["docker compose + healthchecks", "Build das imagens de runtime e espera todos os serviços ficarem healthy", "job docker-build-healthcheck", "sim"],
];

// ---------------------------------------------------------------------------
// 5. Parâmetros de conformidade (o núcleo da página)
// ---------------------------------------------------------------------------
type Param = { key: string; value: string; layer: string; norm: string; how: string };
type ParamGroup = { icon: LucideIcon; group: string; intro: string; items: Param[] };

const paramGroups: ParamGroup[] = [
  {
    icon: Lock,
    group: "Segurança HTTP",
    intro:
      "Cabeçalhos defensivos e política de conteúdo. O CSP forte (com nonce por requisição) é do frontend; os cabeçalhos estáticos valem em toda resposta da API.",
    items: [
      {
        key: "Content-Security-Policy (páginas)",
        value: "nonce + strict-dynamic",
        layer: "frontend · proxy.ts",
        norm: "OWASP A03/A05 · SGD Módulo 1",
        how: "proxy.ts gera um nonce novo a cada requisição (base64 de crypto.randomUUID()), injeta no header CSP e num x-nonce interno que o Next.js usa para carimbar os próprios scripts. script-src usa nonce + strict-dynamic + wasm-unsafe-eval, sem unsafe-inline em produção. object-src 'none', base-uri 'self', form-action 'self', frame-ancestors 'none'.",
      },
      {
        key: "Content-Security-Policy (API)",
        value: "default-src 'none'",
        layer: "backend · SecurityHeaders",
        norm: "OWASP A05 · gap G-03",
        how: "Toda resposta da API emite default-src 'none'; frame-ancestors 'none'; base-uri 'none' — uma resposta JSON não executa nada, então a política mais restritiva é a correta. A rota /docs (Swagger UI) sobrescreve com uma política própria que libera o CDN.",
      },
      {
        key: "Strict-Transport-Security",
        value: "max-age=63072000; includeSubDomains",
        layer: "backend + frontend",
        norm: "OWASP A02/A05 · gap G-03",
        how: "Backend: 2 anos, incondicional (antes só saía com r.TLS != nil, nulo atrás de proxy TLS). Frontend (next.config.ts): max-age=31536000; includeSubDomains. O navegador ignora o header em HTTP puro, então enviar sempre é inofensivo.",
      },
      {
        key: "X-Frame-Options",
        value: "DENY",
        layer: "backend + frontend",
        norm: "OWASP A05",
        how: "Anti-clickjacking. No frontend é reforçado por frame-ancestors 'none' na CSP (mecanismo moderno equivalente).",
      },
      {
        key: "X-Content-Type-Options",
        value: "nosniff",
        layer: "backend + frontend",
        norm: "OWASP A05",
        how: "Impede o navegador de adivinhar um Content-Type diferente do declarado — vetor clássico de servir texto/imagem que vira script.",
      },
      {
        key: "Referrer-Policy",
        value: "no-referrer (API) · strict-origin-when-cross-origin (páginas)",
        layer: "backend / frontend",
        norm: "LGPD art. 6º · OWASP A05",
        how: "Evita vazar a URL completa (com ids de recurso no path) para destinos externos. Nas páginas, o referrer completo ainda vai entre páginas do mesmo site.",
      },
      {
        key: "Permissions-Policy",
        value: "camera=(), microphone=(), geolocation=(), usb=(), payment=(), interest-cohort=()",
        layer: "frontend · next.config.ts",
        norm: "OWASP A05",
        how: "Desliga explicitamente APIs de hardware/sensor que o painel nunca usa — um XSS que escapasse da CSP ainda esbarraria nisso.",
      },
      {
        key: "CORS",
        value: "1 origem + AllowCredentials",
        layer: "backend · httpserver.New",
        norm: "OWASP A05",
        how: "AllowedOrigins = só FRONTEND_URL; métodos GET/POST/PUT/PATCH/DELETE/OPTIONS; MaxAge 300s. O construtor dá panic se alguém tentar \"*\" com credenciais.",
      },
      {
        key: "Cache-Control",
        value: "no-store",
        layer: "backend · /auth/login, /auth/logout",
        norm: "OWASP A02",
        how: "As respostas que carregam bearer token nunca podem ficar em cache de disco do navegador nem em proxy/CDN intermediário.",
      },
    ],
  },
  {
    icon: KeyRound,
    group: "Autenticação",
    intro:
      "Dois caminhos independentes: OIDC Keycloak/Gov.br (SSO) e login local por usuário/senha (dev + conta de emergência). Chaves criptográficas nunca compartilhadas entre eles.",
    items: [
      {
        key: "minRSAKeyBits",
        value: "2048",
        layer: "backend · auth/local.go",
        norm: "OWASP A02 · ADR 003",
        how: "Piso do NIST SP 800-57 para assinatura RSA. NewLocalSigner recusa subir com uma chave menor — a aplicação falha no startup em vez de emitir tokens com assinatura fraca.",
      },
      {
        key: "Algoritmo do token local",
        value: "RS256",
        layer: "backend",
        norm: "Portaria SGD 2.154 · ADR 003",
        how: "Par de chaves RSA próprio deste subsistema, nunca o do Keycloak. iss = aud = projeto-aurora-local. Um token real do Keycloak é rejeitado aqui por assinatura.",
      },
      {
        key: "LOCAL_AUTH_TOKEN_TTL",
        value: "1h (default)",
        layer: "env · config.Load",
        norm: "OWASP A07",
        how: "Validade do token do login local. Curto de propósito: o login local não tem refresh token — uma sessão expirada exige logar de novo.",
      },
      {
        key: "bcrypt cost",
        value: "10",
        layer: "backend · seedadmin + comparação",
        norm: "OWASP A02/A07",
        how: "Fator de trabalho do hash de senha. O mesmo custo do dummyHash usado nos caminhos de rejeição, para que o tempo de comparação de conta bloqueada seja indistinguível de senha errada.",
      },
      {
        key: "maxFailedAttempts",
        value: "5",
        layer: "backend · localauth",
        norm: "OWASP A07",
        how: "Tentativas seguidas de senha errada antes do bloqueio temporário da conta. Defesa em profundidade além do rate limit por IP. Incremento atômico numa única instrução SQL.",
      },
      {
        key: "lockoutDuration",
        value: "15 min",
        layer: "backend · localauth",
        norm: "OWASP A07",
        how: "Duração do bloqueio ao atingir maxFailedAttempts — faixa recomendada pela OWASP. Zerado a cada login bem-sucedido.",
      },
      {
        key: "dummyHash",
        value: "defesa de timing",
        layer: "backend · localauth",
        norm: "OWASP A07",
        how: "Hash bcrypt válido de uma senha que não é de nenhuma conta. Rodado em todo caminho de rejeição para que a latência não vire um oráculo revelando qual caso ocorreu.",
      },
      {
        key: "Cookie de sessão (NextAuth)",
        value: "HttpOnly · SameSite=Lax · Secure*",
        layer: "frontend · options.ts",
        norm: "OWASP A02/A05 · §30",
        how: "Estratégia JWT: a sessão é um cookie criptografado no lado do servidor. Secure automático quando NEXTAUTH_URL é https. O access token bruto nunca chega ao objeto session exposto a Client Components.",
      },
      {
        key: "Fluxo OIDC Keycloak",
        value: "Authorization Code + PKCE + state",
        layer: "frontend · options.ts",
        norm: "Portaria SGD 2.154 · §30",
        how: "Checks padrão do KeycloakProvider. O refresh usa AbortSignal.timeout(5000); expires_in inválido é rejeitado (evita loop de refresh).",
      },
      {
        key: "Verificação do token no backend",
        value: "JWKS local, sem I/O por requisição",
        layer: "backend · auth/oidc.go",
        norm: "OWASP A01/A07 · e-PING",
        how: "Assinatura, iss e aud conferidos a cada requisição contra o JWKS em cache. A fronteira de autorização real é sempre o backend (RequirePermission).",
      },
      {
        key: "RBAC — roles",
        value: "aurora-user / -admin / -integration-manager / -auditor",
        layer: "backend · auth/rbac.go",
        norm: "OWASP A01 · gap G-10",
        how: "aurora-admin tem toda permissão implicitamente. aurora-user não tem nenhuma permissão de diretório — só se vê via /me. Listar terceiros exige aurora-auditor/-admin. feature_flags:manage e keycloak:manage são exclusivas do admin.",
      },
      {
        key: "Mascaramento PII em log",
        value: "MaskCPF / MaskEmail / MaskPhone",
        layer: "backend · logging",
        norm: "LGPD art. 6º",
        how: "Tipos PIICPF, PIIEmail, PIIPhone implementam slog.LogValuer: 123.***.***-45, u***r@dominio.com, (66) 9****-1234. SecretString vira [REDACTED].",
      },
    ],
  },
  {
    icon: Gauge,
    group: "Rate limiting",
    intro:
      "Janela fixa em PostgreSQL, compartilhada por todas as réplicas da API (a plataforma não usa Redis por decisão — §7). Cada limite tem um bucket próprio na tabela rate_limit_buckets.",
    items: [
      {
        key: "API_RATE_LIMIT_MAX",
        value: "600 (default)",
        layer: "env · bucket api_global",
        norm: "OWASP A04/A07 · gap G-01",
        how: "Teto por identidade autenticada (identity.Subject; fallback ClientIPKey) aplicado como middleware a todo o grupo /api/v1. Antes só login e /ws/ticket tinham teto.",
      },
      {
        key: "API_RATE_LIMIT_WINDOW_SECONDS",
        value: "60 (default)",
        layer: "env · bucket api_global",
        norm: "OWASP A04",
        how: "Tamanho da janela do api_global. Excedente → 429 com código RATE_LIMITED. Janela fixa: um cliente pode em teoria enviar até 2x o limite na fronteira entre janelas — trade-off aceito por ser um UPSERT atômico sem lock.",
      },
      {
        key: "bucket local_login",
        value: "5 / 60s por IP",
        layer: "backend · dependencies.go",
        norm: "OWASP A07",
        how: "Mais apertado de propósito — desacelera força bruta de senha sem travar um usuário que só errou uma ou duas vezes. Chave por IP porque não há identidade autenticada ainda.",
      },
      {
        key: "bucket ws_ticket",
        value: "5 / 5s",
        layer: "backend · dependencies.go",
        norm: "OWASP A04",
        how: "Emissão de tickets de WebSocket, chave pelo subject do usuário.",
      },
      {
        key: "Comportamento em erro do limiter",
        value: "fail-open",
        layer: "backend · RateLimit middleware",
        norm: "Resiliência · ADR 002",
        how: "Se o Postgres por trás do limiter fica brevemente inalcançável, a requisição passa e o erro é só logado — uma instabilidade do limiter não vira indisponibilidade do endpoint.",
      },
      {
        key: "Limpeza de janelas antigas",
        value: "a cada 5 min · janelas > 1h",
        layer: "worker · ratelimit.Cleanup",
        norm: "Operacional",
        how: "Impede a tabela de crescer indefinidamente com IPs/usuários que apareceram uma vez.",
      },
    ],
  },
  {
    icon: ScrollText,
    group: "Auditoria e observabilidade",
    intro: "Trilha imutável exigida pela LAI e pelo §49, mais a correlação de logs ponta a ponta.",
    items: [
      {
        key: "audit_logs",
        value: "append-only (triggers)",
        layer: "migration 000001",
        norm: "LAI · §49",
        how: "Gatilhos prevent_audit_logs_mutation e prevent_audit_logs_truncate recusam UPDATE/DELETE/TRUNCATE em nível de banco. Colunas: user_id, action, resource_type/id, metadata (JSONB), correlation_id, ip_address (inet), created_at (timestamptz, UTC).",
      },
      {
        key: "audit.FromRequest(r)",
        value: "ip + correlation_id",
        layer: "backend · audit/context.go",
        norm: "§49 · gap G-07/G-08",
        how: "Todo handler de mutação constrói a audit.Entry a partir deste helper — preenche IPAddress (de r.RemoteAddr; sanitizeIP tira a porta) e CorrelationID (de logging.CorrelationID). Aplicado a feature flags, config do Keycloak, login e logout.",
      },
      {
        key: "audit.NewWriter(tx)",
        value: "atomicidade",
        layer: "backend · módulos",
        norm: "§49 · gap G-06",
        how: "Quando a mutação roda em database.WithTx, a linha de auditoria é gravada com um Writer preso à mesma transação — item, evento de outbox e trilha commitam ou revertem juntos (blueprint example).",
      },
      {
        key: "X-Request-ID → correlation_id",
        value: "1 valor por fluxo",
        layer: "backend · middleware RequestID",
        norm: "§50 · e-PING",
        how: "Respeita um X-Request-ID de entrada ou gera um UUID; devolve no header e guarda no contexto. O mesmo valor vira o correlation_id, propagado por HTTP → Postgres → RabbitMQ → worker → WebSocket.",
      },
      {
        key: "Logger",
        value: "log/slog · JSON",
        layer: "backend · logging.New",
        norm: "§50",
        how: "service e environment presos em todo registro; request_id/correlation_id/user_id via FromContext. Formato text só para dev local (LOG_FORMAT).",
      },
      {
        key: "METRICS_SCRAPE_TOKEN",
        value: '"" (aberto) → Bearer',
        layer: "env · gap G-02",
        norm: "OWASP A05 · gap G-02",
        how: "Vazio: /metrics aberto (só aceitável em dev / rede interna). Definido: exige Authorization: Bearer <token>, comparado em tempo constante (crypto/subtle). Aceita METRICS_SCRAPE_TOKEN_FILE.",
      },
      {
        key: "Métricas Prometheus",
        value: "rótulo = padrão de rota",
        layer: "backend · middleware Metrics",
        norm: "§53",
        how: "nix_http_requests_total e nix_http_request_duration_seconds rotulados pelo padrão que o chi casou (/api/v1/users/{id}), nunca o path bruto — senão a cardinalidade explode com uma série por id.",
      },
      {
        key: "OpenTelemetry",
        value: "span por requisição, no-op sem config",
        layer: "backend · telemetry",
        norm: "§51",
        how: "otelhttp envolve cada requisição; propaga o contexto de trace nos headers de entrada até o worker. Vira no-op se OTEL_EXPORTER_OTLP_ENDPOINT não está definido.",
      },
    ],
  },
  {
    icon: Network,
    group: "Limites e timeouts",
    intro: "Defesas anti-DoS/Slowloris e limites de payload.",
    items: [
      {
        key: "MaxRequestBodyBytes",
        value: "1 MiB (1<<20)",
        layer: "backend · httputil.DecodeJSON",
        norm: "OWASP A04 · §57",
        how: "http.MaxBytesReader em todo corpo JSON; excedente → 400 request body too large. DisallowUnknownFields e rejeição de lixo após o objeto também.",
      },
      {
        key: "MAX_PAGE_SIZE",
        value: "100 (default)",
        layer: "env · paginação",
        norm: "OWASP A04 · §46",
        how: "Teto de page_size em toda listagem — pagination.New satura o valor pedido a este máximo.",
      },
      {
        key: "ReadHeaderTimeout",
        value: "5s",
        layer: "backend · http.Server",
        norm: "OWASP A04",
        how: "Tempo máximo para o cliente terminar de enviar os cabeçalhos — defesa direta contra Slowloris.",
      },
      {
        key: "ReadTimeout / WriteTimeout",
        value: "30s / 30s",
        layer: "backend · http.Server",
        norm: "OWASP A04",
        how: "Corpo de requisição e escrita de resposta.",
      },
      {
        key: "IdleTimeout",
        value: "120s",
        layer: "backend · http.Server",
        norm: "OWASP A04",
        how: "Conexões keep-alive ociosas.",
      },
      {
        key: "RequestTimeout (chi)",
        value: "30s · exceto WebSocket",
        layer: "backend · timeoutExceptWebSocket",
        norm: "OWASP A04",
        how: "Timeout por requisição no contexto do handler. O upgrade de /ws é pulado (fica aberto horas) — sob o timeout comum, cada conexão encerrada gerava um 504 com duração absurda no log.",
      },
      {
        key: "TRUSTED_PROXIES",
        value: '"" (nunca confia em XFF)',
        layer: "env · gap G-04",
        norm: "OWASP A04 · LGPD art. 8º · gap G-04",
        how: "Lista de CIDRs dos proxies reversos. httpserver.ClientIP só lê o X-Forwarded-For quando a conexão TCP vem de uma dessas faixas — senão usa só o RemoteAddr. Validado no boot (CIDR inválido = o processo recusa subir). lgpd.handleAccept usa isso para o IP da prova de consentimento.",
      },
      {
        key: "Health / readiness",
        value: "/health · /ready · +/livez /healthz /readyz",
        layer: "backend",
        norm: "§54 · gap G-14",
        how: "/health (liveness) nunca toca dependência externa — Postgres fora do ar não pode fazer o orquestrador matar um processo saudável. /ready checa Postgres + RabbitMQ + MinIO com timeout de 3s (503 se algum falhar). Aliases canônicos k8s adicionados.",
      },
    ],
  },
  {
    icon: Accessibility,
    group: "Acessibilidade e-MAG",
    intro:
      "Aplicados server-side antes do primeiro paint (atributos data-* em <html>), sem flash na hidratação.",
    items: [
      {
        key: "Teclas de acesso e-MAG",
        value: "Alt+1 / Alt+2 / Alt+3 / Alt+4",
        layer: "frontend · EMagAccessibilityBar",
        norm: "e-MAG 2.0 · WCAG 2.4.1",
        how: "Alt+1 → conteúdo (#conteudo), Alt+2 → menu (#menu), Alt+3 → busca (#busca, só quando há busca global), Alt+4 → rodapé (#rodape). Skip links invisíveis até receber foco por teclado.",
      },
      {
        key: "Escala de fonte",
        value: "90% – 130% · data-font-scale",
        layer: "frontend · BrandingProvider",
        norm: "e-MAG · WCAG 1.4.4",
        how: "Botões A+ / A- / A ajustam a escala do documento; o valor é persistido em cookie e reaplicado server-side. Tipografia toda em rem/em, então a escala afeta tudo proporcionalmente.",
      },
      {
        key: "Alto contraste",
        value: "paleta e-MAG preto/amarelo · data-high-contrast",
        layer: "frontend",
        norm: "e-MAG · WCAG 1.4.3/1.4.11",
        how: "Toggle na barra; sobrescreve os tokens de cor. Cookie BRANDING_COOKIE, lido no layout.tsx antes do render.",
      },
      {
        key: "Contraste dos tokens",
        value: "≥ 4.5:1 (texto) · documentado no CSS",
        layer: "frontend · globals.css",
        norm: "WCAG 1.4.3",
        how: "Ex.: --muted: #52606d ≈ 5.4:1 sobre surface-hover — folga sobre o mínimo AA, com a razão anotada no próprio arquivo.",
      },
      {
        key: "lang",
        value: "pt-BR",
        layer: "frontend · <html>",
        norm: "WCAG 3.1.1",
        how: "Idioma declarado no elemento raiz — leitores de tela usam a pronúncia correta.",
      },
      {
        key: "Campos de formulário",
        value: "<label htmlFor> + aria-invalid + aria-describedby",
        layer: "frontend · components/ui/Input",
        norm: "WCAG 1.3.1/3.3.2",
        how: "Todo Input associa rótulo e mensagem de erro (role=\"alert\") ao campo pelo id.",
      },
      {
        key: "VLibras",
        value: "player oficial, montado globalmente",
        layer: "frontend · providers.tsx",
        norm: "Lei 13.146/2015 · e-MAG",
        how: "Tradução automática de todo o conteúdo para Libras. Carregado de vlibras.gov.br/app; a exceção de CSP (style-src unsafe-inline) que ele exige está registrada no ADR 007.",
      },
      {
        key: "prefers-reduced-motion",
        value: "respeitado",
        layer: "frontend · globals.css",
        norm: "WCAG 2.3.3",
        how: "Animações de transição do shell e dos modais são suprimidas quando o usuário pede menos movimento.",
      },
    ],
  },
  {
    icon: FileCheck2,
    group: "LGPD",
    intro: "Consentimento registrável, minimização e a fundação dos direitos do titular.",
    items: [
      {
        key: "CurrentTermVersion",
        value: "v1.0.0-2026",
        layer: "backend · lgpd",
        norm: "LGPD art. 7º I / art. 9º",
        how: "Versão vigente dos Termos de Uso / Política de Privacidade. GET /api/v1/lgpd/status diz se o usuário já aceitou esta versão; POST /api/v1/lgpd/accept registra o aceite. Trocar a versão reobriga o aceite.",
      },
      {
        key: "user_consents",
        value: "user_id + term_version + ip + user_agent + accepted_at",
        layer: "migration 000002",
        norm: "LGPD art. 8º §1º",
        how: "UNIQUE(user_id, term_version). Um trigger AFTER INSERT espelha cada aceite em audit_logs (ação lgpd_consent_given) — a prova do consentimento fica também na trilha imutável.",
      },
      {
        key: "IP da prova de consentimento",
        value: "via httpserver.ClientIP",
        layer: "backend · lgpd.handleAccept",
        norm: "LGPD art. 8º · gap G-04",
        how: "Antes lia o X-Forwarded-For cru (spoofável). Agora só confia no XFF vindo de TRUSTED_PROXIES; senão usa o RemoteAddr direto.",
      },
      {
        key: "GET /api/v1/users",
        value: "UserListItem (sem e-mail)",
        layer: "backend · users/transport",
        norm: "LGPD art. 6º III · gap G-10",
        how: "A listagem devolve só id, username, display_name, active, created_at. O e-mail completo só aparece em GET /users/{id} (auditor/admin) e em /me (o próprio titular).",
      },
      {
        key: "data_subject_requests",
        value: "kind ∈ {export, erasure}",
        layer: "migration 000005",
        norm: "LGPD art. 18 / art. 19",
        how: "Fila + trilha do SLA legal das solicitações do art. 18. export = acesso + portabilidade; erasure = anonimização das colunas de PII (nunca um DELETE que quebre integridade referencial ou a trilha). Trigger de auditoria em toda abertura. Endpoints + worker: em andamento (Fase 3).",
      },
      {
        key: "Segredos em runtime",
        value: "AES-256-GCM · CONFIG_ENCRYPTION_KEY",
        layer: "backend · secretcrypto",
        norm: "LGPD art. 46 · OWASP A02",
        how: "O Client Secret do Keycloak configurado pela tela é cifrado antes de ir ao Postgres — um dump/réplica/backup do banco não expõe o segredo. Decifragem só em memória, nunca em SQL. Chave de 32 bytes base64; o backend recusa subir em produção com a chave de exemplo.",
      },
    ],
  },
];

// ---------------------------------------------------------------------------
// 6. Fluxo por camada
// ---------------------------------------------------------------------------
const camadas = [
  "proxy.ts (Edge) — gera o nonce da CSP, injeta os cabeçalhos de página, e redireciona a visita não autenticada a rota protegida para /login antes de qualquer render.",
  "Proxy BFF (/api/backend/[...path]) — lê o access token do cookie criptografado no servidor e anexa como Authorization: Bearer; propaga/gera o X-Request-ID. O token nunca chega ao JavaScript do navegador.",
  "Pilha de middlewares chi — RequestID → AccessLog → Recoverer → CORS → SecurityHeaders (CSP+HSTS) → timeoutExceptWebSocket(30s) → Metrics. Depois, no grupo /api/v1: RequireAuthentication (JWKS) → RateLimit(api_global) → Idempotency.",
  "Handler — httputil.DecodeJSON (1 MiB + campos desconhecidos rejeitados) → httputil.Validate (struct-tags) → RequirePermission(...) específico da rota.",
  "Serviço — database.WithTx: a mutação, o outbox.Write do evento e audit.NewWriter(tx).Record(...) (com IP + correlation de audit.FromRequest) commitam ou revertem juntos.",
  "PostgreSQL — gatilhos recusam UPDATE/DELETE/TRUNCATE em audit_logs; NOTIFY no mesmo commit acorda o dispatcher do outbox (latência ~zero). O worker consome o evento carregando o mesmo correlation_id.",
];

// ---------------------------------------------------------------------------
// Sub-componentes de render
// ---------------------------------------------------------------------------
function SimpleTable({
  caption,
  head,
  rows,
}: {
  caption: string;
  head: string[];
  rows: string[][];
}) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          {head.map((h) => (
            <TableHeaderCell key={h}>{h}</TableHeaderCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((cells, ri) => (
          <TableRow key={ri}>
            {cells.map((c, ci) => (
              <TableCell
                key={ci}
                className={
                  ci === 0
                    ? "align-top font-mono text-xs font-medium text-foreground"
                    : "align-top text-xs text-muted leading-relaxed"
                }
              >
                {c}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export default async function PadroesPage() {
  await connection();

  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-4xl flex-1 flex-col gap-12 px-6 py-12">
        {/* Cabeçalho */}
        <section className="flex flex-col gap-4">
          <div className="flex items-center gap-2 text-xs font-bold font-mono text-primary uppercase tracking-wider">
            <ListChecks size={16} className="text-success" />
            Padrões & Parâmetros de Conformidade
          </div>
          <h1 className="text-3xl font-bold text-foreground">
            Padrões, Ferramentas e Parâmetros do Projeto Aurora
          </h1>
          <p className="text-muted leading-relaxed">
            Documento de referência para a equipe, para auditoria (CGU/TCE) e para quem entra no
            projeto: cada <strong>norma seguida</strong>, cada <strong>parceiro oficial</strong>{" "}
            (Gov.br, VLibras, DSGov), a <strong>stack</strong>, as{" "}
            <strong>ferramentas de CI</strong> e <strong>cada parâmetro de configuração</strong> —
            com a norma que atende e como é aplicado. Complementa a página{" "}
            <Link href="/sobre" className="text-primary hover:underline">
              Sobre &amp; Conformidade
            </Link>
            .
          </p>
        </section>

        {/* 1. Normas */}
        <section className="flex flex-col gap-4">
          <div>
            <h2 className="text-xl font-semibold text-foreground">1. Normas que seguimos</h2>
            <p className="mt-1 text-sm text-muted">
              Os cinco módulos de conformidade da Secretaria de Governo Digital (SGD/MGI) mais os
              padrões transversais de segurança e interoperabilidade.
            </p>
          </div>
          <Table caption="Normas seguidas pelo Projeto Aurora, o que cada uma exige e onde é aplicada.">
            <TableHead>
              <TableRow>
                <TableHeaderCell>Norma</TableHeaderCell>
                <TableHeaderCell>O que exige</TableHeaderCell>
                <TableHeaderCell>Onde no Aurora</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {normas.map((n) => (
                <TableRow key={n.norma}>
                  <TableCell className="align-top">
                    <span className="block text-xs font-bold text-primary">{n.norma}</span>
                    <span className="block text-[11px] font-mono text-seal">{n.base}</span>
                  </TableCell>
                  <TableCell className="align-top text-xs text-muted leading-relaxed">
                    {n.exige}
                  </TableCell>
                  <TableCell className="align-top font-mono text-xs text-muted leading-relaxed">
                    {n.onde}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </section>

        {/* 2. Parceiros */}
        <section className="flex flex-col gap-4">
          <div>
            <h2 className="text-xl font-semibold text-foreground">
              2. Parceiros e órgãos oficiais
            </h2>
            <p className="mt-1 text-sm text-muted">
              Serviços e padrões mantidos pelo Governo Federal que a plataforma consome diretamente —
              nenhum é hospedado ou administrado pelo Aurora, só integrado. O projeto{" "}
              <strong>não usa agência externa de design</strong>: a identidade visual segue o
              GovBR-DS (DSGov) oficial.
            </p>
          </div>
          <Table caption="Parceiros e padrões oficiais do Governo Federal e como são integrados.">
            <TableHead>
              <TableRow>
                <TableHeaderCell>Parceiro / padrão</TableHeaderCell>
                <TableHeaderCell>Mantenedor</TableHeaderCell>
                <TableHeaderCell>Origem oficial</TableHeaderCell>
                <TableHeaderCell>Como é integrado</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {parceiros.map((p) => (
                <TableRow key={p.nome}>
                  <TableCell className="align-top text-xs font-bold text-primary leading-relaxed">
                    {p.nome}
                  </TableCell>
                  <TableCell className="align-top text-xs text-muted">{p.mantenedor}</TableCell>
                  <TableCell className="align-top font-mono text-[11px] text-seal">
                    {p.origem}
                  </TableCell>
                  <TableCell className="align-top text-xs text-muted leading-relaxed">
                    {p.integracao}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </section>

        {/* 3. Stack */}
        <section className="flex flex-col gap-6">
          <div>
            <h2 className="text-xl font-semibold text-foreground">3. Stack tecnológico</h2>
            <p className="mt-1 text-sm text-muted">
              Arquitetura Microkernel (plug-in): o kernel em <code>internal/platform</code> concentra
              infraestrutura, segurança e resiliência; os módulos de negócio em{" "}
              <code>internal/modules</code> são plug-ins isolados em Clean Architecture.
            </p>
          </div>
          <div className="flex flex-col gap-3">
            <h3 className="text-sm font-semibold text-foreground">Backend — Go 1.25.14</h3>
            <SimpleTable
              caption="Bibliotecas do backend Go, versão e papel."
              head={["Componente", "Versão", "Papel"]}
              rows={stackBackend}
            />
          </div>
          <div className="flex flex-col gap-3">
            <h3 className="text-sm font-semibold text-foreground">
              Frontend — Next.js 16.3.2 / React 19.2.8
            </h3>
            <SimpleTable
              caption="Bibliotecas do frontend, versão e papel."
              head={["Componente", "Versão", "Papel"]}
              rows={stackFrontend}
            />
          </div>
          <div className="flex flex-col gap-3">
            <h3 className="text-sm font-semibold text-foreground">Infraestrutura</h3>
            <SimpleTable
              caption="Serviços de infraestrutura, imagem e papel."
              head={["Serviço", "Imagem", "Papel"]}
              rows={stackInfra}
            />
          </div>
        </section>

        {/* 4. Ferramentas */}
        <section className="flex flex-col gap-4">
          <div>
            <h2 className="text-xl font-semibold text-foreground">
              4. Ferramentas de qualidade e segurança
            </h2>
            <p className="mt-1 text-sm text-muted">
              Rodam no CI (<code>.github/workflows/ci.yml</code>) a cada push e PR. &quot;Bloqueante&quot;
              = reprova o merge.
            </p>
          </div>
          <SimpleTable
            caption="Ferramentas de CI, o que fazem, onde rodam e se bloqueiam o merge."
            head={["Ferramenta", "O que faz", "Onde", "Bloqueante"]}
            rows={ferramentas}
          />
        </section>

        {/* 5. Parâmetros */}
        <section className="flex flex-col gap-8">
          <div>
            <h2 className="text-xl font-semibold text-foreground">5. Parâmetros de conformidade</h2>
            <p className="mt-1 text-sm text-muted">
              Cada parâmetro com o valor padrão, a camada onde vive, a norma que atende e uma
              explicação do mecanismo. Todos os parâmetros de ambiente aceitam o padrão{" "}
              <code>&lt;VAR&gt;_FILE</code> (Docker/K8s secrets).
            </p>
          </div>

          {paramGroups.map((g) => {
            const Icon = g.icon;
            return (
              <div key={g.group} className="flex flex-col gap-3">
                <div className="flex items-center gap-2.5">
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Icon size={16} aria-hidden="true" />
                  </span>
                  <h3 className="text-base font-semibold text-foreground">{g.group}</h3>
                </div>
                <p className="text-xs text-muted leading-relaxed">{g.intro}</p>
                <Table caption={`Parâmetros de ${g.group}: valor, camada, norma e como é usado.`}>
                  <TableHead>
                    <TableRow>
                      <TableHeaderCell>Parâmetro</TableHeaderCell>
                      <TableHeaderCell>Valor</TableHeaderCell>
                      <TableHeaderCell>Camada</TableHeaderCell>
                      <TableHeaderCell>Como é usado</TableHeaderCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {g.items.map((p) => (
                      <TableRow key={p.key}>
                        <TableCell className="align-top">
                          <span className="block font-mono text-xs font-semibold text-foreground break-words">
                            {p.key}
                          </span>
                          <span className="mt-1 block font-mono text-[10px] text-seal">
                            {p.norm}
                          </span>
                        </TableCell>
                        <TableCell className="align-top">
                          <span className="inline-block rounded border border-surface-border bg-primary/5 px-1.5 py-0.5 font-mono text-[11px] text-primary">
                            {p.value}
                          </span>
                        </TableCell>
                        <TableCell className="align-top font-mono text-[11px] text-muted">
                          {p.layer}
                        </TableCell>
                        <TableCell className="align-top text-xs text-muted leading-relaxed">
                          {p.how}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            );
          })}
        </section>

        {/* 6. Fluxo por camada */}
        <section className="flex flex-col gap-3 rounded-xl border border-surface-border bg-surface p-6">
          <div className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <ShieldCheck size={18} aria-hidden="true" />
            </span>
            <h2 className="text-lg font-semibold text-foreground">
              6. Como cada camada aplica o padrão
            </h2>
          </div>
          <p className="text-sm text-muted">
            O caminho de uma requisição autenticada de escrita, e o controle que cada etapa impõe.
          </p>
          <ol className="mt-1 flex flex-col gap-2.5">
            {camadas.map((step, i) => (
              <li
                key={i}
                className="grid grid-cols-[1.6rem_1fr] items-baseline gap-3 rounded-lg border border-surface-border bg-black/5 px-3 py-2.5 text-xs text-muted leading-relaxed dark:bg-white/5"
              >
                <span className="font-mono text-xs font-semibold text-primary">{i + 1}</span>
                <span>{step}</span>
              </li>
            ))}
          </ol>
        </section>

        <section className="flex flex-col gap-1 border-t border-surface-border pt-6 text-xs text-muted">
          <p>
            <Landmark size={13} className="mr-1.5 inline text-primary" aria-hidden="true" />
            Valores conferidos no código-fonte. Rastreamento por fase em{" "}
            <code>docs/ROADMAP_CONFORMIDADE.md</code>; decisões de arquitetura nos ADRs 005–007.
          </p>
          <p>Escopo: backend Go 1.25 (microkernel) + frontend Next.js 16 · Prefeitura Municipal de Rondonópolis.</p>
        </section>
      </div>
    </PublicShell>
  );
}
