package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AIClient defines the interface for calling an LLM to generate team suggestions.
type AIClient interface {
	GenerateTeamSuggestion(ctx context.Context, prompt string) (string, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// Gemini REST client (no SDK dependency — uses the v1beta generateContent API)
// ─────────────────────────────────────────────────────────────────────────────

const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
	aiTimeout     = 45 * time.Second
)

// GeminiClient calls Google Gemini via the REST API.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewGeminiClient creates a production-ready Gemini client.
func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: aiTimeout,
		},
	}
}

// geminiRequest is the request body shape for the Gemini generateContent API.
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// geminiResponse is the subset of the Gemini REST response we need.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// GenerateTeamSuggestion sends prompt to Gemini and returns the text response.
func (g *GeminiClient) GenerateTeamSuggestion(ctx context.Context, prompt string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("ai_client: GEMINI_API_KEY is not configured")
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.4, // Low temp for deterministic FPL advice
			MaxOutputTokens: 1024,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ai_client: marshal request: %w", err)
	}

	url := fmt.Sprintf(geminiBaseURL, g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ai_client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai_client: http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ai_client: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai_client: gemini API error (HTTP %d): %s", resp.StatusCode, string(raw))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(raw, &gemResp); err != nil {
		return "", fmt.Errorf("ai_client: decode response: %w", err)
	}

	if gemResp.PromptFeedback != nil && gemResp.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("ai_client: prompt blocked by Gemini: %s", gemResp.PromptFeedback.BlockReason)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("ai_client: empty response from Gemini")
	}

	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

// NoOpClient is used when no API key is configured — returns a placeholder.
type NoOpClient struct{}

func NewNoOpClient() *NoOpClient { return &NoOpClient{} }

func (n *NoOpClient) GenerateTeamSuggestion(_ context.Context, _ string) (string, error) {
	return "⚠️ AI suggestions require a GEMINI_API_KEY to be configured in your .env file.", nil
}
