package runner_test

import (
	"context"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
)

func TestLocalRunner(t *testing.T) {
	r := runner.NewLocalRunner(2000)
	ctx := context.Background()

	problem := model.Problem{
		ID:         "two_sum",
		Language:   "python",
		Entrypoint: "solve",
		Tests: []model.TestCase{
			{
				ID:             "1",
				Input:          "{\"nums\": [2, 7, 11, 15], \"target\": 9}",
				ExpectedOutput: "[0, 1]",
			},
			{
				ID:             "2",
				Input:          "{\"nums\": [3, 2, 4], \"target\": 6}",
				ExpectedOutput: "[1, 2]",
			},
		},
	}

	// Case 1: Valid Solution
	validCode := `
def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
`
	res, err := r.Run(ctx, model.Submission{SourceCode: validCode}, problem)
	if err != nil {
		t.Fatalf("unexpected error running tests: %v", err)
	}
	if res.Status != model.TestStatusPass {
		t.Errorf("expected PASS, got %s (error: %s)", res.Status, res.Error)
	}
	if res.PassedCount != 2 || res.FailedCount != 0 {
		t.Errorf("expected 2 passed 0 failed, got passed=%d failed=%d", res.PassedCount, res.FailedCount)
	}

	// Case 2: Incorrect Solution
	wrongCode := `
def solve(nums, target):
    return [0, 0]
`
	res, err = r.Run(ctx, model.Submission{SourceCode: wrongCode}, problem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != model.TestStatusFail {
		t.Errorf("expected FAIL, got %s", res.Status)
	}
	if res.FailedCount == 0 {
		t.Errorf("expected at least 1 failed test, got 0")
	}

	// Case 3: Syntax Error
	badSyntax := `def solve(nums, target) return`
	res, err = r.Run(ctx, model.Submission{SourceCode: badSyntax}, problem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != model.TestStatusFail {
		t.Errorf("expected FAIL for syntax error, got %s", res.Status)
	}

	// Case 4: Infinite loop / Timeout
	timeoutRunner := runner.NewLocalRunner(500)
	infLoop := `
def solve(nums, target):
    while True:
        pass
`
	res, err = timeoutRunner.Run(ctx, model.Submission{SourceCode: infLoop}, problem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != model.TestStatusFail {
		t.Errorf("expected timeout FAIL, got %s", res.Status)
	}

	// Case 5: LeetCode class Solution
	solutionClassCode := `
class Solution:
    def solve(self, nums, target):
        seen = {}
        for i, n in enumerate(nums):
            diff = target - n
            if diff in seen:
                return [seen[diff], i]
            seen[n] = i
        return []
`
	res, err = r.Run(ctx, model.Submission{SourceCode: solutionClassCode}, problem)
	if err != nil {
		t.Fatalf("unexpected error running class Solution: %v", err)
	}
	if res.Status != model.TestStatusPass {
		t.Errorf("expected class Solution to PASS, got %s (error: %s)", res.Status, res.Error)
	}

	// Case 6: Single-parameter function receiving list input
	singleArgProb := model.Problem{
		ID:         "second_max",
		Language:   "python",
		Entrypoint: "solve",
		Tests: []model.TestCase{
			{
				ID:             "1",
				Input:          "[10, 20, 4, 45, 99]",
				ExpectedOutput: "45",
			},
		},
	}
	singleArgCode := `
def solve(nums):
    unique = sorted(list(set(nums)))
    return unique[-2]
`
	res, err = r.Run(ctx, model.Submission{SourceCode: singleArgCode}, singleArgProb)
	if err != nil {
		t.Fatalf("unexpected error running single list argument: %v", err)
	}
	if res.Status != model.TestStatusPass {
		t.Errorf("expected single list argument to PASS, got %s (error: %s)", res.Status, res.Error)
	}
}
