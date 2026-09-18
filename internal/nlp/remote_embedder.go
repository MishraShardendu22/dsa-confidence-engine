package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// RemoteHTTPEmbedder queries a remote embedding API (OpenAI, LiteLLM, Ollama, Voyage)
// using standard HTTP POST requests with zero local model weights.
type RemoteHTTPEmbedder struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
	fallback Embedder
}

type RemoteEmbeddingOption func(*RemoteHTTPEmbedder)

func WithEmbedderHTTPClient(client *http.Client) RemoteEmbeddingOption {
	return func(r *RemoteHTTPEmbedder) {
		if client != nil {
			r.client = client
		}
	}
}

func WithEmbedderFallback(fallback Embedder) RemoteEmbeddingOption {
	return func(r *RemoteHTTPEmbedder) {
		r.fallback = fallback
	}
}

// NewRemoteHTTPEmbedder creates a new RemoteHTTPEmbedder.
func NewRemoteHTTPEmbedder(endpoint, apiKey, modelName string, opts ...RemoteEmbeddingOption) *RemoteHTTPEmbedder {
	if modelName == "" {
		modelName = "text-embedding-3-small"
	}
	r := &RemoteHTTPEmbedder{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		model:    modelName,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

type openAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (r *RemoteHTTPEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if r.endpoint == "" {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, errors.New("remote embedding endpoint is empty and no fallback configured")
	}

	reqBody := openAIEmbeddingRequest{
		Input: text,
		Model: r.model,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("embedding http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("embedding remote error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var parsed openAIEmbeddingResponse
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("failed to unmarshal embedding response: %w", err)
	}

	if parsed.Error != nil {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, fmt.Errorf("embedding api returned error: %s", parsed.Error.Message)
	}

	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		if r.fallback != nil {
			return r.fallback.Embed(ctx, text)
		}
		return nil, errors.New("empty embedding vector returned from remote endpoint")
	}

	vec := parsed.Data[0].Embedding

	// L2 Unit Normalization
	var sumSq float64
	for _, v := range vec {
		sumSq += float64(v * v)
	}
	if sumSq > 0 {
		norm := float32(math.Sqrt(sumSq))
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec, nil
}
