-- +goose Up
-- Retry automático de edições que falharam na ingestão. Antes, uma edição que
-- ia para FAILED (ex.: bytes UTF-8 inválidos vindos do pdftotext) ficava presa
-- para sempre: o watcher só reprocessa status='PENDING'. Agora o watcher
-- reencaminha FAILED -> PENDING até retry_count atingir o limite.
ALTER TABLE diario_oficial_editions
    ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_diario_oficial_editions_status_retry
    ON diario_oficial_editions(status, retry_count);

-- +goose Down
DROP INDEX IF EXISTS idx_diario_oficial_editions_status_retry;
ALTER TABLE diario_oficial_editions DROP COLUMN IF EXISTS retry_count;
