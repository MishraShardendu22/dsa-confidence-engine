package controller

import (
	"errors"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/api"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/service"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/util"
	"github.com/gofiber/fiber/v2"
)

type EvaluationController struct {
	svc *service.EvaluationService
}

func NewEvaluationController(svc *service.EvaluationService) *EvaluationController {
	return &EvaluationController{svc: svc}
}

func (ctrl *EvaluationController) Evaluate(c *fiber.Ctx) error {
	var sub model.Submission
	if err := c.BodyParser(&sub); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_SUBMISSION", "Failed to parse submission JSON")
	}

	if sub.ProblemID == "" || sub.SourceCode == "" {
		return api.Error(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "problem_id and source_code are required")
	}

	eval, err := ctrl.svc.Evaluate(c.Context(), sub)
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "EVALUATION_FAILED", err.Error())
	}

	return api.Success(c, eval)
}

func (ctrl *EvaluationController) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	eval, err := ctrl.svc.GetEvaluation(c.Context(), id)
	if err != nil {
		if errors.Is(err, util.ErrEvaluationNotFound) {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Evaluation not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return api.Success(c, eval)
}
