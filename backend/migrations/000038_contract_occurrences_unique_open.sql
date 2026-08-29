-- +goose Up
-- Torna único o índice parcial de pendências abertas: no máximo UMA
-- ocorrência não-resolvida por (demanda, tipo). Isso deixa EnsureSLAOccurrence
-- usar INSERT ... ON CONFLICT DO NOTHING (race-safe) em vez do
-- INSERT ... WHERE NOT EXISTS, que tem janela TOCTOU se duas replicas do
-- worker rodarem a varredura de SLA ao mesmo tempo.
DROP INDEX IF EXISTS idx_contract_occurrences_open;
CREATE UNIQUE INDEX IF NOT EXISTS idx_contract_occurrences_open
    ON contract_occurrences (demanda_id, tipo)
    WHERE resolvida = false;

-- +goose Down
DROP INDEX IF EXISTS idx_contract_occurrences_open;
CREATE INDEX IF NOT EXISTS idx_contract_occurrences_open
    ON contract_occurrences (demanda_id, tipo)
    WHERE resolvida = false;
