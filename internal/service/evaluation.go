package service

import (
	"context"
	"fmt"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/evaluator"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/repository"
)

type EvaluationService struct {
	evaluator *evaluator.Evaluator
	repo      *repository.SQLiteRepository
}

func NewEvaluationService(
	e *evaluator.Evaluator,
	repo *repository.SQLiteRepository,
) *EvaluationService {
	return &EvaluationService{
		evaluator: e,
		repo:      repo,
	}
}

func (s *EvaluationService) Evaluate(ctx context.Context, sub model.Submission) (*model.Evaluation, error) {
	problem, err := s.repo.GetProblem(ctx, sub.ProblemID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve problem %s: %w", sub.ProblemID, err)
	}

	eval, err := s.evaluator.Evaluate(ctx, sub, *problem)
	if err != nil {
		return nil, fmt.Errorf("evaluation failed: %w", err)
	}

	if err := s.repo.SaveEvaluation(ctx, eval); err != nil {
		return nil, fmt.Errorf("failed to save evaluation: %w", err)
	}

	return eval, nil
}

func (s *EvaluationService) GetEvaluation(ctx context.Context, id string) (*model.Evaluation, error) {
	return s.repo.GetEvaluation(ctx, id)
}
