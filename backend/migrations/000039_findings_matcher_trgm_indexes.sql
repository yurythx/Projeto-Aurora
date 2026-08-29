-- +goose Up
-- O casador automático Contrato <-> Diário (worker contratos.diario_matcher)
-- roda MatchFindingsForContract por contrato elegível, com
--   raw_content    ILIKE '%'||numero||'%'
--   portaria_number ILIKE '%'||numero||'%'
-- Wildcard à esquerda não é indexável por btree -> seq scan em
-- diario_oficial_findings por contrato (N+1 num loop de 6h). pg_trgm já
-- está habilitado (000028); estes índices GIN trigram tornam o ILIKE
-- '%...%' indexável.
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_raw_trgm
    ON diario_oficial_findings USING gin (raw_content gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_portaria_trgm
    ON diario_oficial_findings USING gin (portaria_number gin_trgm_ops)
    WHERE portaria_number IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_diario_oficial_findings_portaria_trgm;
DROP INDEX IF EXISTS idx_diario_oficial_findings_raw_trgm;
