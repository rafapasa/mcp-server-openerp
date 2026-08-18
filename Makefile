# ================================================================
# Makefile - MCP Server Openerp
# ================================================================

DOCKER_USERNAME := rafapasa
IMAGE_TAG := latest
PLATFORM := linux/arm64
DOCKERFILE := Dockerfile.multistage

# ================================================================
# DESENVOLVIMENTO LOCAL
# ================================================================

.PHONY: build build-stdio build-http build-wh build-api
build: build-stdio build-http build-wh build-api

build-stdio:
	go build -o bin/stdio ./cmd/stdio

build-http:
	go build -o bin/http ./cmd/http

build-wh:
	go build -o bin/webhook ./cmd/webhook

build-api:
	go build -o bin/api ./cmd/api

.PHONY: run-stdio run-http run-wh run-api
run-stdio: ; go run ./cmd/stdio
run-http: ; go run ./cmd/http
run-wh: ; go run ./cmd/webhook
run-api: ; go run ./cmd/api

.PHONY: dev
dev:
	air

.PHONY: test fmt lint
test:
	go test -v ./...

fmt:
	gofmt -w -s .
	goimports -w .

lint:
	golangci-lint run ./...

.PHONY: deps install-tools
deps:
	go mod download && go mod tidy

install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest

.PHONY: clean
clean:
	rm -rf bin/ tmp/
	go clean

# ================================================================
# DOCKER BUILD E PUSH (MULTI-ARQUITETURA)
# ================================================================

.PHONY: login
login:
	docker login

.PHONY: docker-build-webhook
docker-build-webhook:
	docker buildx build --platform $(PLATFORM) -f $(DOCKERFILE) --target webhook -t $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) .

.PHONY: docker-build-api
docker-build-api:
	docker buildx build --platform $(PLATFORM) -f $(DOCKERFILE) --target api -t $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG) .

.PHONY: docker-build-all
docker-build-all: docker-build-webhook docker-build-api

.PHONY: docker-push-webhook
docker-push-webhook:
	docker push $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)

.PHONY: docker-push-api
docker-push-api:
	docker push $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)

.PHONY: docker-push-all
docker-push-all: docker-push-webhook docker-push-api

.PHONY: docker-build-push
docker-build-push: login docker-build-all docker-push-all
	@echo "✅ Imagens publicadas: $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) e $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)"

# ================================================================
# 🚀 DEPLOY NA OCI
# ================================================================

# --- Banco de Dados (MySQL + Redis) ---
# Executar UMA ÚNICA VEZ, ou quando quiser resetar o banco

.PHONY: oci-db-up oci-db-down oci-db-logs
oci-db-up:
	@echo "🐳 Iniciando MySQL e Redis (uma vez)..."
	docker-compose -f docker-compose.database.yml up -d
	@echo "✅ Banco de dados em execução"
	@echo "   MySQL: mcp-mysql:3306"
	@echo "   Redis: mcp-redis:6379"

oci-db-down:
	docker-compose -f docker-compose.database.yml down

oci-db-logs:
	docker-compose -f docker-compose.database.yml logs -f

# --- Aplicação (API + Webhook) ---
# Executar SEMPRE que atualizar as imagens

.PHONY: oci-pull oci-app-up oci-app-down oci-app-restart oci-app-logs
oci-pull:
	@echo "📥 Baixando imagens do Docker Hub..."
	docker pull $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)
	docker pull $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)
	@echo "✅ Imagens baixadas"

oci-app-up:
	@echo "🐳 Iniciando API e Webhook..."
	docker-compose -f docker-compose.app.yml up -d
	@echo "✅ Aplicação em execução"
	@echo "   API:      http://localhost:8081"
	@echo "   Webhook:  http://localhost:8080"

oci-app-down:
	docker-compose -f docker-compose.app.yml down

oci-app-restart: oci-app-down oci-pull oci-app-up

oci-app-logs:
	docker-compose -f docker-compose.app.yml logs -f

# --- Deploy Completo (Banco + Aplicação) ---

.PHONY: oci-deploy
oci-deploy: oci-db-up oci-pull oci-app-up
	@echo "============================================"
	@echo "  ✅ DEPLOY COMPLETO FINALIZADO"
	@echo "============================================"
	@echo "  Banco:  MySQL e Redis"
	@echo "  App:    API e Webhook"
	@echo "============================================"

# --- Comandos de Utilitários ---

.PHONY: oci-ps oci-status
oci-ps:
	docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

oci-status: oci-ps

.PHONY: oci-logs-all
oci-logs-all:
	docker-compose -f docker-compose.database.yml logs -f &
	docker-compose -f docker-compose.app.yml logs -f

# ================================================================
# AMBIENTE LOCAL (Desenvolvimento)
# ================================================================

.PHONY: local-up local-down
local-up:
	docker-compose -f docker-compose.database.yml up -d
	docker-compose -f docker-compose.app.yml up -d

local-down:
	docker-compose -f docker-compose.database.yml down
	docker-compose -f docker-compose.app.yml down