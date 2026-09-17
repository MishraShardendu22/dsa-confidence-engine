package fidelity

import (
	"fmt"
	"math"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

type FidelityResult struct {
	Score       float64
	Decision    model.Decision
	Matched     []model.ConceptMatch
	Missing     []model.ConceptMatch
	Extra       []model.ConceptMatch
	Diagnostics []string
	Explanation string
}

type Scorer struct {
	ontology   *dsa.Ontology
	thresholds Thresholds
}

func NewScorer(ontology *dsa.Ontology, thresholds Thresholds) *Scorer {
	return &Scorer{
		ontology:   ontology,
		thresholds: thresholds,
	}
}

func (s *Scorer) Score(
	actual []model.DetectedConcept,
	claimed []model.ClaimedConcept,
	problem model.Problem,
) FidelityResult {
	if len(claimed) == 0 {
		return FidelityResult{
			Score:       0.0,
			Decision:    model.DecisionReject,
			Diagnostics: []string{"No DSA concepts were identified in candidate explanation."},
			Explanation: "Candidate did not articulate any recognizable DSA approach.",
		}
	}

	actualMap := make(map[string]model.DetectedConcept, len(actual))
	for _, a := range actual {
		actualMap[a.ConceptID] = a
	}

	primarySet := make(map[string]bool)
	for _, p := range problem.PrimaryConcepts {
		primarySet[p] = true
	}
	for _, r := range problem.RequiredConcepts {
		primarySet[r] = true
	}

	optionalSet := make(map[string]bool)
	for _, o := range problem.OptionalConcepts {
		optionalSet[o] = true
	}

	var matched []model.ConceptMatch
	var missing []model.ConceptMatch
	var extra []model.ConceptMatch
	var diagnostics []string

	var totalWeight float64
	var earnedWeight float64

	claimedMatchedActuals := make(map[string]bool)

	// Preliminary check: determine whether all claimed primary concepts are matched and output-relevant
	primaryClaimedCount := 0
	primaryMatchedCount := 0
	allPrimaryOutputRelevant := true

	for _, c := range claimed {
		if primarySet[c.ConceptID] {
			primaryClaimedCount++
			var foundActual *model.DetectedConcept
			if a, ok := actualMap[c.ConceptID]; ok {
				foundActual = &a
			} else {
				for actID, a := range actualMap {
					if s.ontology != nil && (s.ontology.IsAncestor(actID, c.ConceptID) || s.ontology.IsAncestor(c.ConceptID, actID)) {
						foundActual = &a
						break
					}
				}
			}
			if foundActual != nil && foundActual.Reachable {
				if foundActual.OutputRelevant {
					primaryMatchedCount++
				} else {
					allPrimaryOutputRelevant = false
				}
			} else {
				allPrimaryOutputRelevant = false
			}
		}
	}

	allPrimaryFullyMatched := primaryClaimedCount > 0 && primaryMatchedCount == primaryClaimedCount && allPrimaryOutputRelevant

	// 1. Evaluate claimed concepts
	for _, c := range claimed {
		role := model.RoleSupporting
		weight := 1.0

		if primarySet[c.ConceptID] {
			role = model.RolePrimary
			weight = 2.0
		} else if optionalSet[c.ConceptID] {
			role = model.RoleSupporting
			weight = 1.0
		} else {
			role = model.RoleAuxiliary
			weight = 0.5
		}

		// Search for actual match directly or hierarchically
		var foundActual *model.DetectedConcept
		if a, ok := actualMap[c.ConceptID]; ok {
			foundActual = &a
		} else {
			for actID, a := range actualMap {
				if s.ontology != nil && (s.ontology.IsAncestor(actID, c.ConceptID) || s.ontology.IsAncestor(c.ConceptID, actID)) {
					foundActual = &a
					break
				}
			}
		}

		if foundActual != nil {
			claimedMatchedActuals[foundActual.ConceptID] = true
			totalWeight += weight

			if foundActual.Reachable && foundActual.OutputRelevant {
				earnedWeight += weight
				matched = append(matched, model.ConceptMatch{
					ConceptID:      c.ConceptID,
					Name:           c.Name,
					Role:           role,
					OutputRelevant: true,
					Contribution:   1.0,
					Notes:          "Confirmed in code and output-relevant",
				})
			} else if foundActual.Reachable && !foundActual.OutputRelevant {
				// Present and executes, but not output relevant
				partial := weight * 0.85
				earnedWeight += partial
				matched = append(matched, model.ConceptMatch{
					ConceptID:      c.ConceptID,
					Name:           c.Name,
					Role:           role,
					OutputRelevant: false,
					Contribution:   0.85,
					Notes:          "Present in reachable code but not output-relevant",
				})
				diagnostics = append(diagnostics, fmt.Sprintf(
					"Candidate claims %s. Detected in reachable code, but static analysis shows it does not contribute to the returned value.",
					c.ConceptID,
				))
			} else {
				// Found only in dead/unreachable code
				missing = append(missing, model.ConceptMatch{
					ConceptID:      c.ConceptID,
					Name:           c.Name,
					Role:           role,
					OutputRelevant: false,
					Contribution:   0.0,
					Notes:          "Found only in unreachable/dead code",
				})
				diagnostics = append(diagnostics, fmt.Sprintf(
					"Candidate claims %s, but evidence exists only in unreachable dead code.",
					c.ConceptID,
				))
			}
		} else {
			// Concept not implemented in code
			if allPrimaryFullyMatched && (role == model.RoleAuxiliary || isGenericConcept(c.ConceptID)) {
				missing = append(missing, model.ConceptMatch{
					ConceptID:      c.ConceptID,
					Name:           c.Name,
					Role:           role,
					OutputRelevant: false,
					Contribution:   0.0,
					Notes:          "Generic concept omitted or implicit; primary approach fully verified",
				})
				diagnostics = append(diagnostics, fmt.Sprintf(
					"Candidate mentioned %s in explanation; core primary strategy is fully verified in code.",
					c.Name,
				))
			} else {
				totalWeight += weight
				missing = append(missing, model.ConceptMatch{
					ConceptID:      c.ConceptID,
					Name:           c.Name,
					Role:           role,
					OutputRelevant: false,
					Contribution:   0.0,
					Notes:          "Not found in code",
				})
				diagnostics = append(diagnostics, fmt.Sprintf(
					"Candidate claims %s, but no corresponding operations were detected in the source code.",
					c.ConceptID,
				))
			}
		}
	}

	// 2. Evaluate actual concepts that were NOT claimed
	var extraPenalty float64
	for _, a := range actual {
		if !a.Reachable || !a.OutputRelevant {
			continue
		}
		if claimedMatchedActuals[a.ConceptID] {
			continue
		}

		// Check if a belongs to the same ontological family as a claimed concept
		isFamilyMatched := false
		for _, c := range claimed {
			if s.ontology != nil && (s.ontology.IsAncestor(c.ConceptID, a.ConceptID) || s.ontology.IsAncestor(a.ConceptID, c.ConceptID)) {
				isFamilyMatched = true
				break
			}
		}
		if isFamilyMatched {
			continue
		}

		if primarySet[a.ConceptID] && !isGenericConcept(a.ConceptID) {
			extraPenalty += 0.15
			extra = append(extra, model.ConceptMatch{
				ConceptID:      a.ConceptID,
				Name:           a.Name,
				Role:           model.RolePrimary,
				OutputRelevant: a.OutputRelevant,
				Contribution:   -0.15,
				Notes:          "Unclaimed primary concept used in solution",
			})
			diagnostics = append(diagnostics, fmt.Sprintf(
				"Code heavily utilizes %s, but this was omitted from the candidate explanation.",
				a.ConceptID,
			))
		} else {
			extra = append(extra, model.ConceptMatch{
				ConceptID:      a.ConceptID,
				Name:           a.Name,
				Role:           model.RoleAuxiliary,
				OutputRelevant: a.OutputRelevant,
				Contribution:   0.0,
				Notes:          "Auxiliary concept present in code",
			})
		}
	}

	var rawScore float64
	if totalWeight > 0 {
		rawScore = (earnedWeight / totalWeight) - extraPenalty
	}
	if rawScore < 0.0 {
		rawScore = 0.0
	}
	if rawScore > 1.0 {
		rawScore = 1.0
	}

	// Round to 3 decimal places
	finalScore := math.Round(rawScore*1000) / 1000

	decision := DetermineDecision(finalScore, s.thresholds)

	var explanation string
	switch decision {
	case model.DecisionAccept:
		explanation = fmt.Sprintf("High approach fidelity (score %.3f >= %.2f). Candidate implemented the claimed DSA approach with verified output relevance.", finalScore, s.thresholds.AcceptThreshold)
	case model.DecisionRejustify:
		explanation = fmt.Sprintf("Marginal approach fidelity (score %.3f between %.2f and %.2f). Claimed concepts were detected, but some lack direct output relevance or have minor structural divergences.", finalScore, s.thresholds.RejustifyThreshold, s.thresholds.AcceptThreshold)
	case model.DecisionReject:
		explanation = fmt.Sprintf("Fidelity mismatch (score %.3f < %.2f). Primary claimed concepts were missing from the implementation or contradictory algorithms were detected.", finalScore, s.thresholds.RejustifyThreshold)
	}

	return FidelityResult{
		Score:       finalScore,
		Decision:    decision,
		Matched:     matched,
		Missing:     missing,
		Extra:       extra,
		Diagnostics: diagnostics,
		Explanation: explanation,
	}
}

// isGenericConcept identifies foundational sequence, collection, or syntax constructs
// that represent context rather than high-level algorithmic strategies.
func isGenericConcept(id string) bool {
	switch id {
	case "arrays", "iteration", "strings", "primitives", "linear_scan", "variables":
		return true
	default:
		return false
	}
}
