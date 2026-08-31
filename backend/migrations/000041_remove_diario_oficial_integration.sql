-- +goose Up
DELETE FROM integrations WHERE key = 'diario-oficial';

INSERT INTO integrations (key, name, type, enabled, status) VALUES
    ('example-service', 'Serviço de Exemplo', 'example', true, 'online')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM integrations WHERE key = 'example-service';
INSERT INTO integrations (key, name, type) VALUES ('diario-oficial', 'Diário Oficial', 'diario_oficial') ON CONFLICT (key) DO NOTHING;
