-- +goose Up
-- Enriquece diario_oficial_findings com os campos estruturados que antes o
-- Service re-derivava do raw_content por regex a cada request (divergindo do
-- que ia pro Typesense), e corrige dois defeitos de dados já observados em
-- produção: act_type em vocabulário legado e edition_number com sufixo " (PDF)".

ALTER TABLE diario_oficial_findings
    ADD COLUMN IF NOT EXISTS secretaria      TEXT,
    ADD COLUMN IF NOT EXISTS job_role        TEXT,
    ADD COLUMN IF NOT EXISTS das_level       VARCHAR(12),
    ADD COLUMN IF NOT EXISTS portaria_number VARCHAR(60),
    ADD COLUMN IF NOT EXISTS pdf_page_number INTEGER NOT NULL DEFAULT 1;

-- Uma edição sem data de publicação confiável deve ficar NULL — a ordenação
-- cronológica usa `ORDER BY edition_date DESC NULLS LAST`, então NULL afunda
-- para o fim da lista em vez de now() jogando a edição para o topo.
ALTER TABLE diario_oficial_editions ALTER COLUMN edition_date DROP NOT NULL;

-- Remapeia act_type legado -> vocabulário canônico (internal/gazette/act_type.go).
UPDATE diario_oficial_findings SET act_type = 'NOMEACAO_COMISSIONADO'  WHERE act_type = 'NOMEACAO';
UPDATE diario_oficial_findings SET act_type = 'RELOTACAO'              WHERE act_type IN ('MUDANCA_SETOR', 'TRANSFERENCIA', 'REMANEJAMENTO');
UPDATE diario_oficial_findings SET act_type = 'CONTRATACAO_TEMPORARIA' WHERE act_type = 'CONTRATACAO';
UPDATE diario_oficial_findings SET act_type = 'DESIGNACAO_FUNCAO'      WHERE act_type = 'DESIGNACAO';

-- Limpa edition_number: "6265 (PDF)" -> "6265". edition_number é UNIQUE e todos
-- os valores atuais são "<n> (PDF)" distintos, então não há colisão.
UPDATE diario_oficial_editions
   SET edition_number = regexp_replace(edition_number, '^\s*(\d+).*$', '\1')
 WHERE edition_number ~ '^\s*\d+.*[^0-9]$';

CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_secretaria_trgm
    ON diario_oficial_findings USING gin(secretaria gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_created_at
    ON diario_oficial_findings(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_diario_oficial_findings_secretaria_trgm;
DROP INDEX IF EXISTS idx_diario_oficial_findings_created_at;

ALTER TABLE diario_oficial_findings
    DROP COLUMN IF EXISTS secretaria,
    DROP COLUMN IF EXISTS job_role,
    DROP COLUMN IF EXISTS das_level,
    DROP COLUMN IF EXISTS portaria_number,
    DROP COLUMN IF EXISTS pdf_page_number;

-- edition_date volta a NOT NULL só se não houver linhas NULL.
UPDATE diario_oficial_editions SET edition_date = created_at::date WHERE edition_date IS NULL;
ALTER TABLE diario_oficial_editions ALTER COLUMN edition_date SET NOT NULL;
