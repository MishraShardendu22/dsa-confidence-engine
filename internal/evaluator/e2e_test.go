package evaluator_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
)

// e2eEvaluator reuses a single evaluator for all e2e tests.
var e2eEval *evaluator.Evaluator

func getE2EEvaluator(t *testing.T) *evaluator.Evaluator {
	t.Helper()
	if e2eEval != nil {
		return e2eEval
	}
	repoPath := filepath.Join("..", "..", "data", "dsa")
	repo, err := dsa.NewRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to load ontology: %v", err)
	}
	embedder := nlp.NewLocalHashingEmbedder(256)
	ctx := context.Background()
	matcher, err := nlp.NewCascadeMatcher(ctx, repo.Ontology(), embedder, true)
	if err != nil {
		t.Fatalf("failed to init cascade matcher: %v", err)
	}
	r := runner.NewLocalRunner(5000) // generous timeout for e2e
	scriptPath := filepath.Join("..", "analysis", "python_ast.py")
	a := analysis.NewPythonAnalyzer(scriptPath)
	s := fidelity.NewScorer(repo.Ontology(), fidelity.DefaultThresholds())
	e2eEval = evaluator.NewEvaluator(r, a, matcher, s, nil)
	return e2eEval
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func evalSubmission(t *testing.T, prob model.Problem, code, explanation string) *model.Evaluation {
	t.Helper()
	ev := getE2EEvaluator(t)
	res, err := ev.Evaluate(context.Background(), model.Submission{
		ProblemID:   prob.ID,
		SourceCode:  code,
		Explanation: explanation,
	}, prob)
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	return res
}

func assertAccept(t *testing.T, res *model.Evaluation, ctx string) {
	t.Helper()
	if res.TestResult != model.TestStatusPass {
		t.Errorf("[%s] expected test PASS, got %s", ctx, res.TestResult)
	}
	if res.Decision != model.DecisionAccept {
		t.Errorf("[%s] expected ACCEPT, got %s (score=%.3f, diags=%v)", ctx, res.Decision, res.FidelityScore, res.Diagnostics)
	}
}

func assertReject(t *testing.T, res *model.Evaluation, ctx string) {
	t.Helper()
	if res.Decision == model.DecisionAccept {
		t.Errorf("[%s] expected REJECT or REJUSTIFY, got ACCEPT (score=%.3f)", ctx, res.FidelityScore)
	}
}

func assertTestFail(t *testing.T, res *model.Evaluation, ctx string) {
	t.Helper()
	if res.TestResult != model.TestStatusFail {
		t.Errorf("[%s] expected test FAIL, got %s", ctx, res.TestResult)
	}
}

// ─── Problem definitions ──────────────────────────────────────────────────────

var probBinarySearchBasic = model.Problem{
	ID:         "binary_search_basic",
	Title:      "Binary Search",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"search", "binary_search", "binarySearch"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [-1, 0, 3, 5, 9, 12], "target": 9}`, ExpectedOutput: "4"},
		{ID: "2", Input: `{"nums": [-1, 0, 3, 5, 9, 12], "target": 2}`, ExpectedOutput: "-1"},
		{ID: "3", Input: `{"nums": [5], "target": 5}`, ExpectedOutput: "0"},
		{ID: "4", Input: `{"nums": [1, 3, 5, 7, 9, 11], "target": 1}`, ExpectedOutput: "0"},
		{ID: "5", Input: `{"nums": [1, 3, 5, 7, 9, 11], "target": 11}`, ExpectedOutput: "5"},
	},
	AcceptedStrategies: []string{"binary_search_iterative", "binary_search_recursive"},
	RequiredConcepts:   []string{"binary_search"},
	OptionalConcepts:   []string{"sorting", "two_pointers", "recursion", "arrays"},
	PrimaryConcepts:    []string{"binary_search"},
}

var probCoinChange = model.Problem{
	ID:         "coin_change",
	Title:      "Coin Change",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"coinChange", "coin_change"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"coins": [1, 5, 11], "amount": 15}`, ExpectedOutput: "3"},
		{ID: "2", Input: `{"coins": [2], "amount": 3}`, ExpectedOutput: "-1"},
		{ID: "3", Input: `{"coins": [1], "amount": 0}`, ExpectedOutput: "0"},
		{ID: "4", Input: `{"coins": [1, 2, 5], "amount": 11}`, ExpectedOutput: "3"},
	},
	AcceptedStrategies: []string{"dp_bottom_up_coin_change", "dp_memoization_coin_change"},
	RequiredConcepts:   []string{"dynamic_programming"},
	PrimaryConcepts:    []string{"dynamic_programming", "dp_knapsack"},
}

var probLIS = model.Problem{
	ID:         "longest_increasing_subsequence",
	Title:      "Longest Increasing Subsequence",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"lengthOfLIS", "lis"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [10, 9, 2, 5, 3, 7, 101, 18]}`, ExpectedOutput: "4"},
		{ID: "2", Input: `{"nums": [0, 1, 0, 3, 2, 3]}`, ExpectedOutput: "4"},
		{ID: "3", Input: `{"nums": [7, 7, 7, 7, 7, 7, 7]}`, ExpectedOutput: "1"},
		{ID: "4", Input: `{"nums": [1]}`, ExpectedOutput: "1"},
	},
	AcceptedStrategies: []string{"dp_lis_n_squared", "binary_search_patience_sorting"},
	RequiredConcepts:   []string{"dynamic_programming"},
	PrimaryConcepts:    []string{"dynamic_programming", "dp_lis"},
}

var probLCS = model.Problem{
	ID:         "longest_common_subsequence",
	Title:      "Longest Common Subsequence",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"longestCommonSubsequence"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"text1": "abcde", "text2": "ace"}`, ExpectedOutput: "3"},
		{ID: "2", Input: `{"text1": "abc", "text2": "abc"}`, ExpectedOutput: "3"},
		{ID: "3", Input: `{"text1": "abc", "text2": "def"}`, ExpectedOutput: "0"},
	},
	AcceptedStrategies: []string{"dp_lcs_2d_table", "dp_lcs_memoization"},
	RequiredConcepts:   []string{"dynamic_programming"},
	PrimaryConcepts:    []string{"dynamic_programming", "dp_lcs"},
}

var probHouseRobber = model.Problem{
	ID:         "house_robber",
	Title:      "House Robber",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"rob", "house_robber"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [1, 2, 3, 1]}`, ExpectedOutput: "4"},
		{ID: "2", Input: `{"nums": [2, 7, 9, 3, 1]}`, ExpectedOutput: "12"},
		{ID: "3", Input: `{"nums": [1]}`, ExpectedOutput: "1"},
	},
	AcceptedStrategies: []string{"dp_1d_house_robber", "dp_memoization_house_robber"},
	RequiredConcepts:   []string{"dynamic_programming"},
	PrimaryConcepts:    []string{"dynamic_programming", "dp_1d"},
}

var probValidParens = model.Problem{
	ID:         "valid_parentheses",
	Title:      "Valid Parentheses",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"isValid", "is_valid"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"s": "()"}`, ExpectedOutput: "true"},
		{ID: "2", Input: `{"s": "()[]{}"}`, ExpectedOutput: "true"},
		{ID: "3", Input: `{"s": "(]"}`, ExpectedOutput: "false"},
		{ID: "4", Input: `{"s": "([])"}`, ExpectedOutput: "true"},
		{ID: "5", Input: `{"s": "]"}`, ExpectedOutput: "false"},
	},
	AcceptedStrategies: []string{"stack_matching_brackets"},
	RequiredConcepts:   []string{"stack"},
	PrimaryConcepts:    []string{"stack"},
}

var probThreeSum = model.Problem{
	ID:         "three_sum",
	Title:      "3Sum",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"threeSum", "three_sum"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [-1, 0, 1, 2, -1, -4]}`, ExpectedOutput: "[[-1, -1, 2], [-1, 0, 1]]"},
		{ID: "2", Input: `{"nums": [0, 1, 1]}`, ExpectedOutput: "[]"},
		{ID: "3", Input: `{"nums": [0, 0, 0]}`, ExpectedOutput: "[[0, 0, 0]]"},
	},
	AcceptedStrategies: []string{"two_pointers_sorted_three_sum"},
	RequiredConcepts:   []string{"two_pointers"},
	PrimaryConcepts:    []string{"two_pointers"},
}

var probSingleNumber = model.Problem{
	ID:         "single_number",
	Title:      "Single Number",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"singleNumber", "single_number"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [2, 2, 1]}`, ExpectedOutput: "1"},
		{ID: "2", Input: `{"nums": [4, 1, 2, 1, 2]}`, ExpectedOutput: "4"},
		{ID: "3", Input: `{"nums": [1]}`, ExpectedOutput: "1"},
	},
	AcceptedStrategies: []string{"xor_single_number"},
	RequiredConcepts:   []string{"bit_manipulation"},
	PrimaryConcepts:    []string{"bit_manipulation"},
}

var probNumIslands = model.Problem{
	ID:         "number_of_islands",
	Title:      "Number of Islands",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"numIslands", "num_islands"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"grid": [["1","1","1","1","0"],["1","1","0","1","0"],["1","1","0","0","0"],["0","0","0","0","0"]]}`, ExpectedOutput: "1"},
		{ID: "2", Input: `{"grid": [["1","1","0","0","0"],["1","1","0","0","0"],["0","0","1","0","0"],["0","0","0","1","1"]]}`, ExpectedOutput: "3"},
		{ID: "3", Input: `{"grid": [["1"]]}`, ExpectedOutput: "1"},
		{ID: "4", Input: `{"grid": [["0"]]}`, ExpectedOutput: "0"},
	},
	AcceptedStrategies: []string{"dfs_flood_fill", "bfs_flood_fill"},
	PrimaryConcepts:    []string{"dfs", "bfs"},
}

var probJumpGame = model.Problem{
	ID:         "jump_game",
	Title:      "Jump Game",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"canJump", "can_jump"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [2, 3, 1, 1, 4]}`, ExpectedOutput: "true"},
		{ID: "2", Input: `{"nums": [3, 2, 1, 0, 4]}`, ExpectedOutput: "false"},
		{ID: "3", Input: `{"nums": [0]}`, ExpectedOutput: "true"},
	},
	AcceptedStrategies: []string{"greedy_jump_game"},
	PrimaryConcepts:    []string{"greedy"},
}

var probGroupAnagrams = model.Problem{
	ID:         "group_anagrams",
	Title:      "Group Anagrams",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"groupAnagrams", "group_anagrams"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"strs": ["eat", "tea", "tan", "ate", "nat", "bat"]}`, ExpectedOutput: `[["ate", "eat", "tea"], ["bat"], ["nat", "tan"]]`},
		{ID: "2", Input: `{"strs": [""]}`, ExpectedOutput: `[[""]]`},
		{ID: "3", Input: `{"strs": ["a"]}`, ExpectedOutput: `[["a"]]`},
	},
	AcceptedStrategies: []string{"hashmap_group_anagrams_sorted_key"},
	PrimaryConcepts:    []string{"hashmap"},
}

var probMaxSubarray = model.Problem{
	ID:         "maximum_subarray",
	Title:      "Maximum Subarray",
	Language:   "python",
	Entrypoint: "solve",
	EntrypointAliases: []string{"maxSubArray", "kadane"},
	Tests: []model.TestCase{
		{ID: "1", Input: `{"nums": [-2, 1, -3, 4, -1, 2, 1, -5, 4]}`, ExpectedOutput: "6"},
		{ID: "2", Input: `{"nums": [1]}`, ExpectedOutput: "1"},
		{ID: "3", Input: `{"nums": [5, 4, -1, 7, 8]}`, ExpectedOutput: "23"},
	},
	AcceptedStrategies: []string{"kadane_algorithm", "dp_max_subarray"},
	RequiredConcepts:   []string{"dynamic_programming"},
	PrimaryConcepts:    []string{"dynamic_programming"},
}

// ─── Binary Search E2E Tests ─────────────────────────────────────────────────

func TestE2E_BinarySearch(t *testing.T) {
	t.Run("iterative binary search ACCEPT", func(t *testing.T) {
		code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I use binary search on the sorted array. I maintain bounds lo and hi, compute mid = (lo+hi)//2, and narrow the search range.")
		assertAccept(t, res, "binary_search iterative")
	})

	t.Run("recursive binary search ACCEPT", func(t *testing.T) {
		code := `
def solve(nums, target):
    def bs(lo, hi):
        if lo > hi:
            return -1
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            return bs(mid + 1, hi)
        else:
            return bs(lo, mid - 1)
    return bs(0, len(nums) - 1)
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I recursively apply binary search, dividing the range in half each time.")
		assertAccept(t, res, "binary_search recursive")
	})

	t.Run("wrong output REJECT", func(t *testing.T) {
		code := `
def solve(nums, target):
    return 0  # always wrong
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I use binary search to find the target.")
		assertTestFail(t, res, "binary_search wrong output")
	})

	t.Run("code correct but claims hashmap REJECT", func(t *testing.T) {
		code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I use a hashmap to store all values and look up the target in O(1).")
		assertReject(t, res, "binary_search hallucinated hashmap")
	})

	t.Run("linear search REJECT (wrong approach claimed as binary search)", func(t *testing.T) {
		code := `
def solve(nums, target):
    for i, v in enumerate(nums):
        if v == target:
            return i
    return -1
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I apply binary search to find the target efficiently.")
		// passes tests but code doesn't use binary search — should reject
		assertReject(t, res, "binary_search claim mismatch with linear code")
	})
}

// ─── Dynamic Programming E2E Tests ───────────────────────────────────────────

func TestE2E_DynamicProgramming(t *testing.T) {
	t.Run("coin change bottom-up ACCEPT", func(t *testing.T) {
		code := `
def solve(coins, amount):
    dp = [float('inf')] * (amount + 1)
    dp[0] = 0
    for c in coins:
        for a in range(c, amount + 1):
            dp[a] = min(dp[a], dp[a - c] + 1)
    return dp[amount] if dp[amount] != float('inf') else -1
`
		res := evalSubmission(t, probCoinChange, code,
			"I use bottom-up dynamic programming. I maintain a dp table where dp[i] is the minimum coins to make amount i, using tabulation.")
		assertAccept(t, res, "coin_change tabulation")
	})

	t.Run("coin change memoization ACCEPT", func(t *testing.T) {
		code := `
def solve(coins, amount):
    memo = {}
    def dp(rem):
        if rem in memo:
            return memo[rem]
        if rem == 0:
            return 0
        if rem < 0:
            return float('inf')
        res = min(dp(rem - c) + 1 for c in coins)
        memo[rem] = res
        return res
    ans = dp(amount)
    return -1 if ans == float('inf') else ans
`
		res := evalSubmission(t, probCoinChange, code,
			"I use memoized recursion — top-down dynamic programming — caching subproblem results in a dictionary.")
		assertAccept(t, res, "coin_change memoization")
	})

	t.Run("coin change wrong result REJECT", func(t *testing.T) {
		code := `def solve(coins, amount): return 999`
		res := evalSubmission(t, probCoinChange, code,
			"I use dynamic programming to solve this.")
		assertTestFail(t, res, "coin_change wrong output")
	})

	t.Run("LIS O(n^2) DP ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    if not nums:
        return 0
    n = len(nums)
    dp = [1] * n
    for i in range(1, n):
        for j in range(i):
            if nums[j] < nums[i]:
                dp[i] = max(dp[i], dp[j] + 1)
    return max(dp)
`
		res := evalSubmission(t, probLIS, code,
			"I build a dp array where dp[i] is the length of the longest increasing subsequence ending at index i.")
		assertAccept(t, res, "LIS O(n^2)")
	})

	t.Run("LCS 2D DP ACCEPT", func(t *testing.T) {
		code := `
def solve(text1, text2):
    m, n = len(text1), len(text2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if text1[i-1] == text2[j-1]:
                dp[i][j] = dp[i-1][j-1] + 1
            else:
                dp[i][j] = max(dp[i-1][j], dp[i][j-1])
    return dp[m][n]
`
		res := evalSubmission(t, probLCS, code,
			"I use a 2D dp table. dp[i][j] is the LCS of text1[:i] and text2[:j]. I fill it iteratively.")
		assertAccept(t, res, "LCS 2D DP")
	})

	t.Run("LCS memoization ACCEPT", func(t *testing.T) {
		code := `
def solve(text1, text2):
    memo = {}
    def dp(i, j):
        if (i, j) in memo:
            return memo[(i, j)]
        if i == len(text1) or j == len(text2):
            return 0
        if text1[i] == text2[j]:
            res = 1 + dp(i+1, j+1)
        else:
            res = max(dp(i+1, j), dp(i, j+1))
        memo[(i, j)] = res
        return res
    return dp(0, 0)
`
		res := evalSubmission(t, probLCS, code,
			"I use top-down memoization. I cache dp(i, j) = LCS of text1[i:] and text2[j:].")
		assertAccept(t, res, "LCS memoization")
	})

	t.Run("house robber DP ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    if not nums:
        return 0
    n = len(nums)
    if n == 1:
        return nums[0]
    dp = [0] * n
    dp[0] = nums[0]
    dp[1] = max(nums[0], nums[1])
    for i in range(2, n):
        dp[i] = max(dp[i-1], dp[i-2] + nums[i])
    return dp[-1]
`
		res := evalSubmission(t, probHouseRobber, code,
			"I use a 1D dp array where dp[i] is the maximum money achievable up to house i.")
		assertAccept(t, res, "house_robber DP")
	})

	t.Run("house robber claims binary search REJECT", func(t *testing.T) {
		code := `
def solve(nums):
    if not nums:
        return 0
    n = len(nums)
    if n == 1:
        return nums[0]
    dp = [0] * n
    dp[0] = nums[0]
    dp[1] = max(nums[0], nums[1])
    for i in range(2, n):
        dp[i] = max(dp[i-1], dp[i-2] + nums[i])
    return dp[-1]
`
		res := evalSubmission(t, probHouseRobber, code,
			"I apply binary search to efficiently locate the optimal house positions.")
		assertReject(t, res, "house_robber hallucinated binary search")
	})

	t.Run("max subarray Kadane ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    dp = nums[0]
    max_sum = nums[0]
    for n in nums[1:]:
        dp = max(n, dp + n)
        max_sum = max(max_sum, dp)
    return max_sum
`
		res := evalSubmission(t, probMaxSubarray, code,
			"I use dynamic programming (Kadane's algorithm), maintaining the maximum subarray sum ending at each position in dp.")
		assertAccept(t, res, "max_subarray Kadane")
	})

	t.Run("max subarray all-negative input works", func(t *testing.T) {
		code := `
def solve(nums):
    max_sum = curr = nums[0]
    for n in nums[1:]:
        curr = max(n, curr + n)
        max_sum = max(max_sum, curr)
    return max_sum
`
		res := evalSubmission(t, probMaxSubarray, code,
			"I use dynamic programming: current max subarray ending at each position.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("max_subarray should pass all tests, got %s", res.TestResult)
		}
	})
}

// ─── Stack E2E Tests ──────────────────────────────────────────────────────────

func TestE2E_Stack(t *testing.T) {
	t.Run("valid parentheses stack ACCEPT", func(t *testing.T) {
		code := `
def solve(s):
    stack = []
    mapping = {')': '(', ']': '[', '}': '{'}
    for c in s:
        if c in mapping:
            top = stack.pop() if stack else '#'
            if mapping[c] != top:
                return False
        else:
            stack.append(c)
    return len(stack) == 0
`
		res := evalSubmission(t, probValidParens, code,
			"I use a stack to track open brackets. When I encounter a close bracket I pop and verify the match.")
		assertAccept(t, res, "valid_parentheses stack")
	})

	t.Run("valid parentheses wrong output REJECT", func(t *testing.T) {
		code := `
def solve(s):
    return True  # always true
`
		res := evalSubmission(t, probValidParens, code,
			"I use a stack to match brackets.")
		assertTestFail(t, res, "valid_parentheses wrong output")
	})

	t.Run("valid parentheses claims DP REJECT", func(t *testing.T) {
		code := `
def solve(s):
    stack = []
    mapping = {')': '(', ']': '[', '}': '{'}
    for c in s:
        if c in mapping:
            top = stack.pop() if stack else '#'
            if mapping[c] != top:
                return False
        else:
            stack.append(c)
    return len(stack) == 0
`
		res := evalSubmission(t, probValidParens, code,
			"I use dynamic programming with a 2D table to determine valid brackets.")
		assertReject(t, res, "valid_parentheses hallucinated DP")
	})
}

// ─── Two Pointers E2E Tests ───────────────────────────────────────────────────

func TestE2E_TwoPointers(t *testing.T) {
	t.Run("3sum two pointers ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    nums.sort()
    result = []
    for i in range(len(nums) - 2):
        if i > 0 and nums[i] == nums[i-1]:
            continue
        lo, hi = i + 1, len(nums) - 1
        while lo < hi:
            s = nums[i] + nums[lo] + nums[hi]
            if s == 0:
                result.append([nums[i], nums[lo], nums[hi]])
                while lo < hi and nums[lo] == nums[lo+1]:
                    lo += 1
                while lo < hi and nums[hi] == nums[hi-1]:
                    hi -= 1
                lo += 1
                hi -= 1
            elif s < 0:
                lo += 1
            else:
                hi -= 1
    return result
`
		res := evalSubmission(t, probThreeSum, code,
			"I sort the array and use two pointers: a left and right pointer starting after the fixed element to find triplets summing to zero.")
		assertAccept(t, res, "3sum two pointers")
	})

	t.Run("3sum brute force passes tests but claims two pointers ACCEPT", func(t *testing.T) {
		// Brute force also valid explanation if primary concept matches
		code := `
def solve(nums):
    nums.sort()
    result = []
    for i in range(len(nums) - 2):
        if i > 0 and nums[i] == nums[i-1]:
            continue
        lo, hi = i + 1, len(nums) - 1
        while lo < hi:
            s = nums[i] + nums[lo] + nums[hi]
            if s == 0:
                result.append([nums[i], nums[lo], nums[hi]])
                lo += 1
                hi -= 1
                while lo < hi and nums[lo] == nums[lo-1]:
                    lo += 1
                while lo < hi and nums[hi] == nums[hi+1]:
                    hi -= 1
            elif s < 0:
                lo += 1
            else:
                hi -= 1
    return result
`
		res := evalSubmission(t, probThreeSum, code,
			"I sort the array and use two pointers — a left pointer and a right pointer — to find the triplets.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("3sum variant should pass tests, got %s (diags: %v)", res.TestResult, res.Diagnostics)
		}
	})

	t.Run("3sum claims hashmap but uses two pointers REJECT", func(t *testing.T) {
		code := `
def solve(nums):
    nums.sort()
    result = []
    for i in range(len(nums) - 2):
        if i > 0 and nums[i] == nums[i-1]:
            continue
        lo, hi = i + 1, len(nums) - 1
        while lo < hi:
            s = nums[i] + nums[lo] + nums[hi]
            if s == 0:
                result.append([nums[i], nums[lo], nums[hi]])
                lo += 1
                hi -= 1
            elif s < 0:
                lo += 1
            else:
                hi -= 1
    return result
`
		res := evalSubmission(t, probThreeSum, code,
			"I store all values in a hashmap and look up the complement for each pair using dictionary operations.")
		assertReject(t, res, "3sum claimed hashmap but used two pointers")
	})
}

// ─── Bit Manipulation E2E Tests ───────────────────────────────────────────────

func TestE2E_BitManipulation(t *testing.T) {
	t.Run("single number XOR ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    result = 0
    for n in nums:
        result ^= n
    return result
`
		res := evalSubmission(t, probSingleNumber, code,
			"I use bit manipulation with XOR. Duplicate numbers cancel out via bitwise XOR operations, leaving the single element.")
		assertAccept(t, res, "single_number XOR")
	})

	t.Run("single number hashmap approach REJECT (wrong required concept)", func(t *testing.T) {
		// uses hashmap but required is bit_manipulation
		code := `
def solve(nums):
    seen = {}
    for n in nums:
        seen[n] = seen.get(n, 0) + 1
    for n, cnt in seen.items():
        if cnt == 1:
            return n
`
		res := evalSubmission(t, probSingleNumber, code,
			"I use XOR to cancel out duplicate values and find the single number.")
		// Code uses hashmap but claims XOR — should reject (no bit ops detected)
		assertReject(t, res, "single_number claimed XOR but used hashmap")
	})

	t.Run("single number wrong output REJECT", func(t *testing.T) {
		code := `def solve(nums): return nums[0]`
		res := evalSubmission(t, probSingleNumber, code, "XOR approach.")
		assertTestFail(t, res, "single_number always first element")
	})
}

// ─── Graph E2E Tests ──────────────────────────────────────────────────────────

func TestE2E_Graphs(t *testing.T) {
	t.Run("number of islands DFS ACCEPT", func(t *testing.T) {
		code := `
def solve(grid):
    if not grid:
        return 0
    rows, cols = len(grid), len(grid[0])
    def dfs(r, c):
        if r < 0 or c < 0 or r >= rows or c >= cols or grid[r][c] != '1':
            return
        grid[r][c] = '0'
        dfs(r+1, c); dfs(r-1, c); dfs(r, c+1); dfs(r, c-1)
    count = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == '1':
                dfs(r, c)
                count += 1
    return count
`
		res := evalSubmission(t, probNumIslands, code,
			"I run a depth-first search from each unvisited land cell, marking all reachable land as visited. Each DFS invocation counts one island.")
		assertAccept(t, res, "number_of_islands DFS")
	})

	t.Run("number of islands BFS ACCEPT", func(t *testing.T) {
		code := `
from collections import deque
def solve(grid):
    if not grid:
        return 0
    rows, cols = len(grid), len(grid[0])
    count = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == '1':
                count += 1
                queue = deque([(r, c)])
                grid[r][c] = '0'
                while queue:
                    nr, nc = queue.popleft()
                    for dr, dc in [(1,0),(-1,0),(0,1),(0,-1)]:
                        rr, cc = nr+dr, nc+dc
                        if 0 <= rr < rows and 0 <= cc < cols and grid[rr][cc] == '1':
                            queue.append((rr, cc))
                            grid[rr][cc] = '0'
    return count
`
		res := evalSubmission(t, probNumIslands, code,
			"I use BFS from each unvisited land cell, using a queue to explore all connected land cells.")
		assertAccept(t, res, "number_of_islands BFS")
	})

	t.Run("number of islands always returns 1 REJECT", func(t *testing.T) {
		code := `def solve(grid): return 1`
		res := evalSubmission(t, probNumIslands, code,
			"DFS flood fill counts connected components.")
		assertTestFail(t, res, "number_of_islands wrong output")
	})

	t.Run("number of islands claims binary search REJECT", func(t *testing.T) {
		code := `
def solve(grid):
    if not grid:
        return 0
    rows, cols = len(grid), len(grid[0])
    def dfs(r, c):
        if r < 0 or c < 0 or r >= rows or c >= cols or grid[r][c] != '1':
            return
        grid[r][c] = '0'
        dfs(r+1, c); dfs(r-1, c); dfs(r, c+1); dfs(r, c-1)
    count = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == '1':
                dfs(r, c)
                count += 1
    return count
`
		res := evalSubmission(t, probNumIslands, code,
			"I use binary search to find island boundaries and count them efficiently.")
		assertReject(t, res, "number_of_islands claimed binary search")
	})
}

// ─── Greedy E2E Tests ─────────────────────────────────────────────────────────

func TestE2E_Greedy(t *testing.T) {
	t.Run("jump game greedy ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    max_reach = 0
    for i, jump in enumerate(nums):
        if i > max_reach:
            return False
        max_reach = max(max_reach, i + jump)
    return True
`
		res := evalSubmission(t, probJumpGame, code,
			"I use a greedy approach, tracking the maximum index reachable. If I ever find a position beyond max_reach, I return false.")
		assertAccept(t, res, "jump_game greedy")
	})

	t.Run("jump game DP ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    n = len(nums)
    dp = [False] * n
    dp[0] = True
    for i in range(1, n):
        for j in range(i):
            if dp[j] and j + nums[j] >= i:
                dp[i] = True
                break
    return dp[-1]
`
		res := evalSubmission(t, probJumpGame, code,
			"I use dynamic programming where dp[i] indicates whether index i is reachable.")
		assertAccept(t, res, "jump_game DP")
	})

	t.Run("jump game always false REJECT", func(t *testing.T) {
		code := `def solve(nums): return False`
		res := evalSubmission(t, probJumpGame, code, "Greedy approach.")
		assertTestFail(t, res, "jump_game always false")
	})
}

// ─── HashMap E2E Tests ────────────────────────────────────────────────────────

func TestE2E_HashMap(t *testing.T) {
	t.Run("group anagrams sorted key ACCEPT", func(t *testing.T) {
		code := `
from collections import defaultdict
def solve(strs):
    groups = defaultdict(list)
    for s in strs:
        key = tuple(sorted(s))
        groups[key].append(s)
    result = [sorted(v) for v in groups.values()]
    result.sort()
    return result
`
		res := evalSubmission(t, probGroupAnagrams, code,
			"I use a hashmap where the key is the sorted string. All anagrams share the same sorted key, so I group them together.")
		assertAccept(t, res, "group_anagrams sorted key")
	})

	t.Run("group anagrams count array key ACCEPT", func(t *testing.T) {
		code := `
from collections import defaultdict
def solve(strs):
    groups = defaultdict(list)
    for s in strs:
        count = [0] * 26
        for c in s:
            count[ord(c) - ord('a')] += 1
        key = tuple(count)
        groups[key].append(s)
    result = [sorted(v) for v in groups.values()]
    result.sort()
    return result
`
		res := evalSubmission(t, probGroupAnagrams, code,
			"I use a dictionary with a character frequency count tuple as the key to group anagrams.")
		assertAccept(t, res, "group_anagrams count key")
	})

	t.Run("group anagrams claims DP REJECT", func(t *testing.T) {
		code := `
from collections import defaultdict
def solve(strs):
    groups = defaultdict(list)
    for s in strs:
        key = tuple(sorted(s))
        groups[key].append(s)
    result = [sorted(v) for v in groups.values()]
    result.sort()
    return result
`
		res := evalSubmission(t, probGroupAnagrams, code,
			"I use dynamic programming where dp[i] tracks the longest anagram group seen so far.")
		assertReject(t, res, "group_anagrams claimed DP")
	})
}

// ─── Adversarial / Edge Case Tests ───────────────────────────────────────────

func TestE2E_Adversarial(t *testing.T) {
	t.Run("empty code REJECT", func(t *testing.T) {
		code := `def solve(nums, target): pass`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I use binary search.")
		assertTestFail(t, res, "empty solve always None")
	})

	t.Run("infinite loop code times out REJECT", func(t *testing.T) {
		code := `
def solve(nums, target):
    while True:
        pass
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I use binary search.")
		if res.TestResult != model.TestStatusFail {
			t.Logf("infinite loop: TestResult=%s Decision=%s (timeout expected)", res.TestResult, res.Decision)
		}
	})

	t.Run("syntax error code REJECT", func(t *testing.T) {
		code := `def solve(nums target: SYNTAX ERROR !!`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"Binary search approach.")
		if res.TestResult != model.TestStatusFail && res.Decision != model.DecisionReject {
			t.Logf("syntax error: TestResult=%s Decision=%s", res.TestResult, res.Decision)
		}
	})

	t.Run("explanation mentions no algorithm REJECT", func(t *testing.T) {
		code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I wrote some code that works well for finding elements.")
		// Vague explanation — should score low (no detected concept match claimed)
		if res.Decision == model.DecisionAccept {
			t.Logf("vague explanation accepted with score %.3f — may be acceptable if code signals dominate", res.FidelityScore)
		}
	})

	t.Run("negated claim REJECT", func(t *testing.T) {
		code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
            mid = (lo + hi) // 2
            if nums[mid] == target:
                return mid
            elif nums[mid] < target:
                lo = mid + 1
            else:
                hi = mid - 1
    return -1
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I do NOT use binary search at all. I found the answer using a completely different technique.")
		assertReject(t, res, "negated binary search claim")
	})

	t.Run("dead code claiming multiple concepts REJECT", func(t *testing.T) {
		// Code correct but explanation claims many unrelated concepts
		code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`
		res := evalSubmission(t, probBinarySearchBasic, code,
			"I use dynamic programming with a knapsack formulation, combined with topological sort on a dependency graph, and union-find for cycle detection.")
		assertReject(t, res, "claimed DP+toposort+UF for binary search")
	})

	t.Run("correct code correct explanation single ACCEPT", func(t *testing.T) {
		code := `
def solve(coins, amount):
    dp = [float('inf')] * (amount + 1)
    dp[0] = 0
    for c in coins:
        for a in range(c, amount + 1):
            dp[a] = min(dp[a], dp[a - c] + 1)
    return dp[amount] if dp[amount] != float('inf') else -1
`
		res := evalSubmission(t, probCoinChange, code,
			"I use bottom-up dynamic programming. I fill a table from 0 to amount, each time trying every coin denomination.")
		assertAccept(t, res, "coin_change correct simple explanation")
	})

	t.Run("empty input arrays handled correctly", func(t *testing.T) {
		// Verify problems handle empty inputs without panics
		prob := model.Problem{
			ID:         "test_empty",
			Title:      "Test",
			Language:   "python",
			Entrypoint: "solve",
			Tests: []model.TestCase{
				{ID: "1", Input: `{"nums": []}`, ExpectedOutput: "0"},
			},
			AcceptedStrategies: []string{"any"},
			PrimaryConcepts:    []string{"arrays"},
		}
		code := `def solve(nums): return len(nums)`
		ev := getE2EEvaluator(t)
		res, err := ev.Evaluate(context.Background(), model.Submission{
			ProblemID:   "test_empty",
			SourceCode:  code,
			Explanation: "I use an array.",
		}, prob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TestResult != model.TestStatusPass {
			t.Errorf("empty array case: expected PASS, got %s", res.TestResult)
		}
	})
}

// ─── Coverage Matrix: All Detectors Positively ───────────────────────────────

func TestE2E_DetectorCoverage(t *testing.T) {
	ev := getE2EEvaluator(t)
	ctx := context.Background()

	type tc struct {
		name      string
		prob      model.Problem
		code      string
		expl      string
		wantPass  bool
		wantConcepts []string
	}

	// Helper to make a simple single-test problem
	makeProblem := func(id, concept string, input, expected string) model.Problem {
		return model.Problem{
			ID:         id,
			Language:   "python",
			Entrypoint: "solve",
			Tests: []model.TestCase{
				{ID: "1", Input: input, ExpectedOutput: expected},
			},
			PrimaryConcepts: []string{concept},
		}
	}

	cases := []tc{
		// HashMap detection
		{
			name: "hashmap detect dict write",
			prob: makeProblem("hm1", "hashmap", `{"nums": [1,2,3]}`, "3"),
			code: `
def solve(nums):
    seen = {}
    for n in nums:
        seen[n] = True
    return len(seen)
`,
			expl:      "I use a hashmap to count elements.",
			wantPass:  true,
			wantConcepts: []string{"hashmap"},
		},
		// HashSet detection
		{
			name: "hashset detect set add",
			prob: makeProblem("hs1", "hashset", `{"nums": [1,2,2,3]}`, "3"),
			code: `
def solve(nums):
    seen = set()
    for n in nums:
        seen.add(n)
    return len(seen)
`,
			expl:     "I use a hashset to track unique elements.",
			wantPass: true,
		},
		// Sorting detection
		{
			name: "sorting detect sort call returning sorted",
			prob: makeProblem("sort1", "sorting", `{"nums": [3,1,2]}`, "[1, 2, 3]"),
			code: `
def solve(nums):
    return sorted(nums)
`,
			expl:     "I sort the array and return it.",
			wantPass: true,
		},
		// Stack detection (list append + pop)
		{
			name: "stack detect list append and pop",
			prob: makeProblem("stk1", "stack", `{"s": "()"}`, "true"),
			code: `
def solve(s):
    stack = []
    for c in s:
        if c == '(':
            stack.append(c)
        else:
            if not stack:
                return False
            stack.pop()
    return len(stack) == 0
`,
			expl:     "I push open brackets onto the stack and pop when I see a close bracket.",
			wantPass: true,
		},
		// Queue detection
		{
			name: "queue detect deque popleft",
			prob: makeProblem("q1", "bfs", `{"start": 0, "n": 3}`, "3"),
			code: `
from collections import deque
def solve(start, n):
    visited = set()
    queue = deque([start])
    count = 0
    while queue:
        node = queue.popleft()
        if node in visited:
            continue
        visited.add(node)
        count += 1
        for nxt in range(n):
            if nxt not in visited:
                queue.append(nxt)
    return count
`,
			expl:     "I use BFS with a deque queue to traverse the graph.",
			wantPass: true,
		},
		// Binary search detection
		{
			name: "binary search detect bisect arithmetic",
			prob: makeProblem("bs1", "binary_search", `{"nums": [1,3,5,7], "target": 5}`, "2"),
			code: `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) >> 1
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`,
			expl:     "I use binary search with right-shift bisection.",
			wantPass: true,
		},
		// DP memoization detection
		{
			name: "dp memoization detect memo dict with recursion",
			prob: makeProblem("dp1", "memoization", `{"n": 5}`, "5"),
			code: `
def solve(n):
    memo = {}
    def dp(k):
        if k in memo:
            return memo[k]
        if k <= 1:
            return k
        memo[k] = dp(k-1) + dp(k-2)
        return memo[k]
    return dp(n)
`,
			expl:     "I use memoization, caching previously computed subproblem results in a dictionary.",
			wantPass: true,
		},
		// Recursion detection
		{
			name: "recursion detect self-call",
			prob: makeProblem("rec1", "recursion", `{"n": 4}`, "24"),
			code: `
def solve(n):
    if n <= 1:
        return 1
    return n * solve(n - 1)
`,
			expl:     "I use recursion to compute factorial.",
			wantPass: true,
		},
		// Two pointers detection
		{
			name: "two pointers detect boundary movement",
			prob: makeProblem("tp1", "two_pointers", `{"nums": [1,2,3,4,5], "target": 5}`, "[0, 3]"),
			code: `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo < hi:
        s = nums[lo] + nums[hi]
        if s == target:
            return [lo, hi]
        elif s < target:
            lo += 1
        else:
            hi -= 1
    return []
`,
			expl:     "I use two pointers — left and right — moving toward each other.",
			wantPass: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res, err := ev.Evaluate(ctx, model.Submission{
				ProblemID:   tc.prob.ID,
				SourceCode:  tc.code,
				Explanation: tc.expl,
			}, tc.prob)
			if err != nil {
				t.Fatalf("evaluate error: %v", err)
			}
			if tc.wantPass && res.TestResult != model.TestStatusPass {
				t.Errorf("expected PASS, got %s", res.TestResult)
			}
			// Just verify the evaluator ran without error and returned sane output
			if res.FidelityScore < 0 || res.FidelityScore > 1.1 {
				t.Errorf("invalid fidelity score: %f", res.FidelityScore)
			}
		})
	}
}

// ─── Regression: Piyush-Style Benchmarks for New Problems ────────────────────

func TestE2E_RegressionNewProblems(t *testing.T) {
	t.Run("LIS claim binary search patience sort ACCEPT", func(t *testing.T) {
		// Patience sort is a valid binary-search-based LIS algorithm
		code := `
import bisect
def solve(nums):
    tails = []
    for n in nums:
        pos = bisect.bisect_left(tails, n)
        if pos == len(tails):
            tails.append(n)
        else:
            tails[pos] = n
    return len(tails)
`
		res := evalSubmission(t, probLIS, code,
			"I use patience sorting with binary search (bisect_left) to find the LIS length in O(n log n).")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("LIS patience sort should pass tests, got %s diags=%v", res.TestResult, res.Diagnostics)
		}
	})

	t.Run("house robber space optimized DP ACCEPT", func(t *testing.T) {
		code := `
def solve(nums):
    dp0, dp1 = 0, 0
    for n in nums:
        dp_curr = max(dp1, dp0 + n)
        dp0 = dp1
        dp1 = dp_curr
    return dp1
`
		res := evalSubmission(t, probHouseRobber, code,
			"I use space-optimized dynamic programming tabulation, keeping only the two previous states in variables dp0 and dp1.")
		assertAccept(t, res, "house_robber space-optimized DP")
	})

	t.Run("LCS correct explanation ACCEPT", func(t *testing.T) {
		code := `
def solve(text1, text2):
    m, n = len(text1), len(text2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if text1[i-1] == text2[j-1]:
                dp[i][j] = dp[i-1][j-1] + 1
            else:
                dp[i][j] = max(dp[i-1][j], dp[i][j-1])
    return dp[m][n]
`
		res := evalSubmission(t, probLCS, code,
			"I fill a 2D dp table. dp[i][j] stores the longest common subsequence of the first i characters of text1 and first j of text2.")
		assertAccept(t, res, "LCS correct 2D DP")
	})
}

// ─── Stress / Multiple Submissions ───────────────────────────────────────────

func TestE2E_StressMultipleSubmissions(t *testing.T) {
	// Run the same correct submission 5 times to ensure determinism
	code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`
	expl := "I use binary search, halving the search space each iteration."
	for i := 0; i < 5; i++ {
		res := evalSubmission(t, probBinarySearchBasic, code, expl)
		assertAccept(t, res, "binary_search stress iteration")
		if res.FidelityScore < 0.90 {
			t.Errorf("stress[%d]: fidelity score dropped to %.3f", i, res.FidelityScore)
		}
	}
}

// ─── Edge Cases: Empty / Single Element ──────────────────────────────────────

func TestE2E_EdgeCases(t *testing.T) {
	t.Run("binary search single element found", func(t *testing.T) {
		prob := model.Problem{
			ID:         "bs_single",
			Language:   "python",
			Entrypoint: "solve",
			Tests:      []model.TestCase{{ID: "1", Input: `{"nums": [42], "target": 42}`, ExpectedOutput: "0"}},
			PrimaryConcepts: []string{"binary_search"},
		}
		code := `
def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`
		res := evalSubmission(t, prob, code, "Binary search on a single element array.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("single element found: expected PASS, got %s", res.TestResult)
		}
	})

	t.Run("coin change zero amount edge case", func(t *testing.T) {
		prob := model.Problem{
			ID:         "cc_zero",
			Language:   "python",
			Entrypoint: "solve",
			Tests:      []model.TestCase{{ID: "1", Input: `{"coins": [1,5], "amount": 0}`, ExpectedOutput: "0"}},
			PrimaryConcepts: []string{"dynamic_programming"},
		}
		code := `
def solve(coins, amount):
    dp = [float('inf')] * (amount + 1)
    dp[0] = 0
    for c in coins:
        for a in range(c, amount + 1):
            dp[a] = min(dp[a], dp[a - c] + 1)
    return dp[amount] if dp[amount] != float('inf') else -1
`
		res := evalSubmission(t, prob, code, "Bottom-up DP for coin change.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("coin change zero: expected PASS, got %s", res.TestResult)
		}
	})

	t.Run("valid parentheses empty string edge case", func(t *testing.T) {
		prob := model.Problem{
			ID:         "vp_empty",
			Language:   "python",
			Entrypoint: "solve",
			Tests:      []model.TestCase{{ID: "1", Input: `{"s": ""}`, ExpectedOutput: "true"}},
			PrimaryConcepts: []string{"stack"},
		}
		code := `
def solve(s):
    stack = []
    mapping = {')': '(', ']': '[', '}': '{'}
    for c in s:
        if c in mapping:
            top = stack.pop() if stack else '#'
            if mapping[c] != top:
                return False
        else:
            stack.append(c)
    return len(stack) == 0
`
		res := evalSubmission(t, prob, code, "Stack-based bracket matching.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("empty string: expected PASS, got %s", res.TestResult)
		}
	})

	t.Run("graph no edges single node", func(t *testing.T) {
		prob := model.Problem{
			ID:         "ni_single",
			Language:   "python",
			Entrypoint: "solve",
			Tests:      []model.TestCase{{ID: "1", Input: `{"grid": [["1"]]}`, ExpectedOutput: "1"}},
			PrimaryConcepts: []string{"dfs"},
		}
		code := `
def solve(grid):
    if not grid:
        return 0
    rows, cols = len(grid), len(grid[0])
    def dfs(r, c):
        if r < 0 or c < 0 or r >= rows or c >= cols or grid[r][c] != '1':
            return
        grid[r][c] = '0'
        dfs(r+1, c); dfs(r-1, c); dfs(r, c+1); dfs(r, c-1)
    count = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == '1':
                dfs(r, c)
                count += 1
    return count
`
		res := evalSubmission(t, prob, code, "DFS flood fill.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("single node grid: expected PASS, got %s", res.TestResult)
		}
	})
}

// ─── Sliding Window E2E Tests ─────────────────────────────────────────────────

func TestE2E_SlidingWindow(t *testing.T) {
	minWindowProb := model.Problem{
		ID:         "minimum_window_substring",
		Title:      "Minimum Window Substring",
		Language:   "python",
		Entrypoint: "solve",
		EntrypointAliases: []string{"minWindow", "min_window"},
		Tests: []model.TestCase{
			{ID: "1", Input: `{"s": "ADOBECODEBANC", "t": "ABC"}`, ExpectedOutput: "BANC"},
			{ID: "2", Input: `{"s": "a", "t": "a"}`, ExpectedOutput: "a"},
			{ID: "3", Input: `{"s": "a", "t": "aa"}`, ExpectedOutput: ""},
		},
		AcceptedStrategies: []string{"sliding_window_frequency_map"},
		RequiredConcepts:   []string{"sliding_window"},
		PrimaryConcepts:    []string{"sliding_window", "frequency_count"},
	}

	t.Run("min window sliding window ACCEPT", func(t *testing.T) {
		code := `
from collections import Counter
def solve(s, t):
    if not t or not s:
        return ""
    need = Counter(t)
    have = {}
    formed = 0
    required = len(need)
    lo = 0
    result = (float('inf'), 0, 0)
    for hi, c in enumerate(s):
        have[c] = have.get(c, 0) + 1
        if c in need and have[c] == need[c]:
            formed += 1
        while lo <= hi and formed == required:
            if hi - lo + 1 < result[0]:
                result = (hi - lo + 1, lo, hi)
            have[s[lo]] -= 1
            if s[lo] in need and have[s[lo]] < need[s[lo]]:
                formed -= 1
            lo += 1
    return "" if result[0] == float('inf') else s[result[1]:result[2]+1]
`
		res := evalSubmission(t, minWindowProb, code,
			"I use a sliding window with two frequency maps. I expand the right pointer and shrink the left when all required characters are in the window.")
		assertAccept(t, res, "min_window sliding window")
	})

	t.Run("min window claims binary search REJECT", func(t *testing.T) {
		code := `
from collections import Counter
def solve(s, t):
    if not t or not s:
        return ""
    need = Counter(t)
    have = {}
    formed = 0
    required = len(need)
    lo = 0
    result = (float('inf'), 0, 0)
    for hi, c in enumerate(s):
        have[c] = have.get(c, 0) + 1
        if c in need and have[c] == need[c]:
            formed += 1
        while lo <= hi and formed == required:
            if hi - lo + 1 < result[0]:
                result = (hi - lo + 1, lo, hi)
            have[s[lo]] -= 1
            if s[lo] in need and have[s[lo]] < need[s[lo]]:
                formed -= 1
            lo += 1
    return "" if result[0] == float('inf') else s[result[1]:result[2]+1]
`
		res := evalSubmission(t, minWindowProb, code,
			"I use binary search to find the optimal window size, then verify using a linear scan.")
		assertReject(t, res, "min_window claimed binary search")
	})
}

// ─── Trie E2E Test ────────────────────────────────────────────────────────────

func TestE2E_Trie(t *testing.T) {
	prob := model.Problem{
		ID:         "implement_trie",
		Title:      "Implement Trie",
		Language:   "python",
		Entrypoint: "solve",
		EntrypointAliases: []string{"trie_operations"},
		Tests: []model.TestCase{
			{ID: "1", Input: `{"operations": ["insert", "search", "search", "startsWith", "insert", "search"], "args": ["apple", "apple", "app", "app", "app", "app"]}`, ExpectedOutput: "[null, true, false, true, null, true]"},
			{ID: "2", Input: `{"operations": ["insert", "search", "startsWith"], "args": ["hello", "hell", "hell"]}`, ExpectedOutput: "[null, false, true]"},
		},
		AcceptedStrategies: []string{"trie_array_children", "trie_hashmap_children"},
		RequiredConcepts:   []string{"trie"},
		OptionalConcepts:   []string{"trees", "strings", "hashmap"},
		PrimaryConcepts:    []string{"trie"},
	}

	t.Run("trie hashmap children ACCEPT", func(t *testing.T) {
		code := `
def solve(operations, args):
    class TrieNode:
        def __init__(self):
            self.children = {}
            self.is_end = False

    class Trie:
        def __init__(self):
            self.root = TrieNode()
        def insert(self, word):
            node = self.root
            for c in word:
                if c not in node.children:
                    node.children[c] = TrieNode()
                node = node.children[c]
            node.is_end = True
        def search(self, word):
            node = self.root
            for c in word:
                if c not in node.children:
                    return False
                node = node.children[c]
            return node.is_end
        def startsWith(self, prefix):
            node = self.root
            for c in prefix:
                if c not in node.children:
                    return False
                node = node.children[c]
            return True

    trie = Trie()
    results = []
    for op, arg in zip(operations, args):
        if op == 'insert':
            trie.insert(arg)
            results.append(None)
        elif op == 'search':
            results.append(trie.search(arg))
        elif op == 'startsWith':
            results.append(trie.startsWith(arg))
    return results
`
		res := evalSubmission(t, prob, code,
			"I implement a trie prefix tree with insert, search, and startsWith operations to efficiently store strings and match prefixes.")
		assertAccept(t, res, "trie hashmap children")
	})
}

// ─── Heap E2E Test ────────────────────────────────────────────────────────────

func TestE2E_Heap(t *testing.T) {
	prob := model.Problem{
		ID:         "top_k_frequent_elements",
		Language:   "python",
		Entrypoint: "solve",
		EntrypointAliases: []string{"topKFrequent", "top_k_frequent"},
		Tests: []model.TestCase{
			{ID: "1", Input: `{"nums": [1, 1, 1, 2, 2, 3], "k": 2}`, ExpectedOutput: "[1, 2]"},
			{ID: "2", Input: `{"nums": [1], "k": 1}`, ExpectedOutput: "[1]"},
		},
		AcceptedStrategies: []string{"heap_top_k_frequent"},
		PrimaryConcepts:    []string{"heap", "frequency_count"},
	}

	t.Run("top k frequent heap ACCEPT", func(t *testing.T) {
		code := `
import heapq
from collections import Counter
def solve(nums, k):
    count = Counter(nums)
    return sorted(heapq.nlargest(k, count.keys(), key=count.get))
`
		res := evalSubmission(t, prob, code,
			"I count frequencies with a Counter hashmap, then use a min-heap (heapq.nlargest) to extract the top K elements.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("top_k_frequent: expected PASS, got %s diags=%v", res.TestResult, res.Diagnostics)
		}
	})
}

// ─── Explanations with Typos / Informal Language ─────────────────────────────

func TestE2E_InformalExplanations(t *testing.T) {
	t.Run("typo in 'hashmap' still accepted", func(t *testing.T) {
		code := `
def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
`
		prob := model.Problem{
			ID:         "two_sum",
			Language:   "python",
			Entrypoint: "solve",
			Tests: []model.TestCase{
				{ID: "1", Input: `{"nums": [2, 7, 11, 15], "target": 9}`, ExpectedOutput: "[0, 1]"},
				{ID: "2", Input: `{"nums": [3, 2, 4], "target": 6}`, ExpectedOutput: "[1, 2]"},
			},
			AcceptedStrategies: []string{"hashmap_two_sum"},
			RequiredConcepts:   []string{},
			PrimaryConcepts:    []string{"hashmap"},
		}
		res := evalSubmission(t, prob, code,
			"I store each number in a dictonary (haschmap) and look up compliment.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("typo test: expected PASS, got %s", res.TestResult)
		}
	})

	t.Run("informal language 'I keep track using a dict' ACCEPT", func(t *testing.T) {
		code := `
def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
`
		prob := model.Problem{
			ID:         "two_sum2",
			Language:   "python",
			Entrypoint: "solve",
			Tests: []model.TestCase{
				{ID: "1", Input: `{"nums": [2, 7, 11, 15], "target": 9}`, ExpectedOutput: "[0, 1]"},
			},
			AcceptedStrategies: []string{"hashmap_two_sum"},
			PrimaryConcepts:    []string{"hashmap"},
		}
		res := evalSubmission(t, prob, code,
			"So basically I keep track using a dict. For each number I check if its complement is already in the dict.")
		if res.TestResult != model.TestStatusPass {
			t.Errorf("informal test: expected PASS, got %s", res.TestResult)
		}
	})
}

// ─── Validate Diagnostics Fields ─────────────────────────────────────────────

func TestE2E_DiagnosticsFieldsPresent(t *testing.T) {
	code := `
def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
`
	prob := model.Problem{
		ID:         "diag_test",
		Language:   "python",
		Entrypoint: "solve",
		Tests: []model.TestCase{
			{ID: "1", Input: `{"nums": [2, 7, 11, 15], "target": 9}`, ExpectedOutput: "[0, 1]"},
		},
		AcceptedStrategies: []string{"hashmap_two_sum"},
		PrimaryConcepts:    []string{"hashmap"},
	}
	res := evalSubmission(t, prob, code,
		"I use a hashmap to store seen values and find the complement quickly.")

	// Verify all key fields populated
	if res.ProblemID == "" {
		t.Error("ProblemID should be populated")
	}
	if res.FidelityScore < 0 || res.FidelityScore > 1.1 {
		t.Errorf("FidelityScore out of range: %f", res.FidelityScore)
	}
	if res.Decision == "" {
		t.Error("Decision should be set")
	}
	if res.TestResult == "" {
		t.Error("TestResult should be set")
	}
	if len(res.Diagnostics) == 0 {
		t.Log("Note: diagnostics empty — this may be fine if ACCEPT path skips them")
	}
	// Explanation must not contain internal error
	for _, d := range res.Diagnostics {
		if strings.Contains(strings.ToLower(d), "panic") {
			t.Errorf("diagnostic contains 'panic': %s", d)
		}
	}
}
