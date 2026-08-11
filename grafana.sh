# Rodar Grafana
docker run -d --name grafana \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_USER=admin \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  grafana/grafana:latest

# Verificar se está rodando
docker ps | grep grafana

# Acessar
# http://localhost:3000
# Usuário: admin
# Senha: admin