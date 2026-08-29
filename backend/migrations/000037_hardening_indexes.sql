-- +goose Up
-- Índices de suporte às consultas adicionadas nesta fase.
--
-- audit_logs: o painel de histórico de uma demanda
-- (GET /demands/{id}/history -> audit.Reader.ListByResource) filtra por
-- (resource_type, resource_id). audit_logs é imutável e só cresce; sem
-- este índice cada abertura do modal fazia um seq scan da tabela inteira.
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource
    ON audit_logs (resource_type, resource_id, created_at DESC);

-- monthly_demands.etapa: usado pelo funil do dashboard (CountByEtapa,
-- GROUP BY etapa), pela varredura de SLA (ListStale: etapa < 6) e pelo
-- painel de certidões a vencer (JOIN ... WHERE m.etapa < 6).
CREATE INDEX IF NOT EXISTS idx_monthly_demands_etapa
    ON monthly_demands (etapa);

-- contract_occurrences: EnsureSLAOccurrence checa
-- (demanda_id, tipo, resolvida) e ListOpenOccurrences filtra resolvida=false.
CREATE INDEX IF NOT EXISTS idx_contract_occurrences_open
    ON contract_occurrences (demanda_id, tipo)
    WHERE resolvida = false;

-- +goose Down
DROP INDEX IF EXISTS idx_contract_occurrences_open;
DROP INDEX IF EXISTS idx_monthly_demands_etapa;
DROP INDEX IF EXISTS idx_audit_logs_resource;
