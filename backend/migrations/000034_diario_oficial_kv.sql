-- +goose Up
-- Chave-valor pequeno do módulo diario_oficial. Uso atual: guardar o valor da
-- API key search-only do Typesense (o Typesense só devolve o valor completo
-- de uma key no momento da criação), para não expor a admin key no navegador.
CREATE TABLE IF NOT EXISTS diario_oficial_kv (
    k          TEXT PRIMARY KEY,
    v          TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS diario_oficial_kv;
