@echo off
echo === Iniciando MCP Inspector ===
echo Conectando ao WSL...

REM Executa o MCP Server no WSL
wsl bash -c "cd ~/mcp-server-openerp && go run ./cmd/stdio/main.go"