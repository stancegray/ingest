.DEFAULT_GOAL := deploy

PORT ?= 8080
POSTGRES_PORT ?= 5433

# Server-friendly: support both "docker compose" (v2 plugin) and "docker-compose" (v1).
COMPOSE := $(shell \
	if docker compose version >/dev/null 2>&1; then \
		echo "docker compose"; \
	elif command -v docker-compose >/dev/null 2>&1; then \
		echo "docker-compose"; \
	else \
		echo "missing"; \
	fi \
)

.PHONY: deploy up down stop restart logs ps env keys dev run check install-deps

ifeq ($(COMPOSE),missing)
deploy up down stop restart logs ps dev postgres:
	@echo "Docker Compose not found."
	@echo ""
	@echo "On Ubuntu/Debian server, run:"
	@echo "  make install-deps"
	@echo ""
	@echo "Or install manually:"
	@echo "  sudo apt update && sudo apt install -y docker.io docker-compose-plugin curl"
	@echo "  sudo usermod -aG docker \$$USER && newgrp docker"
	@exit 1
endif

## Verify docker + compose are available
check:
	@command -v docker >/dev/null || (echo "Missing: docker" && exit 1)
	@$(COMPOSE) version >/dev/null || (echo "Missing: docker compose" && exit 1)
	@command -v curl >/dev/null || echo "Warning: curl not found (health wait will be skipped)"
	@echo "OK: $$(docker --version)"
	@echo "OK: $$($(COMPOSE) version | head -1)"

## Install Docker + Compose on Ubuntu/Debian (run on fresh server)
install-deps:
	@command -v apt-get >/dev/null || (echo "install-deps supports apt-based systems only" && exit 1)
	sudo apt-get update
	sudo apt-get install -y docker.io docker-compose-plugin curl
	sudo systemctl enable --now docker
	@echo ""
	@echo "Docker installed. If needed: sudo usermod -aG docker $$USER && newgrp docker"

## Start postgres + ingest (build if needed)
deploy up: check env keys
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
dev: check env keys postgres run

postgres:
	$(COMPOSE) up -d postgres
	@echo "Waiting for postgres..."
	@until $(COMPOSE) exec -T postgres pg_isready -U ingest -d ingest >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres ready on port $(POSTGRES_PORT)."

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
