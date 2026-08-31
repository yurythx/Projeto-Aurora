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
import { Header } from "@/components/layout/Header";

export default function ${CAP_MODULE_NAME}Page() {
  return (
    <div className="space-y-6">
      <Header
        title="Módulo ${CAP_MODULE_NAME}"
        description="Módulo gerado automaticamente pelo scaffolding do Projeto Aurora."
      />
      <div className="p-6 bg-surface border border-surface-border rounded-xl">
        <p className="text-muted">Desenvolva a interface do módulo ${MODULE_NAME} aqui.</p>
      </div>
    </div>
  );
}
EOF

# 3. Permissões de Execução
chmod +x "$0"

echo "✅ Módulo '${MODULE_NAME}' criado com sucesso!"
echo "   - Backend: ${BACKEND_MOD_DIR}"
echo "   - Frontend: ${FRONTEND_PAGE_DIR}"
