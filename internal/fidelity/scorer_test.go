package fidelity_test

import (
	"path/filepath"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

func TestScorer(t *testing.T) {
	repoPath := filepath.Join("..", "..", "data", "dsa")
	repo, err := dsa.NewRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to load ontology: %v", err)
	}

	thresholds := fidelity.DefaultThresholds()
	scorer := fidelity.NewScorer(repo.Ontology(), thresholds)

	problem := model.Problem{
		ID:               "two_sum",
		PrimaryConcepts:  []string{"hashmap"},
		RequiredConcepts: []string{"hashmap"},
		OptionalConcepts: []string{"two_pointers"},
	}

	// 1. Strong Match -> ACCEPT
	t.Run("strong match output relevant -> ACCEPT", func(t *testing.T) {
		claimed := []model.ClaimedConcept{
			{ConceptID: "hashmap", Name: "Hash Map"},
		}
		actual := []model.DetectedConcept{
			{
				ConceptID:      "hashmap",
				Name:           "Hash Map",
				Reachable:      true,
				OutputRelevant: true,
			},
		}

		res := scorer.Score(actual, claimed, problem)
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f)", res.Decision, res.Score)
		}
		if res.Score < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.Score)
		}
	})

	// 2. Claimed two concepts, one output relevant, one not output relevant -> REJUSTIFY
	t.Run("two_pointers + non-output-relevant hashmap -> REJUSTIFY", func(t *testing.T) {
		claimed := []model.ClaimedConcept{
			{ConceptID: "two_pointers", Name: "Two Pointers"},
			{ConceptID: "hashmap", Name: "Hash Map"},
		}
		actual := []model.DetectedConcept{
			{
				ConceptID:      "two_pointers",
				Name:           "Two Pointers",
				Reachable:      true,
				OutputRelevant: true,
			},
			{
				ConceptID:      "hashmap",
				Name:           "Hash Map",
				Reachable:      true,
				OutputRelevant: false, // NOT output relevant
			},
		}

		res := scorer.Score(actual, claimed, problem)
		if res.Decision != model.DecisionRejustify {
			t.Errorf("expected REJUSTIFY, got %s (score: %f)", res.Decision, res.Score)
		}
		if res.Score < 0.90 || res.Score >= 0.95 {
			t.Errorf("expected score in [0.90, 0.95), got %f", res.Score)
		}
	})

	// 3. Completely Missing Primary Concept -> REJECT
	t.Run("missing primary concept -> REJECT", func(t *testing.T) {
		claimed := []model.ClaimedConcept{
			{ConceptID: "hashmap", Name: "Hash Map"},
		}
		actual := []model.DetectedConcept{
			{
				ConceptID:      "sorting",
				Name:           "Sorting",
				Reachable:      true,
				OutputRelevant: true,
			},
		}

		res := scorer.Score(actual, claimed, problem)
		if res.Decision != model.DecisionReject {
			t.Errorf("expected REJECT, got %s (score: %f)", res.Decision, res.Score)
		}
		if res.Score >= 0.90 {
			t.Errorf("expected score < 0.90, got %f", res.Score)
		}
	})

	// 4. Primary Concept Matched and Generic Auxiliary Concept Missing -> ACCEPT (No Dilution)
	t.Run("primary matched + missing generic auxiliary concept -> ACCEPT without dilution", func(t *testing.T) {
		claimed := []model.ClaimedConcept{
			{ConceptID: "hashmap", Name: "Hash Map"},
			{ConceptID: "arrays", Name: "Arrays"},
		}
		// Code only detected hashmap; arrays not detected
		actual := []model.DetectedConcept{
			{
				ConceptID:      "hashmap",
				Name:           "Hash Map",
				Reachable:      true,
				OutputRelevant: true,
			},
		}

		res := scorer.Score(actual, claimed, problem)
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.Score, res.Diagnostics)
		}
		if res.Score < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.Score)
		}
	})

	// 5. Hallucinated non-generic algorithm (Trie/Segment Tree) claimed but not implemented -> Penalized
	t.Run("hallucinated non-generic algorithm is penalized and not excused", func(t *testing.T) {
		claimed := []model.ClaimedConcept{
			{ConceptID: "hashmap", Name: "Hash Map"},
			{ConceptID: "segment_tree", Name: "Segment Tree"},
			{ConceptID: "trie", Name: "Trie"},
		}
		actual := []model.DetectedConcept{
			{
				ConceptID:      "hashmap",
				Name:           "Hash Map",
				Reachable:      true,
				OutputRelevant: true,
			},
		}

		res := scorer.Score(actual, claimed, problem)
		if res.Score >= 0.90 {
			t.Errorf("hallucinated segment_tree and trie must NOT be excused, expected score < 0.90, got %f", res.Score)
		}
		if len(res.Missing) != 2 {
			t.Errorf("expected 2 missing concepts, got %d", len(res.Missing))
		}
	})

	// 6. Problem required concept missing from code is penalized even if omitted from explanation
	t.Run("problem required concept missing from code is penalized", func(t *testing.T) {
		strictProblem := model.Problem{
			ID:               "two_sum",
			PrimaryConcepts:  []string{"hashmap"},
			RequiredConcepts: []string{"hashmap"},
			OptionalConcepts: []string{"two_pointers"},
		}
		// Candidate claimed two_pointers and implemented two_pointers, but omitted required hashmap
		claimed := []model.ClaimedConcept{
			{ConceptID: "two_pointers", Name: "Two Pointers"},
		}
		actual := []model.DetectedConcept{
			{
				ConceptID:      "two_pointers",
				Name:           "Two Pointers",
				Reachable:      true,
				OutputRelevant: true,
			},
		}

		res := scorer.Score(actual, claimed, strictProblem)
		if res.Decision == model.DecisionAccept {
			t.Errorf("expected rejection or rejustification due to missing required concept, got ACCEPT (score: %f)", res.Score)
		}
		hasRequiredMissing := false
		for _, m := range res.Missing {
			if m.ConceptID == "hashmap" {
				hasRequiredMissing = true
			}
		}
		if !hasRequiredMissing {
			t.Errorf("expected required concept 'hashmap' to be recorded in missing concepts")
		}
	})

	// 7. Determinism across 20 repeated runs
	t.Run("deterministic score across repeated runs", func(t *testing.T) {
		claimed := []model.ClaimedConcept{
			{ConceptID: "hashmap", Name: "Hash Map"},
			{ConceptID: "two_pointers", Name: "Two Pointers"},
		}
		actual := []model.DetectedConcept{
			{
				ConceptID:      "hashmap",
				Name:           "Hash Map",
				Reachable:      true,
				OutputRelevant: true,
			},
			{
				ConceptID:      "sorting",
				Name:           "Sorting",
				Reachable:      true,
				OutputRelevant: true,
			},
		}

		firstRes := scorer.Score(actual, claimed, problem)
		for i := 0; i < 20; i++ {
			res := scorer.Score(actual, claimed, problem)
			if res.Score != firstRes.Score {
				t.Fatalf("run %d produced score %f, expected deterministic %f", i, res.Score, firstRes.Score)
			}
		}
	})
}
