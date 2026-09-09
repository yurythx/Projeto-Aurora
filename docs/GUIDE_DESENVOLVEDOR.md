# 📖 Guia do Desenvolvedor — Plataforma Projeto Aurora

Bem-vindo ao guia de desenvolvimento e arquitetura do **Projeto Aurora**. Este documento foi elaborado para capacitar engenheiros de software, arquitetos e equipes de TI municipais a desenvolver, manter e expandir aplicações corporativas sobre a base governamental padronizada.

---

## 🏛️ 1. Visão Geral e Arquitetura

O **Projeto Aurora** é uma plataforma corporativa modular construída segundo o padrão de **Arquitetura Microkernel (Plug-in Architecture)** combinado a um **Monólito Modular & Clean Architecture**:

- **Core System / Kernel (`internal/platform/`)**: Infraestrutura central reutilizável que fornece os serviços fundamentais da plataforma:
  - Autenticação OIDC Gov.br / Local RSA.
  - Segurança HTTP (Headers OWASP, CSP Nonce, Rate Limiter).
  - Privacidade LGPD (Mascaramento PII em logs e consentimento).
  - Resiliência e Concorrência (Transactional Outbox, RabbitMQ com DLQ, Idempotência, Circuit Breaker).
  - Notificações em Tempo Real (WebSocket Hub com Auth via Tickets).
  - Auditoria Imutável (PostgreSQL Append-Only).
- **Plug-ins / Módulos de Negócio (`internal/modules/`)**: Componentes de domínio de negócio totalmente isolados e desacoplados:
  - Cada plug-in possui sua própria divisão Clean Architecture (`domain`, `application`, `infrastructure`, `transport`).
  - São acoplados ao Kernel via injeção de dependências e registrados dinamicamente no roteador com controle por Feature Flags (`feature_flags`).
- **Frontend (Next.js 16 App Router & React 19)**:
  - Componentes acessíveis em conformidade com o **e-MAG 2.0 / WCAG 2.1 AA**.
  - Estilização Tailwind CSS v4 com suporte White-Label dinâmico (`BrandingProvider`).
  - Suporte a acessibilidade nativa (VLibras, Alto Contraste, atalhos de teclado).

---

## ⚡ 2. Criando um Novo Módulo Corporativo

Para criar um novo módulo funcional (ex: *Módulo de Contratos* ou *Módulo de Protocolos*), utilize a automação do Makefile:

```bash
make new-module NAME=contratos
```

Este comando gera automaticamente:
1. A estrutura Clean Architecture Go em `backend/internal/modules/contratos/`.
2. A página React em `frontend/src/app/(protected)/contratos/page.tsx`.
3. A migration Goose de banco registrando a Feature Flag do novo módulo em `backend/migrations/`.

Após gerar o módulo, aplique a migration:
```bash
make migrate-up
```

---

## 🇧🇷 3. Normas de Conformidade Governamental

Todo código desenvolvido na plataforma deve obedecer estritamente aos 5 pilares de conformidade:

### 3.1. Acessibilidade Digital (e-MAG 2.0 / WCAG 2.1 AA)
- **Atalhos de Teclado**: Todo componente deve respeitar os marcadores de foco e âncoras e-MAG:
  - `Alt+1`: Conteúdo Principal (`#conteudo`)
  - `Alt+2`: Menu Principal (`#menu`)
  - `Alt+4`: Rodapé Institucional (`#rodape`)
- **Semântica HTML**: Utilize elementos semânticos (`<header>`, `<main>`, `<nav>`, `<footer>`) e atributos ARIA explicitados (`role="dialog"`, `aria-modal="true"`, `aria-label`).

### 3.2. Privacidade e LGPD (Lei 13.709/2018)
- **Mascaramento PII em Logs**: Nunca imprima CPF, e-mail ou telefone brutos em logs de aplicação. Utilize os envelopadores nativos do pacote `logging`:
  ```go
  logger.Info("registro acessado", slog.Any("cpf", logging.PIICPF(cpfBruto)))
  ```
- **Aceite de Termos**: O componente `<LGPDConsentModal />` verifica automaticamente o aceite formal dos termos pelo usuário e registra o evento auditável na tabela imutável `audit_logs`.

### 3.3. Autenticação e Níveis Gov.br (Portaria SGD/SEDGG Nº 2.154)
- O sistema aceita autenticação via OIDC Keycloak / Gov.br e Login Local.
- As identidades autenticadas contêm o nível de confiabilidade do Gov.br (`BRONZE`, `PRATA`, `OURO`):
  ```go
  identity, ok := auth.IdentityFromContext(ctx)
  if identity.HasGovBRLevelAtLeast(auth.GovBRLevelPrata) {
      // Permitir ação sensível
  }
  ```

---

## 📑 4. Transparência e Lei de Acesso à Informação (LAI)

Para auditorias e relatórios de controle interno (CGU/TCE):
- Toda mutação no banco dispara entradas na tabela imutável `audit_logs`, com **ator, IP de origem e `correlation_id`** (`audit.FromRequest`). Há ainda uma **cópia WORM diária** para o object storage, encadeada por SHA-256 (`audit.WORMExporter`).
- Exportação da trilha (permissão `audit:read`): `GET /api/v1/audit/export?from=&to=&action=&format=&cursor=` — `format` ∈ `csv|json|xml` (ou header `Accept`), `from/to` em `AAAA-MM-DD` ou RFC3339 (default: últimos 30 dias, teto 366), paginação keyset via `cursor` (`X-Next-Cursor`). Cada exportação é ela mesma auditada (`audit.exported`).
- **Transparência ativa** (LAI art. 8º), rotas **públicas** em `/api/v1/transparencia/*` (JSON/CSV/XML): `datasets` (dicionário), `plataforma`, `integracoes`, `auditoria/acoes`. Datasets agregados, sem PII — o órgão define os datasets reais (`internal/platform/transparency`).
- **Direitos do titular (LGPD art. 18)**: `GET /api/v1/lgpd/meus-dados` (acesso + portabilidade), `POST /api/v1/lgpd/solicitar-exclusao` (anonimização pelo worker), `GET /api/v1/lgpd/minhas-solicitacoes`.

---

## 🧭 4.1. Convenções de conformidade (roadmap SGD/MGI)

Ver `docs/ROADMAP_CONFORMIDADE.md` e `docs/adr/005`. Vale para todo PR:

### Trilha de auditoria em toda mutação
Todo handler de POST/PUT/PATCH/DELETE constrói a entrada a partir de
`audit.FromRequest(r)` — nunca uma `audit.Entry{}` literal — para que
`ip_address` e `correlation_id` sejam sempre preenchidos (§49):

```go
entry := audit.FromRequest(r)
entry.Action = "modulo.recurso.acao"
entry.ResourceType = "recurso"
entry.ResourceID = id.String()
entry.Metadata = map[string]any{...} // NUNCA segredo/senha/token aqui
_ = h.audit.Record(r.Context(), entry)
```

Quando a mutação roda dentro de `database.WithTx`, use `audit.NewWriter(tx)`
para que a linha de auditoria commite/reverta junto (ver o blueprint
`internal/modules/example`).

### RBAC — sempre testar o caminho negativo
Toda rota nova sob `/api/v1` que não seja auto-serviço (`/me`) passa por
`auth.RequirePermission(...)`. O teste tem que cobrir o **403 sem a
permissão**, não só o 200 com ela.

### Entrada
`httputil.DecodeJSON` + `httputil.Validate` (struct-tags
`validate:"required,max=..."`). `json.NewDecoder` cru é proibido — não
aplica limite de corpo nem rejeita campo desconhecido.

### Migrations
Toda migration tem `-- +goose Down`. Exercite antes de abrir o PR:
`make migrate-redo` (up → down-to 0 → up). O CI roda o mesmo.

### SAST local
`make backend-sec` roda `govulncheck` + `staticcheck` + `gosec` — o mesmo
conjunto que bloqueia o CI.

### Parâmetros de hardening
`API_RATE_LIMIT_WINDOW_SECONDS`, `API_RATE_LIMIT_MAX`,
`METRICS_SCRAPE_TOKEN`, `TRUSTED_PROXIES` — ver `.env.example` e a tabela
em `docs/ROADMAP_CONFORMIDADE.md`.

---

## 🛠️ 5. Comandos Úteis do Makefile

| Comando | Descrição |
| :--- | :--- |
| `make dev` | Sobe toda a stack (Postgres, RabbitMQ, MinIO, Go API, Frontend) em modo desenvolvimento |
| `make build` | Compila o backend Go e gera o bundle de produção do Next.js |
| `make test` | Executa 100% das suítes de teste unitário do backend e frontend |
| `make lint` | Executa auditoria estática de código e acessibilidade (`jsx-a11y`) |
| `make backend-sec` | SAST do backend: `govulncheck` + `staticcheck` + `gosec` (mesmo do CI) |
| `make migrate-up` | Executa as migrations de banco pendentes via Goose |
| `make migrate-redo` | Testa a reversibilidade: up → down-to 0 → up |
| `make seed-admin` | Cria ou reseta a conta do administrador local com senha segura |
