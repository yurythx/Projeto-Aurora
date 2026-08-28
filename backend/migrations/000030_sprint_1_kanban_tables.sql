-- +goose Up
-- Sprint 1: Tabelas base para o Kanban de 6 etapas de Demandas Mensais (Projeto Nova)

-- Tipos de contrato afetam as regras de negócio
CREATE TYPE contract_type AS ENUM ('COMPRA_CONSUMO', 'SERVICOS_TERCEIRIZADOS', 'OBRAS');

-- Atualização na tabela de contratos (adiciona tipo)
ALTER TABLE contratos ADD COLUMN IF NOT EXISTS tipo_contrato contract_type DEFAULT 'COMPRA_CONSUMO';

-- Itens do Contrato (Para gerar a OS e OF detalhadas)
CREATE TABLE IF NOT EXISTS contract_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id   UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    descricao     TEXT NOT NULL,
    marca         TEXT,
    unidade       TEXT NOT NULL,
    quantidade    NUMERIC(15,3) NOT NULL,
    valor_unitario NUMERIC(15,2) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contract_items_contrato_id ON contract_items(contrato_id);

-- Fiscais do Contrato (Relaciona o usuário com o contrato de forma nominal)
CREATE TABLE IF NOT EXISTS contract_officials (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id    UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    user_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    nome           TEXT NOT NULL,
    matricula      TEXT NOT NULL,
    num_portaria   TEXT,
    is_titular     BOOLEAN DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contract_officials_contrato_id ON contract_officials(contrato_id);

-- O Card do Kanban: Demanda Mensal (Acompanhamento do ciclo de pagamento)
CREATE TABLE IF NOT EXISTS monthly_demands (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id    UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    ano_mes        TEXT NOT NULL, -- ex: '2026-08'
    etapa          INTEGER NOT NULL DEFAULT 1, -- 1 a 6 (Kanban Stages)
    status_etapa   TEXT NOT NULL DEFAULT 'pendente', -- pendente, aguardando_documento, concluida
    observacoes    TEXT,
    -- Controle de SLAs (IN 04/2021)
    etapa_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (contrato_id, ano_mes)
);
CREATE INDEX IF NOT EXISTS idx_monthly_demands_contrato_id ON monthly_demands(contrato_id);

-- Tipos de Documentos Exigidos pelas regras
CREATE TYPE document_type AS ENUM (
    'OF_PRE_EMPENHO', 'OFICIO_PLANEJAMENTO', 
    'EMPENHO_ASSINADO', 'COMPROVANTE_ENVIO',
    'NOTA_FISCAL', 'ORDEM_RECEPCAO',
    'EXTRATO_EMPENHO', 'RELATORIO_PAGAMENTO',
    'CERTIDAO_SIMPLES', 'CERTIDAO_CNDT', 'CERTIDAO_FGTS', 
    'CERTIDAO_MUNICIPAL', 'CERTIDAO_ESTADUAL', 'CERTIDAO_FEDERAL',
    'GUIA_DAM_ISSQN', 'PLANILHA_MEDICAO'
);

-- Documentos Anexados ao Card (Kanban)
CREATE TABLE IF NOT EXISTS demand_documents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    demanda_id     UUID NOT NULL REFERENCES monthly_demands(id) ON DELETE CASCADE,
    doc_type       document_type NOT NULL,
    file_path      TEXT NOT NULL, -- Caminho no MinIO
    file_name      TEXT NOT NULL,
    uploaded_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    validade_ate   DATE -- Importante para as certidões
);
CREATE INDEX IF NOT EXISTS idx_demand_documents_demanda_id ON demand_documents(demanda_id);

-- Ocorrências (Antingerência IN 04/2021)
CREATE TABLE IF NOT EXISTS contract_occurrences (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id    UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    demanda_id     UUID REFERENCES monthly_demands(id) ON DELETE CASCADE,
    tipo           TEXT NOT NULL, -- NOTIFICACAO_PREPOSTO, ADVERTENCIA
    descricao      TEXT NOT NULL,
    sla_vence_em   DATE,
    resolvida      BOOLEAN DEFAULT FALSE,
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Relatórios oficiais gerados pelo sistema
CREATE TABLE IF NOT EXISTS contract_reports (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id    UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    demanda_id     UUID REFERENCES monthly_demands(id) ON DELETE CASCADE,
    tipo_relatorio TEXT NOT NULL, -- MENSAL, QUADRIMESTRAL, FINAL
    file_path      TEXT NOT NULL,
    gerado_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS contract_reports;
DROP TABLE IF EXISTS contract_occurrences;
DROP TABLE IF EXISTS demand_documents;
DROP TYPE IF EXISTS document_type;
DROP TABLE IF EXISTS monthly_demands;
DROP TABLE IF EXISTS contract_officials;
DROP TABLE IF EXISTS contract_items;
ALTER TABLE contratos DROP COLUMN IF EXISTS tipo_contrato;
DROP TYPE IF EXISTS contract_type;
