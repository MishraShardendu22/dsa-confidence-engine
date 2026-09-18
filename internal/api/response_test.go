package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/api"
	"github.com/gofiber/fiber/v2"
)

func TestSuccessResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test-success", func(c *fiber.Ctx) error {
		return api.Success(c, map[string]string{"foo": "bar"})
	})
	app.Get("/test-created", func(c *fiber.Ctx) error {
		return api.Success(c, "created-item", fiber.StatusCreated)
	})

	// 1. Default status 200
	req := httptest.NewRequest(http.MethodGet, "/test-success", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env api.Response
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !env.Success || env.Error != nil {
		t.Errorf("expected success=true and error=nil, got %+v", env)
	}

	// 2. Custom status 201
	req = httptest.NewRequest(http.MethodGet, "/test-created", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestErrorResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test-error", func(c *fiber.Ctx) error {
		return api.Error(c, fiber.StatusBadRequest, "INVALID_INPUT", "Bad argument")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var env api.Response
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if env.Success {
		t.Errorf("expected success=false, got true")
	}
	if env.Error == nil || env.Error.Code != "INVALID_INPUT" || env.Error.Message != "Bad argument" {
		t.Errorf("unexpected error payload: %+v", env.Error)
	}
}
