package controller

import (
	"errors"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/api"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/service"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/util"
	"github.com/gofiber/fiber/v2"
)

type ProblemController struct {
	svc *service.ProblemService
}

func NewProblemController(svc *service.ProblemService) *ProblemController {
	return &ProblemController{svc: svc}
}

func (ctrl *ProblemController) List(c *fiber.Ctx) error {
	problems, err := ctrl.svc.ListProblems(c.Context())
	if err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return api.Success(c, problems)
}

func (ctrl *ProblemController) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	p, err := ctrl.svc.GetProblem(c.Context(), id)
	if err != nil {
		if errors.Is(err, util.ErrProblemNotFound) {
			return api.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Problem not found")
		}
		return api.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return api.Success(c, p)
}

func (ctrl *ProblemController) Create(c *fiber.Ctx) error {
	var p model.Problem
	if err := c.BodyParser(&p); err != nil {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Failed to parse problem JSON")
	}

	if p.ID == "" || p.Title == "" || p.Entrypoint == "" {
		return api.Error(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "Problem ID, title, and entrypoint are required")
	}

	if len(p.Tests) == 0 {
		return api.Error(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "At least one test case is required")
	}

	if err := ctrl.svc.CreateProblem(c.Context(), &p); err != nil {
		return api.Error(c, fiber.StatusInternalServerError, "STORAGE_ERROR", err.Error())
	}

	return api.Success(c, p, fiber.StatusCreated)
}
