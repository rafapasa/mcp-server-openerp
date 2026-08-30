//go:build generate
// +build generate

package mocks

// go:generate go run go.uber.org/mock/mockgen -destination=redis_interface_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/database RedisInterface
// go:generate go run go.uber.org/mock/mockgen -destination=cardapio_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service CardapioServiceInterface
// go:generate go run go.uber.org/mock/mockgen -destination=pedido_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service PedidoServiceInterface
// go:generate go run go.uber.org/mock/mockgen -destination=produto_repo_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/repository ProdutoRepository
// go:generate go run go.uber.org/mock/mockgen -destination=llm_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service LLMServiceInterface
// go:generate go run go.uber.org/mock/mockgen -destination=tenant_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service TenantServiceInterface
// go:generate go run go.uber.org/mock/mockgen -destination=cliente_repo_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/repository ClienteRepositoryInterface
// go:generate go run go.uber.org/mock/mockgen -destination=endereco_repo_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/repository EnderecoRepositoryInterface

// go run go.uber.org/mock/mockgen -destination=internal/mocks/cardapio_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service CardapioServiceInterface
// go run go.uber.org/mock/mockgen -destination=internal/mocks/pedido_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service PedidoServiceInterface
// go run go.uber.org/mock/mockgen -destination=internal/mocks/produto_repo_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/repository ProdutoRepository
// go run go.uber.org/mock/mockgen -destination=internal/mocks/llm_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service LLMServiceInterface
// go run go.uber.org/mock/mockgen -destination=internal/mocks/tenant_service_mock.go -package=mocks github.com/rafapasa/mcp-server-openerp/internal/service TenantServiceInterface
