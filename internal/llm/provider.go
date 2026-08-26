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

type LLMKeywordItemResult struct {
	Nome       string  `json:"nome"`
	Quantidade float32 `json:"qtd"`
}

// ============ STRUCTS - DONO É O PROVIDER ============
type LLMKeywordResult struct {
	Keywords []LLMKeywordItemResult `json:"keywords"`
}

// ToString formats the keyword result as a list of strings.
func (l *LLMKeywordResult) ToString() []string {
	var result []string
	for _, item := range l.Keywords {
		result = append(result, fmt.Sprintf("Keywords:%s  Quantidade:%g", item.Nome, item.Quantidade))
	}
	return result
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
		logger.Info(
			context.Background(), "[PROVIDER] textClient OK",
			zap.String("provider", textClient.GetProvider()),
			zap.String("model", textClient.GetModel()),
		)
	}
	if audioClient != nil {
		logger.Info(
			context.Background(), "[PROVIDER] audioClient OK",
			zap.String("provider", audioClient.GetProvider()),
		)
	}
	if visionClient != nil {
		logger.Info(
			context.Background(), "[PROVIDER] visionClient OK",
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

// ============ IMPLEMENTA LLMClient ============
func (u *UnifiedLLM) GetProvider() string { return "unified" }
func (u *UnifiedLLM) GetModel() string    { return "router" }

func (u *UnifiedLLM) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	logger.Info(
		ctx, "[PROVIDER] GenerateResponse -> textClient",
		zap.Int("prompt_len", len(prompt)),
	)
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
	logger.Info(
		ctx, "[PROVIDER] TranscribeAudio -> audioClient",
		zap.Int("audio_len", len(audio)),
	)
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
	logger.Info(
		ctx, "[PROVIDER] DescribeImage -> visionClient",
		zap.Int("image_len", len(image)),
	)
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

// ============ ENTRADA DO HANDLER ============
func (u *UnifiedLLM) ExtractIntent(ctx context.Context, input dto.MessageInput, cardapio []dto.ProdutoItem) (*dto.IntencaoCliente, error) {
	logger.Info(
		ctx, "[PROVIDER] ExtractIntent entrada",
		zap.String("source", string(input.Source)),
		zap.Int("cardapio_len", len(cardapio)),
	)

	var textoBase string
	switch input.Source {
	case models.SourceAudio:
		logger.LogInfo("[PROVIDER] Source=audio")
		t, err := u.TranscribeAudio(ctx, input.Audio, PromptTranscribeSimple)
		if err != nil {
			return nil, err
		}
		textoBase = t

	case models.SourceImage:
		logger.LogInfo("[PROVIDER] Source=image")
		d, err := u.DescribeImage(ctx, input.Image, input.Text)
		if err != nil {
			return nil, err
		}
		textoBase = d

	default:
		logger.LogInfo("[PROVIDER] Source=text")
		textoBase = input.Text
	}

	logger.Info(ctx, "[PROVIDER] Texto base", zap.String("texto_base", textoBase))

	// HIGIENIZAÇÃO - PRIMEIRA COISA APÓS TRANSCRIBE/DESCRIBE
	pp := u.preprocessor.Process(textoBase)
	textoLimpo := pp.Cleaned
	logger.Info(
		ctx, "[PROVIDER] Texto higienizado",
		zap.String("original", pp.Original),
		zap.String("cleaned", textoLimpo),
	)

	// Usa textoLimpo pra keywords, mantém textoBase como Mensagem original
	textoParaKeywords := textoLimpo
	if textoParaKeywords == "" {
		textoParaKeywords = textoBase
	}

	// Keywords via provider da origem
	var keywordClient LLMClient
	switch input.Source {
	case models.SourceAudio:
		keywordClient = u.audioClient
	case models.SourceImage:
		keywordClient = u.visionClient
	default:
		keywordClient = u.textClient
	}

	promptKw := fmt.Sprintf(PromptExtractKeywords, textoParaKeywords)
	rawKw, err := keywordClient.GenerateResponse(ctx, promptKw)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro keywords", zap.Error(err))
		return nil, err
	}

	kwRes := LLMKeywordResult{}
	if err := json.Unmarshal([]byte(extractJSON(rawKw)), &kwRes); err != nil {
		logger.Error(ctx, "[PROVIDER] Parse keywords falhou", zap.Error(err))
		kwRes.Keywords = []LLMKeywordItemResult{}
	}
	logger.Info(ctx, "[PROVIDER] Keywords", zap.Any("keywords", kwRes.Keywords))

	if len(kwRes.Keywords) == 0 {
		return &dto.IntencaoCliente{Acao: "visualizar", Itens: []dto.ItemCarrinho{}, Mensagem: textoBase}, nil
	}

	// Resolve IDs sempre texto
	lista := formatCardapioForPrompt(cardapio)
	promptIDs := fmt.Sprintf(PromptResolverIDs, lista, strings.Join(kwRes.ToString(), ", "), lista)
	rawIds, err := u.textClient.GenerateResponse(ctx, promptIDs)
	if err != nil {
		logger.Error(ctx, "[PROVIDER] Erro resolver IDs", zap.Error(err))
		return nil, err
	}

	idRes := LLMResolverResult{}
	if err := json.Unmarshal([]byte(extractJSON(rawIds)), &idRes); err != nil {
		logger.Error(ctx, "[PROVIDER] Parse IDs falhou", zap.Error(err))
	}

	itens := mapResolverToCarrinho(idRes, cardapio)
	logger.Info(ctx, "[PROVIDER] IDs resolvidos", zap.Int("itens_len", len(itens)))

	return &dto.IntencaoCliente{Acao: "adicionar", Itens: itens, Mensagem: textoBase}, nil
}

func formatCardapioForPrompt(cardapio []dto.ProdutoItem) string {
	var sb strings.Builder
	for _, p := range cardapio {
		sb.WriteString(fmt.Sprintf("%d - %s", p.ID, p.Nome))
	}
	return sb.String()
}

func mapResolverToCarrinho(res LLMResolverResult, cardapio []dto.ProdutoItem) []dto.ItemCarrinho {
	mapCard := make(map[int]dto.ProdutoItem)
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
			out = append(out, dto.ItemCarrinho{ProdutoItem: prod, Quantidade: qtd, Observacao: it.Obs, Preco: prod.Preco})
		}
	}
	return out
}
