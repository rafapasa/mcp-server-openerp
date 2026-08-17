# ================================================================
# Makefile Unificado - MCP Server Openerp
# ================================================================

# ================================================================
# VARIÁVEIS GLOBAIS
# ================================================================
DOCKER_USERNAME := rafapasa
IMAGE_TAG := latest
PLATFORM := linux/arm64
DOCKERFILE := deploy/Dockerfile.multistage

# Cores para output
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# ================================================================
# AJUDA (default)
# ================================================================
.PHONY: help
help: ## Mostra ajuda com todos os comandos disponíveis
	@echo "$(BLUE)============================================================$(NC)"
	@echo "$(GREEN)  MCP Server Openerp - Makefile$(NC)"
	@echo "$(BLUE)============================================================$(NC)"
	@echo ""
	@echo "$(YELLOW)Desenvolvimento Local:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -v "docker\|build-webhook\|build-api\|push" | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Docker/OCI (Build e Deploy):$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E "docker|build-webhook|build-api|push|login" | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Limpeza e Utilitários:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E "clean|test|fmt|lint|check|deps|install" | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# ================================================================
# DESENVOLVIMENTO LOCAL (Go)
# ================================================================

.PHONY: build build-stdio build-http build-wh build-api
build: build-stdio build-http build-wh build-api ## Build all Go binaries
build-stdio: ## Build stdio binary
	@echo "$(GREEN)🔨 Building stdio...$(NC)"
	go build -o bin/stdio ./cmd/stdio

build-http: ## Build http binary
	@echo "$(GREEN)🔨 Building http...$(NC)"
	go build -o bin/http ./cmd/http

build-wh: ## Build webhook binary
	@echo "$(GREEN)🔨 Building webhook...$(NC)"
	go build -o bin/webhook ./cmd/webhook

build-api: ## Build api binary
	@echo "$(GREEN)🔨 Building api...$(NC)"
	go build -o bin/api ./cmd/api

.PHONY: run-stdio run-http run-wh run-api
run-stdio: ## Run stdio
	go run ./cmd/stdio

run-http: ## Run http
	go run ./cmd/http

run-wh: ## Run webhook
	go run ./cmd/webhook

run-api: ## Run api
	go run ./cmd/api

.PHONY: ngrok
ngrok: ## Start ngrok tunnel on port 8080
	ngrok http 8080

.PHONY: dev
dev: ## Development with live reload (requires air)
	@echo "$(GREEN)🔄 Starting dev mode with air...$(NC)"
	air

.PHONY: mcp
mcp: ## Run MCP inspector
	@echo "$(GREEN)🔍 Starting MCP inspector...$(NC)"
	npx @modelcontextprotocol/inspector -- wsl bash -c "cd ~/mcp-server-openerp && go run ./cmd/stdio/main.go"

# ================================================================
# TESTES E QUALIDADE DE CÓDIGO
# ================================================================

.PHONY: test
test: ## Run tests
	@echo "$(GREEN)🧪 Running tests...$(NC)"
	go test -v ./...

.PHONY: fmt
fmt: ## Format code (gofmt + goimports)
	@echo "$(GREEN)🎨 Formatting code...$(NC)"
	gofmt -w -s .
	@which goimports > /dev/null && goimports -w . || echo "⚠️  goimports not installed (run 'make install-tools')"

.PHONY: lint
lint: ## Lint code (requires golangci-lint)
	@echo "$(GREEN)🔍 Linting code...$(NC)"
	@which golangci-lint > /dev/null && golangci-lint run ./... || echo "⚠️  golangci-lint not installed (run 'make install-tools')"

.PHONY: check
check: fmt lint test ## Run all checks (fmt, lint, test)

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "$(GREEN)📦 Updating dependencies...$(NC)"
	go mod download
	go mod tidy

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(GREEN)🔧 Installing tools...$(NC)"
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest

# ================================================================
# DOCKER BUILD (Local e OCI)
# ================================================================

.PHONY: login
login: ## Login no Docker Hub
	@echo "$(YELLOW)🔑 Login no Docker Hub...$(NC)"
	docker login

.PHONY: docker-build-webhook
docker-build-webhook: ## Build webhook Docker image (ARM64)
	@echo "$(GREEN)🔨 Build webhook Docker image (ARM64)...$(NC)"
	docker buildx build \
		--platform $(PLATFORM) \
		--cache-from $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) \
		--cache-from $(DOCKER_USERNAME)/mcp-webhook:latest \
		-f $(DOCKERFILE) \
		--target webhook \
		-t $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) \
		-t $(DOCKER_USERNAME)/mcp-webhook:latest \
		.

.PHONY: docker-build-mcp
docker-build-mcp: ## Build MCP Docker image (ARM64)
	@echo "$(GREEN)🔨 Build MCP Docker image (ARM64)...$(NC)"
	docker buildx build \
		--platform $(PLATFORM) \
		--cache-from $(DOCKER_USERNAME)/mcp:$(IMAGE_TAG) \
		--cache-from $(DOCKER_USERNAME)/mcp:latest \
		-f $(DOCKERFILE) \
		--target mcp \
		-t $(DOCKER_USERNAME)/mcp:$(IMAGE_TAG) \
		-t $(DOCKER_USERNAME)/mcp:latest \
		.

.PHONY: docker-build-all
docker-build-all: docker-build-webhook docker-build-mcp ## Build all Docker images

.PHONY: docker-push-webhook
docker-push-webhook: ## Push webhook image to Docker Hub
	@echo "$(GREEN)📤 Pushing webhook...$(NC)"
	docker push $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-webhook:latest

.PHONY: docker-push-mcp
docker-push-mcp: ## Push MCP image to Docker Hub
	@echo "$(GREEN)📤 Pushing MCP...$(NC)"
	docker push $(DOCKER_USERNAME)/mcp:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp:latest

.PHONY: docker-push-all
docker-push-all: docker-push-webhook docker-push-mcp ## Push all images

.PHONY: docker-build-push
docker-build-push: login docker-build-all docker-push-all ## Build and push all images
	@echo "$(GREEN)✅ Build e push concluídos!$(NC)"
	@echo ""
	@echo "Imagens disponíveis:"
	@echo "  - $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)"
	@echo "  - $(DOCKER_USERNAME)/mcp:$(IMAGE_TAG)"

# ================================================================
# DOCKER COMPOSE (Testes Locais)
# ================================================================

.PHONY: compose-up
compose-up: ## Start containers with docker-compose
	@echo "$(GREEN)🚀 Starting containers...$(NC)"
	docker-compose -f deploy/docker-compose.yml up -d
	@echo "✅ Containers rodando:"
	docker-compose -f deploy/docker-compose.yml ps

.PHONY: compose-down
compose-down: ## Stop containers
	@echo "$(YELLOW)🛑 Stopping containers...$(NC)"
	docker-compose -f deploy/docker-compose.yml down

.PHONY: compose-logs
compose-logs: ## View container logs
	docker-compose -f deploy/docker-compose.yml logs -f

.PHONY: compose-restart
compose-restart: compose-down compose-up ## Restart containers

# ================================================================
# LIMPEZA
# ================================================================

.PHONY: clean
clean: ## Clean all (binaries, Docker builders, cache)
	@echo "$(YELLOW)🧹 Cleaning...$(NC)"
	rm -rf bin/ tmp/
	go clean
	docker buildx rm arm64-builder 2>/dev/null || true
	docker system prune -f 2>/dev/null || true
	@echo "$(GREEN)✅ Clean complete!$(NC)"

.PHONY: clean-docker
clean-docker: ## Clean only Docker resources
	@echo "$(YELLOW)🧹 Cleaning Docker...$(NC)"
	docker buildx rm arm64-builder 2>/dev/null || true
	docker system prune -f

.PHONY: clean-bin
clean-bin: ## Clean only Go binaries
	@echo "$(YELLOW)🧹 Cleaning binaries...$(NC)"
	rm -rf bin/ tmp/
	go clean

# ================================================================
# COMANDOS COMBINADOS
# ================================================================

.PHONY: all
all: build ## Default: build all Go binaries

.PHONY: dev-full
dev-full: deps build install-tools ## Setup full development environment

# ================================================================
# DEPLOY OCI
# ================================================================

.PHONY: oci-deploy
oci-deploy: docker-build-push ## Build, push and prepare for OCI deploy
	@echo ""
	@echo "$(BLUE)============================================================$(NC)"
	@echo "$(GREEN)✅ Pronto para deploy na OCI!$(NC)"
	@echo "$(BLUE)============================================================$(NC)"
	@echo ""
	@echo "$(YELLOW)Na instância OCI (Ampere), execute:$(NC)"
	@echo "  ssh opc@<IP_OCI>"
	@echo "  docker pull $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)"
	@echo "  docker pull $(DOCKER_USERNAME)/mcp:$(IMAGE_TAG)"
	@echo "  docker-compose -f deploy/docker-compose.yml --env-file .env up -d"
	@echo ""

# ================================================================
# COMANDOS ESPECÍFICOS PARA OCI (executar na instância)
# ================================================================

.PHONY: oci-pull
oci-pull: ## Pull images na OCI
	@echo "$(GREEN)📦 Pulling images on OCI...$(NC)"
	docker pull $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)
	docker pull $(DOCKER_USERNAME)/mcp:$(IMAGE_TAG)

.PHONY: oci-up
oci-up: ## Start containers na OCI (com .env)
	@echo "$(GREEN)🚀 Starting containers on OCI...$(NC)"
	docker-compose -f deploy/docker-compose.yml --env-file .env up -d
	docker-compose -f deploy/docker-compose.yml ps

.PHONY: oci-down
oci-down: ## Stop containers na OCI
	@echo "$(YELLOW)🛑 Stopping containers on OCI...$(NC)"
	docker-compose -f deploy/docker-compose.yml down

.PHONY: oci-restart
oci-restart: oci-down oci-up ## Restart containers na OCI

.PHONY: oci-logs
oci-logs: ## View logs na OCI
	docker-compose -f deploy/docker-compose.yml logs -f

.PHONY: oci-status
oci-status: ## Check status na OCI
	docker-compose -f deploy/docker-compose.yml ps
	@echo ""
	@echo "Containers:"
	docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"