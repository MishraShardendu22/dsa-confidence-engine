# DSA Approach Fidelity Evaluator

A deterministic, evidence-based coding interview evaluation engine that validates:
1. **Functional Correctness**: Whether candidate code passes problem test cases (immediate first gate).
2. **Approach Fidelity**: Whether the candidate actually implemented the DSA strategy and concepts claimed in their explanation.

---

## 1. Core Architecture & Pipeline

```text
                         Candidate Submission
                     (Code + Natural Explanation)
                                  |
                                  v
                         +----------------+
                         | Run Test Cases | (Subprocess Sandbox Gate)
                         +--------+-------+
                                  |
                    +-------------+-------------+
                    |                           |
                 [FAIL]                      [PASS]
                    |                           |
                    v                           v
              Reject directly         Run two independent analyses
             (Zero AST/NLP cost)                |
                                                v
                           +--------------------+--------------------+
                           |                                         |
                           v                                         v
                     [ POINT A ]                                [ POINT B ]
                Deterministic Program                    Natural Language Cascade
                      Analysis                                (Claimed Concepts)
             - Entrypoint Call Graph                     - Stage 1: Exact Keywords
             - Reachable Code Filter                     - Stage 2: Phrase & Aliases
             - Def-Use & Mutations                       - Stage 3: Local Embeddings
             - Backward Return Slice                     - Stage 4: Optional LLM
             - Concrete Line Evidence
                           |                                         |
                           v                                         v
                     Actual Concepts                          Claimed Concepts
                           |                                         |
                           +--------------------+--------------------+
                                                |
                                                v
                                  Fidelity Scorer & Diagnostics
                                  - Evidence-weighted matching
                                  - Primary/Supporting role weighting
                                                |
                           +--------------------+--------------------+
                           |                    |                    |
                           v                    v                    v
                        < 90%                90 - 95%              >= 95%
                       REJECT               REJUSTIFY              ACCEPT
```

### Threshold Enforcement
- `score < 0.90` => `REJECT`
- `0.90 <= score < 0.95` => `REJUSTIFY`
- `score >= 0.95` => `ACCEPT`

---

## 2. Key Subsystems

### The Test Gate
Untrusted candidate code executes in an isolated Python subprocess with configurable execution timeouts (default 3,000ms). If test cases fail, evaluation halts immediately (`test_result = FAIL`, `decision = REJECT`), consuming zero resources on AST parsing, embeddings, or scoring.

### Point A: Deterministic Program Analysis
Point A performs backward dataflow slicing from the `return` statement of reachable functions:
- Unrelated functions (e.g. dead helpers with 10,000 dictionary operations) are marked unreachable and discarded.
- Populated data structures that do not contribute to the return value are flagged as `output_relevant = false`, preventing cheating via unused data structures.
- Detectors emit concrete line-numbered evidence for all verified operations.

### Point B: Natural Language Cascade
Candidate explanations are mapped to the DSA ontology through a 4-stage cascade:
1. Exact keyword normalization
2. Longest-match alias and synonym phrase matching
3. Deterministic local subword embedding similarity against precomputed ontology concept vectors
4. Optional structured LLM escalation (disabled by default)

### Extensible Data-Driven Ontology
The ontology lives in modular YAML files (`data/dsa/*.yaml`) across 29 categories. Adding concepts, aliases, signals, and problem contracts requires modifying YAML data only—zero Go code modifications.

---

## 3. How to Run

### Prerequisites
- Go 1.25+ or Go 1.26+
- Python 3.10+
- `templ` CLI (`go install github.com/a-h/templ/cmd/templ@latest`)

### Quick Start with Make
```bash
# Generate templates and start the application
make run

# Run full automated test suite
make test

# Run tests with race detector
make test-race

# Build production binary
make build

# View all available targets
make help
```

### Manual Build and Launch
```bash
# 1. Compile Templ templates
templ generate

# 2. Build the server binary
go build -o server ./cmd/server

# 3. Start the application
./server
```

The web interface will be accessible at `http://localhost:8080`.

---

## 4. End-to-End Evaluation Example: Two Sum

### 1. Problem Contract
```yaml
id: two_sum
title: Two Sum
entrypoint: solve
primary_concepts:
  - hashmap
required_concepts:
  - hashmap
```

### 2. Candidate Explanation
> "I'll store each number and its index in a hashmap, and then look up whether the target complement exists in constant time."

### 3. Candidate Code
```python
def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
```

### 4. Evaluation Outcome
```json
{
  "test_result": "PASS",
  "passed_tests": 3,
  "total_tests": 3,
  "actual_concepts": [
    {
      "id": "hashmap",
      "name": "Hash Map",
      "reachable": true,
      "output_relevant": true,
      "confidence": 1.0,
      "evidence": [
        {"line": 3, "type": "map_allocation", "description": "Dictionary allocated for 'seen'"},
        {"line": 7, "type": "map_read", "description": "Subscript lookup on map 'seen'"},
        {"line": 8, "type": "map_write", "description": "Key-value write on map 'seen'"}
      ]
    }
  ],
  "claimed_concepts": [
    {
      "id": "hashmap",
      "name": "Hash Map",
      "matched_phrase": "hashmap",
      "stage": "ALIAS",
      "confidence": 1.0
    }
  ],
  "fidelity_score": 1.0,
  "decision": "ACCEPT",
  "diagnostics": [
    "Approach fidelity confirmed: candidate implemented claimed concepts with direct output relevance."
  ]
}
```

---

## 5. Developer Guide: Extending the Engine

### How to Add a DSA Concept
Add an entry to the corresponding YAML file under `data/dsa/` (e.g. `data/dsa/dynamic_programming.yaml`):
```yaml
- id: dp_tree_rerooting
  name: Tree DP Rerooting
  parent: dp_tree
  description: Re-rooting dynamic programming technique on trees.
  aliases:
    - tree rerooting
    - rerooting dp
  keywords:
    - rerooting
    - tree
```
The engine automatically loads and indexes the new concept upon restart with zero Go code modifications.

### How to Add an Alias or Synonym
Add the phrase to the concept's `aliases` list in `data/dsa/*.yaml`:
```yaml
  aliases:
    - hash map
    - dictionary
    - lookup table
    - table of seen entries
```

### How to Add a Problem
Add problem records to `data/problems/dataset_all.json` (or any `.json` file under `data/problems/`) conforming to [`data/schemas/problem.schema.json`](data/schemas/problem.schema.json):
```json
{
  "id": "coin_change",
  "title": "Coin Change",
  "difficulty": "Medium",
  "topic_tags": ["dynamic-programming", "array"],
  "language": "python",
  "entrypoint": "solve",
  "entrypoint_aliases": ["solve", "coinChange"],
  "tests": [
    {
      "id": "1",
      "name": "Standard example 1",
      "input": "{\"coins\": [1, 2, 5], \"amount\": 11}",
      "expected_output": "3",
      "description": "Returns minimum coins needed"
    }
  ],
  "accepted_strategies": ["dp_unbounded_knapsack"],
  "required_concepts": ["dynamic_programming"],
  "optional_concepts": ["memoization", "tabulation"],
  "primary_concepts": ["dynamic_programming"],
  "time_limit_ms": 2000,
  "memory_limit_mb": 256
}
```

### How to Add a Strategy / Concept Detector
Implement the `ConceptDetector` interface in `internal/analysis/detectors.go`:
```go
type MyCustomDetector struct{}

func (d *MyCustomDetector) Detect(
    functions []FunctionNode,
    reachableFuncs map[string]*FunctionNode,
    relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
    // Inspect functions, reachableFuncs, and relevanceMap
    // Return detected concepts with concrete evidence
}
```
Register the detector in `NewDetectorRegistry()` in `internal/analysis/detectors.go`.

### How to Add a New Language Analyzer
Implement the `Analyzer` interface in `internal/analysis/analyzer.go`:
```go
type Analyzer interface {
    Analyze(ctx context.Context, source []byte, contract AnalysisContract) (AnalysisResult, error)
}
```
Create a new file `internal/analysis/go.go` or `internal/analysis/typescript.go` implementing `Analyze`.

### How to Enable or Configure Local Embeddings
Configure environment variables in `.env` or execution shell:
```bash
export EMBEDDING_ENABLED=true
export EMBEDDING_MODEL=local-hashing-embedder
```

---

## 6. HTTP API Reference

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/health` | Health status |
| `GET` | `/api/problems` | List all problem definitions |
| `GET` | `/api/problems/:id` | Get single problem details |
| `POST` | `/api/problems` | Create new problem definition |
| `POST` | `/api/evaluate` | Submit candidate solution for evaluation |
| `GET` | `/api/evaluations/:id` | Fetch evaluation result by ID |

---

## 7. Web Interface

- `/`: Interactive problem selector, code editor, candidate explanation textarea, and sample pre-fills.
- `/evaluation/:id`: Full evaluation report showing test case status, fidelity score, decision, collapsible line evidence, and diagnostics.
- `/problems/new`: Problem creation form with multi-select concepts dynamically populated from the ontology.
