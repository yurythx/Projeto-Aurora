#!/usr/bin/env bash
# ==============================================================================
# Script de Scaffolding para novos módulos no Projeto Aurora
# Uso: ./scripts/create-module.sh <nome-do-modulo>
# Exemplo: ./scripts/create-module.sh financeiro
# ==============================================================================

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Erro: Forneça o nome do módulo em caixa baixa (ex: financeiro)"
    echo "Uso: $0 <nome-do-modulo>"
    exit 1
fi

MODULE_NAME=$(echo "$1" | tr '[:upper:]' '[:lower:]' | tr '-' '_')
CAP_MODULE_NAME="$(tr '[:lower:]' '[:upper:]' <<< ${MODULE_NAME:0:1})${MODULE_NAME:1}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BACKEND_MOD_DIR="${ROOT_DIR}/backend/internal/modules/${MODULE_NAME}"
FRONTEND_PAGE_DIR="${ROOT_DIR}/frontend/src/app/(protected)/${MODULE_NAME}"
FRONTEND_COMP_DIR="${ROOT_DIR}/frontend/src/components/${MODULE_NAME}"
MIGRATION_DIR="${ROOT_DIR}/backend/migrations"

echo "🚀 Criando novo módulo '${MODULE_NAME}' no Projeto Aurora..."

# 1. Estrutura do Backend
mkdir -p "${BACKEND_MOD_DIR}/domain"
mkdir -p "${BACKEND_MOD_DIR}/application"
mkdir -p "${BACKEND_MOD_DIR}/infrastructure"
mkdir -p "${BACKEND_MOD_DIR}/transport"

cat <<EOF > "${BACKEND_MOD_DIR}/domain/entity.go"
package domain

import (
	"time"
	"github.com/google/uuid"
)

type ${CAP_MODULE_NAME}Item struct {
	ID          uuid.UUID \`json:"id"\`
	Nome        string    \`json:"nome"\`
	Descricao   string    \`json:"descricao"\`
	CreatedAt   time.Time \`json:"created_at"\`
}

const Event${CAP_MODULE_NAME}Created = "${MODULE_NAME}.created"
EOF

# 2. Estrutura do Frontend
mkdir -p "${FRONTEND_PAGE_DIR}"
mkdir -p "${FRONTEND_COMP_DIR}"

cat <<EOF > "${FRONTEND_PAGE_DIR}/page.tsx"
"use client";

export default function ${CAP_MODULE_NAME}Page() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Módulo ${CAP_MODULE_NAME}</h1>
        <p className="text-sm text-muted">Módulo gerado automaticamente pelo scaffolding do Projeto Aurora.</p>
      </div>
      <div className="p-6 bg-surface border border-surface-border rounded-xl">
        <p className="text-muted">Desenvolva a interface do módulo ${MODULE_NAME} aqui.</p>
      </div>
    </div>
  );
}
EOF

# 3. Gerar Migration da Feature Flag para o Módulo
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

# 4. Permissões de Execução
chmod +x "$0"

echo "✅ Módulo '${MODULE_NAME}' criado com sucesso!"
echo "   - Backend: ${BACKEND_MOD_DIR}"
echo "   - Frontend: ${FRONTEND_PAGE_DIR}"
echo "   - Migration Feature Flag: ${MIGRATION_FILE}"
