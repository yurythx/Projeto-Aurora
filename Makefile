.PHONY: dev up down logs build test lint format \
	migrate-up migrate-down migrate-status migrate-redo seed-admin \
	backend-shell frontend-shell rabbitmq-status clean \
	backend-build backend-test backend-lint backend-sec backend-format \
	frontend-build frontend-test frontend-lint frontend-format

ifneq (,$(wildcard .env))
include .env
export
endif

COMPOSE      := docker compose -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_PROD := docker compose -f docker-compose.yml
GOOSE_DIR    := backend/migrations
DB_USER      ?= aurora
DB_PASSWORD  ?= aurora_pass
DB_NAME      ?= aurora
HOST_DB_PORT ?= 5433
DB_DSN       ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(HOST_DB_PORT)/$(DB_NAME)?sslmode=disable

## --- Orquestração local ---

dev: ## Sobe todos os serviços em modo desenvolvimento (com overrides de dev)
	$(COMPOSE) up --build

up: ## Sobe todos os serviços em modo produção
	$(COMPOSE_PROD) up --build -d

down: ## Para e remove todos os serviços
	$(COMPOSE) down

logs: ## Acompanha os logs de todos os serviços
	$(COMPOSE) logs -f

clean: ## Para os serviços e remove os volumes (DESTRÓI os dados locais)
	$(COMPOSE) down -v

## --- Build ---

build: backend-build frontend-build ## Compila os binários/artefatos do backend e do frontend

backend-build:
	cd backend && go build ./...

frontend-build:
	cd frontend && npm run build

## --- Testes ---

test: backend-test frontend-test ## Roda as suítes de teste do backend e do frontend

backend-test:
	cd backend && go test ./... -p 1

frontend-test:
	cd frontend && npm test

## --- Lint / format ---

lint: backend-lint frontend-lint ## Roda lint no backend e no frontend

backend-lint:
	cd backend && go vet ./...

backend-sec: ## SAST do backend (govulncheck + staticcheck + gosec) — mesmo conjunto do CI
	cd backend && govulncheck ./...
	cd backend && staticcheck ./...
	cd backend && gosec -quiet -exclude=G104 ./...

frontend-lint:
	cd frontend && npm run lint

format: backend-format frontend-format ## Formata o backend e o frontend

backend-format:
	cd backend && go fmt ./...

frontend-format:
	cd frontend && npm run format

## --- Migrations (Goose) ---

migrate-up: ## Aplica todas as migrations pendentes
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" up

migrate-down: ## Reverte a última migration
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" down

migrate-status: ## Mostra o status das migrations
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" status

migrate-redo: ## Exercita a reversibilidade: up -> down-to 0 -> up (toda migration precisa de Down válido)
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" up \
		&& goose postgres "$(DB_DSN)" down-to 0 \
		&& goose postgres "$(DB_DSN)" up \
		&& goose postgres "$(DB_DSN)" status

seed-admin: ## Cria/reseta o usuário admin local com senha ALEATÓRIA
	cd backend && DB_HOST=localhost DB_PORT=$(HOST_DB_PORT) DB_NAME=$(DB_NAME) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) \
		go run ./cmd/seedadmin

new-module: ## Gera o esqueleto Clean Architecture de um novo módulo (Uso: make new-module NAME=contratos)
	@./scripts/create-module.sh $(NAME)

## --- Shells ---

backend-shell: ## Abre um shell no container backend-api em execução
	$(COMPOSE) exec backend-api sh

frontend-shell: ## Abre um shell no container frontend em execução
	$(COMPOSE) exec frontend sh

## --- RabbitMQ ---

rabbitmq-status: ## Mostra o status do nó/filas do RabbitMQ
	$(COMPOSE) exec rabbitmq rabbitmqctl status
