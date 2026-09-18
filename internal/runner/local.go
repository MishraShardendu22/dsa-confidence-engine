package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

type LocalRunner struct {
	defaultTimeout time.Duration
}

func NewLocalRunner(timeoutMs int) *LocalRunner {
	if timeoutMs <= 0 {
		timeoutMs = 3000
	}
	return &LocalRunner{
		defaultTimeout: time.Duration(timeoutMs) * time.Millisecond,
	}
}

type runnerPayload struct {
	Code              string           `json:"code"`
	ProblemID         string           `json:"problem_id"`
	Entrypoint        string           `json:"entrypoint"`
	EntrypointAliases []string         `json:"entrypoint_aliases"`
	Tests             []model.TestCase `json:"tests"`
}

type runnerResponse struct {
	Status      string           `json:"status"`
	PassedCount int              `json:"passed_count"`
	FailedCount int              `json:"failed_count"`
	TotalCount  int              `json:"total_count"`
	Details     []TestCaseDetail `json:"details"`
	Error       string           `json:"error,omitempty"`
}

const pythonHarness = `
import sys
import io
import json
import ast
import traceback

def run():
    real_stdout = sys.stdout
    real_stderr = sys.stderr

    try:
        raw_input = sys.stdin.read()
        payload = json.loads(raw_input)
    except Exception as e:
        real_stdout.write(json.dumps({
            "status": "FAIL",
            "passed_count": 0,
            "failed_count": 1,
            "total_count": 1,
            "error": f"Failed to parse test payload: {str(e)}",
            "details": []
        }) + "\n")
        return

    code = payload.get("code", "")
    entrypoint_name = payload.get("entrypoint", "solve")
    entrypoint_aliases = payload.get("entrypoint_aliases") or []
    problem_id = payload.get("problem_id", "")
    tests = payload.get("tests") or []

    env = {}

    # 1. Parse AST and hoist functions/classes before executing to avoid forward-reference NameErrors
    try:
        tree = ast.parse(code)
        funcs = [n for n in tree.body if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef))]
        others = [n for n in tree.body if not isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef))]
        tree.body = funcs + others
        ast.fix_missing_locations(tree)
        compiled = compile(tree, "<submission>", "exec")
    except Exception as e:
        tb = traceback.format_exc()
        real_stdout.write(json.dumps({
            "status": "FAIL",
            "passed_count": 0,
            "failed_count": len(tests) if tests else 1,
            "total_count": len(tests) if tests else 1,
            "error": f"Syntax/Compilation error: {str(e)}\n{tb}",
            "details": []
        }) + "\n")
        return

    if not tests:
        real_stdout.write(json.dumps({
            "status": "FAIL",
            "passed_count": 0,
            "failed_count": 0,
            "total_count": 0,
            "error": "No test cases configured for problem",
            "details": []
        }) + "\n")
        return

    # 2. Redirect stdout/stderr so candidate prints do not break runner JSON
    capture_buf = io.StringIO()
    sys.stdout = capture_buf
    sys.stderr = capture_buf
    try:
        exec(compiled, env)
    except Exception as e:
        sys.stdout = real_stdout
        sys.stderr = real_stderr
        tb = traceback.format_exc()
        real_stdout.write(json.dumps({
            "status": "FAIL",
            "passed_count": 0,
            "failed_count": len(tests) if tests else 1,
            "total_count": len(tests) if tests else 1,
            "error": f"Execution error: {str(e)}\n{tb}",
            "details": []
        }) + "\n")
        return
    finally:
        sys.stdout = real_stdout
        sys.stderr = real_stderr

    # 3. Resolve entrypoint intelligently
    norm_prob = problem_id.replace("_", "").lower()
    norm_entry = entrypoint_name.replace("_", "").lower()

    entrypoint = None
    if entrypoint_name in env and callable(env[entrypoint_name]):
        entrypoint = env[entrypoint_name]

    if entrypoint is None:
        for alias in entrypoint_aliases:
            if alias in env and callable(env[alias]):
                entrypoint = env[alias]
                break

    if entrypoint is None:
        all_funcs = [n.name for n in funcs if isinstance(n, ast.FunctionDef)]
        for fname in all_funcs:
            norm_fname = fname.replace("_", "").lower()
            if norm_fname == norm_prob or norm_fname == norm_entry or norm_fname in ("solve", "solution"):
                if fname in env and callable(env[fname]):
                    entrypoint = env[fname]
                    break

    if entrypoint is None:
        all_funcs = [n.name for n in funcs if isinstance(n, ast.FunctionDef)]
        if len(all_funcs) == 1 and all_funcs[0] in env and callable(env[all_funcs[0]]):
            entrypoint = env[all_funcs[0]]

    # Support LeetCode style class Solution
    if entrypoint is None and "Solution" in env and isinstance(env["Solution"], type):
        try:
            sol_instance = env["Solution"]()
            if hasattr(sol_instance, entrypoint_name) and callable(getattr(sol_instance, entrypoint_name)):
                entrypoint = getattr(sol_instance, entrypoint_name)
            if entrypoint is None:
                for alias in entrypoint_aliases:
                    if hasattr(sol_instance, alias) and callable(getattr(sol_instance, alias)):
                        entrypoint = getattr(sol_instance, alias)
                        break
            if entrypoint is None:
                for attr in dir(sol_instance):
                    if not attr.startswith("_") and callable(getattr(sol_instance, attr)):
                        norm_attr = attr.replace("_", "").lower()
                        if norm_attr == norm_prob or norm_attr == norm_entry or norm_attr in ("solve", "solution"):
                            entrypoint = getattr(sol_instance, attr)
                            break
        except Exception:
            pass

    if entrypoint is None:
        real_stdout.write(json.dumps({
            "status": "FAIL",
            "passed_count": 0,
            "failed_count": len(tests) if tests else 1,
            "total_count": len(tests) if tests else 1,
            "error": f"Entrypoint function '{entrypoint_name}' (or alias) not found or not callable",
            "details": []
        }) + "\n")
        return

    passed_count = 0
    failed_count = 0
    details = []

    def normalize(val):
        if val is None:
            return "None"
        if isinstance(val, bool):
            return str(val)
        if isinstance(val, (int, float, str)):
            return str(val)
        if isinstance(val, (set, frozenset)):
            try:
                return json.dumps(sorted(list(val)))
            except Exception:
                return str(val)
        try:
            return json.dumps(val, sort_keys=True)
        except Exception:
            return str(val)

    def compare(actual, expected_str):
        actual_str = normalize(actual)
        if actual_str == expected_str:
            return True
        try:
            exp_json = json.loads(expected_str)
            act_json = json.loads(actual_str)
            if exp_json == act_json:
                return True
            if isinstance(exp_json, list) and isinstance(act_json, list):
                if exp_json == act_json:
                    return True
        except Exception:
            pass
        if expected_str.lower() in ("true", "false"):
            if str(actual).lower() == expected_str.lower():
                return True
        return False

    for t in tests:
        test_id = t.get("id", "")
        test_input_str = t.get("input", "")
        expected_str = t.get("expected_output", "")

        try:
            parsed_input = json.loads(test_input_str)

            # Isolate prints during individual test case execution
            call_buf = io.StringIO()
            sys.stdout = call_buf
            sys.stderr = call_buf
            try:
                if isinstance(parsed_input, dict):
                    result = entrypoint(**parsed_input)
                elif isinstance(parsed_input, list):
                    try:
                        import inspect
                        sig = inspect.signature(entrypoint)
                        params = [p for p in sig.parameters.values() if p.kind in (p.POSITIONAL_ONLY, p.POSITIONAL_OR_KEYWORD)]
                        if len(params) == 1 and len(parsed_input) != 1:
                            result = entrypoint(parsed_input)
                        else:
                            result = entrypoint(*parsed_input)
                    except Exception:
                        try:
                            result = entrypoint(*parsed_input)
                        except TypeError:
                            result = entrypoint(parsed_input)
                else:
                    result = entrypoint(parsed_input)
            finally:
                sys.stdout = real_stdout
                sys.stderr = real_stderr

            actual_str = normalize(result)
            is_pass = compare(result, expected_str)
            if is_pass:
                passed_count += 1
                details.append({
                    "id": test_id,
                    "passed": True,
                    "expected": expected_str,
                    "actual": actual_str
                })
            else:
                failed_count += 1
                details.append({
                    "id": test_id,
                    "passed": False,
                    "expected": expected_str,
                    "actual": actual_str,
                    "error": f"Expected {expected_str}, got {actual_str}"
                })
        except Exception as e:
            sys.stdout = real_stdout
            sys.stderr = real_stderr
            failed_count += 1
            details.append({
                "id": test_id,
                "passed": False,
                "expected": expected_str,
                "actual": "",
                "error": f"Runtime error: {str(e)}"
            })

    status = "PASS" if failed_count == 0 and passed_count > 0 else "FAIL"
    real_stdout.write(json.dumps({
        "status": status,
        "passed_count": passed_count,
        "failed_count": failed_count,
        "total_count": len(tests),
        "details": details
    }) + "\n")

if __name__ == "__main__":
    run()
`

func (r *LocalRunner) Run(ctx context.Context, submission model.Submission, problem model.Problem) (TestResult, error) {
	start := time.Now()

	execCtx, cancel := context.WithTimeout(ctx, r.defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "python3", "-c", pythonHarness)

	payload := runnerPayload{
		Code:              submission.SourceCode,
		ProblemID:         problem.ID,
		Entrypoint:        problem.Entrypoint,
		EntrypointAliases: problem.EntrypointAliases,
		Tests:             problem.Tests,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return TestResult{
			Status: model.TestStatusFail,
			Error:  fmt.Sprintf("failed to serialize test payload: %v", err),
		}, nil
	}

	cmd.Stdin = bytes.NewReader(payloadBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	if execCtx.Err() == context.DeadlineExceeded {
		return TestResult{
			Status:      model.TestStatusFail,
			PassedCount: 0,
			FailedCount: len(problem.Tests),
			TotalCount:  len(problem.Tests),
			Error:       "Execution timed out",
			DurationMs:  durationMs,
			Output:      stderr.String(),
		}, nil
	}

	stdoutStr := strings.TrimSpace(stdout.String())
	if stdoutStr == "" {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" && err != nil {
			errMsg = err.Error()
		}
		return TestResult{
			Status:      model.TestStatusFail,
			PassedCount: 0,
			FailedCount: len(problem.Tests),
			TotalCount:  len(problem.Tests),
			Error:       errMsg,
			DurationMs:  durationMs,
		}, nil
	}

	var resp runnerResponse
	if parseErr := json.Unmarshal([]byte(stdoutStr), &resp); parseErr != nil {
		return TestResult{
			Status:      model.TestStatusFail,
			PassedCount: 0,
			FailedCount: len(problem.Tests),
			TotalCount:  len(problem.Tests),
			Error:       fmt.Sprintf("invalid test runner response: %s (stderr: %s)", stdoutStr, stderr.String()),
			DurationMs:  durationMs,
			Output:      stdoutStr,
		}, nil
	}

	return TestResult{
		Status:      model.TestStatus(resp.Status),
		PassedCount: resp.PassedCount,
		FailedCount: resp.FailedCount,
		TotalCount:  resp.TotalCount,
		Details:     resp.Details,
		DurationMs:  durationMs,
		Error:       resp.Error,
		Output:      stdoutStr,
	}, nil
}
