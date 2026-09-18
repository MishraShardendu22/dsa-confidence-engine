package analysis

import (
	"strings"
)

type FunctionRelevance struct {
	RelevantVars map[string]bool
}

func containsIdentifier(raw, ident string) bool {
	if ident == "" || raw == "" {
		return false
	}
	idx := 0
	for {
		pos := strings.Index(raw[idx:], ident)
		if pos == -1 {
			return false
		}
		actualPos := idx + pos
		endPos := actualPos + len(ident)

		leftOk := actualPos == 0 || !isIdentRune(rune(raw[actualPos-1]))
		rightOk := endPos == len(raw) || !isIdentRune(rune(raw[endPos]))

		if leftOk && rightOk {
			return true
		}
		idx = actualPos + 1
	}
}

func isIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// ComputeOutputRelevance computes backward dataflow slice from return statements
// of all reachable functions.
func ComputeOutputRelevance(reachableFuncs map[string]*FunctionNode) map[string]*FunctionRelevance {
	relevanceMap := make(map[string]*FunctionRelevance, len(reachableFuncs))

	for name, fn := range reachableFuncs {
		relevant := make(map[string]bool)

		// 1. Seed with variables directly present in Return statements and control guards
		for _, ret := range fn.Returns {
			for _, v := range ret.Vars {
				relevant[v] = true
			}
			// Also inspect raw return expression for variable tokens
			if ret.Raw != "" {
				for _, op := range fn.Operations {
					if op.Var != "" && containsIdentifier(ret.Raw, op.Var) {
						relevant[op.Var] = true
					}
				}
			}
		}

		if len(fn.Returns) > 0 {
			for _, wl := range fn.WhileLoops {
				for _, cv := range wl.CondVars {
					relevant[cv] = true
				}
			}
			for _, ig := range fn.IfGuards {
				if ig.HasReturn {
					for _, cv := range ig.CondVars {
						relevant[cv] = true
					}
				}
			}
		}

		// 2. Backward propagation loop until convergence
		changed := true
		for changed {
			changed = false

			// Check var defs: if target var is relevant, all its deps are relevant
			for varName, def := range fn.VarDefs {
				if relevant[varName] {
					for _, dep := range def.Deps {
						if !relevant[dep] {
							relevant[dep] = true
							changed = true
						}
					}
				}
			}

			// Check mutations: if target var is mutated by other variables
			for _, mut := range fn.VarMutations {
				if relevant[mut.Var] {
					for _, d := range mut.ValDeps {
						if !relevant[d] {
							relevant[d] = true
							changed = true
						}
					}
					for _, k := range mut.KeyDeps {
						if !relevant[k] {
							relevant[k] = true
							changed = true
						}
					}
				}
			}

			// Check reads: if a read influences an assignment to a relevant variable
			for _, r := range fn.VarReads {
				for varName, def := range fn.VarDefs {
					if relevant[varName] {
						for _, dep := range def.Deps {
							if dep == r.Var && !relevant[r.Var] {
								relevant[r.Var] = true
								changed = true
							}
						}
					}
				}
			}
		}

		relevanceMap[name] = &FunctionRelevance{RelevantVars: relevant}
	}

	return relevanceMap
}

// IsVariableOutputRelevant checks whether the given variable name belongs to the backward slice.
func IsVariableOutputRelevant(relevanceMap map[string]*FunctionRelevance, funcName, varName string) bool {
	if rel, ok := relevanceMap[funcName]; ok {
		return rel.RelevantVars[varName]
	}
	return false
}
