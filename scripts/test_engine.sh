#!/usr/bin/env bash
# ==============================================================================
# DSA Confidence Engine — Interactive End-to-End Test Suite for Humans
# ==============================================================================
# Tests the engine against 3 canonical evaluation archetypes:
#   1. Valid implementation + matching explanation (Fast-path ACCEPT)
#   2. Valid implementation + colloquial explanation (Semantic ACCEPT)
#   3. Valid implementation + fraudulent bluffing explanation (Security REJECT)
# ==============================================================================

set -euo pipefail

PORT="${TEST_PORT:-8089}"
BASE_URL="http://localhost:${PORT}"
SERVER_PID=""

cleanup() {
    if [ -n "${SERVER_PID}" ]; then
        echo ""
        echo "Shutting down test server (PID: ${SERVER_PID})..."
        kill -9 "${SERVER_PID}" 2>/dev/null || true
        wait "${SERVER_PID}" 2>/dev/null || true
    fi
    rm -f test_server data/test_quickstart.db*
}
trap cleanup EXIT

# 1. Compile server
echo "==> Compiling server binary for testing..."
go build -o test_server ./cmd/server

# 2. Start server in background
echo "==> Starting test server on port ${PORT}..."
APP_ADDR=":${PORT}" DATABASE_PATH="data/test_quickstart.db" ./test_server > /dev/null 2>&1 &
SERVER_PID=$!

# 3. Wait for health check
echo "==> Waiting for server to become healthy..."
RETRIES=30
until curl -s "${BASE_URL}/health" | grep -q '"status":"healthy"' || [ $RETRIES -eq 0 ]; do
    sleep 0.2
    RETRIES=$((RETRIES - 1))
done

if [ $RETRIES -eq 0 ]; then
    echo "ERROR: Server failed to start on port ${PORT}."
    exit 1
fi
echo "==> Server is healthy and listening on ${BASE_URL}."
echo ""

# Helper function to submit and evaluate
run_test_case() {
    local title="$1"
    local code="$2"
    local expl="$3"
    local expected_decision="$4"

    echo "--------------------------------------------------------------------------------"
    echo "TEST CASE: ${title}"
    echo "--------------------------------------------------------------------------------"
    echo "Explanation : \"${expl}\""

    local payload
    payload=$(jq -n \
        --arg prob "two_sum" \
        --arg code "${code}" \
        --arg expl "${expl}" \
        '{problem_id: $prob, source_code: $code, explanation: $expl}')

    local response
    response=$(curl -s -X POST "${BASE_URL}/api/evaluate" \
        -H "Content-Type: application/json" \
        -d "${payload}")

    local decision
    decision=$(echo "${response}" | jq -r '.data.decision // "ERROR"')
    local score
    score=$(echo "${response}" | jq -r '.data.fidelity_score // 0')
    local reason
    reason=$(echo "${response}" | jq -r '.data.reason // ""')
    local test_result
    test_result=$(echo "${response}" | jq -r '.data.test_result // "FAIL"')
    local actual
    actual=$(echo "${response}" | jq -c '.data.actual_concepts // [] | map(.id // .concept_id)')
    local claimed
    claimed=$(echo "${response}" | jq -c '.data.claimed_concepts // [] | map(.id // .concept_id)')

    echo "Tests Pass  : ${test_result}"
    echo "Actual DSA  : ${actual}"
    echo "Claimed DSA : ${claimed}"
    echo "Fidelity    : ${score}"
    echo "Decision    : ${decision} (Expected: ${expected_decision})"
    echo "Reason      : ${reason}"

    if [ "${decision}" != "${expected_decision}" ]; then
        echo "RESULT      : FAILED (Expected ${expected_decision}, got ${decision})"
        return 1
    else
        echo "RESULT      : PASSED"
    fi
    echo ""
}

TWO_SUM_CODE='def solve(nums, target):
    seen = {}
    for i, n in enumerate(nums):
        diff = target - n
        if diff in seen:
            return [seen[diff], i]
        seen[n] = i
    return []
'

# Case 1: Standard Exact Match
run_test_case \
    "Standard Exact Match (Fast-Path)" \
    "${TWO_SUM_CODE}" \
    "I used a hashmap to store numbers and their indices, checking for the target difference." \
    "ACCEPT"

# Case 2: Colloquial Vernacular Match
run_test_case \
    "Colloquial Vernacular Match" \
    "${TWO_SUM_CODE}" \
    "I stored previously visited elements in a dictionary lookup to check if the complement was already seen." \
    "ACCEPT"

# Case 3: Cheating / Bluffing Attempt
run_test_case \
    "Cheating / Bluffing Detection" \
    "${TWO_SUM_CODE}" \
    "I constructed a directed weighted graph and ran Dijkstra shortest path algorithm with a priority queue." \
    "REJECT"

echo "================================================================================"
echo "ALL END-TO-END VERIFICATION TESTS PASSED (3/3)"
echo "================================================================================"
