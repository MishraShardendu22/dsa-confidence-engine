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
	textPadded := " " + normText + " "
	for _, entry := range m.sortedAliases {
		needle := " " + entry.phrase + " "
		if !strings.Contains(textPadded, needle) {
			continue
		}

		// Check all occurrences for non-negated usage
		hasPositiveOccurrence := false
		searchStart := 0
		for {
			idx := strings.Index(textPadded[searchStart:], needle)
			if idx == -1 {
				break
			}
			actualIdx := searchStart + idx
			if !isNegatedOccurrence(textPadded, actualIdx) {
				hasPositiveOccurrence = true
				break
			}
			searchStart = actualIdx + len(needle)
		}

		if hasPositiveOccurrence {
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
