# Guia de Desenvolvimento de Novos Módulos — Projeto Aurora

Este documento é o guia oficial para engenheiros que vão criar novos módulos de negócio sobre a plataforma **Projeto Aurora**. Siga estas convenções para manter a modularidade, desacoplamento e consistência arquitetural.

---

## 🏗️ Visão Geral da Arquitetura

O Projeto Aurora utiliza uma arquitetura em **Camadas Limpas (Clean Architecture)** orientada a eventos via **Transactional Outbox**.

```
   Navegador (React / Next.js)
        │  ▲
        │  │ (WebSocket Events)
        ▼  │
   BFF Proxy (/api/backend/*)
        │
        ▼
   Backend Core (Go REST API)
        │
  ┌─────┴─────────────────────┐
  │ Handler -> Service -> Repo│
  └─────┬─────────────────────┘
        │ (Transação ACID)
  ┌─────┴─────────────────────┐
  │ PostgreSQL (Outbox Table) │
  └─────┬─────────────────────┘
        │ (Outbox Worker)
        ▼
     RabbitMQ
        │
        ▼
   WebSocket Hub -> Cliente
```

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
Crie um novo arquivo de migration com numeracao sequencial via `goose`:
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

### 5. Consumir e Notificar no Frontend (`Next.js`)
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
