package service_test

import (
	"context"
	"os"
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
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/service"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/util"
)

func setupTestServices(t *testing.T) (*service.ProblemService, *service.EvaluationService, *repository.SQLiteRepository) {
	tmpDir, err := os.MkdirTemp("", "svc-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	dbPath := filepath.Join(tmpDir, "test.db")
	sqliteRepo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create sqlite repo: %v", err)
	}
	t.Cleanup(func() { sqliteRepo.Close() })

	ontologyPath := filepath.Join("..", "..", "data", "dsa")
	ontRepo, err := dsa.NewRepository(ontologyPath)
	if err != nil {
		t.Fatalf("failed to load ontology: %v", err)
	}

	ctx := context.Background()
	embedder := nlp.NewLocalHashingEmbedder(256)
	matcher, err := nlp.NewCascadeMatcher(ctx, ontRepo.Ontology(), embedder, true)
	if err != nil {
		t.Fatalf("failed to create matcher: %v", err)
	}

	r := runner.NewLocalRunner(2000)
	scriptPath := filepath.Join("..", "analysis", "python_ast.py")
	a := analysis.NewPythonAnalyzer(scriptPath)
	s := fidelity.NewScorer(ontRepo.Ontology(), fidelity.DefaultThresholds())
	evalEngine := evaluator.NewEvaluator(r, a, matcher, s, nil)

	probService := service.NewProblemService(sqliteRepo)
	evalService := service.NewEvaluationService(evalEngine, sqliteRepo)

	return probService, evalService, sqliteRepo
}

func TestProblemService(t *testing.T) {
	probService, _, _ := setupTestServices(t)
	ctx := context.Background()

	p := &model.Problem{
		ID:          "p1",
		Title:       "Test Problem",
		Description: "Desc",
		Language:    "python",
		Entrypoint:  "solve",
		Tests: []model.TestCase{
			{ID: "1", Input: `{"x": 1}`, ExpectedOutput: "1"},
		},
		AcceptedStrategies: []string{"strat1"},
		RequiredConcepts:   []string{"concept1"},
	}

	if err := probService.CreateProblem(ctx, p); err != nil {
		t.Fatalf("CreateProblem failed: %v", err)
	}

	got, err := probService.GetProblem(ctx, "p1")
	if err != nil {
		t.Fatalf("GetProblem failed: %v", err)
	}
	if got.Title != "Test Problem" {
		t.Errorf("expected Title Test Problem, got %s", got.Title)
	}

	list, err := probService.ListProblems(ctx)
	if err != nil {
		t.Fatalf("ListProblems failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 problem, got %d", len(list))
	}

	_, err = probService.GetProblem(ctx, "nonexistent")
	if err != util.ErrProblemNotFound {
		t.Errorf("expected ErrProblemNotFound, got %v", err)
	}
}

func TestEvaluationService(t *testing.T) {
	probService, evalService, _ := setupTestServices(t)
	ctx := context.Background()

	p := &model.Problem{
		ID:          "two_sum",
		Title:       "Two Sum",
		Description: "Find two indices that sum to target",
		Language:    "python",
		Entrypoint:  "two_sum",
		Tests: []model.TestCase{
			{ID: "1", Input: `{"nums": [2, 7, 11, 15], "target": 9}`, ExpectedOutput: "[0, 1]"},
		},
		AcceptedStrategies: []string{"hashmap_two_sum"},
		RequiredConcepts:   []string{"hashmap"},
		PrimaryConcepts:    []string{"hashmap"},
	}
	if err := probService.CreateProblem(ctx, p); err != nil {
		t.Fatalf("failed to create problem: %v", err)
	}

	sub := model.Submission{
		ProblemID: "two_sum",
		SourceCode: `
def two_sum(nums, target):
    seen = {}
    for i, x in enumerate(nums):
        d = target - x
        if d in seen:
            return [seen[d], i]
        seen[x] = i
    return []
`,
		Explanation: "I used a hash map for constant time complement lookup.",
	}

	eval, err := evalService.Evaluate(ctx, sub)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if eval.Decision != model.DecisionAccept {
		t.Errorf("expected ACCEPT, got %s", eval.Decision)
	}

	gotEval, err := evalService.GetEvaluation(ctx, eval.ID)
	if err != nil {
		t.Fatalf("GetEvaluation failed: %v", err)
	}
	if gotEval.ID != eval.ID {
		t.Errorf("expected eval ID %s, got %s", eval.ID, gotEval.ID)
	}

	// Missing problem evaluate
	_, err = evalService.Evaluate(ctx, model.Submission{ProblemID: "missing", SourceCode: "pass"})
	if err == nil {
		t.Errorf("expected error for nonexistent problem")
	}

	// Missing evaluation
	_, err = evalService.GetEvaluation(ctx, "missing_eval")
	if err != util.ErrEvaluationNotFound {
		t.Errorf("expected ErrEvaluationNotFound, got %v", err)
	}
}
