// internal/media/groq_transcriber.go
package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/rafapasa/mcp-server-openerp/internal/config"
)

type GroqTranscriber struct {
	apiKey     string
	httpClient *http.Client
	cfg        *config.Config
}

func NewGroqTranscriber(cfg *config.Config) *GroqTranscriber {
	return &GroqTranscriber{
		apiKey:     cfg.LlmGroqApiKey,
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

func (t *GroqTranscriber) Transcribe(ctx context.Context, audioBytes []byte) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não configurada")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "audio.ogg")
	if err != nil {
		return "", err
	}
	part.Write(audioBytes)
	_ = writer.WriteField("model", t.cfg.LlmGroqWhisperModel)
	_ = writer.WriteField("language", "pt")
	_ = writer.WriteField("response_format", "json")
	writer.Close()

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/audio/transcriptions", body)
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("groq whisper %d: %s", resp.StatusCode, string(bb))
	}
	var r struct {
		Text string `json:"text"`
	}
	json.Unmarshal(bb, &r)
	return r.Text, nil
}
