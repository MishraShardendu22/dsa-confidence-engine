package model

type TestCase struct {
	ID             string `json:"id" yaml:"id"`
	Name           string `json:"name,omitempty" yaml:"name,omitempty"`
	Input          string `json:"input" yaml:"input"`
	ExpectedOutput string `json:"expected_output" yaml:"expected_output"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Problem struct {
	ID                 string     `json:"id" yaml:"id"`
	Title              string     `json:"title" yaml:"title"`
	Description        string     `json:"description" yaml:"description"`
	Difficulty         string     `json:"difficulty,omitempty" yaml:"difficulty,omitempty"`
	TopicTags          []string   `json:"topic_tags,omitempty" yaml:"topic_tags,omitempty"`
	Language           string     `json:"language" yaml:"language"`
	Entrypoint         string     `json:"entrypoint" yaml:"entrypoint"`
	EntrypointAliases  []string   `json:"entrypoint_aliases,omitempty" yaml:"entrypoint_aliases,omitempty"`
	StarterCode        string     `json:"starter_code,omitempty" yaml:"starter_code,omitempty"`
	Tests              []TestCase `json:"tests" yaml:"tests"`
	AcceptedStrategies []string   `json:"accepted_strategies" yaml:"accepted_strategies"`
	RequiredConcepts   []string   `json:"required_concepts" yaml:"required_concepts"`
	OptionalConcepts   []string   `json:"optional_concepts" yaml:"optional_concepts"`
	PrimaryConcepts    []string   `json:"primary_concepts" yaml:"primary_concepts"`
	Hints              []string   `json:"hints,omitempty" yaml:"hints,omitempty"`
	TimeLimitMs        int        `json:"time_limit_ms,omitempty" yaml:"time_limit_ms,omitempty"`
	MemoryLimitMB      int        `json:"memory_limit_mb,omitempty" yaml:"memory_limit_mb,omitempty"`
}
