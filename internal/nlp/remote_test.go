package nlp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
)

func TestRemoteHTTPEmbedder_Success(t *testing.T) {
	ctx := context.Background()

	var reqCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)

		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := openAIEmbeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Embedding: []float32{3.0, 4.0}, // norm = 5.0, normalized = [0.6, 0.8]
					Index:     0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewRemoteHTTPEmbedder(server.URL, "test-key", "text-embedding-3-small")
	vec, err := embedder.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vec) != 2 {
		t.Fatalf("expected 2 dimensions, got %d", len(vec))
	}

	// 3.0 / 5.0 = 0.6, 4.0 / 5.0 = 0.8
	if vec[0] < 0.59 || vec[0] > 0.61 || vec[1] < 0.79 || vec[1] > 0.81 {
		t.Errorf("unexpected normalized vector: %v", vec)
	}

	if atomic.LoadInt32(&reqCount) != 1 {
		t.Errorf("expected 1 request, got %d", reqCount)
	}
}

func TestRemoteHTTPEmbedder_FallbackOnError(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	fallback := NewLocalHashingEmbedder(64)
	embedder := NewRemoteHTTPEmbedder(server.URL, "test-key", "text-embedding-3-small", WithEmbedderFallback(fallback))

	vec, err := embedder.Embed(ctx, "fallback test query")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if len(vec) != 64 {
		t.Errorf("expected 64 dimensions from fallback embedder, got %d", len(vec))
	}
}

func TestRemoteLLMEscalator_Success(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-llm-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "```json\n{\n  \"resolved_claimed_concepts\": [\"hashmap\", \"frequency_counter\"],\n  \"confidence\": 0.96,\n  \"reasoning\": \"The candidate described tallying items in a dictionary table.\"\n}\n```",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	escalator := NewRemoteLLMEscalator(server.URL, "test-llm-key", "gpt-4o-mini")

	payload := LLMPayload{
		CandidateExplanation: "I stored occurrences in a table and checked for dupes",
		ClaimedConcepts:      []string{},
		DetectedConcepts:     []string{"hashmap"},
	}

	decision, err := escalator.Escalate(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decision.Confidence < 0.95 {
		t.Errorf("expected confidence >= 0.95, got %f", decision.Confidence)
	}
	if len(decision.ResolvedClaimedConcepts) != 2 {
		t.Fatalf("expected 2 resolved concepts, got %d", len(decision.ResolvedClaimedConcepts))
	}
	if decision.ResolvedClaimedConcepts[0] != "hashmap" {
		t.Errorf("expected hashmap, got %s", decision.ResolvedClaimedConcepts[0])
	}
}

func TestCascadeMatcher_MatchWithEscalation(t *testing.T) {
	ctx := context.Background()

	ontology := dsa.NewOntology()
	ontology.AddConcept(&dsa.Concept{
		ID:       "hashmap",
		Name:     "Hash Map",
		Category: "hashing",
		Aliases:  []string{"hashmap", "hash table", "hash map"},
	})
	ontology.AddConcept(&dsa.Concept{
		ID:       "two_pointers",
		Name:     "Two Pointers",
		Category: "two_pointers",
		Aliases:  []string{"two pointers", "pointer pair"},
	})

	var llmCallCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&llmCallCount, 1)

		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "{\"resolved_claimed_concepts\": [\"hashmap\"], \"confidence\": 0.92, \"reasoning\": \"Colloquial lookup description refers to hash table.\"}",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	escalator := NewRemoteLLMEscalator(server.URL, "dummy-key", "gpt-4o-mini")
	matcher, err := NewCascadeMatcher(ctx, ontology, NewLocalHashingEmbedder(64), false)
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}
	matcher.SetEscalator(escalator)

	t.Run("high confidence exact match bypasses LLM escalation", func(t *testing.T) {
		atomic.StoreInt32(&llmCallCount, 0)
		claimed, err := matcher.MatchWithEscalation(ctx, "I solved this using a hashmap for lookups.", []string{"hashmap"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(claimed) == 0 || claimed[0].ConceptID != "hashmap" {
			t.Errorf("expected hashmap claimed, got %v", claimed)
		}

		if atomic.LoadInt32(&llmCallCount) != 0 {
			t.Errorf("expected 0 LLM calls for exact match, got %d", llmCallCount)
		}
	})

	t.Run("colloquial vernacular escalates to LLM", func(t *testing.T) {
		atomic.StoreInt32(&llmCallCount, 0)
		// Explanation contains zero keywords/aliases: "I just used a dictionary lookup table to track things"
		// But AST detected hashmap:
		claimed, err := matcher.MatchWithEscalation(ctx, "I recorded every number in a fast lookup store to find complements.", []string{"hashmap"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if atomic.LoadInt32(&llmCallCount) != 1 {
			t.Errorf("expected 1 LLM escalation call, got %d", llmCallCount)
		}

		found := false
		for _, c := range claimed {
			if c.ConceptID == "hashmap" && c.Stage == "LLM_ESCALATION" {
				found = true
				if c.Confidence < 0.90 {
					t.Errorf("expected high confidence from LLM, got %f", c.Confidence)
				}
			}
		}
		if !found {
			t.Errorf("expected hashmap resolved via LLM_ESCALATION, got %v", claimed)
		}
	})
}
