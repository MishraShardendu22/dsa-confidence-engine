package analysis_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
)

func TestPythonAnalyzer_AdversarialSuite(t *testing.T) {
	scriptPath := filepath.Join("python_ast.py")
	analyzer := analysis.NewPythonAnalyzer(scriptPath)
	ctx := context.Background()

	// 1. Unrelated helper contains hashmap 10,000 times
	t.Run("unrelated helper with hashmap", func(t *testing.T) {
		code := `
def useless():
    d = {}
    for i in range(10000):
        d[i] = i

def solve(nums):
    return max(nums)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" {
				if c.Reachable {
					t.Errorf("expected hashmap in useless() to be unreachable, got reachable=true")
				}
				if c.OutputRelevant {
					t.Errorf("expected hashmap in useless() to NOT be output-relevant")
				}
			}
		}
	})

	// 2. Hashmap populated but not used in return
	t.Run("hashmap populated but not output relevant", func(t *testing.T) {
		code := `
def solve(nums):
    counts = {}
    for x in nums:
        counts[x] = counts.get(x, 0) + 1
    return max(nums)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var mapConcept *struct {
			Reachable      bool
			OutputRelevant bool
		}
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" {
				mapConcept = &struct {
					Reachable      bool
					OutputRelevant bool
				}{
					Reachable:      c.Reachable,
					OutputRelevant: c.OutputRelevant,
				}
			}
		}

		if mapConcept == nil {
			t.Fatalf("expected hashmap to be detected")
		}
		if !mapConcept.Reachable {
			t.Errorf("expected hashmap to be reachable")
		}
		if mapConcept.OutputRelevant {
			t.Errorf("expected hashmap to NOT be output-relevant")
		}
	})

	// 3. Hashmap populated AND used in return
	t.Run("hashmap populated and output relevant", func(t *testing.T) {
		code := `
def solve(nums):
    counts = {}
    for x in nums:
        counts[x] = counts.get(x, 0) + 1
    return max(counts, key=counts.get)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var found bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" {
				found = true
				if !c.Reachable {
					t.Errorf("expected hashmap to be reachable")
				}
				if !c.OutputRelevant {
					t.Errorf("expected hashmap to be output-relevant")
				}
			}
		}
		if !found {
			t.Fatalf("expected hashmap to be detected")
		}
	})

	// 4. Hashmap used in relevant helper
	t.Run("hashmap in relevant helper", func(t *testing.T) {
		code := `
def build_map(nums):
    seen = {}
    for x in nums:
        seen[x] = seen.get(x, 0) + 1
    return seen

def solve(nums):
    res = build_map(nums)
    return max(res)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var found bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" {
				found = true
				if !c.Reachable {
					t.Errorf("expected helper hashmap to be reachable")
				}
				if !c.OutputRelevant {
					t.Errorf("expected helper hashmap to be output-relevant")
				}
			}
		}
		if !found {
			t.Fatalf("expected hashmap to be detected in helper")
		}
	})

	// 5. Two Pointers with unusual variable names
	t.Run("two pointers with unusual variable names", func(t *testing.T) {
		code := `
def solve(s):
    ptrA = 0
    ptrB = len(s) - 1
    while ptrA < ptrB:
        if s[ptrA] != s[ptrB]:
            return False
        ptrA += 1
        ptrB -= 1
    return True
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var tpFound, oppFound bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "two_pointers" {
				tpFound = true
			}
			if c.ConceptID == "opposite_end_pointers" {
				oppFound = true
			}
		}

		if !tpFound {
			t.Errorf("expected two_pointers to be detected")
		}
		if !oppFound {
			t.Errorf("expected opposite_end_pointers to be detected")
		}
	})

	// 6. DP with memoization
	t.Run("dp with memoization", func(t *testing.T) {
		code := `
def solve(n):
    memo = {}
    def helper(i):
        if i <= 1:
            return 1
        if i in memo:
            return memo[i]
        memo[i] = helper(i - 1) + helper(i - 2)
        return memo[i]
    return helper(n)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var dpFound, memoFound bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "dynamic_programming" {
				dpFound = true
			}
			if c.ConceptID == "memoization" {
				memoFound = true
			}
		}

		if !dpFound {
			t.Errorf("expected dynamic_programming to be detected")
		}
		if !memoFound {
			t.Errorf("expected memoization to be detected")
		}
	})

	// 7. Dead hashmap code (allocated but unused)
	t.Run("dead hashmap allocated but unused", func(t *testing.T) {
		code := `
def solve(nums):
    unused_map = {}
    return len(nums)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" {
				if c.OutputRelevant {
					t.Errorf("expected unused hashmap to NOT be output-relevant")
				}
			}
		}
	})

	// 8. Tabulation bottom-up DP
	t.Run("dp tabulation", func(t *testing.T) {
		code := `
def solve(n):
    dp = [0] * (n + 1)
    dp[0] = 1
    dp[1] = 1
    for i in range(2, n + 1):
        dp[i] = dp[i - 1] + dp[i - 2]
    return dp[n]
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var tabFound, dpFound bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "tabulation" {
				tabFound = true
			}
			if c.ConceptID == "dynamic_programming" {
				dpFound = true
			}
		}

		if !tabFound {
			t.Errorf("expected tabulation to be detected")
		}
		if !dpFound {
			t.Errorf("expected dynamic_programming to be detected")
		}
	})

	// 9. Sorting disguised through helper and multi-strategy (Sorting + Two Pointers)
	t.Run("sorting in helper with two pointers", func(t *testing.T) {
		code := `
def sort_arr(arr):
    arr.sort()
    return arr

def solve(nums, target):
    sorted_nums = sort_arr(nums)
    left = 0
    right = len(sorted_nums) - 1
    while left < right:
        curr = sorted_nums[left] + sorted_nums[right]
        if curr == target:
            return True
        elif curr < target:
            left += 1
        else:
            right -= 1
    return False
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sortFound, tpFound bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "sorting" && c.Reachable {
				sortFound = true
			}
			if c.ConceptID == "two_pointers" && c.Reachable {
				tpFound = true
			}
		}

		if !sortFound {
			t.Errorf("expected reachable sorting to be detected")
		}
		if !tpFound {
			t.Errorf("expected reachable two_pointers to be detected")
		}
	})

	// 10. Pure array/list indexing does NOT detect hashmap
	t.Run("list indexing does not detect hashmap", func(t *testing.T) {
		code := `
def solve(nums):
    arr = [0] * len(nums)
    for i in range(len(nums)):
        arr[i] = nums[i] * 2
    return arr[0]
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" {
				t.Errorf("list indexing must NOT detect hashmap, but got: %+v", c)
			}
		}
	})

	// 11. Recursive factorial does NOT detect DFS
	t.Run("recursive factorial emits recursion but not dfs", func(t *testing.T) {
		code := `
def solve(n):
    if n <= 1:
        return 1
    return n * solve(n - 1)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasRecursion := false
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "recursion" {
				hasRecursion = true
			}
			if c.ConceptID == "dfs" {
				t.Errorf("recursive factorial must NOT be detected as DFS")
			}
		}
		if !hasRecursion {
			t.Errorf("expected recursion to be detected")
		}
	})

	// 12. Two pointers without midpoint does NOT detect binary search
	t.Run("two pointers without midpoint does not detect binary search", func(t *testing.T) {
		code := `
def solve(nums, target):
    l, r = 0, len(nums) - 1
    while l < r:
        s = nums[l] + nums[r]
        if s == target:
            return True
        elif s < target:
            l += 1
        else:
            r -= 1
    return False
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "binary_search" {
				t.Errorf("two pointers without midpoint must NOT detect binary_search")
			}
		}
	})

	// 13. Dummy if guard does not make dead variables output-relevant
	t.Run("dummy if guard does not make dead variables output-relevant", func(t *testing.T) {
		code := `
def solve(nums):
    counts = {}
    for x in nums:
        counts[x] = counts.get(x, 0) + 1
    if len(counts) > 0:
        pass
    return max(nums)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" && c.OutputRelevant {
				t.Errorf("hashmap guarded by dummy 'pass' if condition must NOT be output-relevant")
			}
		}
	})

	// 14. Uncalled nested helper is NOT marked reachable
	t.Run("uncalled nested helper is unreachable", func(t *testing.T) {
		code := `
def solve(nums):
    def unused_helper():
        return {}
    return nums[0]
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "hashmap" && c.Reachable {
				t.Errorf("uncalled nested helper must NOT be marked reachable")
			}
		}
	})

	// 15. Simple queue drain without enqueue or graph context does NOT detect BFS (Test 7)
	t.Run("simple queue drain is not bfs", func(t *testing.T) {
		code := `
from collections import deque

def solve(items):
    q = deque(items)
    while q:
        x = q.popleft()
    return 0
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "bfs" {
				t.Errorf("simple queue drain without enqueue or graph context must NOT detect BFS")
			}
		}
	})

	// 16. Single counter while loop with dict write does NOT detect sliding window (Test 8)
	t.Run("single counter while loop is not sliding window", func(t *testing.T) {
		code := `
def solve(nums):
    state = {}
    i = 0
    while i < len(nums):
        state[nums[i]] = i
        i += 1
    return len(state)
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "sliding_window" {
				t.Errorf("single counter while loop must NOT detect sliding window")
			}
		}
	})

	// 17. Bisection calculation without mid name detects binary search
	t.Run("bisection calculation detects binary search", func(t *testing.T) {
		code := `
def solve(nums, target):
    l = 0
    r = len(nums) - 1
    while l <= r:
        m = (l + r) // 2
        if nums[m] == target:
            return m
        elif nums[m] < target:
            l = m + 1
        else:
            r = m - 1
    return -1
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var bsFound bool
		for _, c := range res.ActualConcepts {
			if c.ConceptID == "binary_search" {
				bsFound = true
				break
			}
		}
		if !bsFound {
			t.Errorf("expected binary_search to be detected via bisection operation")
		}
	})

	// 18. Unused sorting is NOT marked output-relevant (Test 5)
	t.Run("unused sorting is not output-relevant", func(t *testing.T) {
		code := `
def solve(nums):
    copy = nums[:]
    copy.sort()
    return nums[0]
`
		res, err := analyzer.Analyze(ctx, []byte(code), analysis.AnalysisContract{Entrypoint: "solve"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, c := range res.ActualConcepts {
			if c.ConceptID == "sorting" && c.OutputRelevant {
				t.Errorf("unused sorting of detached copy must NOT be output-relevant")
			}
		}
	})
}
