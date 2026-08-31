package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
	"github.com/rafapasa/mcp-server-openerp/internal/dto"
	"github.com/rafapasa/mcp-server-openerp/internal/models"
	"github.com/rafapasa/mcp-server-openerp/internal/observability/logger"
	"go.uber.org/zap"
)

type LLMIntencaoResult struct {
	Acao     string `json:"acao"`
	Resposta string `json:"resposta"`
	Filtro   string `json:"filtro"`
}

type LLMKeywordItemResult struct {
	Nome       string  `json:"nome"`
	Quantidade float32 `json:"qtd"`
	Unidade    string  `json:"unidade,omitempty"`
}

type LLMKeywordResult struct {
	Keywords []LLMKeywordItemResult `json:"keywords"`
}

func (l *LLMKeywordResult) ToString() []string {
	var result []string
	for _, item := range l.Keywords {
		s := fmt.Sprintf("%s qtd:%.0f", item.Nome, item.Quantidade)
		if item.Unidade != "" {
			s += " " + item.Unidade
		}
		result = append(result, s)
	}
	return result
}

type IntencaoEKeywordsResult struct {
	Acao     string                 `json:"acao"`
	Resposta string                 `json:"resposta"`
	Filtro   string                 `json:"filtro"`
	Keywords []LLMKeywordItemResult `json:"keywords"`
}

type LLMResolverResult struct {
	Itens []struct {
		ID  int    `json:"id"`
		Qtd int    `json:"qtd"`
		Obs string `json:"obs"`
	} `json:"itens"`
}

type UnifiedLLM struct {
	pool         map[string]LLMClient
	textClient   TextLLM
	audioClient  AudioLLM
	visionClient VisionLLM
	preprocessor *Preprocessor
}

func NewUnifiedLLM(cfg *config.Config) *UnifiedLLM {
	logger.LogInfo("[PROVIDER] Inicializando UnifiedLLM gestor/fachada")
	pool := NewAllClients(cfg)
	logger.LogInfo(fmt.Sprintf("[PROVIDER] Pool criado %d clients", len(pool)))

	textClient := NewLLMText(cfg)
	audioClient := NewLLMAudio(cfg)
	visionClient := NewLLMVision(cfg)
	preprocessor := NewPreprocessor()

	if textClient != nil {
		logger.Info(context.Background(), "[PROVIDER] textClient OK",
			zap.String("provider", textClient.GetProvider()),
			zap.String("model", textClient.GetModel()),
		)
	}
	if audioClient != nil {
		logger.Info(context.Background(), "[PROVIDER] audioClient OK",
			zap.String("provider", audioClient.GetProvider()),
		)
	}
	if visionClient != nil {
		logger.Info(context.Background(), "[PROVIDER] visionClient OK",
			zap.String("provider", visionClient.GetProvider()),
		)
	}

	return &UnifiedLLM{
		pool:         pool,
		textClient:   textClient,
		audioClient:  audioClient,
		visionClient: visionClient,
		preprocessor: preprocessor,
	}
}

func (u *UnifiedLLM) GetProvider() string { return "unified" }
func (u *UnifiedLLM) GetModel() string    { return "router" }

func (u *UnifiedLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	logger.Info(ctx, "[PROVIDER] GenerateResponse -> textClient", zap.Int("prompt_len", len(prompt)))
	if u.textClient == nil {
		return "", fmt.Errorf("textClient nil")
	}
	raw, err := u.textClient.GenerateResponse(ctx, prompt)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro GenerateResponse", zap.Error(err))
		return "", err
	}
	logger.Info(ctx, "[PROVIDER] GenerateResponse OK", zap.Int("raw_len", len(raw)))
	return raw, nil
}

func (u *UnifiedLLM) TranscribeAudio(ctx context.Context, audio []byte, prompt string) (string, error) {
	logger.Info(ctx, "[PROVIDER] TranscribeAudio -> audioClient", zap.Int("audio_len", len(audio)))
	if u.audioClient == nil {
		return "", fmt.Errorf("audioClient nil")
	}
	if prompt == "" {
		prompt = PromptTranscribeSimple
	}
	text, err := u.audioClient.TranscribeAudio(ctx, audio, prompt)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro TranscribeAudio", zap.Error(err))
		return "", err
	}
	logger.Info(ctx, "[PROVIDER] TranscribeAudio OK", zap.String("texto", text))
	return text, nil
}

func (u *UnifiedLLM) DescribeImage(ctx context.Context, image []byte, prompt string) (string, error) {
	logger.Info(ctx, "[PROVIDER] DescribeImage -> visionClient", zap.Int("image_len", len(image)))
	if u.visionClient == nil {
		return "", fmt.Errorf("visionClient nil")
	}
	if prompt == "" {
		prompt = PromptVisionDescribe
	}
	desc, err := u.visionClient.DescribeImage(ctx, image, prompt)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro DescribeImage", zap.Error(err))
		return "", err
	}
	logger.Info(ctx, "[PROVIDER] DescribeImage OK", zap.Int("desc_len", len(desc)))
	return desc, nil
}

func (u *UnifiedLLM) Higienizar(texto string) string {
	if u.preprocessor == nil {
		return strings.TrimSpace(strings.ToLower(texto))
	}
	pp := u.preprocessor.Process(texto)
	return pp.Cleaned
}

func (u *UnifiedLLM) ClassificarEExtrairKeywords(
	ctx context.Context,
	textoHigienizado string,
	contextoCarrinho string,
	contextoLoja string,
) (*IntencaoEKeywordsResult, error) {
	logger.Info(ctx, "[PROVIDER] ClassificarEExtrairKeywords", zap.String("texto", textoHigienizado))

	if textoHigienizado == "" {
		return &IntencaoEKeywordsResult{
			Acao:     "conversa",
			Resposta: "Desculpe, não entendi. Pode repetir?",
		}, nil
	}

	prompt := fmt.Sprintf(PromptClassificarEExtrairKeywords, contextoLoja, textoHigienizado, contextoCarrinho)
	raw, err := u.GenerateResponse(ctx, prompt)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro ClassificarEExtrairKeywords", zap.Error(err))
		return nil, err
	}

	var result IntencaoEKeywordsResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &result); err != nil {
		logger.Error(ctx, "[PROVIDER] Parse ClassificarEExtrairKeywords falhou", zap.Error(err), zap.String("raw", raw))
		return &IntencaoEKeywordsResult{
			Acao:     "conversa",
			Resposta: "Desculpe, não consegui processar. Pode reformular?",
		}, nil
	}

	for i := range result.Keywords {
		if result.Keywords[i].Quantidade <= 0 {
			result.Keywords[i].Quantidade = 1
		}
	}

	logger.Info(ctx, "[PROVIDER] ClassificarEExtrairKeywords OK",
		zap.String("acao", result.Acao),
		zap.Int("keywords_len", len(result.Keywords)),
	)
	return &result, nil
}

func (u *UnifiedLLM) ResolverItensByKeyWords(
	ctx context.Context,
	keywords []LLMKeywordItemResult,
	cardapioReduzido []dto.ProdutoItem,
) ([]dto.ItemCarrinho, error) {
	logger.Info(ctx, "[PROVIDER] ResolverItensByKeyWords",
		zap.Int("keywords_len", len(keywords)),
		zap.Int("cardapio_len", len(cardapioReduzido)),
	)

	if len(keywords) == 0 || len(cardapioReduzido) == 0 {
		return []dto.ItemCarrinho{}, nil
	}

	kwStr := strings.Join((&LLMKeywordResult{Keywords: keywords}).ToString(), ", ")
	lista := formatarCardapioParaPrompt(cardapioReduzido)

	prompt := fmt.Sprintf(PromptResolverByKeywords, kwStr, lista)
	raw, err := u.GenerateResponse(ctx, prompt)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro ResolverItensByKeyWords", zap.Error(err))
		return nil, err
	}

	var parsed LLMResolverResult
	if err := json.Unmarshal([]byte(extractJSON(raw)), &parsed); err != nil {
		logger.Error(ctx, "[PROVIDER] Parse ResolverItensByKeyWords falhou", zap.Error(err), zap.String("raw", raw))
		return []dto.ItemCarrinho{}, nil
	}

	itens := mapResolverToCarrinho(parsed, cardapioReduzido)
	logger.Info(ctx, "[PROVIDER] ResolverItensByKeyWords OK", zap.Int("itens_len", len(itens)))
	return itens, nil
}

func (u *UnifiedLLM) ResolveItemsByMenu(
	ctx context.Context,
	input dto.MessageInput,
	cardapio []dto.ProdutoItem,
	contextoLoja string,
) (*dto.IntencaoCliente, error) {
	logger.Info(ctx, "[PROVIDER] ResolveItemsByMenu",
		zap.String("source", string(input.Source)),
		zap.Int("cardapio_len", len(cardapio)),
	)

	textoBase, err := u.obterTextoBase(ctx, input)
	if err != nil {
		return nil, err
	}

	textoLimpo := u.Higienizar(textoBase)
	if textoLimpo == "" {
		textoLimpo = textoBase
	}

	lista := formatarCardapioParaPrompt(cardapio)
	prompt := fmt.Sprintf(PromptResolveByMenu, contextoLoja, textoLimpo, lista)

	raw, err := u.GenerateResponse(ctx, prompt)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro ResolveItemsByMenu", zap.Error(err))
		return nil, err
	}

	result := struct {
		Acao     string `json:"acao"`
		Resposta string `json:"resposta"`
		Filtro   string `json:"filtro"`
		Itens    []struct {
			ID  int    `json:"id"`
			Qtd int    `json:"qtd"`
			Obs string `json:"obs"`
		} `json:"itens"`
	}{}

	if err := json.Unmarshal([]byte(extractJSON(raw)), &result); err != nil {
		logger.Error(ctx, "[PROVIDER] Parse ResolveItemsByMenu falhou", zap.Error(err), zap.String("raw", raw))
		return &dto.IntencaoCliente{
			Acao:     "visualizar",
			Itens:    []dto.ItemCarrinho{},
			Mensagem: textoBase,
		}, nil
	}

	parsed := LLMResolverResult{Itens: result.Itens}
	itens := mapResolverToCarrinho(parsed, cardapio)

	logger.Info(ctx, "[PROVIDER] ResolveItemsByMenu OK",
		zap.String("acao", result.Acao),
		zap.Int("itens_len", len(itens)),
	)

	return &dto.IntencaoCliente{
		Acao:     result.Acao,
		Resposta: result.Resposta,
		Filtro:   result.Filtro,
		Itens:    itens,
		Mensagem: textoBase,
	}, nil
}

func (u *UnifiedLLM) obterTextoBase(ctx context.Context, input dto.MessageInput) (string, error) {
	switch input.Source {
	case models.SourceAudio:
		if len(input.Audio) == 0 {
			return "", fmt.Errorf("áudio vazio")
		}
		return u.TranscribeAudio(ctx, input.Audio, PromptTranscribeSimple)
	case models.SourceImage:
		if len(input.Image) == 0 {
			return "", fmt.Errorf("imagem vazia")
		}
		return u.DescribeImage(ctx, input.Image, input.Text)
	default:
		return input.Text, nil
	}
}

func mapResolverToCarrinho(res LLMResolverResult, cardapio []dto.ProdutoItem) []dto.ItemCarrinho {
	mapCard := make(map[int]dto.ProdutoItem, len(cardapio))
	for _, p := range cardapio {
		mapCard[int(p.ID)] = p
	}
	var out []dto.ItemCarrinho
	for _, it := range res.Itens {
		if prod, ok := mapCard[it.ID]; ok {
			qtd := it.Qtd
			if qtd <= 0 {
				qtd = 1
			}
			out = append(out, dto.ItemCarrinho{
				ProdutoItem: prod,
				Quantidade:  qtd,
				Observacao:  it.Obs,
				Preco:       prod.Preco,
			})
		}
	}
	return out
}
