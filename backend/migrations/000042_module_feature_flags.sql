-- +goose Up
INSERT INTO feature_flags (key, enabled, description) VALUES
    ('module_exemplos_enabled', true, 'Habilita a exibição e uso do Módulo Modelo (Exemplo Blueprint).')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM feature_flags WHERE key = 'module_exemplos_enabled';
