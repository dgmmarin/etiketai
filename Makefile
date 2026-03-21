COMPOSE = docker compose -f infra/docker-compose.yml

# ── Start / Stop ──────────────────────────────────────────────────────────────

.PHONY: start
start:
	$(COMPOSE) up -d --build
	@echo "Waiting for databases to be healthy..."
	@$(COMPOSE) exec auth-db      sh -c 'until pg_isready -U postgres; do sleep 1; done' 2>/dev/null
	@$(COMPOSE) exec workspace-db sh -c 'until pg_isready -U postgres; do sleep 1; done' 2>/dev/null
	@$(COMPOSE) exec label-db     sh -c 'until pg_isready -U postgres; do sleep 1; done' 2>/dev/null
	@$(COMPOSE) exec agent-db     sh -c 'until pg_isready -U postgres; do sleep 1; done' 2>/dev/null
	@$(COMPOSE) exec print-db     sh -c 'until pg_isready -U postgres; do sleep 1; done' 2>/dev/null
	@$(MAKE) migrate
	@echo ""
	@echo "✓ API Gateway: http://localhost:8080"
	@echo "✓ MinIO console: http://localhost:9001  (minioadmin / minioadmin123)"

.PHONY: stop
stop:
	$(COMPOSE) down

.PHONY: restart
restart: stop start

# ── Database migrations ───────────────────────────────────────────────────────

.PHONY: migrate
migrate:
	@echo "Running migrations..."
	@$(COMPOSE) exec -T auth-db      psql -U postgres -d auth_db      -v ON_ERROR_STOP=0 -f /dev/stdin < migrations/auth-svc/000001_init.up.sql      2>/dev/null; true
	@$(COMPOSE) exec -T auth-db      psql -U postgres -d auth_db      -v ON_ERROR_STOP=0 -f /dev/stdin < migrations/auth-svc/000002_superadmin.up.sql  2>/dev/null; true
	@$(COMPOSE) exec -T workspace-db psql -U postgres -d workspace_db -v ON_ERROR_STOP=0 -f /dev/stdin < migrations/workspace-svc/000001_init.up.sql 2>/dev/null; true
	@$(COMPOSE) exec -T label-db     psql -U postgres -d label_db     -v ON_ERROR_STOP=0 -f /dev/stdin < migrations/label-svc/000001_init.up.sql     2>/dev/null; true
	@$(COMPOSE) exec -T agent-db     psql -U postgres -d agent_db     -v ON_ERROR_STOP=0 -f /dev/stdin < migrations/agent-svc/000001_init.up.sql     2>/dev/null; true
	@$(COMPOSE) exec -T print-db     psql -U postgres -d print_db     -v ON_ERROR_STOP=0 -f /dev/stdin < migrations/print-svc/000001_init.up.sql     2>/dev/null; true
	@echo "Migrations done."

# ── Seed dev data ────────────────────────────────────────────────────────────

.PHONY: seed
seed:
	@bash scripts/seed-dev.sh

# ── Logs ─────────────────────────────────────────────────────────────────────

.PHONY: logs
logs:
	$(COMPOSE) logs -f --tail=50

.PHONY: logs-gw
logs-gw:
	$(COMPOSE) logs -f --tail=50 api-gateway

# ── Status ────────────────────────────────────────────────────────────────────

.PHONY: ps
ps:
	$(COMPOSE) ps

# ── API Docs (Swagger UI) ─────────────────────────────────────────────────────

.PHONY: docs
docs:
	@echo "Swagger UI → http://localhost:8090"
	@docker run --rm -p 8090:8080 \
	  -e SWAGGER_JSON=/spec/openapi.yaml \
	  -v $(PWD)/docs:/spec \
	  swaggerapi/swagger-ui

# ── Web dashboard ─────────────────────────────────────────────────────────────

.PHONY: web
web:
	cd web && npm run dev

.PHONY: web-build
web-build:
	cd web && npm run build

# ── Nuke everything (including volumes) ──────────────────────────────────────

.PHONY: destroy
destroy:
	$(COMPOSE) down -v
