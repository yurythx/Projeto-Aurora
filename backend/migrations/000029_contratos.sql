-- +goose Up
-- Módulo de Contratos Municipais (Projeto-Nova)
-- Tabela principal de contratos
CREATE TABLE IF NOT EXISTS contratos (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero               TEXT NOT NULL,
    objeto               TEXT NOT NULL,
    contratante          TEXT NOT NULL,
    contratado           TEXT NOT NULL,
    cnpj                 TEXT,
    valor                NUMERIC(15,2),
    data_assinatura      DATE,
    data_vigencia_inicio DATE,
    data_vigencia_fim    DATE,
    status               TEXT NOT NULL DEFAULT 'rascunho',
    -- Metadados de auditoria
    created_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- numero é único dentro do sistema (ex.: "001/2026")
CREATE UNIQUE INDEX IF NOT EXISTS idx_contratos_numero ON contratos(numero);
CREATE INDEX IF NOT EXISTS idx_contratos_status ON contratos(status);
CREATE INDEX IF NOT EXISTS idx_contratos_data_vigencia_fim ON contratos(data_vigencia_fim);
CREATE INDEX IF NOT EXISTS idx_contratos_contratado ON contratos(contratado);
-- Busca textual por objeto/contratado
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_contratos_objeto_trgm ON contratos USING gin(objeto gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_contratos_contratado_trgm ON contratos USING gin(contratado gin_trgm_ops);

-- Aditivos de contrato
CREATE TABLE IF NOT EXISTS contrato_aditivos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id     UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    numero          TEXT,
    objeto          TEXT,
    valor_adicional NUMERIC(15,2),
    data_assinatura DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contrato_aditivos_contrato_id ON contrato_aditivos(contrato_id);

-- Referências a publicações do Diário Oficial que originaram / citam o contrato
CREATE TABLE IF NOT EXISTS contrato_diario_refs (
    contrato_id       UUID NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    edition_number    TEXT NOT NULL,
    tipo_evento       TEXT,   -- ex.: CONTRATO, ADITIVO, RESCISAO, EXTRATO
    publicado_em      DATE,
    contexto          TEXT,   -- trecho relevante da publicação
    doc_url           TEXT,
    PRIMARY KEY (contrato_id, edition_number)
);

CREATE INDEX IF NOT EXISTS idx_contrato_diario_refs_contrato_id ON contrato_diario_refs(contrato_id);

-- +goose Down
DROP TABLE IF EXISTS contrato_diario_refs;
DROP TABLE IF EXISTS contrato_aditivos;
DROP TABLE IF EXISTS contratos;
