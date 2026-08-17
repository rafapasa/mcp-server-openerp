#!/bin/bash
# oci-run.sh

set -e

DOCKER_USERNAME="rafapasa"
IMAGE_TAG="latest"

echo "============================================"
echo "  MCP Server - OCI (ARM64)"
echo "============================================"

# Verifica se .env existe
if [ ! -f .env ]; then
    echo "❌ Arquivo .env não encontrado!"
    echo "   Copie .env.example para .env e configure"
    exit 1
fi

docker network create mcp-network 2>/dev/null || true

# MySQL
echo "🐳 Iniciando MySQL..."
docker rm -f mcp-mysql 2>/dev/null || true
docker run -d --name mcp-mysql --restart unless-stopped \
    -p 3306:3306 --network mcp-network \
    --env-file .env \
    -v mysql_data:/var/lib/mysql \
    mysql:8.4

sleep 15

# Redis
echo "🐳 Iniciando Redis..."
docker rm -f mcp-redis 2>/dev/null || true
docker run -d --name mcp-redis --restart unless-stopped \
    -p 6379:6379 --network mcp-network \
    -v redis_data:/data \
    redis:7-alpine

# Webhook
echo "📥 Pull webhook..."
docker pull ${DOCKER_USERNAME}/mcp-webhook:${IMAGE_TAG}

echo "🐳 Iniciando Webhook..."
docker rm -f mcp-webhook 2>/dev/null || true
docker run -d --name mcp-webhook --restart unless-stopped \
    -p 8080:8080 --network mcp-network \
    --env-file .env \
    -v $(pwd)/logs:/app/logs \
    ${DOCKER_USERNAME}/mcp-webhook:${IMAGE_TAG}

# API
echo "📥 Pull API..."
docker pull ${DOCKER_USERNAME}/mcp-api:${IMAGE_TAG}

echo "🐳 Iniciando API..."
docker rm -f mcp-api 2>/dev/null || true
docker run -d --name mcp-api --restart unless-stopped \
    -p 8081:8081 --network mcp-network \
    --env-file .env \
    -v $(pwd)/logs:/app/logs \
    ${DOCKER_USERNAME}/mcp-api:${IMAGE_TAG}

echo ""
echo "============================================"
echo "  ✅ Concluído!"
echo "============================================"
docker ps
echo ""
echo "  Testar:"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8081/health"
echo "============================================"