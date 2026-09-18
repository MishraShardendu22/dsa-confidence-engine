package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppAddr            string
	AppEnv             string
	DatabasePath       string
	OntologyPath       string
	ProblemsPath       string
	RunnerTimeoutMs    int
	AcceptThreshold    float64
	RejustifyThreshold float64
	EmbeddingEnabled   bool
	EmbeddingModel     string
	LLMEnabled         bool
	LLMProvider        string
	LLMEndpoint        string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppAddr:            getEnv("APP_ADDR", ":8080"),
		AppEnv:             getEnv("APP_ENV", "development"),
		DatabasePath:       getEnv("DATABASE_PATH", "data/evaluator.db"),
		OntologyPath:       getEnv("ONTOLOGY_PATH", "data/dsa"),
		ProblemsPath:       getEnv("PROBLEMS_PATH", "data/problems"),
		RunnerTimeoutMs:    getEnvInt("RUNNER_TIMEOUT_MS", 3000),
		AcceptThreshold:    getEnvFloat("ACCEPT_THRESHOLD", 0.95),
		RejustifyThreshold: getEnvFloat("REJUSTIFY_THRESHOLD", 0.90),
		EmbeddingEnabled:   getEnvBool("EMBEDDING_ENABLED", true),
		EmbeddingModel:     getEnv("EMBEDDING_MODEL", "local-hashing-embedder"),
		LLMEnabled:         getEnvBool("LLM_ENABLED", false),
		LLMProvider:        getEnv("LLM_PROVIDER", "disabled"),
		LLMEndpoint:        getEnv("LLM_ENDPOINT", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.RunnerTimeoutMs <= 0 {
		return fmt.Errorf("RUNNER_TIMEOUT_MS must be > 0, got %d", c.RunnerTimeoutMs)
	}
	if c.RejustifyThreshold <= 0 || c.RejustifyThreshold >= 1.0 {
		return fmt.Errorf("REJUSTIFY_THRESHOLD must be between 0 and 1, got %f", c.RejustifyThreshold)
	}
	if c.AcceptThreshold <= 0 || c.AcceptThreshold > 1.0 {
		return fmt.Errorf("ACCEPT_THRESHOLD must be between 0 and 1, got %f", c.AcceptThreshold)
	}
	if c.RejustifyThreshold >= c.AcceptThreshold {
		return fmt.Errorf("REJUSTIFY_THRESHOLD (%f) must be less than ACCEPT_THRESHOLD (%f)", c.RejustifyThreshold, c.AcceptThreshold)
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		intVal, err := strconv.Atoi(val)
		if err == nil {
			return intVal
		}
		log.Printf("[WARN] invalid integer value %q for %s, falling back to default %d: %v", val, key, defaultVal, err)
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		floatVal, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return floatVal
		}
		log.Printf("[WARN] invalid float value %q for %s, falling back to default %f: %v", val, key, defaultVal, err)
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		boolVal, err := strconv.ParseBool(val)
		if err == nil {
			return boolVal
		}
		log.Printf("[WARN] invalid boolean value %q for %s, falling back to default %t: %v", val, key, defaultVal, err)
	}
	return defaultVal
}
