# ================================================================
# Makefile - MCP Server Openerp
# ================================================================

DOCKER_USERNAME := rafapasa
IMAGE_TAG ?= latest
PLATFORM := linux/arm64
DOCKERFILE := Dockerfile.multistage
NO_CACHE ?=

# ================================================================
# DESENVOLVIMENTO LOCAL - MANTIDO
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
# 🚀 DEPLOY OCI - ORACLE LINUX 9 / AMPERE ARM64
# ================================================================

.PHONY: login
login:
	docker login -u $(DOCKER_USERNAME)

.PHONY: init-db
init-db:
	@echo "🐳 Criando rede mcp-network e subindo MySQL + Redis..."
	docker network create mcp-network || true
	docker compose -f docker-compose.db.yml up -d
	@echo "✅ MySQL mcp-mysql:3306 | Redis mcp-redis:6379"

# 2 - Build na OCI + Push - Sempre cria a versão + latest
.PHONY: build-push
build-push:
ifeq ($(IMAGE_TAG),latest)
	$(error Use: make build-push IMAGE_TAG=0.1.11 - não pode buildar só latest)
endif
	git pull
	DOCKER_BUILDKIT=1 docker build $(NO_CACHE) -f $(DOCKERFILE) --target api -t $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG) -t $(DOCKER_USERNAME)/mcp-api:latest .
	DOCKER_BUILDKIT=1 docker build $(NO_CACHE) -f $(DOCKERFILE) --target webhook -t $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) -t $(DOCKER_USERNAME)/mcp-webhook:latest .
	docker push $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-api:latest
	docker push $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-webhook:latest
	@echo "✅ Build $(IMAGE_TAG) + latest enviado"

# 3 - Deploy App - Se não passar tag, usa latest
.PHONY: deploy
deploy:
	@if ! docker network inspect mcp-network > /dev/null 2>&1; then \
		echo "❌ ERRO: rede mcp-network não existe!"; \
		echo "👉 Rode primeiro: make init-db"; \
		exit 1; \
	fi
	@echo "🚀 Deploy $(IMAGE_TAG)..."
	IMAGE_TAG=$(IMAGE_TAG) docker compose -f docker-compose.app.yml up -d --pull always
	docker image prune -f
	docker ps

.PHONY: rollback
rollback:
	$(MAKE) deploy IMAGE_TAG=$(IMAGE_TAG)

.PHONY: logs logs-tail ps status
logs:
	docker logs -f mcp-api --tail=100

logs-api:
	docker logs -f mcp-api --tail=100

logs-wh:
	docker logs -f mcp-webhook --tail=100	

logs-tail:
	docker compose -f docker-compose.app.yml logs -f --tail=100

ps status:
	docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"