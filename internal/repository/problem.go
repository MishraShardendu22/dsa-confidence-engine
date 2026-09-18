package repository

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
	"gopkg.in/yaml.v3"
)

// LoadProblemsFromDirectory recursively discovers and ingests problem specifications
// from directory hierarchies supporting .yaml, .yml, .json, and .jsonl files.
func LoadProblemsFromDirectory(ctx context.Context, dir string, repo *SQLiteRepository) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	var batch []*model.Problem
	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := repo.SaveProblemsBatch(ctx, batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".yaml", ".yml":
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read problem file %s: %w", path, err)
			}
			var p model.Problem
			if err := yaml.Unmarshal(data, &p); err != nil {
				return fmt.Errorf("failed to parse problem YAML %s: %w", path, err)
			}
			if p.ID == "" {
				p.ID = strings.TrimSuffix(d.Name(), ext)
			}
			batch = append(batch, &p)
			if len(batch) >= 500 {
				if err := flushBatch(); err != nil {
					return err
				}
			}

		case ".json":
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read JSON problem file %s: %w", path, err)
			}
			trimmed := bytes.TrimSpace(data)
			if len(trimmed) == 0 {
				return nil
			}

			// Check if root element is an array of problems
			if trimmed[0] == '[' {
				var probs []*model.Problem
				if err := json.Unmarshal(trimmed, &probs); err != nil {
					return fmt.Errorf("failed to parse problem JSON array %s: %w", path, err)
				}
				for _, p := range probs {
					if p != nil {
						batch = append(batch, p)
						if len(batch) >= 500 {
							if err := flushBatch(); err != nil {
								return err
							}
						}
					}
				}
			} else {
				var p model.Problem
				if err := json.Unmarshal(trimmed, &p); err != nil {
					return fmt.Errorf("failed to parse problem JSON %s: %w", path, err)
				}
				if p.ID == "" {
					p.ID = strings.TrimSuffix(d.Name(), ext)
				}
				batch = append(batch, &p)
				if len(batch) >= 500 {
					if err := flushBatch(); err != nil {
						return err
					}
				}
			}

		case ".jsonl":
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open JSONL file %s: %w", path, err)
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			// Allow up to 10MB per JSONL line for large test suites
			buf := make([]byte, 64*1024)
			scanner.Buffer(buf, 10*1024*1024)

			lineNo := 0
			for scanner.Scan() {
				lineNo++
				line := bytes.TrimSpace(scanner.Bytes())
				if len(line) == 0 {
					continue
				}
				var p model.Problem
				if err := json.Unmarshal(line, &p); err != nil {
					return fmt.Errorf("failed to parse JSONL line %d in %s: %w", lineNo, path, err)
				}
				batch = append(batch, &p)
				if len(batch) >= 500 {
					if err := flushBatch(); err != nil {
						return err
					}
				}
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading JSONL file %s: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return flushBatch()
}
