package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"github.com/MishraShardendu22/dsa-confidence-engine/internal/util"
	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS problems (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		difficulty TEXT DEFAULT 'Medium',
		topic_tags_json TEXT DEFAULT '[]',
		language TEXT NOT NULL,
		entrypoint TEXT NOT NULL,
		entrypoint_aliases_json TEXT DEFAULT '[]',
		starter_code TEXT DEFAULT '',
		tests_json TEXT NOT NULL,
		accepted_strategies_json TEXT NOT NULL,
		required_concepts_json TEXT NOT NULL,
		optional_concepts_json TEXT NOT NULL,
		primary_concepts_json TEXT NOT NULL,
		hints_json TEXT DEFAULT '[]',
		time_limit_ms INTEGER DEFAULT 2000,
		memory_limit_mb INTEGER DEFAULT 256,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS evaluations (
		id TEXT PRIMARY KEY,
		problem_id TEXT NOT NULL,
		submission_code TEXT NOT NULL,
		explanation TEXT NOT NULL,
		test_result TEXT NOT NULL,
		passed_tests INTEGER NOT NULL,
		failed_tests INTEGER NOT NULL,
		total_tests INTEGER NOT NULL,
		actual_concepts_json TEXT NOT NULL,
		claimed_concepts_json TEXT NOT NULL,
		matched_concepts_json TEXT NOT NULL,
		missing_concepts_json TEXT NOT NULL,
		extra_concepts_json TEXT NOT NULL,
		fidelity_score REAL NOT NULL,
		decision TEXT NOT NULL,
		reason TEXT,
		evidence_json TEXT NOT NULL,
		diagnostics_json TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := r.db.Exec(schema); err != nil {
		return err
	}
	// Migrations for existing databases
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN entrypoint_aliases_json TEXT DEFAULT '[]';")
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN starter_code TEXT DEFAULT '';")
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN difficulty TEXT DEFAULT 'Medium';")
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN topic_tags_json TEXT DEFAULT '[]';")
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN hints_json TEXT DEFAULT '[]';")
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN time_limit_ms INTEGER DEFAULT 2000;")
	_, _ = r.db.Exec("ALTER TABLE problems ADD COLUMN memory_limit_mb INTEGER DEFAULT 256;")
	return nil
}

func (r *SQLiteRepository) SaveProblem(ctx context.Context, p *model.Problem) error {
	if err := model.ValidateProblem(p); err != nil {
		return err
	}

	aliasesJSON, err := json.Marshal(p.EntrypointAliases)
	if err != nil {
		return err
	}
	testsJSON, err := json.Marshal(p.Tests)
	if err != nil {
		return err
	}
	strategiesJSON, err := json.Marshal(p.AcceptedStrategies)
	if err != nil {
		return err
	}
	reqJSON, err := json.Marshal(p.RequiredConcepts)
	if err != nil {
		return err
	}
	optJSON, err := json.Marshal(p.OptionalConcepts)
	if err != nil {
		return err
	}
	priJSON, err := json.Marshal(p.PrimaryConcepts)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(p.TopicTags)
	if err != nil {
		return err
	}
	hintsJSON, err := json.Marshal(p.Hints)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO problems (
		id, title, description, difficulty, topic_tags_json,
		language, entrypoint, entrypoint_aliases_json, starter_code,
		tests_json, accepted_strategies_json, required_concepts_json,
		optional_concepts_json, primary_concepts_json,
		hints_json, time_limit_ms, memory_limit_mb
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=excluded.title,
		description=excluded.description,
		difficulty=excluded.difficulty,
		topic_tags_json=excluded.topic_tags_json,
		language=excluded.language,
		entrypoint=excluded.entrypoint,
		entrypoint_aliases_json=excluded.entrypoint_aliases_json,
		starter_code=excluded.starter_code,
		tests_json=excluded.tests_json,
		accepted_strategies_json=excluded.accepted_strategies_json,
		required_concepts_json=excluded.required_concepts_json,
		optional_concepts_json=excluded.optional_concepts_json,
		primary_concepts_json=excluded.primary_concepts_json,
		hints_json=excluded.hints_json,
		time_limit_ms=excluded.time_limit_ms,
		memory_limit_mb=excluded.memory_limit_mb;
	`
	_, err = r.db.ExecContext(ctx, query,
		p.ID, p.Title, p.Description, p.Difficulty, string(tagsJSON),
		p.Language, p.Entrypoint, string(aliasesJSON), p.StarterCode,
		string(testsJSON), string(strategiesJSON), string(reqJSON),
		string(optJSON), string(priJSON),
		string(hintsJSON), p.TimeLimitMs, p.MemoryLimitMB,
	)
	return err
}

func (r *SQLiteRepository) SaveProblemsBatch(ctx context.Context, problems []*model.Problem) error {
	if len(problems) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO problems (
		id, title, description, difficulty, topic_tags_json,
		language, entrypoint, entrypoint_aliases_json, starter_code,
		tests_json, accepted_strategies_json, required_concepts_json,
		optional_concepts_json, primary_concepts_json,
		hints_json, time_limit_ms, memory_limit_mb
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title=excluded.title,
		description=excluded.description,
		difficulty=excluded.difficulty,
		topic_tags_json=excluded.topic_tags_json,
		language=excluded.language,
		entrypoint=excluded.entrypoint,
		entrypoint_aliases_json=excluded.entrypoint_aliases_json,
		starter_code=excluded.starter_code,
		tests_json=excluded.tests_json,
		accepted_strategies_json=excluded.accepted_strategies_json,
		required_concepts_json=excluded.required_concepts_json,
		optional_concepts_json=excluded.optional_concepts_json,
		primary_concepts_json=excluded.primary_concepts_json,
		hints_json=excluded.hints_json,
		time_limit_ms=excluded.time_limit_ms,
		memory_limit_mb=excluded.memory_limit_mb;
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	for _, p := range problems {
		if err := model.ValidateProblem(p); err != nil {
			return fmt.Errorf("invalid problem %s: %w", p.ID, err)
		}

		aliasesJSON, _ := json.Marshal(p.EntrypointAliases)
		testsJSON, _ := json.Marshal(p.Tests)
		strategiesJSON, _ := json.Marshal(p.AcceptedStrategies)
		reqJSON, _ := json.Marshal(p.RequiredConcepts)
		optJSON, _ := json.Marshal(p.OptionalConcepts)
		priJSON, _ := json.Marshal(p.PrimaryConcepts)
		tagsJSON, _ := json.Marshal(p.TopicTags)
		hintsJSON, _ := json.Marshal(p.Hints)

		_, err = stmt.ExecContext(ctx,
			p.ID, p.Title, p.Description, p.Difficulty, string(tagsJSON),
			p.Language, p.Entrypoint, string(aliasesJSON), p.StarterCode,
			string(testsJSON), string(strategiesJSON), string(reqJSON),
			string(optJSON), string(priJSON),
			string(hintsJSON), p.TimeLimitMs, p.MemoryLimitMB,
		)
		if err != nil {
			return fmt.Errorf("failed to insert problem %s in batch: %w", p.ID, err)
		}
	}

	return tx.Commit()
}

func (r *SQLiteRepository) GetProblem(ctx context.Context, id string) (*model.Problem, error) {
	query := `
	SELECT id, title, description, COALESCE(difficulty, 'Medium'), COALESCE(topic_tags_json, '[]'),
	       language, entrypoint, entrypoint_aliases_json, starter_code,
	       tests_json, accepted_strategies_json, required_concepts_json,
	       optional_concepts_json, primary_concepts_json,
	       COALESCE(hints_json, '[]'), COALESCE(time_limit_ms, 2000), COALESCE(memory_limit_mb, 256)
	FROM problems WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var p model.Problem
	var aliasesJSON, testsJSON, stratJSON, reqJSON, optJSON, priJSON, tagsJSON, hintsJSON string
	err := row.Scan(
		&p.ID, &p.Title, &p.Description, &p.Difficulty, &tagsJSON,
		&p.Language, &p.Entrypoint, &aliasesJSON, &p.StarterCode,
		&testsJSON, &stratJSON, &reqJSON, &optJSON, &priJSON,
		&hintsJSON, &p.TimeLimitMs, &p.MemoryLimitMB,
	)
	if err == sql.ErrNoRows {
		return nil, util.ErrProblemNotFound
	}
	if err != nil {
		return nil, err
	}

	if aliasesJSON != "" {
		_ = json.Unmarshal([]byte(aliasesJSON), &p.EntrypointAliases)
	}
	if testsJSON != "" {
		_ = json.Unmarshal([]byte(testsJSON), &p.Tests)
	}
	if stratJSON != "" {
		_ = json.Unmarshal([]byte(stratJSON), &p.AcceptedStrategies)
	}
	if reqJSON != "" {
		_ = json.Unmarshal([]byte(reqJSON), &p.RequiredConcepts)
	}
	if optJSON != "" {
		_ = json.Unmarshal([]byte(optJSON), &p.OptionalConcepts)
	}
	if priJSON != "" {
		_ = json.Unmarshal([]byte(priJSON), &p.PrimaryConcepts)
	}
	if tagsJSON != "" {
		_ = json.Unmarshal([]byte(tagsJSON), &p.TopicTags)
	}
	if hintsJSON != "" {
		_ = json.Unmarshal([]byte(hintsJSON), &p.Hints)
	}

	return &p, nil
}

func (r *SQLiteRepository) ListProblems(ctx context.Context) ([]model.Problem, error) {
	query := `
	SELECT id, title, description, COALESCE(difficulty, 'Medium'), COALESCE(topic_tags_json, '[]'),
	       language, entrypoint, entrypoint_aliases_json, starter_code,
	       tests_json, accepted_strategies_json, required_concepts_json,
	       optional_concepts_json, primary_concepts_json,
	       COALESCE(hints_json, '[]'), COALESCE(time_limit_ms, 2000), COALESCE(memory_limit_mb, 256)
	FROM problems ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []model.Problem
	for rows.Next() {
		var p model.Problem
		var aliasesJSON, testsJSON, stratJSON, reqJSON, optJSON, priJSON, tagsJSON, hintsJSON string
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &p.Difficulty, &tagsJSON,
			&p.Language, &p.Entrypoint, &aliasesJSON, &p.StarterCode,
			&testsJSON, &stratJSON, &reqJSON, &optJSON, &priJSON,
			&hintsJSON, &p.TimeLimitMs, &p.MemoryLimitMB,
		); err != nil {
			return nil, err
		}
		if aliasesJSON != "" {
			_ = json.Unmarshal([]byte(aliasesJSON), &p.EntrypointAliases)
		}
		if testsJSON != "" {
			_ = json.Unmarshal([]byte(testsJSON), &p.Tests)
		}
		if stratJSON != "" {
			_ = json.Unmarshal([]byte(stratJSON), &p.AcceptedStrategies)
		}
		if reqJSON != "" {
			_ = json.Unmarshal([]byte(reqJSON), &p.RequiredConcepts)
		}
		if optJSON != "" {
			_ = json.Unmarshal([]byte(optJSON), &p.OptionalConcepts)
		}
		if priJSON != "" {
			_ = json.Unmarshal([]byte(priJSON), &p.PrimaryConcepts)
		}
		if tagsJSON != "" {
			_ = json.Unmarshal([]byte(tagsJSON), &p.TopicTags)
		}
		if hintsJSON != "" {
			_ = json.Unmarshal([]byte(hintsJSON), &p.Hints)
		}
		problems = append(problems, p)
	}
	return problems, rows.Err()
}

func (r *SQLiteRepository) SaveEvaluation(ctx context.Context, e *model.Evaluation) error {
	actualJSON, err := json.Marshal(e.ActualConcepts)
	if err != nil {
		return fmt.Errorf("failed to marshal actual concepts: %w", err)
	}
	claimedJSON, err := json.Marshal(e.ClaimedConcepts)
	if err != nil {
		return fmt.Errorf("failed to marshal claimed concepts: %w", err)
	}
	matchedJSON, err := json.Marshal(e.MatchedConcepts)
	if err != nil {
		return fmt.Errorf("failed to marshal matched concepts: %w", err)
	}
	missingJSON, err := json.Marshal(e.MissingConcepts)
	if err != nil {
		return fmt.Errorf("failed to marshal missing concepts: %w", err)
	}
	extraJSON, err := json.Marshal(e.ExtraConcepts)
	if err != nil {
		return fmt.Errorf("failed to marshal extra concepts: %w", err)
	}
	evidenceJSON, err := json.Marshal(e.Evidence)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}
	diagJSON, err := json.Marshal(e.Diagnostics)
	if err != nil {
		return fmt.Errorf("failed to marshal diagnostics: %w", err)
	}

	query := `
	INSERT INTO evaluations (
		id, problem_id, submission_code, explanation, test_result,
		passed_tests, failed_tests, total_tests,
		actual_concepts_json, claimed_concepts_json, matched_concepts_json,
		missing_concepts_json, extra_concepts_json, fidelity_score,
		decision, reason, evidence_json, diagnostics_json,
		duration_ms, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	createdAt := e.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err = r.db.ExecContext(ctx, query,
		e.ID, e.ProblemID, e.SubmissionCode, e.Explanation, string(e.TestResult),
		e.PassedTests, e.FailedTests, e.TotalTests,
		string(actualJSON), string(claimedJSON), string(matchedJSON),
		string(missingJSON), string(extraJSON), e.FidelityScore,
		string(e.Decision), e.Reason, string(evidenceJSON), string(diagJSON),
		e.DurationMs, createdAt,
	)
	return err
}

func (r *SQLiteRepository) GetEvaluation(ctx context.Context, id string) (*model.Evaluation, error) {
	query := `
	SELECT id, problem_id, submission_code, explanation, test_result,
	       passed_tests, failed_tests, total_tests,
	       actual_concepts_json, claimed_concepts_json, matched_concepts_json,
	       missing_concepts_json, extra_concepts_json, fidelity_score,
	       decision, reason, evidence_json, diagnostics_json,
	       duration_ms, created_at
	FROM evaluations WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var e model.Evaluation
	var testRes, decision string
	var actualJSON, claimedJSON, matchedJSON, missingJSON, extraJSON, evidenceJSON, diagJSON string
	var reason sql.NullString

	err := row.Scan(
		&e.ID, &e.ProblemID, &e.SubmissionCode, &e.Explanation, &testRes,
		&e.PassedTests, &e.FailedTests, &e.TotalTests,
		&actualJSON, &claimedJSON, &matchedJSON,
		&missingJSON, &extraJSON, &e.FidelityScore,
		&decision, &reason, &evidenceJSON, &diagJSON,
		&e.DurationMs, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, util.ErrEvaluationNotFound
	}
	if err != nil {
		return nil, err
	}

	e.TestResult = model.TestStatus(testRes)
	e.Decision = model.Decision(decision)
	if reason.Valid {
		e.Reason = reason.String
	}

	if err := json.Unmarshal([]byte(actualJSON), &e.ActualConcepts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal actual concepts: %w", err)
	}
	if err := json.Unmarshal([]byte(claimedJSON), &e.ClaimedConcepts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claimed concepts: %w", err)
	}
	if err := json.Unmarshal([]byte(matchedJSON), &e.MatchedConcepts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal matched concepts: %w", err)
	}
	if err := json.Unmarshal([]byte(missingJSON), &e.MissingConcepts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal missing concepts: %w", err)
	}
	if err := json.Unmarshal([]byte(extraJSON), &e.ExtraConcepts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal extra concepts: %w", err)
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &e.Evidence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal evidence: %w", err)
	}
	if err := json.Unmarshal([]byte(diagJSON), &e.Diagnostics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal diagnostics: %w", err)
	}

	return &e, nil
}
