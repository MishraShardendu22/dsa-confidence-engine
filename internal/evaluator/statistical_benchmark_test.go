package evaluator_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

type BenchmarkCase struct {
	Name             string
	Domain           string
	ProblemID        string
	Code             string
	Explanation      string
	ExpectedDecision model.Decision // ACCEPT or REJECT
	GroundTruthValid bool           // True if code is correct AND explanation honestly matches
	Subtype          string         // OPTIMAL, ALTERNATIVE, BUGGY, BLUFFING, DEAD_CODE
}

func TestStatisticalSystemEvaluation(t *testing.T) {
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

	// 2. Load catalog
	datasetPath := filepath.Join("..", "..", "data", "problems", "dataset_all.json")
	data, err := os.ReadFile(datasetPath)
	if err != nil {
		t.Fatalf("failed to read dataset_all.json: %v", err)
	}
	var allProblems []model.Problem
	if err := json.Unmarshal(data, &allProblems); err != nil {
		t.Fatalf("failed to unmarshal dataset: %v", err)
	}
	probMap := make(map[string]model.Problem, len(allProblems))
	for _, p := range allProblems {
		probMap[p.ID] = p
	}

	// 3. Assemble statistical test cases from benchmark suite and canonical cases
	var testCases []BenchmarkCase

	// Load from multi_approach_suite.json
	suitePath := filepath.Join("..", "..", "data", "benchmarks", "multi_approach_suite.json")
	if sData, err := os.ReadFile(suitePath); err == nil {
		var suites []BenchmarkSuiteItem
		if err := json.Unmarshal(sData, &suites); err == nil {
			for _, suite := range suites {
				for _, app := range suite.Approaches {
					expDec := model.DecisionAccept
					if app.ExpectedDecision == "REJECT" {
						expDec = model.DecisionReject
					}
					testCases = append(testCases, BenchmarkCase{
						Name:             fmt.Sprintf("[%s] %s - %s", suite.Category, suite.Name, app.Name),
						Domain:           suite.Category,
						ProblemID:        suite.ProblemID,
						Code:             app.Code,
						Explanation:      app.Explanation,
						ExpectedDecision: expDec,
						GroundTruthValid: (app.Type == "OPTIMAL" || app.Type == "ALTERNATIVE" && app.ExpectedDecision == "ACCEPT"),
						Subtype:          app.Type,
					})
				}
			}
		}
	}

	// Add Piyush Benchmark test cases
	testCases = append(testCases, []BenchmarkCase{
		{
			Name:             "[hashing] Two Sum - Hashmap Complement",
			Domain:           "hashing",
			ProblemID:        "two_sum",
			Code:             "def solve(nums, target):\n    seen = {}\n    for i, x in enumerate(nums):\n        diff = target - x\n        if diff in seen:\n            return [seen[diff], i]\n        seen[x] = i\n    return []\n",
			Explanation:      "I will not use brute force. I use a hash map to store each visited number and its index.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
		{
			Name:             "[hashing] Frequency Count - dict.get",
			Domain:           "hashing",
			ProblemID:        "frequency_count",
			Code:             "def solve(s):\n    freq = {}\n    for x in s:\n        freq[x] = freq.get(x, 0) + 1\n    return freq\n",
			Explanation:      "We iterate through the string and populate a frequency map using a hash table.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
		{
			Name:             "[arrays] Second Maximum - Single Pass",
			Domain:           "arrays",
			ProblemID:        "second_maximum",
			Code:             "def solve(nums):\n    first = second = None\n    for n in nums:\n        if first is None or n > first:\n            second, first = first, n\n        elif n != first and (second is None or n > second):\n            second = n\n    return second\n",
			Explanation:      "I iterate through the array elements in a single pass tracking the top two maximum values with two variables.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
		{
			Name:             "[hashing] First Non-Repeating Char - Counter",
			Domain:           "hashing",
			ProblemID:        "first_non_repeating_character",
			Code:             "from collections import Counter\ndef solve(s):\n    c = Counter(s)\n    for ch in s:\n        if c[ch] == 1:\n            return ch\n    return None\n",
			Explanation:      "We use collections.Counter to count occurrences and return the first unique character.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
		{
			Name:             "[adversarial] Two Sum Code with Binary Search Bluff",
			Domain:           "hashing",
			ProblemID:        "two_sum",
			Code:             "def solve(nums, target):\n    seen = {}\n    for i, x in enumerate(nums):\n        diff = target - x\n        if diff in seen:\n            return [seen[diff], i]\n        seen[x] = i\n    return []\n",
			Explanation:      "I will use binary search to locate the complement of each element.",
			ExpectedDecision: model.DecisionReject,
			GroundTruthValid: false,
			Subtype:          "BLUFFING",
		},
		{
			Name:             "[graphs] Number of Islands - DFS",
			Domain:           "graphs",
			ProblemID:        "clone_graph",
			Code:             "def solve(grid):\n    return []\n",
			Explanation:      "We traverse graph vertices and clone neighbors.",
			ExpectedDecision: model.DecisionReject, // synthetic test fails
			GroundTruthValid: false,
			Subtype:          "BUGGY",
		},
		{
			Name:             "[two_pointers] 3Sum - Two Pointers Sorted",
			Domain:           "two_pointers",
			ProblemID:        "three_sum",
			Code:             "def solve(nums):\n    nums.sort()\n    res = []\n    for i in range(len(nums) - 2):\n        if i > 0 and nums[i] == nums[i-1]:\n            continue\n        l, r = i + 1, len(nums) - 1\n        while l < r:\n            s = nums[i] + nums[l] + nums[r]\n            if s == 0:\n                res.append([nums[i], nums[l], nums[r]])\n                while l < r and nums[l] == nums[l+1]: l += 1\n                while l < r and nums[r] == nums[r-1]: r -= 1\n                l += 1; r -= 1\n            elif s < 0:\n                l += 1\n            else:\n                r -= 1\n    return res\n",
			Explanation:      "We sort the array and use two pointers (left and right) to find matching triplets summing to zero.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
		{
			Name:             "[two_pointers] 3Sum - Bluffing DP Claim",
			Domain:           "two_pointers",
			ProblemID:        "three_sum",
			Code:             "def solve(nums):\n    nums.sort()\n    res = []\n    for i in range(len(nums) - 2):\n        if i > 0 and nums[i] == nums[i-1]:\n            continue\n        l, r = i + 1, len(nums) - 1\n        while l < r:\n            s = nums[i] + nums[l] + nums[r]\n            if s == 0:\n                res.append([nums[i], nums[l], nums[r]])\n                while l < r and nums[l] == nums[l+1]: l += 1\n                while l < r and nums[r] == nums[r-1]: r -= 1\n                l += 1; r -= 1\n            elif s < 0:\n                l += 1\n            else:\n                r -= 1\n    return res\n",
			Explanation:      "We formulate a 2D dynamic programming grid with state transitions and memoization.",
			ExpectedDecision: model.DecisionReject,
			GroundTruthValid: false,
			Subtype:          "BLUFFING",
		},
		{
			Name:             "[trie] Implement Trie - Prefix Tree",
			Domain:           "trie",
			ProblemID:        "implement_trie",
			Code:             "class TrieNode:\n    def __init__(self):\n        self.children = {}\n        self.is_end = False\nclass Trie:\n    def __init__(self):\n        self.root = TrieNode()\n    def insert(self, word):\n        curr = self.root\n        for c in word:\n            if c not in curr.children:\n                curr.children[c] = TrieNode()\n            curr = curr.children[c]\n        curr.is_end = True\n    def search(self, word):\n        curr = self.root\n        for c in word:\n            if c not in curr.children:\n                return False\n            curr = curr.children[c]\n        return curr.is_end\n    def startsWith(self, prefix):\n        curr = self.root\n        for c in prefix:\n            if c not in curr.children:\n                return False\n            curr = curr.children[c]\n        return True\ndef solve(operations, args):\n    trie = Trie()\n    res = []\n    for cmd, arg in zip(operations, args):\n        if cmd == 'insert':\n            trie.insert(arg)\n            res.append(None)\n        elif cmd == 'search':\n            res.append(trie.search(arg))\n        elif cmd == 'startsWith':\n            res.append(trie.startsWith(arg))\n    return res\n",
			Explanation:      "We implement a prefix tree (Trie) using tree nodes with children dictionaries and an is_end terminal flag.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
		{
			Name:             "[backtracking] Subsets - Backtracking DFS",
			Domain:           "backtracking",
			ProblemID:        "subsets",
			Code:             "def solve(nums):\n    res = []\n    def backtrack(start, path):\n        res.append(list(path))\n        for i in range(start, len(nums)):\n            path.append(nums[i])\n            backtrack(i + 1, path)\n            path.pop()\n    backtrack(0, [])\n    return res\n",
			Explanation:      "We generate all subsets using backtracking recursion, appending each path to the result and exploring choices.",
			ExpectedDecision: model.DecisionAccept,
			GroundTruthValid: true,
			Subtype:          "OPTIMAL",
		},
	}...)

	// Run all benchmarks and collect statistical measurements
	var (
		tp, fp, tn, fn int
		latencies      []time.Duration
		scoresValid    []float64
		scoresBluffing []float64
		scoresDeadCode []float64
		domainStats    = make(map[string]struct{ total, correct int })
		subtypeStats   = make(map[string]struct{ total, correct int })
	)

	for _, tc := range testCases {
		prob, exists := probMap[tc.ProblemID]
		if !exists {
			t.Fatalf("problem %s not found in catalog", tc.ProblemID)
		}

		sub := model.Submission{
			ProblemID:   tc.ProblemID,
			SourceCode:  tc.Code,
			Explanation: tc.Explanation,
		}

		start := time.Now()
		res, err := eval.Evaluate(ctx, sub, prob)
		dur := time.Since(start)
		latencies = append(latencies, dur)

		if err != nil {
			t.Fatalf("error evaluating %s: %v", tc.Name, err)
		}

		isAccepted := (res.Decision == model.DecisionAccept)
		decisionCorrect := (res.Decision == tc.ExpectedDecision)

		// Confusion Matrix Accounting
		if tc.GroundTruthValid {
			scoresValid = append(scoresValid, res.FidelityScore)
			if isAccepted {
				tp++ // True Positive
			} else {
				fn++ // False Negative (Valid candidate wrongly rejected)
				fmt.Printf("FALSE NEGATIVE: %s, Score: %.3f, Decision: %s, Diags: %v\n", tc.Name, res.FidelityScore, res.Decision, res.Diagnostics)
			}
		} else {
			if tc.Subtype == "BLUFFING" {
				scoresBluffing = append(scoresBluffing, res.FidelityScore)
			} else if tc.Subtype == "DEAD_CODE" {
				scoresDeadCode = append(scoresDeadCode, res.FidelityScore)
			}

			if isAccepted {
				fp++ // False Positive (Cheating/broken code wrongly accepted!)
			} else {
				tn++ // True Negative
			}
		}

		// Domain Stats
		d := domainStats[tc.Domain]
		d.total++
		if decisionCorrect {
			d.correct++
		}
		domainStats[tc.Domain] = d

		// Subtype Stats
		s := subtypeStats[tc.Subtype]
		s.total++
		if decisionCorrect {
			s.correct++
		}
		subtypeStats[tc.Subtype] = s
	}

	// 4. Compute Statistical Metrics
	total := len(testCases)
	accuracy := float64(tp+tn) / float64(total)
	precision := 0.0
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	recall := 0.0
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	specificity := 0.0
	if tn+fp > 0 {
		specificity = float64(tn) / float64(tn+fp)
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * (precision * recall) / (precision + recall)
	}

	// Latency percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	meanLat := time.Duration(0)
	for _, l := range latencies {
		meanLat += l
	}
	meanLat /= time.Duration(total)
	p50 := latencies[int(float64(total)*0.50)]
	p90 := latencies[int(float64(total)*0.90)]
	p95 := latencies[int(float64(total)*0.95)]
	maxLat := latencies[total-1]

	avgScore := func(scores []float64) float64 {
		if len(scores) == 0 {
			return 0
		}
		sum := 0.0
		for _, s := range scores {
			sum += s
		}
		return math.Round((sum/float64(len(scores)))*1000) / 1000
	}

	// 5. Print Detailed Statistical Report
	fmt.Printf("\n================================================================================\n")
	fmt.Printf("          DSA CONFIDENCE ENGINE -- STATISTICAL EVALUATION REPORT                \n")
	fmt.Printf("================================================================================\n")
	fmt.Printf("Evaluated Submissions : %d\n", total)
	fmt.Printf("Execution Mode        : Local AST + Python Sandbox + NLP Cascade Matcher + Scorer\n")
	fmt.Printf("Problem Dataset Size  : %d indexed algorithmic questions\n", len(probMap))
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("CONFUSION MATRIX:\n")
	fmt.Printf("  True Positives  (TP): %2d  [Valid code + matching explanation -> ACCEPT]\n", tp)
	fmt.Printf("  True Negatives  (TN): %2d  [Buggy / Bluffing / Dead Code      -> REJECT]\n", tn)
	fmt.Printf("  False Positives (FP): %2d  [CRITICAL: Cheater / Broken Code   -> ACCEPT]\n", fp)
	fmt.Printf("  False Negatives (FN): %2d  [Valid submission wrongly rejected -> REJECT]\n", fn)
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("STATISTICAL PERFORMANCE METRICS:\n")
	fmt.Printf("  Accuracy            : %6.2f%% (%d/%d correctly classified)\n", accuracy*100, tp+tn, total)
	fmt.Printf("  Precision           : %6.2f%% (Safety rate: accepted answers are truly valid)\n", precision*100)
	fmt.Printf("  Recall / Sensitivity: %6.2f%% (Fairness rate: valid candidates given credit)\n", recall*100)
	fmt.Printf("  Specificity         : %6.2f%% (Interception rate: bad solutions rejected)\n", specificity*100)
	fmt.Printf("  F1-Score            : %6.4f\n", f1)
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("SECURITY & ADVERSARIAL DEFENSE METRICS:\n")
	bluffingStat := subtypeStats["BLUFFING"]
	deadCodeStat := subtypeStats["DEAD_CODE"]
	buggyStat := subtypeStats["FAILING"]
	if buggyStat.total == 0 {
		buggyStat = subtypeStats["BUGGY"]
	}
	fmt.Printf("  Bluffing Interception Rate   : %6.2f%% (%d/%d caught, Avg Score: %.3f)\n",
		float64(bluffingStat.correct)/float64(bluffingStat.total)*100, bluffingStat.correct, bluffingStat.total, avgScore(scoresBluffing))
	fmt.Printf("  Dead Code Stuffing Penalized : %6.2f%% (%d/%d caught, Avg Score: %.3f)\n",
		float64(deadCodeStat.correct)/float64(deadCodeStat.total)*100, deadCodeStat.correct, deadCodeStat.total, avgScore(scoresDeadCode))
	fmt.Printf("  Buggy Code Instant Reject    : %6.2f%% (%d/%d intercepted at Test Phase)\n",
		float64(buggyStat.correct)/float64(buggyStat.total)*100, buggyStat.correct, buggyStat.total)
	fmt.Printf("  Valid Submissions Fidelity   : Avg Score: %.3f across %d submissions\n",
		avgScore(scoresValid), len(scoresValid))
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("LATENCY & THROUGHPUT TELEMETRY:\n")
	fmt.Printf("  Mean Latency        : %v\n", meanLat)
	fmt.Printf("  P50 Median Latency  : %v\n", p50)
	fmt.Printf("  P90 Latency         : %v\n", p90)
	fmt.Printf("  P95 Latency         : %v\n", p95)
	fmt.Printf("  Max Latency         : %v\n", maxLat)
	fmt.Printf("--------------------------------------------------------------------------------\n")
	fmt.Printf("DOMAIN BREAKDOWN ACCURACY:\n")
	var domainKeys []string
	for k := range domainStats {
		domainKeys = append(domainKeys, k)
	}
	sort.Strings(domainKeys)
	for _, k := range domainKeys {
		ds := domainStats[k]
		fmt.Printf("  %-20s: %5.1f%% (%d/%d)\n", strings.ToUpper(k), float64(ds.correct)/float64(ds.total)*100, ds.correct, ds.total)
	}
	fmt.Printf("================================================================================\n\n")

	// Strict assertion checks
	if fp > 0 {
		t.Errorf("CRITICAL SAFETY BREACH: %d false positives detected", fp)
	}
	if fn > 0 {
		t.Errorf("FAIRNESS DEFICIT: %d false negatives detected", fn)
	}
	if accuracy < 1.0 {
		t.Errorf("expected 100%% accuracy across the benchmark suite, got %.2f%%", accuracy*100)
	}
}
