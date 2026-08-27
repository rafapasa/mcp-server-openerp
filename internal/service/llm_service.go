package service

import (
	"context"
	"fmt"

	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type llmService struct {
	provider        *llm.UnifiedLLM
	cardapioService CardapioServiceInterface
}

func NewLLMService(
	provider *llm.UnifiedLLM,
	cardapioService CardapioServiceInterface,
) LLMServiceInterface {
	return &llmService{
		provider:        provider,
		cardapioService: cardapioService,
	}
}

func (s *llmService) HasValidConfig() bool {
	if s.provider == nil {
		return false
	}
	return s.provider.GetProvider() != ""
}

func (s *llmService) IsOnline(ctx context.Context) (bool, error) {
	if !s.HasValidConfig() {
		return false, fmt.Errorf("llm provider sem configuração válida")
	}
	_, err := s.provider.GenerateResponse(ctx, "Responda apenas com a palavra: ok")
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *llmService) GetProviderInfo() (textProvider, audioProvider, visionProvider string) {
	if s.provider == nil {
		return "none", "none", "none"
	}
	p := s.provider.GetProvider()
	return p, p, p
}

func (s *llmService) ResolveItemsByMenu(
	ctx context.Context,
	tenantID uint,
	input dto.MessageInput,
	cardapio []dto.ProdutoItem,
) (*dto.IntencaoCliente, error) {
	logger.Info(
		ctx, "[LLM_SERVICE] ResolveItemsByMenu",
		zap.Uint("tenant_id", tenantID),
		zap.String("source", string(input.Source)),
		zap.Int("cardapio_len", len(cardapio)),
	)

	if len(cardapio) == 0 {
		var err error
		cardapio, err = s.cardapioService.GetCardapio(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar cardápio: %w", err)
		}
	}

	return s.provider.ResolveItemsByMenu(ctx, input, cardapio)
}

func (s *llmService) ClassificarEExtrairKeywords(
	ctx context.Context,
	textoHigienizado string,
	contextoCarrinho string,
) (*llm.IntencaoEKeywordsResult, error) {
	logger.Info(ctx, "[LLM_SERVICE] ClassificarEExtrairKeywords", zap.String("texto", textoHigienizado))

	limpo := s.provider.Higienizar(textoHigienizado)
	if limpo == "" {
		limpo = textoHigienizado
	}

	return s.provider.ClassificarEExtrairKeywords(ctx, limpo, contextoCarrinho)
}

func (s *llmService) ResolverItensByKeyWords(
	ctx context.Context,
	tenantID uint,
	keywords []llm.LLMKeywordItemResult,
	cardapioReduzido []dto.ProdutoItem,
) ([]dto.ItemCarrinho, error) {
	logger.Info(
		ctx, "[LLM_SERVICE] ResolverItensByKeyWords",
		zap.Uint("tenant_id", tenantID),
		zap.Int("keywords_len", len(keywords)),
		zap.Int("cardapio_reduzido_len", len(cardapioReduzido)),
	)

	if len(keywords) == 0 {
		return []dto.ItemCarrinho{}, nil
	}

	if len(cardapioReduzido) == 0 {
		var err error
		cardapioReduzido, err = s.cardapioService.ReduzirPorKeywords(ctx, tenantID, keywords)
		if err != nil {
			return nil, fmt.Errorf("erro ao reduzir cardápio: %w", err)
		}
	}

	return s.provider.ResolverItensByKeyWords(ctx, keywords, cardapioReduzido)
}
