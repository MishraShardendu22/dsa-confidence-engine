package evaluator_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
)

type ScaleTestCase struct {
	ProblemID        string
	Category         string // OPTIMAL, ALTERNATIVE, BUGGY, BLUFFING, DEAD_CODE, SYNTAX_ERROR
	Code             string
	Explanation      string
	ExpectedDecision model.Decision
	GroundTruthValid bool
}

func generateCodeAndExplanation(concept string, entrypoint string, expectedOutput string) (string, string) {
	retVal := strings.TrimSpace(expectedOutput)
	switch strings.ToLower(retVal) {
	case "true":
		retVal = "True"
	case "false":
		retVal = "False"
	case "null", "":
		retVal = "None"
	}

	c := strings.ToLower(concept)
	var body, expl string

	switch {
	case strings.Contains(c, "hashmap") || strings.Contains(c, "hash_table") || strings.Contains(c, "hashing"):
		body = fmt.Sprintf(`
    seen = {}
    for i in range(3):
        seen[i] = i
    if 0 in seen:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a hash map dictionary to store seen values and perform constant-time O(1) lookups."

	case strings.Contains(c, "binary_search"):
		body = fmt.Sprintf(`
    low, high = 0, 10
    while low <= high:
        mid = (low + high) // 2
        if mid >= 0:
            return %s
        low = mid + 1
    return %s
`, retVal, retVal)
		expl = "I use iterative binary search with a while loop and midpoint bisection to find the target in O(log N) time."

	case strings.Contains(c, "two_pointers") || strings.Contains(c, "sliding_window"):
		body = fmt.Sprintf(`
    left = 0
    right = 5
    while left < right:
        left += 1
        right -= 1
    if left >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a two pointers approach scanning inward from opposite ends to solve the problem in linear time."

	case strings.Contains(c, "dynamic_programming") || strings.Contains(c, "tabulation") || strings.Contains(c, "kadane"):
		body = fmt.Sprintf(`
    dp = [0] * 5
    for i in range(1, 5):
        dp[i] = max(dp[i-1] + 1, dp[i])
    if dp[-1] >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use dynamic programming with a bottom-up tabulation table to iteratively compute state transitions."

	case strings.Contains(c, "memoization") || strings.Contains(c, "recursion"):
		body = fmt.Sprintf(`
    memo = {}
    def helper(n):
        if n in memo:
            return memo[n]
        if n <= 1:
            return 1
        memo[n] = helper(n - 1)
        return memo[n]
    res = helper(3)
    if res > 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use recursion with memoization caching intermediate results in a dictionary to prevent redundant work."

	case strings.Contains(c, "backtracking"):
		body = fmt.Sprintf(`
    res = []
    def backtrack(start, path):
        res.append(list(path))
        for i in range(start, 3):
            path.append(i)
            backtrack(i + 1, path)
            path.pop()
    backtrack(0, [])
    if len(res) > 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use recursive backtracking with state restoration by appending and popping elements from the path."

	case strings.Contains(c, "dfs") || strings.Contains(c, "graph") || strings.Contains(c, "tree"):
		body = fmt.Sprintf(`
    visited = set()
    def dfs(node):
        visited.add(node)
        for nei in [node + 1]:
            if nei not in visited and nei < 3:
                dfs(nei)
    dfs(0)
    if len(visited) > 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use depth-first search (DFS) with a recursive helper function and visited set to traverse all reachable nodes."

	case strings.Contains(c, "bfs") || strings.Contains(c, "queue"):
		body = fmt.Sprintf(`
    from collections import deque
    q = deque([0])
    visited = {0}
    while q:
        curr = q.popleft()
        if curr == 0:
            return %s
    return %s
`, retVal, retVal)
		expl = "I use breadth-first search (BFS) with a FIFO deque queue to explore the state space level by level."

	case strings.Contains(c, "stack") || strings.Contains(c, "monotonic_stack"):
		body = fmt.Sprintf(`
    stack = []
    for x in [1, 2, 3]:
        while stack and stack[-1] > x:
            stack.pop()
        stack.append(x)
    if stack:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a monotonic stack to track elements and resolve nearest boundary relationships in linear time."

	case strings.Contains(c, "heap") || strings.Contains(c, "priority_queue"):
		body = fmt.Sprintf(`
    import heapq
    h = []
    for x in [3, 1, 2]:
        heapq.heappush(h, x)
    val = heapq.heappop(h)
    if val >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a min-heap priority queue with heapq to maintain the smallest elements in logarithmic time."

	case strings.Contains(c, "bit_manipulation") || strings.Contains(c, "bitmask"):
		body = fmt.Sprintf(`
    ans = 0
    for x in [1, 2, 3]:
        ans ^= x
        ans = (ans << 1) & 0xFF
    if ans >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use bit manipulation with XOR cancellation and bit shifts to compute the answer in constant space."

	case strings.Contains(c, "trie"):
		body = fmt.Sprintf(`
    trie = {}
    curr = trie
    for ch in "abc":
        if ch not in curr:
            curr[ch] = {}
        curr = curr[ch]
    if trie:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a prefix tree trie with nested dictionaries for fast prefix lookup and character matching."

	case strings.Contains(c, "prefix_sum") || strings.Contains(c, "difference_array"):
		body = fmt.Sprintf(`
    prefix = [0]
    for x in [1, 2, 3]:
        prefix.append(prefix[-1] + x)
    if prefix[-1] >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a prefix sum running array to compute cumulative range sums in constant time."

	case strings.Contains(c, "greedy"):
		body = fmt.Sprintf(`
    farthest = 0
    for i, x in enumerate([1, 2, 3]):
        farthest = max(farthest, i + x)
    if farthest >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I use a greedy algorithm making local optimal choices at each step to reach the global optimum."

	case strings.Contains(c, "sorting"):
		body = fmt.Sprintf(`
    nums = [3, 1, 2]
    nums.sort()
    if nums[0] >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I sort the elements in ascending order and process them sequentially."

	default:
		body = fmt.Sprintf(`
    arr = [1, 2, 3]
    total = sum(arr)
    if total >= 0:
        return %s
    return %s
`, retVal, retVal)
		expl = "I iterate through the array elements to compute the result."
	}

	code := fmt.Sprintf("def %s(*args, **kwargs):\n%s\n", entrypoint, body)
	return code, expl
}

func TestCatalogScaleEvaluator(t *testing.T) {
	// 1. Setup Engine
	dsaRepoPath := filepath.Join("..", "..", "data", "dsa")
	dsaRepo, err := dsa.NewRepository(dsaRepoPath)
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
	analyzer := analysis.NewPythonAnalyzer(scriptPath)
	scorer := fidelity.NewScorer(dsaRepo.Ontology(), fidelity.DefaultThresholds())

	eval := evaluator.NewEvaluator(r, analyzer, matcher, scorer, nil)

	// 2. Load dataset_all.json
	datasetPath := filepath.Join("..", "..", "data", "problems", "dataset_all.json")
	data, err := os.ReadFile(datasetPath)
	if err != nil {
		t.Fatalf("failed to read dataset_all.json: %v", err)
	}

	var allProblems []model.Problem
	if err := json.Unmarshal(data, &allProblems); err != nil {
		t.Fatalf("failed to unmarshal dataset: %v", err)
	}

	t.Logf("Total problems available in catalog: %d", len(allProblems))

	var testCases []ScaleTestCase

	if testing.Short() {
		// Sample 200 problems with 5 variants = 1,000 evaluations
		sampleSize := 200
		startIdx := 55
		if startIdx+sampleSize > len(allProblems) {
			sampleSize = len(allProblems) - startIdx
		}
		sampleProblems := allProblems[startIdx : startIdx+sampleSize]

		for _, p := range sampleProblems {
			entrypoint := p.Entrypoint
			if entrypoint == "" {
				entrypoint = "solve"
			}
			expectedOut := "0"
			if len(p.Tests) > 0 {
				expectedOut = p.Tests[0].ExpectedOutput
			}
			primaryConcept := "arrays"
			if len(p.PrimaryConcepts) > 0 {
				primaryConcept = p.PrimaryConcepts[0]
			}
			secondaryConcept := "sorting"
			if len(p.AcceptedStrategies) > 1 {
				secondaryConcept = p.AcceptedStrategies[1]
			}

			optCode, optExpl := generateCodeAndExplanation(primaryConcept, entrypoint, expectedOut)
			testCases = append(testCases, ScaleTestCase{
				ProblemID:        p.ID,
				Category:         "OPTIMAL",
				Code:             optCode,
				Explanation:      optExpl,
				ExpectedDecision: model.DecisionAccept,
				GroundTruthValid: true,
			})

			altCode, altExpl := generateCodeAndExplanation(secondaryConcept, entrypoint, expectedOut)
			testCases = append(testCases, ScaleTestCase{
				ProblemID:        p.ID,
				Category:         "ALTERNATIVE",
				Code:             altCode,
				Explanation:      altExpl,
				ExpectedDecision: model.DecisionAccept,
				GroundTruthValid: true,
			})

			bugCode := fmt.Sprintf("def %s(*args, **kwargs):\n    return '__WRONG_RESULT_9999__'\n", entrypoint)
			testCases = append(testCases, ScaleTestCase{
				ProblemID:        p.ID,
				Category:         "BUGGY",
				Code:             bugCode,
				Explanation:      optExpl,
				ExpectedDecision: model.DecisionReject,
				GroundTruthValid: false,
			})

			bluffExpl := "I implemented a 2D dynamic programming knapsack table with memoization, bitwise XOR trie operations, and topological sort."
			testCases = append(testCases, ScaleTestCase{
				ProblemID:        p.ID,
				Category:         "BLUFFING",
				Code:             optCode,
				Explanation:      bluffExpl,
				ExpectedDecision: model.DecisionReject,
				GroundTruthValid: false,
			})

			deadCode := fmt.Sprintf(`class UnusedDeadTrie:
    def __init__(self):
        self.root = {}
    def insert(self, word):
        self.root[word] = True

def %s(*args, **kwargs):
    return %s
`, entrypoint, expectedOut)
			deadExpl := "I implemented a prefix trie data structure to search and insert words efficiently."
			testCases = append(testCases, ScaleTestCase{
				ProblemID:        p.ID,
				Category:         "DEAD_CODE",
				Code:             deadCode,
				Explanation:      deadExpl,
				ExpectedDecision: model.DecisionReject,
				GroundTruthValid: false,
			})
		}
	} else {
		// Full scale: evaluate EVERY SINGLE problem in the 4,052 catalog
		for i, p := range allProblems {
			entrypoint := p.Entrypoint
			if entrypoint == "" {
				entrypoint = "solve"
			}
			expectedOut := "0"
			if len(p.Tests) > 0 {
				expectedOut = p.Tests[0].ExpectedOutput
			}
			primaryConcept := "arrays"
			if len(p.PrimaryConcepts) > 0 {
				primaryConcept = p.PrimaryConcepts[0]
			}
			secondaryConcept := "sorting"
			if len(p.AcceptedStrategies) > 1 {
				secondaryConcept = p.AcceptedStrategies[1]
			}

			mode := i % 5
			switch mode {
			case 0:
				// Optimal
				code, expl := generateCodeAndExplanation(primaryConcept, entrypoint, expectedOut)
				testCases = append(testCases, ScaleTestCase{
					ProblemID:        p.ID,
					Category:         "OPTIMAL",
					Code:             code,
					Explanation:      expl,
					ExpectedDecision: model.DecisionAccept,
					GroundTruthValid: true,
				})
			case 1:
				// Alternative
				code, expl := generateCodeAndExplanation(secondaryConcept, entrypoint, expectedOut)
				testCases = append(testCases, ScaleTestCase{
					ProblemID:        p.ID,
					Category:         "ALTERNATIVE",
					Code:             code,
					Explanation:      expl,
					ExpectedDecision: model.DecisionAccept,
					GroundTruthValid: true,
				})
			case 2:
				// Buggy
				bugCode := fmt.Sprintf("def %s(*args, **kwargs):\n    return '__WRONG_RESULT_9999__'\n", entrypoint)
				_, expl := generateCodeAndExplanation(primaryConcept, entrypoint, expectedOut)
				testCases = append(testCases, ScaleTestCase{
					ProblemID:        p.ID,
					Category:         "BUGGY",
					Code:             bugCode,
					Explanation:      expl,
					ExpectedDecision: model.DecisionReject,
					GroundTruthValid: false,
				})
			case 3:
				// Bluffing
				code, _ := generateCodeAndExplanation(primaryConcept, entrypoint, expectedOut)
				bluffExpl := "I implemented a 2D dynamic programming knapsack table with memoization, bitwise XOR trie operations, and topological sort."
				testCases = append(testCases, ScaleTestCase{
					ProblemID:        p.ID,
					Category:         "BLUFFING",
					Code:             code,
					Explanation:      bluffExpl,
					ExpectedDecision: model.DecisionReject,
					GroundTruthValid: false,
				})
			case 4:
				// Dead Code
				deadCode := fmt.Sprintf(`class UnusedDeadTrie:
    def __init__(self):
        self.root = {}
    def insert(self, word):
        self.root[word] = True

def %s(*args, **kwargs):
    return %s
`, entrypoint, expectedOut)
				deadExpl := "I implemented a prefix trie data structure to search and insert words efficiently."
				testCases = append(testCases, ScaleTestCase{
					ProblemID:        p.ID,
					Category:         "DEAD_CODE",
					Code:             deadCode,
					Explanation:      deadExpl,
					ExpectedDecision: model.DecisionReject,
					GroundTruthValid: false,
				})
			}
		}
	}

	t.Logf("Generated %d scale test cases.", len(testCases))

	// Map problems for fast lookup
	probMap := make(map[string]model.Problem, len(allProblems))
	for _, p := range allProblems {
		probMap[p.ID] = p
	}

	// Concurrency worker pool
	numWorkers := 32
	jobs := make(chan ScaleTestCase, len(testCases))
	for _, tc := range testCases {
		jobs <- tc
	}
	close(jobs)

	var tp, tn, fp, fn int64
	var totalDurationMs int64

	type failureRecord struct {
		ProblemID   string
		Category    string
		Expected    model.Decision
		Actual      model.Decision
		Score       float64
		Diagnostics []string
		Explanation string
	}

	var failuresMu sync.Mutex
	var failures []failureRecord

	start := time.Now()
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tc := range jobs {
				prob, ok := probMap[tc.ProblemID]
				if !ok {
					continue
				}

				sub := model.Submission{
					ProblemID:   tc.ProblemID,
					SourceCode:  tc.Code,
					Explanation: tc.Explanation,
				}

				evalStart := time.Now()
				res, err := eval.Evaluate(ctx, sub, prob)
				dur := time.Since(evalStart).Milliseconds()
				atomic.AddInt64(&totalDurationMs, dur)

				if err != nil {
					failuresMu.Lock()
					failures = append(failures, failureRecord{
						ProblemID:   tc.ProblemID,
						Category:    tc.Category,
						Expected:    tc.ExpectedDecision,
						Actual:      "ERROR",
						Diagnostics: []string{err.Error()},
					})
					failuresMu.Unlock()
					continue
				}

				isAccept := res.Decision == model.DecisionAccept
				expectedAccept := tc.ExpectedDecision == model.DecisionAccept

				if isAccept && expectedAccept {
					atomic.AddInt64(&tp, 1)
				} else if !isAccept && !expectedAccept {
					atomic.AddInt64(&tn, 1)
				} else if isAccept && !expectedAccept {
					atomic.AddInt64(&fp, 1)
					failuresMu.Lock()
					failures = append(failures, failureRecord{
						ProblemID:   tc.ProblemID,
						Category:    tc.Category,
						Expected:    tc.ExpectedDecision,
						Actual:      res.Decision,
						Score:       res.FidelityScore,
						Diagnostics: res.Diagnostics,
						Explanation: tc.Explanation,
					})
					failuresMu.Unlock()
				} else if !isAccept && expectedAccept {
					atomic.AddInt64(&fn, 1)
					failuresMu.Lock()
					failures = append(failures, failureRecord{
						ProblemID:   tc.ProblemID,
						Category:    tc.Category,
						Expected:    tc.ExpectedDecision,
						Actual:      res.Decision,
						Score:       res.FidelityScore,
						Diagnostics: res.Diagnostics,
						Explanation: tc.Explanation,
					})
					failuresMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	totalElapsed := time.Since(start)

	totalTests := tp + tn + fp + fn
	accuracy := float64(tp+tn) / float64(totalTests) * 100
	var precision, recall, specificity, f1 float64
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp) * 100
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn) * 100
	}
	if tn+fp > 0 {
		specificity = float64(tn) / float64(tn+fp) * 100
	}
	if precision+recall > 0 {
		f1 = 2 * (precision * recall) / (precision + recall) / 100
	}

	avgLatency := float64(totalDurationMs) / float64(totalTests)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("       MASSIVE SCALE EVALUATION REPORT (CONCURRENT GOROUTINE POOL)      ")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Evaluated Submissions : %d\n", totalTests)
	fmt.Printf("Concurrency (Workers) : %d\n", numWorkers)
	fmt.Printf("Total Elapsed Time    : %v (%.1f evals/sec)\n", totalElapsed, float64(totalTests)/totalElapsed.Seconds())
	fmt.Printf("Average Latency/Eval  : %.2fms\n", avgLatency)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("CONFUSION MATRIX:")
	fmt.Printf("  True Positives  (TP): %5d  [Valid code + matching explanation -> ACCEPT]\n", tp)
	fmt.Printf("  True Negatives  (TN): %5d  [Buggy / Bluffing / Dead Code      -> REJECT]\n", tn)
	fmt.Printf("  False Positives (FP): %5d  [CRITICAL: Cheater / Broken Code   -> ACCEPT]\n", fp)
	fmt.Printf("  False Negatives (FN): %5d  [Valid submission wrongly rejected -> REJECT]\n", fn)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("STATISTICAL PERFORMANCE:")
	fmt.Printf("  Accuracy            : %6.2f%%\n", accuracy)
	fmt.Printf("  Precision           : %6.2f%%\n", precision)
	fmt.Printf("  Recall / Sensitivity: %6.2f%%\n", recall)
	fmt.Printf("  Specificity         : %6.2f%%\n", specificity)
	fmt.Printf("  F1-Score            : %6.4f\n", f1)
	fmt.Println(strings.Repeat("-", 80))

	if len(failures) > 0 {
		fmt.Printf("LOGGED FAILURES (%d):\n", len(failures))
		for i, f := range failures {
			if i >= 15 {
				fmt.Printf("  ... and %d more failures\n", len(failures)-15)
				break
			}
			fmt.Printf("  [%s] Problem: %s | Expected: %s, Got: %s (Score: %.3f)\n", f.Category, f.ProblemID, f.Expected, f.Actual, f.Score)
			if len(f.Diagnostics) > 0 {
				fmt.Printf("     Diagnostics: %s\n", strings.Join(f.Diagnostics, "; "))
			}
		}
		fmt.Println(strings.Repeat("=", 80))
	} else {
		fmt.Println("ZERO FAILURES DETECTED ACROSS SCALE SUBMISSIONS")
		fmt.Println(strings.Repeat("=", 80))
	}
}
