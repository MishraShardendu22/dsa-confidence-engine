package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateProblem validates a Problem struct, enforces required fields,
// checks test cases, and applies production defaults for missing optional fields.
func ValidateProblem(p *Problem) error {
	if p == nil {
		return errors.New("problem cannot be nil")
	}

	p.ID = strings.TrimSpace(p.ID)
	if p.ID == "" {
		return errors.New("problem id is required")
	}
	if !validIDRegex.MatchString(p.ID) {
		return fmt.Errorf("problem id '%s' contains invalid characters (must be alphanumeric, underscore, or hyphen)", p.ID)
	}

	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" {
		return fmt.Errorf("problem '%s': title is required", p.ID)
	}

	p.Language = strings.ToLower(strings.TrimSpace(p.Language))
	if p.Language == "" {
		p.Language = "python"
	}

	p.Entrypoint = strings.TrimSpace(p.Entrypoint)
	if p.Entrypoint == "" {
		return fmt.Errorf("problem '%s': entrypoint is required", p.ID)
	}

	// Ensure primary entrypoint is included in EntrypointAliases
	hasPrimaryAlias := false
	for _, alias := range p.EntrypointAliases {
		if alias == p.Entrypoint {
			hasPrimaryAlias = true
			break
		}
	}
	if !hasPrimaryAlias {
		p.EntrypointAliases = append([]string{p.Entrypoint}, p.EntrypointAliases...)
	}

	// Validate test cases
	if len(p.Tests) == 0 {
		return fmt.Errorf("problem '%s': must define at least one test case", p.ID)
	}
	for i, tc := range p.Tests {
		if strings.TrimSpace(tc.ID) == "" {
			p.Tests[i].ID = fmt.Sprintf("%d", i+1)
		}
		if strings.TrimSpace(tc.Input) == "" {
			return fmt.Errorf("problem '%s': test case #%d input cannot be empty", p.ID, i+1)
		}
	}

	// Defaults for runtime limits and difficulty
	if p.TimeLimitMs <= 0 {
		p.TimeLimitMs = 2000
	}
	if p.MemoryLimitMB <= 0 {
		p.MemoryLimitMB = 256
	}

	p.Difficulty = strings.TrimSpace(p.Difficulty)
	if p.Difficulty == "" {
		p.Difficulty = "Medium"
	} else {
		// Normalize casing: Easy, Medium, Hard
		dLow := strings.ToLower(p.Difficulty)
		switch dLow {
		case "easy":
			p.Difficulty = "Easy"
		case "hard":
			p.Difficulty = "Hard"
		default:
			p.Difficulty = "Medium"
		}
	}

	if p.RequiredConcepts == nil {
		p.RequiredConcepts = []string{}
	}
	if p.OptionalConcepts == nil {
		p.OptionalConcepts = []string{}
	}
	if p.PrimaryConcepts == nil {
		p.PrimaryConcepts = []string{}
	}
	if p.AcceptedStrategies == nil {
		p.AcceptedStrategies = []string{}
	}
	if p.TopicTags == nil {
		p.TopicTags = []string{}
	}
	if p.Hints == nil {
		p.Hints = []string{}
	}

	return nil
}
