-- +goose Up
-- Nível de confiança da extração de cada finding. A extração por regex sobre
-- prosa não-estruturada tem teto de precisão; em vez de descartar o que é
-- duvidoso (perdendo auditoria) ou indexar tudo (poluindo a busca), cada
-- finding carrega seu grau de confiança:
--   high   -> bloco de Portaria com verbo reconhecido + nome plausível + CPF/matrícula/DAS
--   medium -> bloco de Portaria, nome plausível, sem identificador forte; ou contrato
--   low    -> varredura por linha, ou nome que mal passou nos filtros
-- A busca do usuário (Typesense) recebe só high+medium; low fica no PostgreSQL
-- para auditoria e reprocessamento.
ALTER TABLE diario_oficial_findings
    ADD COLUMN IF NOT EXISTS confidence VARCHAR(10) NOT NULL DEFAULT 'medium';

CREATE INDEX IF NOT EXISTS idx_diario_oficial_findings_confidence
    ON diario_oficial_findings(confidence);

-- +goose Down
DROP INDEX IF EXISTS idx_diario_oficial_findings_confidence;
ALTER TABLE diario_oficial_findings DROP COLUMN IF EXISTS confidence;
