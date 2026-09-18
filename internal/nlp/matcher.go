package nlp

import (
	"context"
	"sort"
	"strings"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

type ConceptEmbedding struct {
	Concept *dsa.Concept
	Vector  []float32
}

type CascadeMatcher struct {
	ontology          *dsa.Ontology
	embedder          Embedder
	escalator         LLMEscalator
	sortedAliases     []aliasEntry
	conceptEmbeddings []ConceptEmbedding
	embeddingEnabled  bool
}

type aliasEntry struct {
	phrase  string
	concept *dsa.Concept
}

func NewCascadeMatcher(ctx context.Context, ontology *dsa.Ontology, embedder Embedder, embeddingEnabled bool) (*CascadeMatcher, error) {
	matcher := &CascadeMatcher{
		ontology:         ontology,
		embedder:         embedder,
		embeddingEnabled: embeddingEnabled,
	}

	allAliases := ontology.AllAliases()
	for alias, c := range allAliases {
		matcher.sortedAliases = append(matcher.sortedAliases, aliasEntry{
			phrase:  alias,
			concept: c,
		})
	}
	sort.Slice(matcher.sortedAliases, func(i, j int) bool {
		return len(matcher.sortedAliases[i].phrase) > len(matcher.sortedAliases[j].phrase)
	})

	if embeddingEnabled && embedder != nil {
		allConcepts := ontology.AllConcepts()
		for _, c := range allConcepts {
			parts := []string{c.Name, c.Description}
			parts = append(parts, c.Aliases...)
			parts = append(parts, c.ExplanationExamples...)
			text := strings.Join(parts, " ")

			vec, err := embedder.Embed(ctx, text)
			if err == nil {
				matcher.conceptEmbeddings = append(matcher.conceptEmbeddings, ConceptEmbedding{
					Concept: c,
					Vector:  vec,
				})
			}
		}
	}

	return matcher, nil
}

func isNegatedOccurrence(fullNormText string, matchIdx int) bool {
	prefix := fullNormText[:matchIdx]
	words := strings.Fields(prefix)
	if len(words) == 0 {
		return false
	}

	windowSize := 5
	if len(words) < windowSize {
		windowSize = len(words)
	}
	recentWords := words[len(words)-windowSize:]

	for i, w := range recentWords {
		switch w {
		case "not", "no", "never", "didnt", "dont", "wont", "cant", "cannot", "without", "avoid", "avoided", "avoiding", "neither", "nor", "against":
			if i+1 < len(recentWords) && recentWords[i+1] == "only" {
				continue
			}
			return true
		case "instead":
			if i+1 < len(recentWords) && recentWords[i+1] == "of" {
				return true
			}
		case "rather":
			if i+1 < len(recentWords) && recentWords[i+1] == "than" {
				return true
			}
		}
	}
	return false
}

func (m *CascadeMatcher) Match(ctx context.Context, text string) ([]model.ClaimedConcept, error) {
	normText := dsa.NormalizeText(text)
	claimedMap := make(map[string]*model.ClaimedConcept)

	// Stage 1 & 2: Exact keyword & Alias matching (longest phrase priority)
	type matchedSpan struct {
		start     int
		end       int
		conceptID string
	}
	var occupiedSpans []matchedSpan

	textPadded := " " + normText + " "
	for _, entry := range m.sortedAliases {
		needle := " " + entry.phrase + " "
		if !strings.Contains(textPadded, needle) {
			continue
		}

		// Check all occurrences for non-negated usage and non-overlapping with already matched longer phrases
		hasPositiveOccurrence := false
		var newSpans []matchedSpan
		searchStart := 0
		for {
			idx := strings.Index(textPadded[searchStart:], needle)
			if idx == -1 {
				break
			}
			actualIdx := searchStart + idx
			matchEnd := actualIdx + len(needle)

			// Check if this occurrence is contained within an already matched longer phrase from an unrelated taxonomy branch
			isSubsumed := false
			for _, span := range occupiedSpans {
				if actualIdx >= span.start && matchEnd <= span.end {
					isStrictSubSpan := (actualIdx > span.start || matchEnd < span.end)
					if isStrictSubSpan && entry.concept.ID != span.conceptID &&
						!m.ontology.IsAncestor(entry.concept.ID, span.conceptID) &&
						!m.ontology.IsAncestor(span.conceptID, entry.concept.ID) {
						isSubsumed = true
						break
					}
				}
			}

			if !isSubsumed && !isNegatedOccurrence(textPadded, actualIdx) {
				hasPositiveOccurrence = true
				newSpans = append(newSpans, matchedSpan{
					start:     actualIdx,
					end:       matchEnd,
					conceptID: entry.concept.ID,
				})
			}
			searchStart = actualIdx + len(needle)
		}

		if hasPositiveOccurrence {
			occupiedSpans = append(occupiedSpans, newSpans...)
			if _, exists := claimedMap[entry.concept.ID]; !exists {
				claimedMap[entry.concept.ID] = &model.ClaimedConcept{
					ConceptID:     entry.concept.ID,
					Name:          entry.concept.Name,
					Category:      entry.concept.Category,
					Confidence:    1.0,
					MatchedPhrase: entry.phrase,
					Stage:         "ALIAS",
					Role:          model.RolePrimary,
				}
			}
		}
	}

	// Stage 3: Local Embedding Similarity
	if m.embeddingEnabled && m.embedder != nil && len(m.conceptEmbeddings) > 0 {
		queryVec, err := m.embedder.Embed(ctx, text)
		if err == nil {
			for _, ce := range m.conceptEmbeddings {
				sim := CosineSimilarity(queryVec, ce.Vector)
				if sim >= 0.70 {
					if existing, exists := claimedMap[ce.Concept.ID]; !exists {
						claimedMap[ce.Concept.ID] = &model.ClaimedConcept{
							ConceptID:     ce.Concept.ID,
							Name:          ce.Concept.Name,
							Category:      ce.Concept.Category,
							Confidence:    sim,
							MatchedPhrase: "semantic embedding match",
							Stage:         "EMBEDDING",
							Role:          model.RolePrimary,
						}
					} else if existing.Confidence < sim {
						existing.Confidence = sim
					}
				}
			}
		}
	}

	results := make([]model.ClaimedConcept, 0, len(claimedMap))
	for _, c := range claimedMap {
		results = append(results, *c)
	}

	// Sort results deterministically
	sort.Slice(results, func(i, j int) bool {
		if results[i].Confidence != results[j].Confidence {
			return results[i].Confidence > results[j].Confidence
		}
		return results[i].ConceptID < results[j].ConceptID
	})

	return results, nil
}

func (m *CascadeMatcher) SetEscalator(escalator LLMEscalator) {
	m.escalator = escalator
}

// MatchWithEscalation performs cascade keyword + alias + embedding matching first.
// If no primary concepts were claimed or confidence is borderline, and code AST detected concepts,
// it escalates to the configured LLMEscalator for semantic resolution of informal vernacular.
func (m *CascadeMatcher) MatchWithEscalation(ctx context.Context, text string, detectedConcepts []string, evidence []model.Evidence) ([]model.ClaimedConcept, error) {
	claimed, err := m.Match(ctx, text)
	if err != nil {
		return nil, err
	}

	// If exact/embedding matching already found primary claimed concepts with high confidence (>= 0.85),
	// return immediately with zero LLM overhead.
	hasHighConfidencePrimary := false
	for _, c := range claimed {
		if c.Role == model.RolePrimary && c.Confidence >= 0.85 {
			hasHighConfidencePrimary = true
			break
		}
	}

	if hasHighConfidencePrimary || m.escalator == nil {
		return claimed, nil
	}

	var claimedIDs []string
	for _, c := range claimed {
		claimedIDs = append(claimedIDs, c.ConceptID)
	}

	payload := LLMPayload{
		CandidateExplanation: text,
		ClaimedConcepts:      claimedIDs,
		DetectedConcepts:     detectedConcepts,
		Evidence:             FormatEvidenceSummaries(evidence),
	}

	decision, err := m.escalator.Escalate(ctx, payload)
	if err != nil {
		return claimed, nil
	}

	if decision != nil && decision.Confidence >= 0.70 {
		claimedMap := make(map[string]model.ClaimedConcept)
		for _, c := range claimed {
			claimedMap[c.ConceptID] = c
		}

		for _, conceptID := range decision.ResolvedClaimedConcepts {
			name := conceptID
			category := "general"
			role := model.RolePrimary
			if m.ontology != nil {
				if concept, found := m.ontology.FindConcept(conceptID); found && concept != nil {
					name = concept.Name
					category = concept.Category
				}
			}

			if existing, exists := claimedMap[conceptID]; !exists {
				claimedMap[conceptID] = model.ClaimedConcept{
					ConceptID:     conceptID,
					Name:          name,
					Category:      category,
					Confidence:    decision.Confidence,
					MatchedPhrase: "llm escalation resolution: " + decision.Reasoning,
					Stage:         "LLM_ESCALATION",
					Role:          role,
				}
			} else if existing.Confidence < decision.Confidence {
				existing.Confidence = decision.Confidence
				existing.Stage = "LLM_ESCALATION"
				claimedMap[conceptID] = existing
			}
		}

		results := make([]model.ClaimedConcept, 0, len(claimedMap))
		for _, c := range claimedMap {
			results = append(results, c)
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].Confidence != results[j].Confidence {
				return results[i].Confidence > results[j].Confidence
			}
			return results[i].ConceptID < results[j].ConceptID
		})
		return results, nil
	}

	return claimed, nil
}
