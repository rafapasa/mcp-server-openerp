# 1. Criar .env
cat > .env << 'EOF'
DB_ROOT_PASSWORD=root_password_secure
DB_NAME=mcp_server_openerp
DB_USER=mcp_user
DB_PASSWORD=mcp_password_secure
REDIS_HOST=redis
REDIS_PORT=6379
JWT_SECRET=your-secret-key
LLM_PROVIDER=gemini
LLM_API_KEY=your_key
LLM_MODEL=gemini-2.0-flash
WHATSAPP_APP_SECRET=your_secret
WHATSAPP_VERIFY_TOKEN=your_token
WHATSAPP_ACCESS_TOKEN=your_token
WHATSAPP_PHONE_NUMBER=5511999999999
LOG_LEVEL=debug
API_PORT=8081
EOF

# 2. Rodar MySQL e Redis
docker run -d --name mcp-mysql --restart unless-stopped -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root_password_secure -e MYSQL_DATABASE=mcp_server_openerp -e MYSQL_USER=mcp_user -e MYSQL_PASSWORD=mcp_password_secure -v mysql_data:/var/lib/mysql mysql:8.4
docker run -d --name mcp-redis --restart unless-stopped -p 6379:6379 -v redis_data:/data redis:7-alpine

# 3. Aguardar MySQL
sleep 15

# 4. Rodar Webhook e API
docker run -d --name mcp-webhook --restart unless-stopped -p 8080:8080 --env-file .env rafapasa/mcp-webhook:latest
docker run -d --name mcp-api --restart unless-stopped -p 8081:8081 --env-file .env rafapasa/mcp-api:latest

# 5. Verificar
docker ps
curl http://localhost:8080/health
curl http://localhost:8081/health