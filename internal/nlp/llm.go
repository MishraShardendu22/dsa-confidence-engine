package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

var ErrLLMDisabled = errors.New("LLM escalation is disabled")

type LLMPayload struct {
	CandidateExplanation string   `json:"candidate_explanation"`
	ClaimedConcepts      []string `json:"claimed_concepts"`
	DetectedConcepts     []string `json:"detected_concepts"`
	Evidence             []string `json:"evidence"`
}

type LLMDecision struct {
	ResolvedClaimedConcepts []string `json:"resolved_claimed_concepts"`
	Confidence              float64  `json:"confidence"`
	Reasoning               string   `json:"reasoning"`
}

type LLMEscalator interface {
	Escalate(ctx context.Context, payload LLMPayload) (*LLMDecision, error)
}

type DisabledLLMEscalator struct{}

func (d *DisabledLLMEscalator) Escalate(ctx context.Context, payload LLMPayload) (*LLMDecision, error) {
	return nil, ErrLLMDisabled
}

func FormatEvidenceSummaries(evidence []model.Evidence) []string {
	var summaries []string
	for _, e := range evidence {
		if e.Reachable {
			summaries = append(summaries, e.Description)
		}
	}
	return summaries
}

// RemoteLLMEscalator evaluates borderline or ambiguous semantic claims using
// a remote OpenAI/LiteLLM/Ollama chat completions endpoint with zero local model weights.
type RemoteLLMEscalator struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
	fallback LLMEscalator
}

type RemoteLLMOption func(*RemoteLLMEscalator)

func WithLLMHTTPClient(client *http.Client) RemoteLLMOption {
	return func(r *RemoteLLMEscalator) {
		if client != nil {
			r.client = client
		}
	}
}

func WithLLMFallback(fallback LLMEscalator) RemoteLLMOption {
	return func(r *RemoteLLMEscalator) {
		r.fallback = fallback
	}
}

// NewRemoteLLMEscalator initializes a RemoteLLMEscalator instance.
func NewRemoteLLMEscalator(endpoint, apiKey, modelName string, opts ...RemoteLLMOption) *RemoteLLMEscalator {
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}
	r := &RemoteLLMEscalator{
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		model:    modelName,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float32                 `json:"temperature"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (r *RemoteLLMEscalator) Escalate(ctx context.Context, payload LLMPayload) (*LLMDecision, error) {
	if r.endpoint == "" {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, errors.New("remote LLM endpoint is empty and no fallback configured")
	}

	sysPrompt := "You are a DSA concept fidelity evaluator. You evaluate whether a candidate's spoken/informal explanation correctly describes the algorithmic approach implemented in code. Output valid JSON only with keys: 'resolved_claimed_concepts' (list of valid concept IDs matching the explanation), 'confidence' (float 0.0 to 1.0), and 'reasoning' (brief string explanation)."

	userContentBytes, _ := json.Marshal(payload)

	reqPayload := chatCompletionRequest{
		Model: r.model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: string(userContentBytes)},
		},
		Temperature: 0.0,
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("failed to marshal chat completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
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
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("llm escalation request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("llm escalation remote error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("failed to unmarshal chat response: %w", err)
	}

	if chatResp.Error != nil {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("llm api error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, errors.New("no completion choices returned from remote LLM")
	}

	rawContent := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Strip optional markdown code fences
	if strings.HasPrefix(rawContent, "```") {
		lines := strings.Split(rawContent, "\n")
		if len(lines) >= 2 {
			if strings.HasPrefix(lines[0], "```") {
				lines = lines[1:]
			}
			if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			rawContent = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}

	var decision LLMDecision
	if err := json.Unmarshal([]byte(rawContent), &decision); err != nil {
		if r.fallback != nil {
			return r.fallback.Escalate(ctx, payload)
		}
		return nil, fmt.Errorf("failed to parse structured LLM decision JSON: %w (raw: %s)", err, rawContent)
	}

	return &decision, nil
}
