# 📖 Guia do Desenvolvedor — Plataforma Projeto Aurora

Bem-vindo ao guia de desenvolvimento e arquitetura do **Projeto Aurora**. Este documento foi elaborado para capacitar engenheiros de software, arquitetos e equipes de TI municipais a desenvolver, manter e expandir aplicações corporativas sobre a base governamental padronizada.

---

## 🏛️ 1. Visão Geral e Arquitetura

O **Projeto Aurora** é uma plataforma corporativa modular construída segundo o conceito de **Monólito Modular & Clean Architecture**:

- **Backend (Go 1.25)**: Estruturado em camadas estritas de separação de responsabilidade:
  - `domain`: Entidades de negócio, interfaces de repositório e eventos.
  - `application`: Casos de uso e orquestração de serviços.
  - `infrastructure`: Persistência SQL no PostgreSQL e conectores de mensageria.
  - `transport`: Handlers REST e adaptadores de entrada HTTP chi.
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
- Toda mutação no banco dispara entradas na tabela imutável `audit_logs`.
- Administradores podem exportar o relatório de auditoria higienizado em formato CSV através da rota `/api/v1/audit/export` ou pelo botão **"Exportar LAI (CSV)"** na tela de Monitoramento da Plataforma.

---

## 🛠️ 5. Comandos Úteis do Makefile

| Comando | Descrição |
| :--- | :--- |
| `make dev` | Sobe toda a stack (Postgres, RabbitMQ, MinIO, Go API, Frontend) em modo desenvolvimento |
| `make build` | Compila o backend Go e gera o bundle de produção do Next.js |
| `make test` | Executa 100% das suítes de teste unitário do backend e frontend |
| `make lint` | Executa auditoria estática de código e acessibilidade (`jsx-a11y`) |
| `make migrate-up` | Executa as migrations de banco pendentes via Goose |
| `make seed-admin` | Cria ou reseta a conta do administrador local com senha segura |
