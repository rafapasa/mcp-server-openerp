#!/bin/bash
echo "============================================"
echo " MCP Inspector - WSL"
echo "============================================"

cd ~/mcp-server-openerp || exit 1

echo "[1/3] Compilando MCP Server..."
go build -o bin/stdio ./cmd/stdio/main.go

echo "[2/3] Verificando binario..."
test -f bin/stdio && echo "OK" || echo "Erro"

echo "[3/3] Iniciando MCP Inspector..."
echo
npx @modelcontextprotocol/inspector -- ./bin/stdio