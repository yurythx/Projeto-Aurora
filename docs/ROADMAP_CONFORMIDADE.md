# Roadmap de Conformidade Governamental — Projeto Aurora

Origem: auditoria de conformidade de **2026-09-09** (e-MAG 2.0 / WCAG 2.1
AA, LGPD 13.709/2018, LAI 12.527/2011 + LC 131/2009, OWASP ASVS, SGD/MGI,
RFC 7807). Decisões de arquitetura em [`docs/adr/005`](adr/005-roadmap-conformidade-governamental.md),
[`006`](adr/006-rfc7807-problem-details.md), [`007`](adr/007-excecao-csp-style-src-vlibras.md).

Legenda: ✅ feito · 🟡 parcial / stub · ⬜ pendente · 🔒 depende de insumo externo

---

## Fase 0 — Fundação de processo

| # | Item | Estado |
|---|---|---|
| F0.1 | ADR obrigatório para contrato de API / RBAC / auditoria / middleware | ✅ ADRs 005–007; checklist no template de PR |
| F0.2 | Gates de CI: `govulncheck` + `staticcheck` + `gosec` + `go vet` + `go test -race -p 1` | ✅ `.github/workflows/ci.yml`, `make backend-sec` |
| F0.3 | Checklist de conformidade no template de PR | ✅ `.github/pull_request_template.md` |
| F0.4 | Convenções de teste + migration (`up→down→up` no CI) | ✅ job `migrations-reversibility`, `make migrate-redo`, guia atualizado |

## Fase 1 — Contenção dos riscos altos

| # | Gap | Item | Estado |
|---|---|---|---|
| F1.1 | G-01 | Rate limit global por identidade em `/api/v1` | ✅ `RateLimiters.APIGlobal`, env `API_RATE_LIMIT_*` |
| F1.2 | G-02 | `/metrics` exige bearer quando `METRICS_SCRAPE_TOKEN` definido | ✅ `requireMetricsToken` (tempo constante) |
| F1.3 | G-10 | Minimização de PII no diretório de usuários | ✅ `RoleUser` sem `PermUsersRead`; `UserListItem` sem e-mail |
| F1.4 | G-07 | IP + `correlation_id` em toda `audit.Entry` de mutação | ✅ `audit.FromRequest` em configflags / keycloakconfig / localauth |
| F1.5 | G-08 | `logout` auditado | ✅ `POST /api/v1/auth/logout` (backend); ⬜ chamada no `signOut` do frontend |

## Fase 2 — Hardening HTTP e integridade da trilha

| # | Gap | Item | Estado |
|---|---|---|---|
| F2.1 | G-03 | CSP + HSTS incondicionais no backend | ✅ `SecurityHeaders`; ⬜ servir Swagger UI localmente |
| F2.2 | G-04 | Cadeia `X-Forwarded-For` confiável (`TRUSTED_PROXIES`) | ✅ `httpserver.ClientIP`; `lgpd.handleAccept` migrado |
| F2.3 | G-09 | Auto-auditoria do acesso à exportação LAI | ⬜ |
| F2.4 | G-05 | Permissão `integrations:read` nas rotas de integrations | ✅ |
| F2.5 | G-06 | Padronizar o blueprint `example` (decode + validate + auditoria na tx) | ✅ |
| F2.6 | — | Export WORM da auditoria (MinIO object-lock) | ⬜ ADR 005 §5.6 |

## Fase 3 — Direitos do titular e transparência ativa

| # | Gap | Item | Estado |
|---|---|---|---|
| F3.1 | — | Página de Política de Privacidade + Termos versionada | 🔒 insumo DPO / jurídico |
| F3.2 | — | Endpoints LGPD art. 18 (`/lgpd/meus-dados`, `/lgpd/solicitar-exclusao`) | 🟡 migration `000005_data_subject_requests.sql` criada; endpoints + worker pendentes |
| F3.3 | G-11 | Consentimento de visitante não autenticado | ⬜ |
| F3.4 | G-12 | Exportação de auditoria completa (`from/to`, paginação, JSON/XML) | ⬜ |
| F3.5 | — | Endpoints de transparência ativa / dados abertos | 🔒 decisão de escopo (produto / jurídico) |

## Fase 4 — Interoperabilidade e conformidade formal

| # | Gap | Item | Estado |
|---|---|---|---|
| F4.1 | G-13 | Respostas RFC 7807 (`application/problem+json`) | 🔒 ADR 006 — aguarda decisão `/api/v2` vs negotiation |
| F4.2 | G-14 | Aliases `/livez` `/healthz` `/readyz` | ✅ |
| F4.3 | G-15 | Skip links / landmarks / teclas de acesso verificados com leitor de tela | ⬜ frontend |
| F4.4 | G-16 | Exceção de CSP `style-src 'unsafe-inline'` registrada | ✅ ADR 007 |
| F4.5 | — | Declaração formal de Acessibilidade + VPAT | ⬜ |

## Fase 5 — Verificação externa e evidências

| # | Item | Estado |
|---|---|---|
| F5.1 | Pentest externo (caixa-cinza, foco authz/RBAC/rate limit) | ⬜ |
| F5.2 | Avaliação e-MAG por avaliador (ASES + manual + T.A.) | ⬜ |
| F5.3 | RIPD / DPIA | ⬜ |
| F5.4 | Dossiê de conformidade SGD/MGI | ⬜ |

---

## Como cada parâmetro novo é usado

| Variável | Default | Efeito |
|---|---|---|
| `API_RATE_LIMIT_WINDOW_SECONDS` | `60` | Janela do rate limiter global de `/api/v1` |
| `API_RATE_LIMIT_MAX` | `600` | Requisições por identidade por janela; excedente → `429 RATE_LIMITED` |
| `METRICS_SCRAPE_TOKEN` | *(vazio)* | Vazio: `/metrics` aberto. Definido: exige `Authorization: Bearer <token>` |
| `TRUSTED_PROXIES` | *(vazio)* | CIDRs cujo `X-Forwarded-For` é confiável; fora deles, só `RemoteAddr` |

Todos os três aceitam o padrão `<VAR>_FILE` (Docker/K8s secrets).
