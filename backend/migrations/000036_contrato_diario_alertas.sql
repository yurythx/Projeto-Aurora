-- +goose Up
-- Dedup dos alertas do casador automático Contrato <-> Diário Oficial
-- (worker contratos.diario_matcher, roda a cada 6h). Sem isto, todo ciclo
-- reabriria a mesma notificação ("fiscal exonerado", "contrato vigente sem
-- publicação vinculada") e o sino do frontend viraria spam.
--
-- ref_key identifica a ocorrência concreta que gerou o alerta:
--   MOVIMENTACAO_PESSOAL -> "<edicao>|<servidor>|<act_type>"
--   SEM_VINCULO_DIARIO   -> "-"
-- Uma linha aqui = "este alerta já foi emitido para este contrato"; o
-- worker só publica o evento quando o INSERT ON CONFLICT DO NOTHING cria a
-- linha de fato.
CREATE TABLE IF NOT EXISTS contrato_diario_alertas (
    contrato_id uuid        NOT NULL REFERENCES contratos(id) ON DELETE CASCADE,
    kind        text        NOT NULL,
    ref_key     text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (contrato_id, kind, ref_key)
);

-- +goose Down
DROP TABLE IF EXISTS contrato_diario_alertas;
