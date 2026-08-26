SUPERSET_VERSION ?= 6.1.0
SUPERSET_PORT    ?= 8088
COMPOSE_FILE     := docker-compose/docker-compose.yml
COMPOSE_PROJECT  := superset-acc

COMPOSE := SUPERSET_VERSION=$(SUPERSET_VERSION) SUPERSET_PORT=$(SUPERSET_PORT) \
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT)

.DEFAULT_GOAL := help

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build the provider binary to dist/
	go build -o ./dist/

# ── Tests ─────────────────────────────────────────────────────────────────────

.PHONY: testacc
testacc: ## Run acceptance tests against Superset (starts/stops docker-compose automatically)
	./scripts/run-acc-tests.sh $(SUPERSET_VERSION) $(TESTARGS)

.PHONY: test-client
test-client: ## Run client unit tests only (no Superset needed)
	go test ./internal/client/ -v $(TESTARGS)

# ── Docker Compose ────────────────────────────────────────────────────────────

.PHONY: compose-pull
compose-pull: ## Pull Superset images for the configured version
	$(COMPOSE) pull

.PHONY: compose-up
compose-up: ## Start Superset stack (waits until healthy)
	$(COMPOSE) up -d --wait
	@echo ""
	@echo "Superset $(SUPERSET_VERSION) is running at http://localhost:$(SUPERSET_PORT)"
	@echo "Credentials: admin / admin"
	@echo "Stop with: make compose-down"

.PHONY: compose-down
compose-down: ## Stop and remove the Superset stack (volumes included)
	$(COMPOSE) down -v --remove-orphans

.PHONY: compose-logs
compose-logs: ## Tail logs from the running Superset stack
	$(COMPOSE) logs -f

.PHONY: compose-ps
compose-ps: ## Show status of the Superset stack containers
	$(COMPOSE) ps

.PHONY: compose-restart
compose-restart: ## Restart the Superset web container
	$(COMPOSE) restart superset

# ── Misc ──────────────────────────────────────────────────────────────────────

.PHONY: generate
generate: ## Regenerate provider documentation
	go generate ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run
