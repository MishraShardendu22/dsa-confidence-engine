package evaluator_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
)

func setupPiyushBenchmarkEvaluator(t *testing.T) (*evaluator.Evaluator, map[string]model.Problem) {
	repoPath := filepath.Join("..", "..", "data", "dsa")
	dsaRepo, err := dsa.NewRepository(repoPath)
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
	a := analysis.NewPythonAnalyzer(scriptPath)
	s := fidelity.NewScorer(dsaRepo.Ontology(), fidelity.DefaultThresholds())

	eval := evaluator.NewEvaluator(r, a, matcher, s, nil)

	probMap := make(map[string]model.Problem)
	datasetPath := filepath.Join("..", "..", "data", "problems", "dataset_all.json")
	if data, err := os.ReadFile(datasetPath); err == nil {
		var list []model.Problem
		if err := json.Unmarshal(data, &list); err == nil {
			for _, p := range list {
				probMap[p.ID] = p
			}
		}
	}
	if len(probMap) == 0 {
		probDir := filepath.Join("..", "..", "data", "problems")
		entries, err := filepath.Glob(filepath.Join(probDir, "*.yaml"))
		if err != nil {
			t.Fatalf("failed to glob problems: %v", err)
		}
		for _, fpath := range entries {
			data, err := os.ReadFile(fpath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", fpath, err)
			}
			var p model.Problem
			if err := yaml.Unmarshal(data, &p); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", fpath, err)
			}
			probMap[p.ID] = p
		}
	}

	return eval, probMap
}

func TestPiyushBenchmark_TwoSum(t *testing.T) {
	eval, probMap := setupPiyushBenchmarkEvaluator(t)
	ctx := context.Background()

	prob, ok := probMap["two_sum"]
	if !ok {
		t.Fatalf("two_sum problem not loaded")
	}

	// 1. Reference Solution: Hash Map Complement Lookup
	// With the exact negation trap phrase "for not found" that crashed Piyush's regex engine
	t.Run("hashmap complement with negation trap explanation -> ACCEPT", func(t *testing.T) {
		code := `def two_sum(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []`

		explanation := "So I will iterate through the array, keep a Hashmap storing their index, match target - current number, check for this value in the Hashmap, if found return these two index the current and the one stored in Hashmap else return empty array for not found."

		sub := model.Submission{
			ProblemID:   "two_sum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 2. Reference Solution: Sorting and Two Pointers
	t.Run("sorting and two pointers -> ACCEPT", func(t *testing.T) {
		code := `def two_sum(nums, target):
    arr = sorted((v, i) for i, v in enumerate(nums))
    l, r = 0, len(arr) - 1
    while l < r:
        s = arr[l][0] + arr[r][0]
        if s == target:
            return [arr[l][1], arr[r][1]]
        elif s < target:
            l += 1
        else:
            r -= 1
    return []`

		explanation := "I sort the array while preserving original indices, then use two pointers converging inward from both ends."

		sub := model.Submission{
			ProblemID:   "two_sum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
	})

	// 3. Reference Solution: Nested Loop Brute Force
	t.Run("nested loop brute force -> ACCEPT", func(t *testing.T) {
		code := `def two_sum(nums, target):
    for i in range(len(nums)):
        for j in range(i + 1, len(nums)):
            if nums[i] + nums[j] == target:
                return [i, j]
    return []`

		explanation := "I use a brute force nested loop to check all pairs of elements."

		sub := model.Submission{
			ProblemID:   "two_sum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
	})

	// 4. Forward reference top-level call with camelCase entrypoint
	t.Run("forward print call and camelCase twoSum -> ACCEPT", func(t *testing.T) {
		code := `nums = [2, 7, 11, 15]
target = 9
print(twoSum(nums, target))

def twoSum(nums, target):
    seen = {}
    for i, num in enumerate(nums):
        diff = target - num
        if diff in seen:
            return [seen[diff], i]
        seen[num] = i
    return []`

		explanation := "I use a hash map to store seen values and check complement in O(1)."

		sub := model.Submission{
			ProblemID:   "two_sum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f)", res.Decision, res.FidelityScore)
		}
	})
}

func TestPiyushBenchmark_FrequencyCount(t *testing.T) {
	eval, probMap := setupPiyushBenchmarkEvaluator(t)
	ctx := context.Background()

	prob, ok := probMap["frequency_count"]
	if !ok {
		t.Fatalf("frequency_count problem not loaded")
	}

	// 1. Reference Solution: Dict .get() Accumulator
	t.Run("hash frequency dict get -> ACCEPT", func(t *testing.T) {
		code := `def char_frequency(s):
    counts = {}
    for c in s:
        counts[c] = counts.get(c, 0) + 1
    return counts`

		explanation := "I iterate through the string and use a hash map dictionary to count the frequency of each character."

		sub := model.Submission{
			ProblemID:   "frequency_count",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 2. Reference Solution: collections.Counter
	t.Run("collections Counter -> ACCEPT", func(t *testing.T) {
		code := `from collections import Counter
def char_frequency(s):
    return dict(Counter(s))`

		explanation := "I use collections.Counter to tally the frequency count of each character and return a dictionary."

		sub := model.Submission{
			ProblemID:   "frequency_count",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 3. Reference Solution: Nested Count Scan
	t.Run("nested count scan dict comprehension -> ACCEPT", func(t *testing.T) {
		code := `def char_frequency(s):
    return {c: s.count(c) for c in set(s)}`

		explanation := "I use a set to get unique characters and build a dictionary frequency count using s.count."

		sub := model.Submission{
			ProblemID:   "frequency_count",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
	})
}

func TestPiyushBenchmark_SecondMaximum(t *testing.T) {
	eval, probMap := setupPiyushBenchmarkEvaluator(t)
	ctx := context.Background()

	prob, ok := probMap["second_maximum"]
	if !ok {
		t.Fatalf("second_maximum problem not loaded")
	}

	// 1. Reference Solution: Single Pass Iterative Two Variables
	t.Run("single pass iterative tracking -> ACCEPT", func(t *testing.T) {
		code := `def second_maximum(nums):
    first = second = None
    for n in nums:
        if first is None or n > first:
            second, first = first, n
        elif n != first and (second is None or n > second):
            second = n
    return second`

		explanation := "I iterate through the array elements in a single pass tracking the top two maximum values with two variables."

		sub := model.Submission{
			ProblemID:   "second_maximum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 2. Reference Solution: Sorting & Indexing Penultimate
	t.Run("sorting and set deduplication indexing -> ACCEPT", func(t *testing.T) {
		code := `def second_maximum(nums):
    unique = sorted(set(nums))
    return unique[-2] if len(unique) >= 2 else None`

		explanation := "I convert the array into a set to deduplicate, sort the elements in ascending order, and return the second to last element."

		sub := model.Submission{
			ProblemID:   "second_maximum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 3. Reference Solution: Set Reduction (Double Max)
	t.Run("set reduction double max -> ACCEPT", func(t *testing.T) {
		code := `def second_maximum(nums):
    unique = set(nums)
    if len(unique) < 2:
        return None
    unique.remove(max(unique))
    return max(unique)`

		explanation := "I store unique elements in a hash set, remove the maximum element, and return the new maximum."

		sub := model.Submission{
			ProblemID:   "second_maximum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 4. Trap: Single Pass with Unused Decoy Sorting (Broke Piyush's AST checker)
	t.Run("decoy unused sorting with single pass iterative -> ACCEPT without false positive", func(t *testing.T) {
		code := `def second_maximum(nums):
    decoy = sorted(nums)
    first = second = None
    for n in nums:
        if first is None or n > first:
            second, first = first, n
        elif n != first and (second is None or n > second):
            second = n
    return second`

		explanation := "I iterate through the array elements in a single pass tracking the top two maximum values with two variables."

		sub := model.Submission{
			ProblemID:   "second_maximum",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
	})
}

func TestPiyushBenchmark_MismatchesAndFailures(t *testing.T) {
	eval, probMap := setupPiyushBenchmarkEvaluator(t)
	ctx := context.Background()

	probTS := probMap["two_sum"]
	probFC := probMap["frequency_count"]
	probSM := probMap["second_maximum"]

	// 1. Approach mismatch: Hashmap code but claimed Binary Search
	t.Run("hashmap implementation with claimed binary search -> REJECT", func(t *testing.T) {
		code := `def two_sum(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []`

		sub := model.Submission{
			ProblemID:   "two_sum",
			SourceCode:  code,
			Explanation: "I will use binary search to locate the complement in logarithmic time.",
		}

		res, err := eval.Evaluate(ctx, sub, probTS)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected test PASS, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionReject {
			t.Errorf("expected Decision REJECT for mismatched binary search, got %s (score: %f)", res.Decision, res.FidelityScore)
		}
	})

	// 2. Approach mismatch: Sorting code but claimed Dynamic Programming
	t.Run("sorting implementation with claimed dynamic programming -> REJECT", func(t *testing.T) {
		code := `def second_maximum(nums):
    unique = sorted(set(nums))
    return unique[-2] if len(unique) >= 2 else None`

		sub := model.Submission{
			ProblemID:   "second_maximum",
			SourceCode:  code,
			Explanation: "I define a DP state table dp[i] storing the subproblem solutions and use memoization.",
		}

		res, err := eval.Evaluate(ctx, sub, probSM)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected test PASS, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionReject {
			t.Errorf("expected Decision REJECT for claimed DP on sorting code, got %s (score: %f)", res.Decision, res.FidelityScore)
		}
	})

	// 3. Test execution failure: Buggy code failing test cases
	t.Run("buggy code failing test cases -> FAIL and immediate REJECT", func(t *testing.T) {
		code := `def char_frequency(s):
    return {"wrong": 999}`

		sub := model.Submission{
			ProblemID:   "frequency_count",
			SourceCode:  code,
			Explanation: "I count character frequencies using a hashmap.",
		}

		res, err := eval.Evaluate(ctx, sub, probFC)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusFail {
			t.Errorf("expected test FAIL, got %s", res.TestResult)
		}
		if res.Decision != model.DecisionReject {
			t.Errorf("expected Decision REJECT on test failure, got %s", res.Decision)
		}
		if len(res.ActualConcepts) != 0 {
			t.Errorf("expected 0 actual concepts extracted on failed tests, got %d", len(res.ActualConcepts))
		}
	})
}

func TestPiyushBenchmark_Question4Expansion(t *testing.T) {
	eval, probMap := setupPiyushBenchmarkEvaluator(t)
	ctx := context.Background()

	prob, ok := probMap["first_non_repeating_character"]
	if !ok {
		t.Fatalf("first_non_repeating_character problem not loaded")
	}

	// 1. Implementation Family 1: Hash Map Frequency Count
	t.Run("first non-repeating character with dict get -> ACCEPT", func(t *testing.T) {
		code := `def first_non_repeating_character(s):
    counts = {}
    for c in s:
        counts[c] = counts.get(c, 0) + 1
    for c in s:
        if counts[c] == 1:
            return c
    return None`

		explanation := "I count character frequencies using a hashmap dictionary, then scan the string to find the first character with frequency count 1."

		sub := model.Submission{
			ProblemID:   "first_non_repeating_character",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 2. Implementation Family 2: collections.Counter
	t.Run("first non-repeating character with collections.Counter -> ACCEPT", func(t *testing.T) {
		code := `from collections import Counter
def first_non_repeating_character(s):
    counts = Counter(s)
    for c in s:
        if counts[c] == 1:
            return c
    return None`

		explanation := "I tally character frequencies with collections.Counter and return the first character with count equal to one."

		sub := model.Submission{
			ProblemID:   "first_non_repeating_character",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
		if res.FidelityScore < 0.95 {
			t.Errorf("expected score >= 0.95, got %f", res.FidelityScore)
		}
	})

	// 3. Implementation Family 3: Repeated str.count() Scan (Correct but O(N^2) inefficient)
	t.Run("first non-repeating character with repeated count scan -> ACCEPT", func(t *testing.T) {
		code := `def first_non_repeating_character(s):
    for c in s:
        if s.count(c) == 1:
            return c
    return None`

		explanation := "I iterate through the string and check character frequency count using s.count for each character."

		sub := model.Submission{
			ProblemID:   "first_non_repeating_character",
			SourceCode:  code,
			Explanation: explanation,
		}

		res, err := eval.Evaluate(ctx, sub, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.TestResult != model.TestStatusPass {
			t.Fatalf("expected PASS, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
		if res.Decision != model.DecisionAccept {
			t.Errorf("expected ACCEPT, got %s (score: %f, diags: %v)", res.Decision, res.FidelityScore, res.Diagnostics)
		}
	})
}
