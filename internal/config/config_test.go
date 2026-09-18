package config_test

import (
	"os"
	"testing"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/config"
)

func TestConfigDefaults(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	if cfg.AppAddr != ":8080" {
		t.Errorf("expected :8080, got %s", cfg.AppAddr)
	}
	if cfg.RunnerTimeoutMs != 3000 {
		t.Errorf("expected 3000, got %d", cfg.RunnerTimeoutMs)
	}
	if cfg.AcceptThreshold != 0.95 {
		t.Errorf("expected 0.95, got %f", cfg.AcceptThreshold)
	}
	if cfg.RejustifyThreshold != 0.90 {
		t.Errorf("expected 0.90, got %f", cfg.RejustifyThreshold)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr bool
	}{
		{
			name: "invalid timeout",
			mutate: func(c *config.Config) {
				c.RunnerTimeoutMs = 0
			},
			wantErr: true,
		},
		{
			name: "invalid rejustify threshold negative",
			mutate: func(c *config.Config) {
				c.RejustifyThreshold = -0.1
			},
			wantErr: true,
		},
		{
			name: "invalid rejustify threshold >= 1.0",
			mutate: func(c *config.Config) {
				c.RejustifyThreshold = 1.0
			},
			wantErr: true,
		},
		{
			name: "invalid accept threshold > 1.0",
			mutate: func(c *config.Config) {
				c.AcceptThreshold = 1.1
			},
			wantErr: true,
		},
		{
			name: "rejustify >= accept threshold",
			mutate: func(c *config.Config) {
				c.RejustifyThreshold = 0.95
				c.AcceptThreshold = 0.95
			},
			wantErr: true,
		},
		{
			name: "valid config",
			mutate: func(c *config.Config) {
				c.RunnerTimeoutMs = 2000
				c.RejustifyThreshold = 0.85
				c.AcceptThreshold = 0.95
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("failed to load initial config: %v", err)
			}
			tc.mutate(cfg)
			err = cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestConfigEnvOverridesAndFallbacks(t *testing.T) {
	os.Setenv("RUNNER_TIMEOUT_MS", "invalid_int")
	os.Setenv("ACCEPT_THRESHOLD", "not_a_float")
	os.Setenv("EMBEDDING_ENABLED", "not_a_bool")
	defer func() {
		os.Unsetenv("RUNNER_TIMEOUT_MS")
		os.Unsetenv("ACCEPT_THRESHOLD")
		os.Unsetenv("EMBEDDING_ENABLED")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config load failed: %v", err)
	}

	if cfg.RunnerTimeoutMs != 3000 {
		t.Errorf("expected fallback 3000, got %d", cfg.RunnerTimeoutMs)
	}
	if cfg.AcceptThreshold != 0.95 {
		t.Errorf("expected fallback 0.95, got %f", cfg.AcceptThreshold)
	}
	if !cfg.EmbeddingEnabled {
		t.Errorf("expected fallback true, got %v", cfg.EmbeddingEnabled)
	}
}
