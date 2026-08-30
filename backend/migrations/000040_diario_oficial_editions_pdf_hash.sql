-- +goose Up
-- Idempotência estrita por hash binário do PDF (CLAUDE.md §1: "verify binary
-- SHA-256 hash").
--
-- O worker de ingestão baixa o PDF em stream, calcula o SHA-256 dos bytes
-- brutos e só grava o hash DEPOIS de SaveFindingsTx ter persistido os findings
-- com sucesso. Assim, hash presente ⟺ aqueles bytes exatos já foram
-- integralmente ingeridos: numa reprocessagem (edição re-enfileirada,
-- RequeueFailedEditions, reprocesso manual) o worker compara o hash recém
-- calculado com o armazenado e, se forem iguais, pula pdftotext + parser +
-- reindex e apenas confirma COMPLETED. Hash diferente = republicação com
-- conteúdo alterado: segue a re-ingestão normal (o DELETE-and-reinsert de
-- SaveFindingsTx já troca o conjunto inteiro) e o hash é atualizado.
--
-- CHAR(64): SHA-256 em hex minúsculo tem exatamente 64 caracteres. NULL =
-- edição ainda não ingerida com sucesso sob este mecanismo.
ALTER TABLE diario_oficial_editions
    ADD COLUMN IF NOT EXISTS pdf_sha256 CHAR(64);

-- Mesmo binário republicado sob outro edition_number: o índice permite achar a
-- edição gêmea por hash sem varrer a tabela.
CREATE INDEX IF NOT EXISTS idx_diario_editions_pdf_sha256
    ON diario_oficial_editions (pdf_sha256)
    WHERE pdf_sha256 IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_diario_editions_pdf_sha256;
ALTER TABLE diario_oficial_editions DROP COLUMN IF EXISTS pdf_sha256;
