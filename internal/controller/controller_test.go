package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/api"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/controller"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/repository"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/service"
	"github.com/gofiber/fiber/v2"
)

func setupTestApp(t *testing.T) (*fiber.App, *repository.SQLiteRepository) {
	tmpDir, err := os.MkdirTemp("", "ctrl-test-*")
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

	problemsPath := filepath.Join("..", "..", "data", "problems")
	ctx := context.Background()
	if err := repository.LoadProblemsFromDirectory(ctx, problemsPath, sqliteRepo); err != nil {
		t.Fatalf("failed to load problems: %v", err)
	}

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

	probCtrl := controller.NewProblemController(probService)
	evalCtrl := controller.NewEvaluationController(evalService)

	app := fiber.New()
	api.RegisterRoutes(app, api.RouterConfig{
		ProblemHandler:    probCtrl,
		EvaluationHandler: evalCtrl,
		ProblemService:    probService,
		EvaluationService: evalService,
		OntologyRepo:      ontRepo,
	})

	return app, sqliteRepo
}

func TestHealthEndpoint(t *testing.T) {
	app, _ := setupTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var envelope api.Response
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !envelope.Success {
		t.Errorf("expected success=true, got %v", envelope.Success)
	}
}

func TestProblemEndpoints(t *testing.T) {
	app, _ := setupTestApp(t)

	// 1. List Problems
	req := httptest.NewRequest(http.MethodGet, "/api/problems", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// 2. Get Problem by ID
	req = httptest.NewRequest(http.MethodGet, "/api/problems/two_sum", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestEvaluateEndpoint(t *testing.T) {
	app, _ := setupTestApp(t)

	submission := model.Submission{
		ProblemID: "two_sum",
		SourceCode: `
def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
`,
		Explanation: "I'll store elements in a hashmap and check for complements in constant time.",
	}

	bodyBytes, _ := json.Marshal(submission)
	req := httptest.NewRequest(http.MethodPost, "/api/evaluate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("evaluate request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var envelope struct {
		Success bool             `json:"success"`
		Data    model.Evaluation `json:"data"`
		Error   *api.APIError    `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode eval response: %v", err)
	}

	if !envelope.Success {
		t.Fatalf("expected success=true, got error: %+v", envelope.Error)
	}
	if envelope.Data.Decision != model.DecisionAccept {
		t.Errorf("expected ACCEPT, got %s", envelope.Data.Decision)
	}
	if envelope.Data.TestResult != model.TestStatusPass {
		t.Errorf("expected PASS, got %s", envelope.Data.TestResult)
	}
	if envelope.Data.FidelityScore < 0.95 {
		t.Errorf("expected score >= 0.95, got %f", envelope.Data.FidelityScore)
	}

	// Fetch via GET /api/evaluations/:id
	req = httptest.NewRequest(http.MethodGet, "/api/evaluations/"+envelope.Data.ID, nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("get evaluation failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestControllerValidationAndErrors(t *testing.T) {
	app, _ := setupTestApp(t)

	// 1. Problem not found -> 404
	req := httptest.NewRequest(http.MethodGet, "/api/problems/non_existent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	// 2. Evaluation not found -> 404
	req = httptest.NewRequest(http.MethodGet, "/api/evaluations/non_existent", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	// 3. Problem creation validation failure (empty ID) -> 400
	invalidProb := model.Problem{Title: "Missing ID", Entrypoint: "solve"}
	probBytes, _ := json.Marshal(invalidProb)
	req = httptest.NewRequest(http.MethodPost, "/api/problems", bytes.NewReader(probBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	// 3b. Problem creation validation failure (empty Tests) -> 400
	noTestsProb := model.Problem{ID: "no_tests", Title: "No Tests", Entrypoint: "solve", Tests: []model.TestCase{}}
	noTestsBytes, _ := json.Marshal(noTestsProb)
	req = httptest.NewRequest(http.MethodPost, "/api/problems", bytes.NewReader(noTestsBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty tests, got %d", resp.StatusCode)
	}

	// 4. Evaluate API validation failure (missing source code) -> 400
	invalidSub := model.Submission{ProblemID: "two_sum", SourceCode: ""}
	subBytes, _ := json.Marshal(invalidSub)
	req = httptest.NewRequest(http.MethodPost, "/api/evaluate", bytes.NewReader(subBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	// 5. Web form evaluate validation failure (missing problem_id) -> 400
	form := url.Values{}
	form.Set("problem_id", "")
	form.Set("source_code", "def solve(): pass")
	req = httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	// 6. Web form evaluate valid submission -> 302 Redirect
	form = url.Values{}
	form.Set("problem_id", "two_sum")
	form.Set("source_code", "def solve(nums, target): return [0, 1]")
	form.Set("explanation", "I used brute force")
	req = httptest.NewRequest(http.MethodPost, "/evaluate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/evaluation/") {
		t.Errorf("expected redirect to /evaluation/:id, got %s", loc)
	}

	// 7. Web GET /evaluation/:id for non-existent -> 404
	req = httptest.NewRequest(http.MethodGet, "/evaluation/non_existent", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
