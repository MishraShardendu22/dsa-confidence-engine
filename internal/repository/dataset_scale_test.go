package repository_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/repository"
)

func TestDatasetScaleIngestionAndValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "scaled_test.db")

	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to init sqlite repo: %v", err)
	}
	defer repo.Close()

	problemsDir := filepath.Join("..", "..", "data", "problems")
	ctx := context.Background()

	start := time.Now()
	err = repository.LoadProblemsFromDirectory(ctx, problemsDir, repo)
	if err != nil {
		t.Fatalf("failed to load problems from %s: %v", problemsDir, err)
	}
	elapsed := time.Since(start)
	t.Logf("Loaded 3,600+ problems in %v", elapsed)

	problems, err := repo.ListProblems(ctx)
	if err != nil {
		t.Fatalf("failed to list problems: %v", err)
	}

	if len(problems) < 4000 {
		t.Fatalf("expected at least 4000 problems, got %d", len(problems))
	}
	t.Logf("Successfully verified %d problems (>= 4000) loaded into repository", len(problems))

	// Validate every single problem against strict schema
	difficultyCounts := make(map[string]int)
	for i := range problems {
		p := problems[i]
		if err := model.ValidateProblem(&p); err != nil {
			t.Errorf("problem %s failed validation: %v", p.ID, err)
		}
		difficultyCounts[p.Difficulty]++
		if p.TimeLimitMs <= 0 {
			t.Errorf("problem %s has invalid time_limit_ms: %d", p.ID, p.TimeLimitMs)
		}
		if p.MemoryLimitMB <= 0 {
			t.Errorf("problem %s has invalid memory_limit_mb: %d", p.ID, p.MemoryLimitMB)
		}
	}

	t.Logf("Difficulty distribution across %d problems: Easy=%d, Medium=%d, Hard=%d",
		len(problems), difficultyCounts["Easy"], difficultyCounts["Medium"], difficultyCounts["Hard"])

	// Spot-check known problems
	twoSum, err := repo.GetProblem(ctx, "lc_1_two_sum")
	if err != nil {
		t.Fatalf("failed to retrieve lc_1_two_sum: %v", err)
	}
	if twoSum.Title != "#1 Two Sum" {
		t.Errorf("expected title '#1 Two Sum', got '%s'", twoSum.Title)
	}
	if twoSum.Difficulty != "Easy" {
		t.Errorf("expected difficulty 'Easy', got '%s'", twoSum.Difficulty)
	}
	if len(twoSum.TopicTags) == 0 {
		t.Errorf("expected topic tags for lc_1_two_sum, got empty")
	}

	// Also verify existing non-prefixed problems still coexist seamlessly
	existingTwoSum, err := repo.GetProblem(ctx, "two_sum")
	if err != nil {
		t.Fatalf("failed to retrieve legacy two_sum: %v", err)
	}
	if existingTwoSum.Title != "Two Sum" {
		t.Errorf("expected legacy title 'Two Sum', got '%s'", existingTwoSum.Title)
	}
}
