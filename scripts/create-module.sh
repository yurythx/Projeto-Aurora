#!/usr/bin/env bash
# ==============================================================================
# Script de Scaffolding para novos módulos no Projeto Aurora (Clean Architecture)
# Uso: ./scripts/create-module.sh <nome-do-modulo>
# Exemplo: ./scripts/create-module.sh contratos
# ==============================================================================

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Erro: Forneça o nome do módulo em caixa baixa (ex: contratos)"
    echo "Uso: $0 <nome-do-modulo>"
    exit 1
fi

MODULE_NAME=$(echo "$1" | tr '[:upper:]' '[:lower:]' | tr '-' '_')
CAP_MODULE_NAME="$(tr '[:lower:]' '[:upper:]' <<< ${MODULE_NAME:0:1})${MODULE_NAME:1}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BACKEND_MOD_DIR="${ROOT_DIR}/backend/internal/modules/${MODULE_NAME}"
FRONTEND_PAGE_DIR="${ROOT_DIR}/frontend/src/app/(protected)/${MODULE_NAME}"
MIGRATION_DIR="${ROOT_DIR}/backend/migrations"

echo "🚀 Criando novo módulo '${MODULE_NAME}' no Projeto Aurora com Clean Architecture..."

# 1. Criar estrutura de diretórios do Backend Go
mkdir -p "${BACKEND_MOD_DIR}/domain"
mkdir -p "${BACKEND_MOD_DIR}/application"
mkdir -p "${BACKEND_MOD_DIR}/infrastructure"
mkdir -p "${BACKEND_MOD_DIR}/transport"

# 1a. Domain Entity
cat <<EOF > "${BACKEND_MOD_DIR}/domain/entity.go"
package domain

import (
	"time"
	"github.com/google/uuid"
)

type ${CAP_MODULE_NAME}Item struct {
	ID        uuid.UUID \`json:"id"\`
	Nome      string    \`json:"nome"\`
	Status    string    \`json:"status"\`
	CreatedAt time.Time \`json:"created_at"\`
	UpdatedAt time.Time \`json:"updated_at"\`
}

type Repository interface {
	Create(item *${CAP_MODULE_NAME}Item) error
	GetByID(id uuid.UUID) (*${CAP_MODULE_NAME}Item, error)
	List() ([]${CAP_MODULE_NAME}Item, error)
}
EOF

# 1b. Application Service
cat <<EOF > "${BACKEND_MOD_DIR}/application/service.go"
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-aurora/internal/modules/${MODULE_NAME}/domain"
)

type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, nome string) (*domain.${CAP_MODULE_NAME}Item, error) {
	item := &domain.${CAP_MODULE_NAME}Item{
		ID:        uuid.New(),
		Nome:      nome,
		Status:    "ativo",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(item); err != nil {
		return nil, fmt.Errorf("${MODULE_NAME}: create: %w", err)
	}
	return item, nil
}
EOF

# 1c. Infrastructure Postgres Repository
cat <<EOF > "${BACKEND_MOD_DIR}/infrastructure/postgres_repository.go"
package infrastructure

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
	"github.com/yurythx/projeto-aurora/internal/modules/${MODULE_NAME}/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(item *domain.${CAP_MODULE_NAME}Item) error {
	// Implementar escrita SQL no Postgres
	return nil
}

func (r *PostgresRepository) GetByID(id uuid.UUID) (*domain.${CAP_MODULE_NAME}Item, error) {
	return nil, nil
}

func (r *PostgresRepository) List() ([]domain.${CAP_MODULE_NAME}Item, error) {
	return []domain.${CAP_MODULE_NAME}Item{}, nil
}
EOF

# 1d. Transport HTTP Handlers
cat <<EOF > "${BACKEND_MOD_DIR}/transport/handlers.go"
package transport

import (
	"net/http"

	"github.com/yurythx/projeto-aurora/internal/modules/${MODULE_NAME}/application"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

type Handlers struct {
	service *application.Service
}

func NewHandlers(service *application.Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) RegisterRoutes(r interface {
	Get(path string, fn http.HandlerFunc)
	Post(path string, fn http.HandlerFunc)
}) {
	r.Get("/api/v1/${MODULE_NAME}", h.handleList)
}

func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	httputil.WriteOK(w, map[string]any{
		"module": "${MODULE_NAME}",
		"items":  []string{},
	})
}
EOF

# 2. Estrutura do Frontend React (Next.js App Router)
mkdir -p "${FRONTEND_PAGE_DIR}"

cat <<EOF > "${FRONTEND_PAGE_DIR}/page.tsx"
"use client";

import { Building2 } from "lucide-react";

export default function ${CAP_MODULE_NAME}Page() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
          <Building2 size={24} aria-hidden="true" />
        </span>
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">Módulo ${CAP_MODULE_NAME}</h1>
          <p className="text-sm text-muted">Módulo corporativo gerado pelo scaffolding do Projeto Aurora.</p>
        </div>
      </div>

      <div className="rounded-2xl border border-surface-border bg-surface p-6 shadow-xs">
        <p className="text-sm text-muted">
          Desenvolva os componentes e telas do módulo **${MODULE_NAME}** aqui.
        </p>
      </div>
    </div>
  );
}
EOF

# 3. Migration da Feature Flag para o Módulo
TIMESTAMP=$(date +%Y%m%d%H%M%S)
MIGRATION_FILE="${MIGRATION_DIR}/${TIMESTAMP}_add_module_${MODULE_NAME}_feature_flag.sql"

cat <<EOF > "${MIGRATION_FILE}"
-- +goose Up
INSERT INTO feature_flags (key, enabled, description) VALUES
    ('module_${MODULE_NAME}_enabled', true, 'Habilita a exibição e uso do Módulo ${CAP_MODULE_NAME}.')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM feature_flags WHERE key = 'module_${MODULE_NAME}_enabled';
EOF

chmod +x "$0"

echo "✅ Módulo '${MODULE_NAME}' gerado com sucesso!"
echo "   - Backend: ${BACKEND_MOD_DIR}"
echo "   - Frontend: ${FRONTEND_PAGE_DIR}"
echo "   - Migration Feature Flag: ${MIGRATION_FILE}"
