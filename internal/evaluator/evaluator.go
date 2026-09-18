package evaluator

import (
	"context"
	"fmt"
	"time"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/analysis"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/fidelity"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/nlp"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/runner"
	"github.com/google/uuid"
)

type Evaluator struct {
	runner   runner.Runner
	analyzer analysis.Analyzer
	matcher  *nlp.CascadeMatcher
	scorer   *fidelity.Scorer
	llm      nlp.LLMEscalator
}

func NewEvaluator(
	r runner.Runner,
	a analysis.Analyzer,
	m *nlp.CascadeMatcher,
	s *fidelity.Scorer,
	llm nlp.LLMEscalator,
) *Evaluator {
	if llm == nil {
		llm = &nlp.DisabledLLMEscalator{}
	}
	if m != nil && llm != nil {
		m.SetEscalator(llm)
	}
	return &Evaluator{
		runner:   r,
		analyzer: a,
		matcher:  m,
		scorer:   s,
		llm:      llm,
	}
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	submission model.Submission,
	problem model.Problem,
) (*model.Evaluation, error) {
	start := time.Now()
	evalID := uuid.NewString()

	// 1. Test Gate: Run tests first.
	testRes, err := e.runner.Run(ctx, submission, problem)
	if err != nil {
		return nil, err
	}

	// If tests fail, stop immediately. Zero Point A, Point B, or LLM overhead.
	if testRes.Status == model.TestStatusFail {
		var diags []string
		if testRes.Error != "" {
			diags = append(diags, testRes.Error)
		}
		for _, d := range testRes.Details {
			if !d.Passed && d.Error != "" {
				diags = append(diags, d.Error)
			}
		}

		return &model.Evaluation{
			ID:              evalID,
			ProblemID:       problem.ID,
			SubmissionCode:  submission.SourceCode,
			Explanation:     submission.Explanation,
			TestResult:      model.TestStatusFail,
			PassedTests:     testRes.PassedCount,
			FailedTests:     testRes.FailedCount,
			TotalTests:      testRes.TotalCount,
			ActualConcepts:  []model.DetectedConcept{},
			ClaimedConcepts: []model.ClaimedConcept{},
			MatchedConcepts: []model.ConceptMatch{},
			MissingConcepts: []model.ConceptMatch{},
			ExtraConcepts:   []model.ConceptMatch{},
			FidelityScore:   0.0,
			Decision:        model.DecisionReject,
			Reason:          "Candidate solution failed required test cases.",
			Evidence:        []model.Evidence{},
			Diagnostics:     diags,
			DurationMs:      time.Since(start).Milliseconds(),
			CreatedAt:       time.Now(),
		}, nil
	}

	// 2. Tests Passed -> Run Point A (Actual code analysis)
	analysisRes, err := e.analyzer.Analyze(ctx, []byte(submission.SourceCode), analysis.AnalysisContract{
		Entrypoint:        problem.Entrypoint,
		EntrypointAliases: problem.EntrypointAliases,
		ProblemID:         problem.ID,
		Language:          problem.Language,
	})
	if err != nil {
		return nil, err
	}
	if analysisRes.Status == "ERROR" || analysisRes.Status == "UNKNOWN" {
		errMsg := analysisRes.Error
		if errMsg == "" {
			errMsg = "Static code analysis failed to extract AST from submission"
		}
		return nil, fmt.Errorf("analysis failed: %s", errMsg)
	}

	// 3. Run Point B (Claimed concepts from explanation with hybrid escalation)
	var detectedIDs []string
	for _, c := range analysisRes.ActualConcepts {
		detectedIDs = append(detectedIDs, c.ConceptID)
	}
	claimedConcepts, err := e.matcher.MatchWithEscalation(ctx, submission.Explanation, detectedIDs, analysisRes.Evidence)
	if err != nil {
		return nil, err
	}

	// 4. Compare Point A vs Point B via Fidelity Scorer
	fidelityRes := e.scorer.Score(analysisRes.ActualConcepts, claimedConcepts, problem)

	return &model.Evaluation{
		ID:              evalID,
		ProblemID:       problem.ID,
		SubmissionCode:  submission.SourceCode,
		Explanation:     submission.Explanation,
		TestResult:      model.TestStatusPass,
		PassedTests:     testRes.PassedCount,
		FailedTests:     testRes.FailedCount,
		TotalTests:      testRes.TotalCount,
		ActualConcepts:  analysisRes.ActualConcepts,
		ClaimedConcepts: claimedConcepts,
		MatchedConcepts: fidelityRes.Matched,
		MissingConcepts: fidelityRes.Missing,
		ExtraConcepts:   fidelityRes.Extra,
		FidelityScore:   fidelityRes.Score,
		Decision:        fidelityRes.Decision,
		Reason:          fidelityRes.Explanation,
		Evidence:        analysisRes.Evidence,
		Diagnostics:     fidelityRes.Diagnostics,
		DurationMs:      time.Since(start).Milliseconds(),
		CreatedAt:       time.Now(),
	}, nil
}
