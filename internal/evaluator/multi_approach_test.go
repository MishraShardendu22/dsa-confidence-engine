package evaluator_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
)

type ApproachVariation struct {
	Name               string   `json:"name"`
	Type               string   `json:"type"` // OPTIMAL, ALTERNATIVE, FAILING, BLUFFING, DEAD_CODE
	Code               string   `json:"code"`
	Explanation        string   `json:"explanation"`
	ExpectedTestStatus string   `json:"expected_test_status"` // PASS or FAIL
	ExpectedDecision   string   `json:"expected_decision"`    // ACCEPT or REJECT
	MinFidelityScore   *float64 `json:"min_fidelity_score,omitempty"`
	MaxFidelityScore   *float64 `json:"max_fidelity_score,omitempty"`
}

type BenchmarkSuiteItem struct {
	ProblemID  string              `json:"problem_id"`
	Category   string              `json:"category"`
	Name       string              `json:"name"`
	Approaches []ApproachVariation `json:"approaches"`
}

func TestMultiApproachRigorousEvaluation(t *testing.T) {
	// 1. Initialize Evaluator
	dsaRepoPath := filepath.Join("..", "..", "data", "dsa")
	dsaRepo, err := dsa.NewRepository(dsaRepoPath)
	if err != nil {
		t.Fatalf("failed to load ontology: %v", err)
	}

	embedder := nlp.NewLocalHashingEmbedder(256)
	ctx := context.Background()

	matcher, err := nlp.NewCascadeMatcher(ctx, dsaRepo.Ontology(), embedder, true)
	if err != nil {
		t.Fatalf("failed to init cascade matcher: %v", err)
	}

	r := runner.NewLocalRunner(3000)
	scriptPath := filepath.Join("..", "analysis", "python_ast.py")
	analyzer := analysis.NewPythonAnalyzer(scriptPath)
	scorer := fidelity.NewScorer(dsaRepo.Ontology(), fidelity.DefaultThresholds())

	eval := evaluator.NewEvaluator(r, analyzer, matcher, scorer, nil)

	// 2. Load problem definitions from dataset_all.json
	datasetPath := filepath.Join("..", "..", "data", "problems", "dataset_all.json")
	data, err := os.ReadFile(datasetPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", datasetPath, err)
	}
	var allProblems []model.Problem
	if err := json.Unmarshal(data, &allProblems); err != nil {
		t.Fatalf("failed to unmarshal dataset: %v", err)
	}
	probMap := make(map[string]model.Problem, len(allProblems))
	for _, p := range allProblems {
		probMap[p.ID] = p
	}

	// 3. Load benchmark suite
	suitePath := filepath.Join("..", "..", "data", "benchmarks", "multi_approach_suite.json")
	suiteData, err := os.ReadFile(suitePath)
	if err != nil {
		t.Fatalf("failed to read multi approach suite: %v", err)
	}
	var suites []BenchmarkSuiteItem
	if err := json.Unmarshal(suiteData, &suites); err != nil {
		t.Fatalf("failed to parse multi approach suite: %v", err)
	}

	totalCases := 0
	passedCases := 0

	for _, suite := range suites {
		suite := suite
		prob, exists := probMap[suite.ProblemID]
		if !exists {
			t.Fatalf("problem %s not found in dataset_all.json", suite.ProblemID)
		}

		t.Run(suite.Name, func(t *testing.T) {
			for _, app := range suite.Approaches {
				app := app
				totalCases++
				t.Run(app.Name, func(t *testing.T) {
					sub := model.Submission{
						ProblemID:   prob.ID,
						SourceCode:  app.Code,
						Explanation: app.Explanation,
					}

					res, err := eval.Evaluate(ctx, sub, prob)
					if err != nil {
						t.Fatalf("eval.Evaluate error: %v", err)
					}

					// Verify Test Status
					if string(res.TestResult) != app.ExpectedTestStatus {
						t.Errorf("expected test status %s, got %s", app.ExpectedTestStatus, res.TestResult)
					}

					// Verify Decision
					if string(res.Decision) != app.ExpectedDecision {
						t.Errorf("expected decision %s, got %s (fidelity=%.3f, diags=%v)",
							app.ExpectedDecision, res.Decision, res.FidelityScore, res.Diagnostics)
					}

					// Verify Fidelity bounds if specified
					if app.MinFidelityScore != nil && res.FidelityScore < *app.MinFidelityScore {
						t.Errorf("expected fidelity >= %.3f, got %.3f (matched=%v)",
							*app.MinFidelityScore, res.FidelityScore, res.MatchedConcepts)
					}
					if app.MaxFidelityScore != nil && res.FidelityScore > *app.MaxFidelityScore {
						t.Errorf("expected fidelity <= %.3f, got %.3f",
							*app.MaxFidelityScore, res.FidelityScore)
					}

					passedCases++
				})
			}
		})
	}

	t.Logf("Multi-approach evaluation completed: %d/%d test matrix cases verified strictly", passedCases, totalCases)
}
