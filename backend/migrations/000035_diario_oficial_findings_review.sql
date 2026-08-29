-- +goose Up
-- Trilha de revisão manual dos findings de baixa confiança (fila em
-- /diario/revisao). Um revisor tem três desfechos para um finding
-- confidence='low':
--   promover  -> corrige os campos e sobe a confiança (sai da fila via confidence)
--   descartar -> DELETE da linha (ruído do parser)
--   manter    -> reconhece que é low mas legítimo; carimba reviewed_at
-- reviewed_at tira o finding "mantido" da fila sem mexer na confiança.
-- Reprocessar a edição apaga e reinsere os findings (SaveFindingsTx), então
-- estes carimbos somem no reprocessamento — correto: parse novo, revisão nova.
ALTER TABLE diario_oficial_findings
    ADD COLUMN IF NOT EXISTS reviewed_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reviewed_by  UUID,
    ADD COLUMN IF NOT EXISTS review_note  TEXT;

-- A fila lê exatamente por este predicado; índice parcial mantém a leitura O(1)
-- conforme a tabela cresce.
CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_review_pending
    ON diario_oficial_findings (created_at DESC)
    WHERE confidence = 'low' AND reviewed_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_diario_oficial_findings_review_pending;
ALTER TABLE diario_oficial_findings
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS review_note;
