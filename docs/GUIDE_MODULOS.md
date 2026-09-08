# Guia de Desenvolvimento de Novos Módulos (Plug-ins) — Projeto Aurora

Este documento é o guia oficial para engenheiros que vão criar novos **módulos de negócio (plug-ins)** sobre o **Core System (Kernel)** da plataforma **Projeto Aurora**. Siga estas convenções para manter o isolamento de domínio, o desacoplamento e a consistência da **Arquitetura Microkernel**.

---

## 🏗️ Visão Geral da Arquitetura Microkernel

O Projeto Aurora utiliza o modelo **Microkernel (Plug-in Architecture)** combinado a **Camadas Limpas (Clean Architecture)** e comunicação assíncrona orientada a eventos via **Transactional Outbox**:

```
   Navegador (React / Next.js DSGov Shell)
        │  ▲
        │  │ (WebSocket Events)
        ▼  │
   BFF Proxy (/api/backend/*)
        │
        ▼
 ┌──────────────────────────────────────────────────────────┐
 │               CORE SYSTEM (KERNEL) — GO                  │
 │  (Auth OIDC, LGPD, Audit, Rate Limit, Outbox, WS Hub)    │
 └──────────────────────────┬───────────────────────────────┘
                            │ (Injeção de Dependências)
                            ▼
 ┌──────────────────────────────────────────────────────────┐
 │          PLUG-IN DE NEGÓCIO (ex.: patrimonio)            │
 │  Transport (Chi) -> Application -> Domain -> Postgres    │
 └──────────────────────────┬───────────────────────────────┘
                            │ (Transação ACID)
 ┌──────────────────────────┴───────────────────────────────┐
 │ PostgreSQL (Tabela do Módulo + Tabela Outbox)            │
 └──────────────────────────┬───────────────────────────────┘
                            │ (Outbox Worker)
                            ▼
                         RabbitMQ
                            │
                            ▼
                       WebSocket Hub -> Cliente
```

---

## ⚡ Scaffolding Automático de Módulos

Para criar a estrutura inicial de um novo módulo com 1 comando:

```bash
./scripts/create-module.sh <nome_do_modulo>
# Exemplo:
./scripts/create-module.sh financeiro
```

O script gera automaticamente:
1. Pastas do Go (`internal/modules/<nome>/domain`, `application`, `infrastructure`, `transport`).
2. Página do Next.js em `app/(protected)/<nome>/page.tsx`.
3. Migration SQL da Feature Flag do módulo em `backend/migrations/`.

---

## 📁 Estrutura de Pastas de um Módulo

Ao criar um novo módulo chamado `financeiro`, a estrutura deve seguir:

### Backend (`backend/internal/modules/financeiro/`)
```text
internal/modules/financeiro/
├── domain/            # Interfaces, Entidades e Erros de Domínio
│   ├── entity.go
│   └── repository.go
├── application/       # Use Cases, DTOs e Lógica de Aplicação
│   ├── dto.go
│   └── service.go
├── infrastructure/    # Implementação de Repositório PostgreSQL e Queries SQL
│   └── postgres_repo.go
└── transport/         # Handlers HTTP / Fiber / Chi e DTOs de Request/Response
    └── http_handler.go
```

### Frontend (`frontend/src/`)
```text
src/
├── app/(protected)/financeiro/      # Páginas Next.js (App Router)
│   └── page.tsx
├── components/financeiro/           # Componentes específicos do módulo
│   ├── FinanceiroList.tsx
│   └── FinanceiroModal.tsx
└── lib/validation/schemas.ts       # Schemas Zod para formulários e eventos WS
```

---

## 🛠️ Passo a Passo para Criar um Novo Módulo

### 1. Criar a Migration SQL (`backend/migrations/`)
Crie um novo arquivo de migration com numeração sequencial via `goose`:
```sql
-- 000042_create_financeiro_table.sql
-- +goose Up
CREATE TABLE IF NOT EXISTS financeiro_titulos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    descricao VARCHAR(255) NOT NULL,
    valor NUMERIC(15, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pendente',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS financeiro_titulos;
```

---

### 2. Definir a Entidade e Evento no Backend (`domain/`)
No backend, defina a struct do modelo de dados e o evento de domínio a ser publicado no Outbox:

```go
package domain

import (
	"time"
	"github.com/google/uuid"
)

type Titulo struct {
	ID        uuid.UUID `json:"id"`
	Descricao string    `json:"descricao"`
	Valor     float64   `json:"valor"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

const EventTituloCriado = "financeiro.titulo.created"
```

---

### 3. Registrar o Evento na Transação do Outbox (`infrastructure/`)
Ao salvar a entidade no banco de dados, insira a mensagem na tabela `outbox` **na mesma transação SQL** para garantir a propriedade ACID (*Exactly-Once delivery*):

```go
func (r *PostgresRepository) Create(ctx context.Context, t *domain.Titulo) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Insere o registro na tabela do módulo
	_, err = tx.Exec(ctx, `INSERT INTO financeiro_titulos (id, descricao, valor) VALUES ($1, $2, $3)`, t.ID, t.Descricao, t.Valor)
	if err != nil {
		return err
	}

	// 2. Registra o evento no Transactional Outbox
	payload, _ := json.Marshal(map[string]any{"id": t.ID, "descricao": t.Descricao})
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox (id, event_type, payload, status)
		VALUES ($1, $2, $3, 'pending')
	`, uuid.New(), domain.EventTituloCriado, payload)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
```

---

### 4. Conectar o Router HTTP (`internal/app/router.go`)
Registre as rotas REST do módulo no roteador principal:

```go
r.Route("/api/v1/financeiro", func(r chi.Router) {
    r.Use(auth.RequireAuthentication)
    r.Get("/", financeiroHandler.List)
    r.Post("/", financeiroHandler.Create)
})
```

---

### 5. Registrar a Feature Flag do Módulo (Controle em /configuracao)
Todo módulo deve possuir uma entrada em `feature_flags` para ser ativado/desativado pelos administradores no painel de configurações:

```sql
-- Migration SQL
INSERT INTO feature_flags (key, enabled, description) VALUES
    ('module_financeiro_enabled', true, 'Habilita a exibição e uso do Módulo Financeiro.')
ON CONFLICT (key) DO NOTHING;
```

No frontend (`Sidebar.tsx`), associe a flag ao item do menu:
```tsx
{ href: "/financeiro", label: "Financeiro", icon: DollarSign, flag: "module_financeiro_enabled" }
```
Quando o administrador desabilitar a flag no painel `/configuracao`, o item do menu desaparecerá automaticamente da interface.

---

### 6. Consumir e Notificar no Frontend (`Next.js`)
No frontend, adicione o manipulador do evento no `NotificationCenter.tsx` para exibir toasts e atualizar contadores em tempo real quando o evento WebSocket for recebido:

```tsx
// NotificationCenter.tsx
case "financeiro.titulo.created":
  showToast({
    title: "Novo Título Financeiro",
    description: `Título criado com sucesso: ${event.payload.descricao}`,
    variant: "success",
  });
  break;
```

---

## ⚡ Boas Práticas e Regras de Ouro

1. **Zero-Mock Policy:** Nunca utilize dados simulados hardcoded em telas de produção. Utilize fallbacks seguros com indicação de tela vazia (`EmptyState`).
2. **Resiliência CSP:** Nunca utilize atributos `style=""` inline. Utilize tokens de design do Tailwind CSS pré-configurados para conformidade com a política Content-Security-Policy com Nonce.
3. **Internacionalização e Acessibilidade (e-MAG):** Mantenha rótulos em português brasileiro (`pt-BR`) e garanta suporte a leitor de tela (`aria-label`, `role`, contraste).
