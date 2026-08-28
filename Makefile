.DEFAULT_GOAL := deploy

COMPOSE := docker compose
PORT ?= 8080

.PHONY: deploy up down stop restart logs ps env keys dev run

## Start postgres + ingest (build if needed)
deploy up: env keys
	$(COMPOSE) up -d --build --remove-orphans
	@echo "Waiting for ingest..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -sf http://localhost:$(PORT)/health >/dev/null 2>&1 && break; \
		sleep 2; \
	done; \
	curl -sf http://localhost:$(PORT)/health >/dev/null 2>&1 || (echo "Ingest not ready yet — run: make logs" && exit 0)
	@echo ""
	@echo "Ingest running at http://localhost:$(PORT)"
	@echo "Health:       curl http://localhost:$(PORT)/health"
	@echo "Logs:         make logs"
	@echo "Stop:         make down"

## Postgres in docker, ingest on host (faster dev loop)
dev: env keys postgres run

postgres:
	$(COMPOSE) up -d postgres
	@echo "Waiting for postgres..."
	@until $(COMPOSE) exec -T postgres pg_isready -U ingest -d ingest >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres ready."

run:
	go run ./cmd/ingest

down stop:
	$(COMPOSE) down

restart: down deploy

logs:
	$(COMPOSE) logs -f ingest

ps:
	$(COMPOSE) ps

env:
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example")

keys:
	@test -f keys/public.pem || (go run ./cmd/keygen && echo "Generated keys/public.pem and keys/private.pem")
