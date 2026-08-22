# ================================================================
# Makefile - MCP Server Openerp - Beta 1
# ================================================================

DOCKER_USERNAME := rafapasa
IMAGE_TAG ?= latest
PLATFORM := linux/arm64
DOCKERFILE := Dockerfile.multistage
NO_CACHE ?=

# ================================================================
# DESENVOLVIMENTO LOCAL
# ================================================================

.PHONY: build build-server build-stdio
build: build-server build-stdio

build-server:
	go build -o bin/server ./cmd/server

build-stdio:
	go build -o bin/stdio ./cmd/stdio

.PHONY: run run-server run-stdio
run: run-server

run-server:
	go run ./cmd/server

run-stdio:
	go run ./cmd/stdio

.PHONY: dev wire
dev:
	air

wire:
	rm -f internal/di/wire_gen.go && wire gen ./internal/di/...

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
	go install github.com/google/wire/cmd/wire@latest

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

# Build server + stdio - sempre cria versão + latest
.PHONY: build-push
build-push:
ifeq ($(IMAGE_TAG),latest)
	$(error Use: make build-push IMAGE_TAG=0.1.11 - não pode buildar só latest)
endif
	git pull
	DOCKER_BUILDKIT=1 docker build $(NO_CACHE) -f $(DOCKERFILE) --target server -t $(DOCKER_USERNAME)/mcp-server:$(IMAGE_TAG) -t $(DOCKER_USERNAME)/mcp-server:latest .
	DOCKER_BUILDKIT=1 docker build $(NO_CACHE) -f $(DOCKERFILE) --target stdio -t $(DOCKER_USERNAME)/mcp-stdio:$(IMAGE_TAG) -t $(DOCKER_USERNAME)/mcp-stdio:latest .
	docker push $(DOCKER_USERNAME)/mcp-server:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-server:latest
	docker push $(DOCKER_USERNAME)/mcp-stdio:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-stdio:latest
	@echo "✅ Build $(IMAGE_TAG) + latest enviado"

.PHONY: deploy
deploy:
	@echo "🚀 Deploy latest..."
	IMAGE_TAG=latest docker compose -f docker-compose.app.yml up -d --pull always --no-deps
	@docker ps | grep mcp-server