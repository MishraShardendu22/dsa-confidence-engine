package util_test

import (
	"errors"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/util"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{util.ErrProblemNotFound, "problem not found"},
		{util.ErrEvaluationNotFound, "evaluation not found"},
		{util.ErrInvalidSubmission, "invalid submission"},
		{util.ErrTestExecutionFailed, "test execution failed"},
		{util.ErrAnalysisFailed, "static code analysis failed"},
		{util.ErrOntologyNotFound, "ontology concept not found"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.want {
			t.Errorf("error string mismatch: got %q, want %q", tt.err.Error(), tt.want)
		}
		// Verify errors.Is compatibility
		wrapped := errors.Join(errors.New("prefix"), tt.err)
		if !errors.Is(wrapped, tt.err) {
			t.Errorf("wrapped error should match target via errors.Is")
		}
	}
}
