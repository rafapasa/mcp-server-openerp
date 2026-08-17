#!/bin/bash
# build-and-push.sh - OTIMIZADO

set -e

DOCKER_USERNAME="rafapasa"
IMAGE_TAG="latest"

echo "============================================"
echo "  Build e Push Multiplataforma (OTIMIZADO)"
echo "============================================"

docker login

# Builder multiplataforma
docker buildx create --name mybuilder --use --bootstrap

# Build Webhook
echo "🔨 Build webhook (multi-platform)..."
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -f deploy/Dockerfile.webhook \
    -t ${DOCKER_USERNAME}/mcp-webhook:${IMAGE_TAG} \
    --push .

# Build API
echo "🔨 Build api (multi-platform)..."
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -f deploy/Dockerfile.api \
    -t ${DOCKER_USERNAME}/mcp-api:${IMAGE_TAG} \
    --push .

echo ""
echo "============================================"
echo "  ✅ Imagens disponíveis:"
echo "  - ${DOCKER_USERNAME}/mcp-webhook:${IMAGE_TAG}"
echo "  - ${DOCKER_USERNAME}/mcp-api:${IMAGE_TAG}"
echo "  Tamanho reduzido (com otimizações)"
echo "============================================"