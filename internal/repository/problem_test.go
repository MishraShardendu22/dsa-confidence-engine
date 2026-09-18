package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/repository"
)

func TestLoadProblemsFromDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prob-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}
	defer repo.Close()

	problemsDir := filepath.Join("..", "..", "data", "problems")
	ctx := context.Background()
	if err := repository.LoadProblemsFromDirectory(ctx, problemsDir, repo); err != nil {
		t.Fatalf("failed to load problems from directory: %v", err)
	}

	problems, err := repo.ListProblems(ctx)
	if err != nil {
		t.Fatalf("failed to list problems: %v", err)
	}

	if len(problems) < 4 {
		t.Errorf("expected at least 4 problems, got %d", len(problems))
	}

	twoSum, err := repo.GetProblem(ctx, "two_sum")
	if err != nil {
		t.Fatalf("failed to get two_sum: %v", err)
	}
	if twoSum.Entrypoint != "solve" || len(twoSum.Tests) != 3 {
		t.Errorf("unexpected two_sum problem content: %+v", twoSum)
	}

	// Test Multi-Format and Recursive Loading
	t.Run("recursive and json formats", func(t *testing.T) {
		multiDir, err := os.MkdirTemp("", "prob-multi-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(multiDir)

		// 1. Nested directory with YAML
		subDir := filepath.Join(multiDir, "nested", "category")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create nested dir: %v", err)
		}
		yamlContent := `
id: nested_problem
title: Nested Problem
entrypoint: solve
tests:
  - id: "1"
    input: '{"x": 1}'
    expected_output: '1'
`
		if err := os.WriteFile(filepath.Join(subDir, "problem.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write yaml: %v", err)
		}

		// 2. JSON array file with multiple problems
		jsonArrayContent := `[
			{
				"id": "json_problem_1",
				"title": "JSON Problem 1",
				"entrypoint": "solve",
				"tests": [{"id": "1", "input": "{}", "expected_output": "0"}]
			},
			{
				"id": "json_problem_2",
				"title": "JSON Problem 2",
				"entrypoint": "solve",
				"tests": [{"id": "1", "input": "{}", "expected_output": "0"}]
			}
		]`
		if err := os.WriteFile(filepath.Join(multiDir, "problems_array.json"), []byte(jsonArrayContent), 0644); err != nil {
			t.Fatalf("failed to write json array: %v", err)
		}

		// 3. JSONL file with streamed lines
		jsonlContent := `{"id": "jsonl_prob_1", "title": "JSONL 1", "entrypoint": "solve", "tests": [{"id": "1", "input": "{}", "expected_output": "0"}]}
{"id": "jsonl_prob_2", "title": "JSONL 2", "entrypoint": "solve", "tests": [{"id": "1", "input": "{}", "expected_output": "0"}]}
`
		if err := os.WriteFile(filepath.Join(multiDir, "stream.jsonl"), []byte(jsonlContent), 0644); err != nil {
			t.Fatalf("failed to write jsonl: %v", err)
		}

		multiDbPath := filepath.Join(multiDir, "multi.db")
		multiRepo, err := repository.NewSQLiteRepository(multiDbPath)
		if err != nil {
			t.Fatalf("failed to init multi repo: %v", err)
		}
		defer multiRepo.Close()

		if err := repository.LoadProblemsFromDirectory(ctx, multiDir, multiRepo); err != nil {
			t.Fatalf("failed to load multi-format directory: %v", err)
		}

		loaded, err := multiRepo.ListProblems(ctx)
		if err != nil {
			t.Fatalf("failed to list loaded: %v", err)
		}

		if len(loaded) != 5 {
			t.Errorf("expected exactly 5 problems from nested+json+jsonl, got %d", len(loaded))
		}

		// Verify retrieval of nested problem
		nested, err := multiRepo.GetProblem(ctx, "nested_problem")
		if err != nil {
			t.Fatalf("failed to get nested_problem: %v", err)
		}
		if nested.Title != "Nested Problem" {
			t.Errorf("expected 'Nested Problem', got %s", nested.Title)
		}
	})
}
