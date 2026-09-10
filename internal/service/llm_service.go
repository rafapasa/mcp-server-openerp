package service

import (
	"context"
	"fmt"

	"github.com/etoolstec/gokit/apperror"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/llm"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type llmService struct {
	provider        *llm.UnifiedLLM
	cardapioService CardapioServiceInterface
	tenantService   TenantServiceInterface
}

func NewLLMService(
	provider *llm.UnifiedLLM,
	cardapioService CardapioServiceInterface,
	tenantService TenantServiceInterface,
) LLMServiceInterface {
	return &llmService{
		provider:        provider,
		cardapioService: cardapioService,
		tenantService:   tenantService,
	}
}

func (s *llmService) HasValidConfig() bool {
	if s.provider == nil {
		return false
	}
	return s.provider.GetProvider() != ""
}

func (s *llmService) IsOnline(ctx context.Context) (bool, error) {
	if s.provider == nil {
		return false, apperror.NewInternalError("llm provider não configurado", nil)
	}
	return s.provider.IsOnline(ctx)
}

func (s *llmService) GetProviderInfo() (textProvider, audioProvider, visionProvider string) {
	if s.provider == nil {
		return "none", "none", "none"
	}
	p := s.provider.GetProvider()
	return p, p, p
}

func (s *llmService) ObterTextoBase(ctx context.Context, tenantID uint, input dto.MessageInput) (string, error) {
	_ = tenantID
	switch input.Source {
	case models.SourceAudio:
		if len(input.Audio) == 0 {
			return "", apperror.NewBadRequestError("áudio vazio")
		}
		return s.provider.TranscribeAudio(ctx, input.Audio, llm.PromptTranscribeSimple)
	case models.SourceImage:
		if len(input.Image) == 0 {
			return "", apperror.NewBadRequestError("imagem vazia")
		}
		return s.provider.DescribeImage(ctx, input.Image, input.Text)
	default:
		return input.Text, nil
	}
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
			return nil, err
		}
	}

	result, err := s.provider.ResolveItemsByMenu(ctx, input, cardapio, s.contextoLoja(ctx, tenantID))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *llmService) ClassificarEExtrairKeywords(
	ctx context.Context,
	tenantID uint,
	textoHigienizado string,
	contextoCarrinho string,
) (*llm.IntencaoEKeywordsResult, error) {
	logger.Info(ctx, "[LLM_SERVICE] ClassificarEExtrairKeywords",
		zap.Uint("tenant_id", tenantID),
		zap.String("texto", textoHigienizado),
	)

	limpo := s.provider.Higienizar(textoHigienizado)
	if limpo == "" {
		limpo = textoHigienizado
	}

	result, err := s.provider.ClassificarEExtrairKeywords(ctx, limpo, contextoCarrinho, s.contextoLoja(ctx, tenantID))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *llmService) contextoLoja(ctx context.Context, tenantID uint) string {
	nome := ""
	segmento := "geral"
	if s.tenantService != nil {
		n, seg, err := s.tenantService.GetPromptContext(ctx, tenantID)
		if err != nil {
			logger.Warn(ctx, "[LLM_SERVICE] sem contexto de loja, usando neutro",
				zap.Uint("tenant_id", tenantID),
				zap.Error(err),
			)
		} else {
			nome = n
			if seg != "" {
				segmento = seg
			}
		}
	}
	return fmt.Sprintf(llm.PromptContextoLoja, nome, segmento)
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
			return nil, err
		}
	}

	result, err := s.provider.ResolverItensByKeyWords(ctx, keywords, cardapioReduzido)
	if err != nil {
		return nil, err
	}
	return result, nil
}
