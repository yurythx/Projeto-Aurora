# Roadmap de Conformidade Governamental — Projeto Aurora

Origem: auditoria de conformidade de **2026-09-09** (e-MAG 2.0 / WCAG 2.1
AA, LGPD 13.709/2018, LAI 12.527/2011 + LC 131/2009, OWASP ASVS, SGD/MGI,
RFC 7807). Decisões de arquitetura em [`docs/adr/005`](adr/005-roadmap-conformidade-governamental.md),
[`006`](adr/006-rfc7807-problem-details.md), [`007`](adr/007-excecao-csp-style-src-vlibras.md).

Legenda: ✅ feito · 🟡 parcial / scaffold · ⬜ pendente · 🔒 depende de insumo externo

---

## Fase 0 — Fundação de processo

| # | Item | Estado |
|---|---|---|
| F0.1 | ADR obrigatório para contrato de API / RBAC / auditoria / middleware | ✅ ADRs 005–007; checklist no template de PR |
| F0.2 | Gates de CI: `govulncheck` + `staticcheck` + `gosec` (bloqueante) + `go vet` + `go test -race -p 1` | ✅ `.github/workflows/ci.yml`, `make backend-sec` |
| F0.3 | Checklist de conformidade no template de PR | ✅ `.github/pull_request_template.md` |
| F0.4 | Convenções de teste + migration (`up→down→up` no CI) | ✅ job `migrations-reversibility`, `make migrate-redo` |

## Fase 1 — Contenção dos riscos altos

| # | Gap | Item | Estado |
|---|---|---|---|
| F1.1 | G-01 | Rate limit global por identidade em `/api/v1` | ✅ `RateLimiters.APIGlobal`, env `API_RATE_LIMIT_*` |
| F1.2 | G-02 | `/metrics` exige bearer quando `METRICS_SCRAPE_TOKEN` definido | ✅ `requireMetricsToken` (tempo constante) |
| F1.3 | G-10 | Minimização de PII no diretório de usuários | ✅ `RoleUser` sem `PermUsersRead`; `UserListItem` sem e-mail — **breaking: confirmar com o produto** (ADR 005 §5.4) |
| F1.4 | G-07 | IP + `correlation_id` em toda `audit.Entry` de mutação | ✅ `audit.FromRequest` |
| F1.5 | G-08 | `logout` auditado | ✅ `POST /api/v1/auth/logout` + chamada no `signOut` do frontend |

## Fase 2 — Hardening HTTP e integridade da trilha

| # | Gap | Item | Estado |
|---|---|---|---|
| F2.1 | G-03 | CSP + HSTS incondicionais no backend + Swagger UI local | ✅ `SecurityHeaders`; ✅ `/docs` serve `swagger-ui-dist@5.17.14` vendorado (sem CDN, CSP `script-src 'self'`) |
| F2.2 | G-04 | Cadeia `X-Forwarded-For` confiável (`TRUSTED_PROXIES`) | ✅ `httpserver.ClientIP` |
| F2.3 | G-09 | Auto-auditoria do acesso à exportação LAI | ✅ `audit.exported` gravado a cada exportação |
| F2.4 | G-05 | Permissão `integrations:read` nas rotas de integrations | ✅ |
| F2.5 | G-06 | Padronizar o blueprint `example` | ✅ decode + validate + auditoria na tx |
| F2.6 | — | Export WORM da auditoria | ✅ `audit.WORMExporter` — bucket dedicado com **S3 Object Lock** (Compliance, `AUDIT_WORM_RETENTION_DAYS`) + cadeia de SHA-256 + migration 000007. Verificado: versão travada não pode ser sobrescrita/apagada |

## Fase 3 — Direitos do titular e transparência ativa

| # | Gap | Item | Estado |
|---|---|---|---|
| F3.1 | — | Página de Política de Privacidade + Termos versionada | 🟡 `/privacidade` publicada como **minuta**; 🔒 homologação do DPO |
| F3.2 | — | Endpoints LGPD art. 18 (`/lgpd/meus-dados`, `/solicitar-exclusao`, `/minhas-solicitacoes`) + worker de anonimização | ✅ migration 000005 + `lgpd.ErasureProcessor` |
| F3.3 | G-11 | Consentimento de visitante não autenticado | ✅ migration 000006 + `POST /lgpd/accept-anon` (público) + modal usa `useSession` |
| F3.4 | G-12 | Exportação de auditoria completa (`from/to`, paginação, JSON/XML) | ✅ |
| F3.5 | — | Endpoints de transparência ativa / dados abertos | 🟡 `internal/platform/transparency` (scaffold: datasets agregados sem PII); o órgão define os datasets reais |

## Fase 4 — Interoperabilidade e conformidade formal

| # | Gap | Item | Estado |
|---|---|---|---|
| F4.1 | G-13 | Respostas RFC 7807 (`application/problem+json`) | ✅ negociação por `Accept` (ADR 006, opção B); ✅ documentado no `openapi.yaml` 1.2.0 |
| F4.2 | G-14 | Aliases `/livez` `/healthz` `/readyz` | ✅ |
| F4.3 | G-15 | Skip links / landmarks / teclas de acesso | ✅ verificado — `#conteudo/#menu/#rodape` presentes e focáveis nos dois shells; `accessKey` 1–4 segue a convenção e-MAG (documentado na Declaração de Acessibilidade) |
| F4.4 | G-16 | Exceção de CSP `style-src 'unsafe-inline'` registrada | ✅ ADR 007 |
| F4.5 | — | Declaração formal de Acessibilidade + VPAT | ✅ seção "Declaração de Acessibilidade" em `/acessibilidade`; 🟡 `docs/VPAT.md` (autoavaliação WCAG 2.1 A/AA) — homologação depende de F5.2 |

## Fase 5 — Verificação externa e evidências

| # | Item | Estado |
|---|---|---|
| F5.1 | Pentest externo | 🔒 contratação externa — checklist em `docs/DOSSIE_CONFORMIDADE.md` §5 |
| F5.2 | Avaliação e-MAG por avaliador (ASES + manual + T.A.) | 🔒 avaliação externa |
| F5.3 | RIPD / DPIA | 🟡 `docs/RIPD_MODELO.md` (minuta técnica); 🔒 versão oficial pelo DPO |
| F5.4 | Dossiê de conformidade SGD/MGI | ✅ `docs/DOSSIE_CONFORMIDADE.md` (índice); itens de execução marcados nele |

---

## Parâmetros novos e como são usados

| Variável | Default | Efeito |
|---|---|---|
| `API_RATE_LIMIT_WINDOW_SECONDS` | `60` | Janela do rate limiter global de `/api/v1` |
| `API_RATE_LIMIT_MAX` | `600` | Requisições por identidade por janela; excedente → `429 RATE_LIMITED` |
| `METRICS_SCRAPE_TOKEN` | *(vazio)* | Vazio: `/metrics` aberto. Definido: exige `Authorization: Bearer <token>` |
| `TRUSTED_PROXIES` | *(vazio)* | CIDRs cujo `X-Forwarded-For` é confiável; fora deles, só `RemoteAddr` |
| `AUDIT_WORM_BUCKET` | `aurora-audit-worm` | Bucket dedicado (com Object Lock) da cópia WORM da auditoria |
| `AUDIT_WORM_RETENTION_DAYS` | `1825` | Retenção Compliance por objeto WORM (5 anos) |

Todos aceitam o padrão `<VAR>_FILE` (Docker/K8s secrets). Limiters
`anon_consent` (10/60s por IP) e `public_read` (60/60s por IP) são fixos.

## Endpoints novos

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| POST | `/api/v1/auth/logout` | sim | audita o fim de sessão |
| GET | `/api/v1/lgpd/meus-dados` | sim | pacote JSON de dados do titular (art. 18) |
| POST | `/api/v1/lgpd/solicitar-exclusao` | sim | pedido de anonimização |
| GET | `/api/v1/lgpd/minhas-solicitacoes` | sim | acompanhamento |
| POST | `/api/v1/lgpd/accept-anon` | **não** | consentimento de visitante (rate-limited por IP) |
| GET | `/api/v1/transparencia/{datasets,plataforma,integracoes,auditoria/acoes}` | **não** | dados abertos (JSON/CSV/XML) |
| GET | `/api/v1/audit/export?from=&to=&action=&format=&cursor=` | `audit:read` | trilha LAI completa; auto-auditada |
| GET | `/livez` `/healthz` `/readyz` | não | aliases de sonda |
