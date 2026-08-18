# ================================================================
# Makefile - MCP Server Openerp
# ================================================================

DOCKER_USERNAME := rafapasa
IMAGE_TAG := latest
PLATFORM := linux/arm64
DOCKERFILE := deploy/Dockerfile.multistage

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
# DOCKER BUILD E PUSH
# ================================================================

.PHONY: login
login:
	docker login

.PHONY: docker-build-webhook
docker-build-webhook:
	docker buildx build --platform $(PLATFORM) -f $(DOCKERFILE) --target webhook -t $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) -t $(DOCKER_USERNAME)/mcp-webhook:latest .

.PHONY: docker-build-api
docker-build-api:
	docker buildx build --platform $(PLATFORM) -f $(DOCKERFILE) --target api -t $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG) -t $(DOCKER_USERNAME)/mcp-api:latest .

.PHONY: docker-build-all
docker-build-all: docker-build-webhook docker-build-api

.PHONY: docker-push-webhook
docker-push-webhook:
	docker push $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-webhook:latest

.PHONY: docker-push-api
docker-push-api:
	docker push $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)
	docker push $(DOCKER_USERNAME)/mcp-api:latest

.PHONY: docker-push-all
docker-push-all: docker-push-webhook docker-push-api

.PHONY: docker-build-push
docker-build-push: login docker-build-all docker-push-all
	@echo "✅ Imagens publicadas: $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG) e $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)"

# ================================================================
# DOCKER COMPOSE (LOCAL)
# ================================================================

.PHONY: compose-up compose-down compose-logs
compose-up:
	docker-compose -f deploy/docker-compose.yml up -d

compose-down:
	docker-compose -f deploy/docker-compose.yml down

compose-logs:
	docker-compose -f deploy/docker-compose.yml logs -f

# ================================================================
# OCI DEPLOY
# ================================================================

.PHONY: oci-pull oci-up oci-down oci-restart oci-logs
oci-pull:
	docker pull $(DOCKER_USERNAME)/mcp-webhook:$(IMAGE_TAG)
	docker pull $(DOCKER_USERNAME)/mcp-api:$(IMAGE_TAG)

oci-up:
	docker-compose -f deploy/docker-compose.yml --env-file .env up -d

oci-down:
	docker-compose -f deploy/docker-compose.yml down

oci-restart: oci-down oci-up

oci-logs:
	docker-compose -f deploy/docker-compose.yml logs -f