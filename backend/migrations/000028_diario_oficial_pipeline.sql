-- +goose Up
-- Especificação de Engenharia: Tabela de controle de edições e achados/findings do Diário Oficial de Rondonópolis
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS diario_oficial_editions (
    id             BIGSERIAL PRIMARY KEY,
    edition_number VARCHAR(100) UNIQUE NOT NULL,
    edition_date   DATE NOT NULL,
    pdf_url        TEXT NOT NULL,
    status         VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    records_count  INTEGER DEFAULT 0,
    error_message  TEXT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_diario_oficial_editions_status ON diario_oficial_editions(status);
CREATE INDEX IF NOT EXISTS idx_diario_oficial_editions_date ON diario_oficial_editions(edition_date DESC);

CREATE TABLE IF NOT EXISTS diario_oficial_findings (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    edition_id    BIGINT NOT NULL REFERENCES diario_oficial_editions(id) ON DELETE CASCADE,
    external_id   VARCHAR(255) UNIQUE NOT NULL,
    act_type      VARCHAR(50) NOT NULL,
    servidor_nome VARCHAR(255) NULL,
    cpf           VARCHAR(14) NULL,
    matricula     VARCHAR(30) NULL,
    empresa_nome  VARCHAR(255) NULL,
    cnpj          VARCHAR(18) NULL,
    valor         NUMERIC(15,2) NULL,
    raw_content   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_servidor_trgm ON diario_oficial_findings USING gin(servidor_nome gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_empresa_trgm ON diario_oficial_findings USING gin(empresa_nome gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_fts ON diario_oficial_findings USING gin(to_tsvector('portuguese', raw_content));
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_cpf ON diario_oficial_findings(cpf) WHERE cpf IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_matricula ON diario_oficial_findings(matricula) WHERE matricula IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_cnpj ON diario_oficial_findings(cnpj) WHERE cnpj IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_edition_id ON diario_oficial_findings(edition_id);
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_act_type ON diario_oficial_findings(act_type);

-- +goose Down
-- DROP TABLE IF EXISTS diario_oficial_findings;
-- DROP TABLE IF EXISTS diario_oficial_editions;
