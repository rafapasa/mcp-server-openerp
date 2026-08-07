// internal/server/resources.go
package server

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *MCPServer) registerResources() {
    // Resource: Cardápio (como recurso somente leitura)
    s.AddResource(
        mcp.NewResource(
            "cardapio",
            "cardapio://{tenant_id}",
            mcp.WithResourceDescription("Cardápio completo do restaurante"),
            mcp.WithMIMEType("application/json"),
        ),
        s.cardapioResourceHandler(),
    )
}

func (s *MCPServer) cardapioResourceHandler() func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
        tenantID := request.Params.URI[12:] // "cardapio://tenant_id"
        
        cardapio, err := s.getCardapio(tenantID)
        if err != nil {
            return nil, err
        }
        
        content, err := json.Marshal(cardapio)
        if err != nil {
            return nil, err
        }
        
        return []mcp.ResourceContents{
            mcp.TextResourceContents{
                URI:      request.Params.URI,
                MIMEType: "application/json",
                Text:     string(content),
            },
        }, nil
    }
}