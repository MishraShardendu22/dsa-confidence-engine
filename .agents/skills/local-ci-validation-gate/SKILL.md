---
name: local-ci-validation-gate
scope: generic
description: >-
  Strict rules and operational runbooks for executing full local CI validation gates
  (formatting, module hygiene, static analysis, race detector unit tests, and binary compilation)
  before any local Git commit is created and before any remote push is attempted.
  Strictly prohibits committing or pushing code with broken local CI.
---

# Local CI Validation Gate & Pre-Commit Enforcement Skill

This skill defines the mandatory protocol for mirroring GitHub Actions CI locally before creating Git commits or pushing branches to remote repositories.

---

## 1. High-Priority Rule: Zero Remote CI Failures

> [!CAUTION]
> **NEVER PUSH OR COMMIT BROKEN CODE**:
> - AI agents and contributors MUST run the full local CI validation gate (`make pre-commit`) and ensure 100% pass rate before committing changes.
> - **Zero Tolerance for Formatting or Test Drift**: Commits containing unformatted Go code (`gofmt -l .`), race detector test failures, compiler errors, or untidy `go.mod`/`go.sum` are strictly prohibited.
> - If any check in the local CI gate fails, the agent MUST resolve the defect immediately before committing.
> - Pushing to remote (`git push`) is forbidden if any local verification test fails.

---

## 2. GitHub Actions CI Parity Matrix

The local verification commands directly map 1:1 to the jobs defined in `.github/workflows/ci.yml`:

| GitHub Actions CI Job | What It Validates | Local Command | Remediation Command |
| :--- | :--- | :--- | :--- |
| **`go-lint`** | `gofmt -l .` zero formatting drift | `make fmt-check` | `make fmt` or `gofmt -w .` |
| **`go-build` (tidy)** | `go mod tidy` and git diff check | `make tidy-check` | `go mod tidy` |
| **`go-test`** | `go test -v -race -count=1 ./...` | `make test-race` | Fix failing test or race condition |
| **`go-build` (compile)** | `go build -v ./...` | `make build` | Fix compilation/syntax error |
| **Static Analysis** | `go vet ./...` | `make vet` | Fix suspicious code constructs |

---

## 3. The Mandatory Pre-Commit Execution Protocol

Before running `git commit -s -S` for any milestone:

```bash
# 1. Run the unified local CI validation gate
make pre-commit
```

The unified `make pre-commit` executes in order:
1. `fmt-check`: Verifies all Go files adhere strictly to `gofmt`.
2. `vet`: Runs Go static analysis across all packages.
3. `tidy-check`: Ensures `go.mod` and `go.sum` are clean without uncommitted dependency changes.
4. `templ`: Generates templ UI components if any templates changed.
5. `test-race`: Runs the test suite with the Go data race detector enabled.
6. `build`: Compiles all binaries to guarantee zero compilation breakages.

---

## 4. Agent Remediation Runbook

When a check fails during `make pre-commit`:

### If `fmt-check` fails:
```bash
# Automatically format all Go files in-place
make fmt
```

### If `tidy-check` fails:
```bash
# Prune unused dependencies and synchronize checksums
go mod tidy
```

### If `test-race` fails:
1. Review the test failure output or race detector stack trace.
2. Fix the underlying synchronization flaw or assertion mismatch.
3. Re-run `make test-race` until clean.

### If `build` fails:
1. Correct syntax errors, missing package imports, or interface mismatches.
2. Re-run `make build`.

Only after `make pre-commit` prints:
`All pre-commit and CI verification checks passed successfully.`
is the agent permitted to execute `git commit -s -S`.
