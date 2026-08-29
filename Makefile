.DEFAULT_GOAL := deploy

PORT ?= 8080
POSTGRES_PORT ?= 5433
INSTALL_DOCKER_SCRIPT := scripts/install-docker-ubuntu.sh
BOOTSTRAP_SCRIPT := scripts/bootstrap-server.sh

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

.PHONY: deploy up down stop restart logs ps env keys dev run check install install-deps install-docker bootstrap pull

ifeq ($(COMPOSE),missing)
deploy up down stop restart logs ps dev postgres:
	@echo "Docker Compose not found."
	@echo ""
	@echo "Run full server setup:"
	@echo "  GITHUB_TOKEN=ghp_... make bootstrap"
	@echo ""
	@echo "Or install Docker only:"
	@echo "  make install-deps"
	@echo ""
	@echo "Docs: https://docs.docker.com/engine/install/ubuntu/"
	@exit 1
endif

## Verify docker + compose are available
check:
	@command -v docker >/dev/null || (echo "Missing: docker" >&2; exit 1)
	@$(COMPOSE) version >/dev/null || (echo "Missing: docker compose" >&2; exit 1)
	@command -v curl >/dev/null || echo "Warning: curl not found (health wait will be skipped)"
	@echo "OK: $$(docker --version)"
	@echo "OK: $$($(COMPOSE) version | head -1)"

## Install base system packages (git, curl, make)
install:
	@test -f scripts/install-system.sh || (echo "Missing scripts/install-system.sh" >&2; exit 1)
	bash scripts/install-system.sh

## Install Docker Engine on Ubuntu via Docker apt repository (official guide)
install-deps install-docker:
	@test -f $(INSTALL_DOCKER_SCRIPT) || (echo "Missing $(INSTALL_DOCKER_SCRIPT)" >&2; exit 1)
	bash $(INSTALL_DOCKER_SCRIPT)

## Pull latest code from GitHub (requires GITHUB_TOKEN)
pull:
	@test -n "$(GITHUB_TOKEN)" || (echo "Usage: GITHUB_TOKEN=ghp_... make pull" >&2; exit 1)
	git remote set-url origin https://github.com/stancegray/ingest.git
	git pull https://x-access-token:$(GITHUB_TOKEN)@github.com/stancegray/ingest.git main

## Full server setup: git pull, Docker install, deploy
bootstrap:
	@test -f $(BOOTSTRAP_SCRIPT) || (echo "Missing $(BOOTSTRAP_SCRIPT)" >&2; exit 1)
	@GITHUB_TOKEN="$(GITHUB_TOKEN)" bash $(BOOTSTRAP_SCRIPT)

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
