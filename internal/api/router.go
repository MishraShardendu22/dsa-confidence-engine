package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/service"
	"github.com/MishraShardendu22/dsa-confidence-engine/web/templates"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

type ProblemHandler interface {
	List(c *fiber.Ctx) error
	Get(c *fiber.Ctx) error
	Create(c *fiber.Ctx) error
}

type EvaluationHandler interface {
	Evaluate(c *fiber.Ctx) error
	Get(c *fiber.Ctx) error
}

type RouterConfig struct {
	ProblemHandler    ProblemHandler
	EvaluationHandler EvaluationHandler
	ProblemService    *service.ProblemService
	EvaluationService *service.EvaluationService
	OntologyRepo      *dsa.Repository
}

func RegisterRoutes(app *fiber.App, cfg RouterConfig) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return Success(c, fiber.Map{
			"status": "healthy",
		})
	})

	// JSON API
	apiGroup := app.Group("/api")
	apiGroup.Get("/problems", cfg.ProblemHandler.List)
	apiGroup.Get("/problems/:id", cfg.ProblemHandler.Get)
	apiGroup.Post("/problems", cfg.ProblemHandler.Create)
	apiGroup.Post("/evaluate", cfg.EvaluationHandler.Evaluate)
	apiGroup.Get("/evaluations/:id", cfg.EvaluationHandler.Get)

	// Web HTML Pages
	app.Get("/", func(c *fiber.Ctx) error {
		problems, err := cfg.ProblemService.ListProblems(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return render(c, templates.Index(problems))
	})

	app.Post("/evaluate", func(c *fiber.Ctx) error {
		sub := model.Submission{
			ProblemID:   c.FormValue("problem_id"),
			Explanation: c.FormValue("explanation"),
			SourceCode:  c.FormValue("source_code"),
		}
		if sub.ProblemID == "" || sub.SourceCode == "" {
			return c.Status(fiber.StatusBadRequest).SendString("problem_id and source_code are required")
		}
		eval, err := cfg.EvaluationService.Evaluate(c.Context(), sub)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.Redirect("/evaluation/" + eval.ID)
	})

	app.Get("/evaluation/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		eval, err := cfg.EvaluationService.GetEvaluation(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Evaluation not found")
		}
		return render(c, templates.Evaluation(eval))
	})

	app.Get("/problems/new", func(c *fiber.Ctx) error {
		concepts := cfg.OntologyRepo.Ontology().AllConcepts()
		return render(c, templates.ProblemNew(concepts))
	})

	app.Post("/problems/create", func(c *fiber.Ctx) error {
		id := c.FormValue("id")
		title := c.FormValue("title")
		desc := c.FormValue("description")
		lang := c.FormValue("language")
		entrypoint := c.FormValue("entrypoint")
		stratRaw := c.FormValue("accepted_strategies")
		testsJSON := c.FormValue("tests_json")

		var strategies []string
		for _, s := range strings.Split(stratRaw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				strategies = append(strategies, s)
			}
		}

		var tests []model.TestCase
		if err := json.Unmarshal([]byte(testsJSON), &tests); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid tests JSON: " + err.Error())
		}

		reqConcepts := c.Context().PostArgs().PeekMulti("required_concepts")
		var required []string
		for _, b := range reqConcepts {
			required = append(required, string(b))
		}

		optConcepts := c.Context().PostArgs().PeekMulti("optional_concepts")
		var optional []string
		for _, b := range optConcepts {
			optional = append(optional, string(b))
		}

		p := &model.Problem{
			ID:                 id,
			Title:              title,
			Description:        desc,
			Language:           lang,
			Entrypoint:         entrypoint,
			Tests:              tests,
			AcceptedStrategies: strategies,
			RequiredConcepts:   required,
			OptionalConcepts:   optional,
			PrimaryConcepts:    required,
		}

		if err := cfg.ProblemService.CreateProblem(c.Context(), p); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.Redirect("/")
	})
}

func render(c *fiber.Ctx, component templ.Component) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	return component.Render(c.Context(), c.Response().BodyWriter())
}
