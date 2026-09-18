package model_test

import (
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

func TestValidateProblem(t *testing.T) {
	t.Run("valid problem passes and sets defaults", func(t *testing.T) {
		p := &model.Problem{
			ID:         "two_sum",
			Title:      "Two Sum",
			Entrypoint: "twoSum",
			Tests: []model.TestCase{
				{ID: "1", Input: `{"nums": [2, 7], "target": 9}`, ExpectedOutput: "[0, 1]"},
			},
		}

		err := model.ValidateProblem(p)
		if err != nil {
			t.Fatalf("expected valid, got: %v", err)
		}

		if p.Language != "python" {
			t.Errorf("expected default language 'python', got '%s'", p.Language)
		}
		if p.Difficulty != "Medium" {
			t.Errorf("expected default difficulty 'Medium', got '%s'", p.Difficulty)
		}
		if p.TimeLimitMs != 2000 {
			t.Errorf("expected default TimeLimitMs 2000, got %d", p.TimeLimitMs)
		}
		if p.MemoryLimitMB != 256 {
			t.Errorf("expected default MemoryLimitMB 256, got %d", p.MemoryLimitMB)
		}
		if len(p.EntrypointAliases) == 0 || p.EntrypointAliases[0] != "twoSum" {
			t.Errorf("expected primary entrypoint in aliases, got %v", p.EntrypointAliases)
		}
	})

	t.Run("missing ID fails", func(t *testing.T) {
		p := &model.Problem{
			Title:      "Two Sum",
			Entrypoint: "twoSum",
			Tests:      []model.TestCase{{Input: "test"}},
		}
		if err := model.ValidateProblem(p); err == nil {
			t.Error("expected error for missing ID")
		}
	})

	t.Run("invalid ID characters fail", func(t *testing.T) {
		p := &model.Problem{
			ID:         "two sum invalid!",
			Title:      "Two Sum",
			Entrypoint: "twoSum",
			Tests:      []model.TestCase{{Input: "test"}},
		}
		if err := model.ValidateProblem(p); err == nil {
			t.Error("expected error for invalid ID characters")
		}
	})

	t.Run("missing title fails", func(t *testing.T) {
		p := &model.Problem{
			ID:         "two_sum",
			Entrypoint: "twoSum",
			Tests:      []model.TestCase{{Input: "test"}},
		}
		if err := model.ValidateProblem(p); err == nil {
			t.Error("expected error for missing title")
		}
	})

	t.Run("missing entrypoint fails", func(t *testing.T) {
		p := &model.Problem{
			ID:    "two_sum",
			Title: "Two Sum",
			Tests: []model.TestCase{{Input: "test"}},
		}
		if err := model.ValidateProblem(p); err == nil {
			t.Error("expected error for missing entrypoint")
		}
	})

	t.Run("missing tests fails", func(t *testing.T) {
		p := &model.Problem{
			ID:         "two_sum",
			Title:      "Two Sum",
			Entrypoint: "twoSum",
			Tests:      []model.TestCase{},
		}
		if err := model.ValidateProblem(p); err == nil {
			t.Error("expected error for empty tests")
		}
	})

	t.Run("empty test input fails", func(t *testing.T) {
		p := &model.Problem{
			ID:         "two_sum",
			Title:      "Two Sum",
			Entrypoint: "twoSum",
			Tests:      []model.TestCase{{ID: "1", Input: "   "}},
		}
		if err := model.ValidateProblem(p); err == nil {
			t.Error("expected error for empty test input")
		}
	})

	t.Run("difficulty normalization", func(t *testing.T) {
		cases := []struct {
			input string
			want  string
		}{
			{"easy", "Easy"},
			{"EASY", "Easy"},
			{"hard", "Hard"},
			{"HARD", "Hard"},
			{"medium", "Medium"},
			{"unknown", "Medium"},
			{"", "Medium"},
		}
		for _, tc := range cases {
			p := &model.Problem{
				ID:         "prob",
				Title:      "Title",
				Entrypoint: "solve",
				Difficulty: tc.input,
				Tests:      []model.TestCase{{Input: "{}"}},
			}
			if err := model.ValidateProblem(p); err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.input, err)
			}
			if p.Difficulty != tc.want {
				t.Errorf("for input %q: expected %q, got %q", tc.input, tc.want, p.Difficulty)
			}
		}
	})
}
