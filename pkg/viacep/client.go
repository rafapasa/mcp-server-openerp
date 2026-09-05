package viacep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var cepPattern = regexp.MustCompile(`^\d{5}-?\d{3}$`)

// Endereco representa um endereço retornado pelo serviço ViaCEP.
type Endereco struct {
	CEP        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Cidade     string `json:"localidade"`
	Estado     string `json:"uf"`
	Erro       bool   `json:"erro"`
}

// Client consulta e normaliza endereços por CEP.
type Client interface {
	Buscar(ctx context.Context, cep string) (*Endereco, error)
}

type httpClient struct {
	baseURL string
	client  *http.Client
}

// NewClient cria um cliente ViaCEP com a URL padrão do serviço.
func NewClient() Client {
	return &httpClient{
		baseURL: "https://viacep.com.br/ws",
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// NewHTTPClient cria um cliente ViaCEP usando uma URL e um cliente HTTP personalizados.
func NewHTTPClient(baseURL string, client *http.Client) Client {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &httpClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

// Buscar consulta e normaliza o endereço correspondente ao CEP informado.
func (c *httpClient) Buscar(ctx context.Context, cep string) (*Endereco, error) {
	cep = strings.TrimSpace(cep)
	if !cepPattern.MatchString(cep) {
		return nil, fmt.Errorf("CEP inválido")
	}
	cepNumerico := strings.ReplaceAll(cep, "-", "")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+cepNumerico+"/json/", nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar consulta ViaCEP: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar ViaCEP: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ViaCEP retornou status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta ViaCEP: %w", err)
	}
	var endereco Endereco
	if err := json.Unmarshal(body, &endereco); err != nil {
		return nil, fmt.Errorf("erro ao interpretar resposta ViaCEP: %w", err)
	}
	if endereco.Erro || endereco.Cidade == "" || endereco.Estado == "" {
		return nil, fmt.Errorf("CEP não encontrado")
	}
	endereco.CEP = formatarCEP(cepNumerico)
	endereco.Logradouro = normalizarNome(endereco.Logradouro)
	endereco.Bairro = normalizarNome(endereco.Bairro)
	endereco.Cidade = normalizarNome(endereco.Cidade)
	endereco.Estado = strings.ToUpper(endereco.Estado)
	return &endereco, nil
}

func formatarCEP(cep string) string {
	return cep[:5] + "-" + cep[5:]
}

func normalizarNome(nome string) string {
	return cases.Title(language.Portuguese).String(strings.ToLower(strings.TrimSpace(nome)))
}
