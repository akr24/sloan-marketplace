# Sloan Marketplace local development.
#
#   make db-up      start Postgres only (port 5434)
#   make up         build and start db + backend + frontend in Docker
#   make down       stop everything (data volume is kept)
#   make dev        run backend and frontend natively against the Docker db
#   make help       list every target

SHELL := /bin/bash
COMPOSE    := docker compose
COMPOSE_DB := docker compose -f docker-compose.db.yml

.DEFAULT_GOAL := help

## ---------- Docker ----------

.PHONY: env
env: .env ## Create .env from .env.example if it doesn't exist
	@:

.env: .env.example
	@test -f .env || { cp .env.example .env && echo "Created .env from .env.example; edit it to add SMTP or WhatsApp settings."; }

.PHONY: db-up
db-up: .env ## Start Postgres in Docker
	$(COMPOSE_DB) up -d --wait

.PHONY: db-down
db-down: ## Stop Postgres (keeps data)
	$(COMPOSE_DB) down

.PHONY: db-reset
db-reset: ## Stop Postgres and delete its data volume (re-runs migrations next start)
	$(COMPOSE_DB) down -v

.PHONY: psql
psql: ## Open a psql shell in the Postgres container
	$(COMPOSE_DB) exec db psql -U sloan -d sloan_marketplace

.PHONY: users
users: ## List accounts
	@$(COMPOSE_DB) exec db psql -U sloan -d sloan_marketplace -c "SELECT id, email, name, whatsapp_phone, created_at FROM users ORDER BY id"

.PHONY: delete-user
delete-user: ## Delete the account with EMAIL=... and all its listings
	@test -n "$(EMAIL)" || { echo "usage: make delete-user EMAIL=you@mit.edu"; exit 1; }
	@$(COMPOSE_DB) exec db psql -U sloan -d sloan_marketplace -c "DELETE FROM users WHERE email = lower('$(EMAIL)')" | sed 's/^DELETE 0$$/No account with that email./; s/^DELETE 1$$/Deleted. You can sign up again with the same email and number./'

.PHONY: build
build: ## Build backend and frontend images
	$(COMPOSE) build

.PHONY: up
up: .env ## Build and start db + backend + frontend
	$(COMPOSE) up -d --build --wait

.PHONY: down
down: ## Stop all containers (keeps data)
	$(COMPOSE) down

.PHONY: logs
logs: ## Tail logs from all containers
	$(COMPOSE) logs -f

.PHONY: ps
ps: ## Show container status
	$(COMPOSE) ps

## ---------- Native (faster iteration) ----------

export DATABASE_URL ?= postgres://sloan:sloan@localhost:5434/sloan_marketplace?sslmode=disable
export JWT_SECRET   ?= dev-only-secret-change-me-0123456789abcdef
export APP_ENV      ?= dev
export APP_URL      ?= http://localhost:5174
export PORT         ?= 8081
# Load optional settings (SMTP, WhatsApp) from .env for native runs too.
-include .env
export

.PHONY: backend
backend: ## Run the Go API natively on :8081 (needs `make db-up`)
	cd backend && go run ./cmd/server

.PHONY: frontend
frontend: ## Run the Vite dev server natively on :5174
	cd frontend && npm run dev

.PHONY: dev
dev: db-up ## Start db in Docker, then backend and frontend natively
	$(MAKE) -j2 backend frontend

.PHONY: install
install: ## Download Go modules and npm packages
	cd backend && go mod download
	cd frontend && npm install

.PHONY: test
test: ## Run backend tests and frontend lint
	cd backend && go test ./...
	cd frontend && npm run lint

## ---------- WhatsApp bot ----------

export FROM ?= 16175550123
export TEXT ?= help
.PHONY: bot
bot: ## Send TEXT to the bot as WhatsApp number FROM, without Meta (APP_ENV=dev)
	@curl -s -X POST localhost:$(PORT)/api/dev/whatsapp \
		-H 'Content-Type: application/json' \
		-d "$$(jq -n --arg f "$$FROM" --arg t "$$TEXT" '{from:$$f,text:$$t}')" | jq -r .reply

.PHONY: wa-link
wa-link: db-up ## Link the group bot's WhatsApp account (WHATSAPP_WEB_PHONE from .env)
	cd backend && go run ./cmd/walink

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
