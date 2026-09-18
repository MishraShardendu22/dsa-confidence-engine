package evaluator_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/repository"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
)

func TestDataset2KEvaluationPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "eval_dataset.db")

	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}
	defer repo.Close()

	problemsDir := filepath.Join("..", "..", "data", "problems")
	ctx := context.Background()

	if err := repository.LoadProblemsFromDirectory(ctx, problemsDir, repo); err != nil {
		t.Fatalf("failed to load dataset: %v", err)
	}

	allProblems, err := repo.ListProblems(ctx)
	if err != nil {
		t.Fatalf("failed to list problems: %v", err)
	}
	if len(allProblems) < 3600 {
		t.Fatalf("expected >= 3600 problems loaded, got %d", len(allProblems))
	}

	// Verify broad coverage across core algorithmic paradigms in dataset
	topicPresence := make(map[string]bool)
	for _, p := range allProblems {
		for _, tag := range p.TopicTags {
			topicPresence[tag] = true
		}
	}
	expectedTags := []string{
		"array", "dynamic-programming", "string", "hash-table",
		"binary-search", "greedy", "depth-first-search", "breadth-first-search",
		"tree", "sorting", "two-pointers", "backtracking",
	}
	for _, tag := range expectedTags {
		if !topicPresence[tag] {
			t.Errorf("expected tag '%s' to be present in ingested 2k dataset", tag)
		}
	}

	// Setup evaluator
	dsaRepoPath := filepath.Join("..", "..", "data", "dsa")
	dsaRepo, err := dsa.NewRepository(dsaRepoPath)
	if err != nil {
		t.Fatalf("failed to load ontology: %v", err)
	}

	embedder := nlp.NewLocalHashingEmbedder(256)
	matcher, err := nlp.NewCascadeMatcher(ctx, dsaRepo.Ontology(), embedder, true)
	if err != nil {
		t.Fatalf("failed to init matcher: %v", err)
	}

	r := runner.NewLocalRunner(3000)
	scriptPath := filepath.Join("..", "analysis", "python_ast.py")
	analyzer := analysis.NewPythonAnalyzer(scriptPath)
	scorer := fidelity.NewScorer(dsaRepo.Ontology(), fidelity.DefaultThresholds())
	eval := evaluator.NewEvaluator(r, analyzer, matcher, scorer, nil)

	// Fetch lc_1_two_sum from SQLite repository
	prob, err := repo.GetProblem(ctx, "lc_1_two_sum")
	if err != nil {
		t.Fatalf("failed to find lc_1_two_sum: %v", err)
	}

	// 1. Valid Solution Matching Code & Explanation
	t.Run("valid submission on 2k dataset problem", func(t *testing.T) {
		sub := model.Submission{
			ProblemID: prob.ID,
			SourceCode: `
def twoSum(nums):
    seen = {}
    for i, x in enumerate(nums):
        if x in seen:
            return seen[x]
        seen[x] = 0
    return seen.get(0, 0)
`,
			Explanation: "We use an array iteration and a hashmap to record numbers and find matching pairs.",
		}

		res, err := eval.Evaluate(ctx, sub, *prob)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}
		t.Logf("Valid evaluation: Score=%.3f, Decision=%s, Matched=%v, Diags=%v",
			res.FidelityScore, res.Decision, res.MatchedConcepts, res.Diagnostics)
		if res.TestResult != model.TestStatusPass {
			t.Errorf("expected tests to PASS, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT decision, got %s (score: %.3f)", res.Decision, res.FidelityScore)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected high fidelity score >= 0.95, got %.3f", res.FidelityScore)
		}
	})

	// 2. Failing Solution -> Immediate Rejection
	t.Run("failing submission on 2k dataset problem", func(t *testing.T) {
		sub := model.Submission{
			ProblemID: prob.ID,
			SourceCode: `
def twoSum(nums):
    return -9999
`,
			Explanation: "We iterate through the array.",
		}

		res, err := eval.Evaluate(ctx, sub, *prob)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}
		if res.TestResult != model.TestStatusFail {
			t.Errorf("expected test FAIL, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionReject {
			t.Errorf("expected REJECT decision, got %s", res.Decision)
		}
	})

	// 3. Bluffing Explanation -> Fidelity Divergence Penalty
	t.Run("bluffing explanation penalized on 2k dataset problem", func(t *testing.T) {
		sub := model.Submission{
			ProblemID: prob.ID,
			SourceCode: `
def twoSum(nums):
    seen = {}
    for i, x in enumerate(nums):
        seen[x] = 0
    return seen.get(0, 0)
`,
			Explanation: "We construct a segment tree with lazy propagation and run Dijkstra shortest path algorithm.",
		}

		res, err := eval.Evaluate(ctx, sub, *prob)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}
		if res.TestResult != model.TestStatusPass {
			t.Errorf("expected test PASS, got %s", res.TestResult)
		}
		// Code has no segment tree or dijkstra -> code/explanation divergence
		if res.Decision == model.DecisionAccept {
			t.Errorf("expected penalized score for bluffing explanation, got score=%.3f, decision=%s", res.FidelityScore, res.Decision)
		}
	})
}
