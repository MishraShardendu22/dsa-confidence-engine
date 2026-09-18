package analysis

import (
	"fmt"
	"strings"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/model"
)

type ConceptDetector interface {
	Detect(
		functions []FunctionNode,
		reachableFuncs map[string]*FunctionNode,
		relevanceMap map[string]*FunctionRelevance,
	) []model.DetectedConcept
}

// Registry holds all modular concept detectors.
type DetectorRegistry struct {
	detectors []ConceptDetector
}

func NewDetectorRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: []ConceptDetector{
			&HashMapDetector{},
			&HashSetDetector{},
			&SortingDetector{},
			&TwoPointersDetector{},
			&SlidingWindowDetector{},
			&StackDetector{},
			&QueueDetector{},
			&HeapDetector{},
			&BinarySearchDetector{},
			&RecursionDFSDetector{},
			&BFSDetector{},
			&DPDetector{},
			&ArraysDetector{},
			&BruteForceDetector{},
			&BitManipulationDetector{},
			&TrieDetector{},
			&GreedyDetector{},
			&TopologicalSortDetector{},
			&DijkstraDetector{},
			&UnionFindDetector{},
		},
	}
}

func (r *DetectorRegistry) RunAll(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var detected []model.DetectedConcept
	for _, d := range r.detectors {
		results := d.Detect(functions, reachableFuncs, relevanceMap)
		detected = append(detected, results...)
	}
	return detected
}

// 1. HashMap & Frequency Count Detector
type HashMapDetector struct{}

func (d *HashMapDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var mapEvidence []model.Evidence
		var freqEvidence []model.Evidence
		mapOutputRelevant := false
		freqOutputRelevant := false

		dictVars := make(map[string]bool)
		listVars := make(map[string]bool)
		for _, arg := range fn.Args {
			argLower := strings.ToLower(arg)
			if strings.Contains(argLower, "num") || strings.Contains(argLower, "arr") ||
				strings.Contains(argLower, "list") || strings.Contains(argLower, "seq") ||
				strings.Contains(argLower, "val") || strings.Contains(argLower, "element") {
				listVars[arg] = true
			}
		}

		for _, op := range fn.Operations {
			vLow := strings.ToLower(op.Var)
			if vLow == "kwargs" || vLow == "args" || vLow == "self" || vLow == "params" {
				continue
			}
			switch op.Type {
			case "list_alloc":
				listVars[op.Var] = true
			case "dict_alloc":
				dictVars[op.Var] = true
				delete(listVars, op.Var)
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				if rel {
					mapOutputRelevant = true
				}
				mapEvidence = append(mapEvidence, model.Evidence{
					Type:           "map_allocation",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Dictionary allocated for '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "dict_write":
				if !listVars[op.Var] {
					dictVars[op.Var] = true
					rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
					if rel {
						mapOutputRelevant = true
					}
					mapEvidence = append(mapEvidence, model.Evidence{
						Type:           "map_write",
						Line:           op.LineNo,
						Description:    fmt.Sprintf("Key-value write on map '%s'", op.Var),
						Reachable:      isReachable,
						OutputRelevant: rel,
					})
				}
			case "dict_read":
				if dictVars[op.Var] && !listVars[op.Var] {
					rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
					if rel {
						mapOutputRelevant = true
					}
					mapEvidence = append(mapEvidence, model.Evidence{
						Type:           "map_read",
						Line:           op.LineNo,
						Description:    fmt.Sprintf("Subscript lookup on map '%s'", op.Var),
						Reachable:      isReachable,
						OutputRelevant: rel,
					})
				}
			case "dict_get":
				dictVars[op.Var] = true
				delete(listVars, op.Var)
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				if rel {
					mapOutputRelevant = true
					freqOutputRelevant = true
				}
				mapEvidence = append(mapEvidence, model.Evidence{
					Type:           "map_get",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Dictionary get() call on '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
				freqEvidence = append(freqEvidence, model.Evidence{
					Type:           "frequency_lookup_get",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Frequency counting get() method on '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "membership_check":
				if dictVars[op.Var] && !listVars[op.Var] {
					rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
					if rel {
						mapOutputRelevant = true
					}
					mapEvidence = append(mapEvidence, model.Evidence{
						Type:           "map_membership_check",
						Line:           op.LineNo,
						Description:    fmt.Sprintf("Key membership check 'in %s'", op.Var),
						Reachable:      isReachable,
						OutputRelevant: rel,
					})
				}
			case "counter_call":
				rel := isReachable
				if op.Var != "" && op.Var != "Counter" && op.Var != "collections.Counter" {
					rel = isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				}
				if rel {
					mapOutputRelevant = true
					freqOutputRelevant = true
				}
				mapEvidence = append(mapEvidence, model.Evidence{
					Type:           "counter_call",
					Line:           op.LineNo,
					Description:    "collections.Counter frequency mapping invoked",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
				freqEvidence = append(freqEvidence, model.Evidence{
					Type:           "collections_counter",
					Line:           op.LineNo,
					Description:    "collections.Counter frequency counter instance",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "dict_comp":
				rel := isReachable
				if rel {
					mapOutputRelevant = true
					freqOutputRelevant = true
				}
				mapEvidence = append(mapEvidence, model.Evidence{
					Type:           "dict_comprehension",
					Line:           op.LineNo,
					Description:    "Dictionary comprehension mapping constructed",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
				freqEvidence = append(freqEvidence, model.Evidence{
					Type:           "frequency_dict_comprehension",
					Line:           op.LineNo,
					Description:    "Dictionary comprehension frequency mapping",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "count_call":
				rel := isReachable
				if rel {
					freqOutputRelevant = true
				}
				freqEvidence = append(freqEvidence, model.Evidence{
					Type:           "count_scan",
					Line:           op.LineNo,
					Description:    fmt.Sprintf(".count() scan invoked on '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		for _, mut := range fn.VarMutations {
			if mut.Kind == "subscript_aug_assign" {
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, mut.Var)
				if rel {
					freqOutputRelevant = true
				}
				freqEvidence = append(freqEvidence, model.Evidence{
					Type:           "frequency_counter_increment",
					Line:           mut.LineNo,
					Description:    fmt.Sprintf("Augmented frequency increment on '%s'", mut.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(mapEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "hashmap",
				Name:           "Hash Map",
				Category:       "hashing",
				Evidence:       mapEvidence,
				Reachable:      isReachable,
				OutputRelevant: mapOutputRelevant,
				Confidence:     1.0,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}

		if len(freqEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "frequency_count",
				Name:           "HashMap Frequency Counting",
				Category:       "hashing",
				Evidence:       freqEvidence,
				Reachable:      isReachable,
				OutputRelevant: freqOutputRelevant,
				Confidence:     0.95,
				Role:           model.RoleSupporting,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "counting",
				Name:           "Element / Frequency Counting",
				Category:       "hashing",
				Evidence:       freqEvidence,
				Reachable:      isReachable,
				OutputRelevant: freqOutputRelevant,
				Confidence:     0.95,
				Role:           model.RoleSupporting,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 2. HashSet Detector
type HashSetDetector struct{}

func (d *HashSetDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		for _, op := range fn.Operations {
			switch op.Type {
			case "set_alloc":
				rel := isReachable
				if op.Var != "" && op.Var != "<return>" {
					rel = isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				}
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "set_allocation",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Set allocated for '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "set_add":
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "set_add",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Element addition to set '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "set_remove":
				rel := isReachable && (op.Var == "" || IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var))
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "set_remove",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Element removal/reduction on set '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "hashset",
				Name:           "Hash Set",
				Category:       "hashing",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     1.0,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 3. Sorting Detector
type SortingDetector struct{}

func (d *SortingDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		for _, op := range fn.Operations {
			if op.Type == "sorting_call" {
				rel := false
				if isReachable {
					if op.Var != "" {
						rel = IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
					} else {
						for _, ret := range fn.Returns {
							if ret.LineNo == op.LineNo || strings.Contains(ret.Raw, "sorted") || strings.Contains(ret.Raw, "sort") {
								rel = true
								break
							}
						}
					}
				}
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "sorting_call",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Sort operation invoked: '%s'", op.Target),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "sorting",
				Name:           "Sorting",
				Category:       "sorting",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     1.0,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 4. Two Pointers Detector
type TwoPointersDetector struct{}

func (d *TwoPointersDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		var oppositeEvidence []model.Evidence
		outRel := false

		hasInc := false
		hasDec := false
		for _, op := range fn.Operations {
			if op.Type == "pointer_step" {
				if op.Op == "inc" {
					hasInc = true
				} else if op.Op == "dec" {
					hasDec = true
				}
			}
		}

		for _, wl := range fn.WhileLoops {
			expr := strings.ToLower(wl.Expr)
			if strings.Contains(expr, "<") || strings.Contains(expr, "<=") {
				isTwoPtr := (hasInc && hasDec) || (len(wl.CondVars) >= 2 && (hasInc || hasDec))
				if isTwoPtr {
					rel := isReachable
					if rel {
						outRel = true
					}
					evidence = append(evidence, model.Evidence{
						Type:           "two_pointer_while_condition",
						Line:           wl.LineNo,
						Description:    fmt.Sprintf("Convergent pointer loop condition: %s", wl.Expr),
						Reachable:      isReachable,
						OutputRelevant: rel,
					})

					if (hasInc && hasDec) || len(wl.CondVars) >= 2 {
						oppositeEvidence = append(oppositeEvidence, model.Evidence{
							Type:           "opposite_end_pointers",
							Line:           wl.LineNo,
							Description:    "Pointers converging from opposite boundaries towards center",
							Reachable:      isReachable,
							OutputRelevant: rel,
						})
					}
				}
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "two_pointers",
				Name:           "Two Pointers",
				Category:       "two_pointers",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
		if len(oppositeEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "opposite_end_pointers",
				Name:           "Opposite-End Pointers",
				Category:       "two_pointers",
				Evidence:       oppositeEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RoleSupporting,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 5. Sliding Window Detector
type SlidingWindowDetector struct{}

func (d *SlidingWindowDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		stepVars := make(map[string]bool)
		for _, op := range fn.Operations {
			if (op.Type == "pointer_step" || op.Type == "list_pop" || op.Type == "set_remove") && op.Var != "" {
				stepVars[op.Var] = true
			}
		}

		hasWindowKeyword := false
		for v := range fn.VarDefs {
			vLow := strings.ToLower(v)
			if strings.Contains(vLow, "window") || strings.Contains(vLow, "sub") || strings.Contains(vLow, "span") {
				hasWindowKeyword = true
				break
			}
		}

		isGenuineWindow := len(stepVars) >= 2 || hasWindowKeyword || len(fn.WhileLoops) > 1

		for _, wl := range fn.WhileLoops {
			expr := strings.ToLower(wl.Expr)
			hasInWord := strings.Contains(expr, " in ") || strings.HasPrefix(expr, "in ") || strings.HasSuffix(expr, " in")
			if (strings.Contains(expr, ">") || strings.Contains(expr, "<") || hasInWord) && isGenuineWindow {
				for _, op := range fn.Operations {
					if op.Type == "dict_alloc" || op.Type == "set_alloc" || op.Type == "dict_write" || op.Type == "set_remove" {
						rel := isReachable
						if rel {
							outRel = true
						}
						evidence = append(evidence, model.Evidence{
							Type:           "sliding_window_condition",
							Line:           wl.LineNo,
							Description:    fmt.Sprintf("Sliding window invariant check: %s", wl.Expr),
							Reachable:      isReachable,
							OutputRelevant: rel,
						})
						break
					}
				}
			}
		}

		// Check for fixed-size sliding window in for_loop with multiple writes/updates on a map or counter
		dictWriteCount := make(map[string]int)
		for _, op := range fn.Operations {
			if (op.Type == "dict_write" || op.Type == "dict_alloc") && op.Var != "" {
				dictWriteCount[op.Var]++
			}
		}
		for varName, count := range dictWriteCount {
			vLow := strings.ToLower(varName)
			if strings.Contains(vLow, "dist") || strings.Contains(vLow, "memo") ||
				strings.Contains(vLow, "graph") || strings.Contains(vLow, "adj") ||
				strings.Contains(vLow, "parent") || strings.Contains(vLow, "cache") {
				continue
			}
			if count >= 2 {
				for _, op := range fn.Operations {
					if op.Type == "for_loop" {
						rel := isReachable
						if rel {
							outRel = true
						}
						evidence = append(evidence, model.Evidence{
							Type:           "fixed_window_frequency_update",
							Line:           op.LineNo,
							Description:    fmt.Sprintf("Fixed-size sliding window update on map '%s'", varName),
							Reachable:      isReachable,
							OutputRelevant: rel,
						})
						break
					}
				}
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "sliding_window",
				Name:           "Sliding Window",
				Category:       "sliding_window",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.90,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 6. Stack Detector
type StackDetector struct{}

func (d *StackDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		hasAppend := false
		hasPop := false
		outRel := false

		for _, op := range fn.Operations {
			if op.Type == "list_append" {
				hasAppend = true
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "stack_push",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Append element to stack '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			} else if op.Type == "list_pop" {
				hasPop = true
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "stack_pop",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Pop element from stack '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if hasAppend && hasPop {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "stack",
				Name:           "Stack",
				Category:       "stack",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 7. Queue Detector
type QueueDetector struct{}

func (d *QueueDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		for _, op := range fn.Operations {
			if op.Type == "queue_op" {
				rel := isReachable
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "queue_operation",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Queue/deque FIFO operation: %s", op.Op),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "queue",
				Name:           "Queue / Deque",
				Category:       "queue",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 8. Heap Detector
type HeapDetector struct{}

func (d *HeapDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		for _, op := range fn.Operations {
			if op.Type == "heap_op" {
				rel := isReachable
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "heap_call",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Priority queue/heap operation: %s", op.Op),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "heap",
				Name:           "Heap / Priority Queue",
				Category:       "heap",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     1.0,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 9. Binary Search Detector
type BinarySearchDetector struct{}

func (d *BinarySearchDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		hasMid := false
		for _, op := range fn.Operations {
			if op.Type == "bisection_calc" {
				hasMid = true
				break
			}
		}
		if !hasMid {
			for varName, def := range fn.VarDefs {
				lowVar := strings.ToLower(varName)
				isMidName := lowVar == "mid" || lowVar == "middle" || lowVar == "midpoint" || strings.HasPrefix(lowVar, "mid_") || strings.HasSuffix(lowVar, "_mid")
				if isMidName {
					hasMid = true
					break
				}
				for _, dep := range def.Deps {
					lowDep := strings.ToLower(dep)
					if lowDep == "mid" || lowDep == "middle" || lowDep == "midpoint" {
						hasMid = true
						break
					}
				}
			}
		}

		for _, wl := range fn.WhileLoops {
			expr := strings.ToLower(wl.Expr)
			if (strings.Contains(expr, "<=") || strings.Contains(expr, "<")) && hasMid {
				rel := isReachable
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "binary_search_loop",
					Line:           wl.LineNo,
					Description:    fmt.Sprintf("Binary search range bisection loop: %s", wl.Expr),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		isRecursive := false
		for _, call := range fn.Calls {
			if call.Name == fn.Name {
				isRecursive = true
				break
			}
		}

		if hasMid && isRecursive {
			rel := isReachable
			if rel {
				outRel = true
			}
			evidence = append(evidence, model.Evidence{
				Type:           "binary_search_recursion",
				Line:           fn.LineNo,
				Description:    fmt.Sprintf("Binary search recursive bisection helper: %s", fn.Name),
				Reachable:      isReachable,
				OutputRelevant: rel,
			})
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "binary_search",
				Name:           "Binary Search",
				Category:       "binary_search",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 10. Recursion & DFS Detector
type RecursionDFSDetector struct{}

func (d *RecursionDFSDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var recEvidence []model.Evidence
		outRel := false

		for _, call := range fn.Calls {
			if call.Name == fn.Name {
				rel := isReachable
				if rel {
					outRel = true
				}
				recEvidence = append(recEvidence, model.Evidence{
					Type:           "recursive_self_call",
					Line:           call.LineNo,
					Description:    fmt.Sprintf("Recursive invocation of '%s'", fn.Name),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(recEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "recursion",
				Name:           "Recursion",
				Category:       "recursion",
				Evidence:       recEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     1.0,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})

			hasDFSSignal := false
			fnLower := strings.ToLower(fn.Name)
			if strings.Contains(fnLower, "dfs") || strings.Contains(fnLower, "traverse") || strings.Contains(fnLower, "search") {
				hasDFSSignal = true
			}
			for _, arg := range fn.Args {
				argLow := strings.ToLower(arg)
				if strings.Contains(argLow, "node") || strings.Contains(argLow, "graph") ||
					strings.Contains(argLow, "tree") || strings.Contains(argLow, "visited") ||
					strings.Contains(argLow, "seen") || strings.Contains(argLow, "neighbor") ||
					strings.Contains(argLow, "adj") || strings.Contains(argLow, "grid") ||
					strings.Contains(argLow, "row") || strings.Contains(argLow, "col") {
					hasDFSSignal = true
					break
				}
			}
			for varName := range fn.VarDefs {
				vLow := strings.ToLower(varName)
				if strings.Contains(vLow, "visited") || strings.Contains(vLow, "seen") ||
					strings.Contains(vLow, "neighbor") || strings.Contains(vLow, "adj") ||
					strings.Contains(vLow, "graph") {
					hasDFSSignal = true
					break
				}
			}

			hasBacktrackingSignal := false
			if strings.Contains(fnLower, "backtrack") || strings.Contains(fnLower, "subset") || strings.Contains(fnLower, "permute") || strings.Contains(fnLower, "comb") {
				hasBacktrackingSignal = true
			}
			for _, op := range fn.Operations {
				if op.Type == "stack_pop" || op.Type == "list_pop" || op.Type == "pop_call" {
					hasBacktrackingSignal = true
					break
				}
			}
			for varName := range fn.VarDefs {
				vLow := strings.ToLower(varName)
				if strings.Contains(vLow, "tmp") || strings.Contains(vLow, "temp") || strings.Contains(vLow, "restore") || strings.Contains(vLow, "backtrack") {
					hasBacktrackingSignal = true
					break
				}
			}

			if hasBacktrackingSignal {
				concepts = append(concepts, model.DetectedConcept{
					ConceptID:      "backtracking",
					Name:           "Backtracking",
					Category:       "backtracking",
					Evidence:       recEvidence,
					Reachable:      isReachable,
					OutputRelevant: outRel,
					Confidence:     0.95,
					Role:           model.RolePrimary,
					Status:         "DETECTED",
				})
			}

			if hasDFSSignal {
				concepts = append(concepts, model.DetectedConcept{
					ConceptID:      "dfs",
					Name:           "Depth-First Search (DFS)",
					Category:       "graphs",
					Evidence:       recEvidence,
					Reachable:      isReachable,
					OutputRelevant: outRel,
					Confidence:     0.90,
					Role:           model.RolePrimary,
					Status:         "DETECTED",
				})
			}
		}
	}

	return concepts
}

// 11. BFS Detector
type BFSDetector struct{}

func (d *BFSDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		hasQueue := false
		hasQueueAppend := false
		hasWhile := len(fn.WhileLoops) > 0
		var evidence []model.Evidence
		outRel := false

		hasGraphSignal := false
		fnLower := strings.ToLower(fn.Name)
		if strings.Contains(fnLower, "bfs") || strings.Contains(fnLower, "traverse") || strings.Contains(fnLower, "level") {
			hasGraphSignal = true
		}
		for _, arg := range fn.Args {
			argLow := strings.ToLower(arg)
			if strings.Contains(argLow, "node") || strings.Contains(argLow, "graph") ||
				strings.Contains(argLow, "tree") || strings.Contains(argLow, "visited") ||
				strings.Contains(argLow, "seen") || strings.Contains(argLow, "adj") ||
				strings.Contains(argLow, "root") || strings.Contains(argLow, "grid") {
				hasGraphSignal = true
				break
			}
		}
		for varName := range fn.VarDefs {
			vLow := strings.ToLower(varName)
			if strings.Contains(vLow, "visited") || strings.Contains(vLow, "seen") ||
				strings.Contains(vLow, "neighbor") || strings.Contains(vLow, "adj") ||
				strings.Contains(vLow, "graph") || strings.Contains(vLow, "level") {
				hasGraphSignal = true
				break
			}
		}

		for _, op := range fn.Operations {
			if op.Type == "queue_op" {
				hasQueue = true
				rel := isReachable
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "bfs_queue_traversal",
					Line:           op.LineNo,
					Description:    "FIFO queue popping in level-order traversal",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			} else if op.Type == "list_append" {
				hasQueueAppend = true
			}
		}

		if hasQueue && hasWhile && (hasQueueAppend || hasGraphSignal) {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "bfs",
				Name:           "Breadth-First Search (BFS)",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 12. Dynamic Programming Detector
type DPDetector struct{}

func (d *DPDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	funcMap := make(map[string]*FunctionNode, len(functions))
	for i := range functions {
		funcMap[functions[i].Name] = &functions[i]
	}

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var dpEvidence []model.Evidence
		var memoEvidence []model.Evidence
		outRel := false

		isRecursive := false
		for _, call := range fn.Calls {
			if call.Name == fn.Name {
				isRecursive = true
			}
		}

		// Also check if any child/helper function is recursive
		for _, child := range funcMap {
			if child.ParentName == fn.Name {
				for _, c := range child.Calls {
					if c.Name == child.Name {
						isRecursive = true
					}
				}
			}
		}

		hasMemoTable := false
		checkVarForMemo := func(fnName, varName string, defLine int) {
			low := strings.ToLower(varName)
			if strings.Contains(low, "memo") || strings.Contains(low, "cache") {
				hasMemoTable = true
				rel := isReachable
				if rel {
					outRel = true
				}
				memoEvidence = append(memoEvidence, model.Evidence{
					Type:           "memo_table_definition",
					Line:           defLine,
					Description:    fmt.Sprintf("Memoization cache structure '%s'", varName),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		for varName, def := range fn.VarDefs {
			checkVarForMemo(fn.Name, varName, def.LineNo)
		}

		// Check parent scope variables if nested
		if fn.ParentName != "" {
			if parent, ok := funcMap[fn.ParentName]; ok {
				for varName, def := range parent.VarDefs {
					checkVarForMemo(parent.Name, varName, def.LineNo)
				}
			}
		}

		// Also check for memo subscript reads/writes in fn
		for _, op := range fn.Operations {
			if op.Type == "dict_write" || op.Type == "dict_read" || op.Type == "membership_check" {
				if strings.Contains(strings.ToLower(op.Var), "memo") || strings.Contains(strings.ToLower(op.Var), "cache") {
					hasMemoTable = true
					rel := isReachable
					if rel {
						outRel = true
					}
					memoEvidence = append(memoEvidence, model.Evidence{
						Type:           "memo_table_access",
						Line:           op.LineNo,
						Description:    fmt.Sprintf("Memoization state access on '%s'", op.Var),
						Reachable:      isReachable,
						OutputRelevant: rel,
					})
				}
			}
		}

		// Check for tabulation: dp array allocation like dp = [0] * (n + 1)
		for varName, def := range fn.VarDefs {
			if strings.HasPrefix(strings.ToLower(varName), "dp") || strings.Contains(strings.ToLower(varName), "table") {
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, varName)
				if rel {
					outRel = true
				}
				dpEvidence = append(dpEvidence, model.Evidence{
					Type:           "dp_table_allocation",
					Line:           def.LineNo,
					Description:    fmt.Sprintf("Dynamic programming table '%s'", varName),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		// Check for Kadane's algorithm / running state DP (e.g. curr_max, max_so_far, max_prod, min_prod)
		var kadaneEvidence []model.Evidence
		hasKadane := false
		for varName, def := range fn.VarDefs {
			vLow := strings.ToLower(varName)
			if strings.Contains(vLow, "cur_max") || strings.Contains(vLow, "curr_max") ||
				strings.Contains(vLow, "cur_min") || strings.Contains(vLow, "curr_min") ||
				strings.Contains(vLow, "max_so_far") || strings.Contains(vLow, "max_ending") ||
				strings.Contains(vLow, "max_prod") || strings.Contains(vLow, "min_prod") ||
				strings.Contains(vLow, "kadane") {
				hasKadane = true
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, varName)
				if rel {
					outRel = true
				}
				kadaneEvidence = append(kadaneEvidence, model.Evidence{
					Type:           "kadane_state_variable",
					Line:           def.LineNo,
					Description:    fmt.Sprintf("Kadane running DP state variable '%s'", varName),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if isRecursive && hasMemoTable {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "memoization",
				Name:           "Memoization (Top-Down DP)",
				Category:       "dynamic_programming",
				Evidence:       memoEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "dynamic_programming",
				Name:           "Dynamic Programming",
				Category:       "dynamic_programming",
				Evidence:       memoEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		} else if len(dpEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "tabulation",
				Name:           "Tabulation (Bottom-Up DP)",
				Category:       "dynamic_programming",
				Evidence:       dpEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "dynamic_programming",
				Name:           "Dynamic Programming",
				Category:       "dynamic_programming",
				Evidence:       dpEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		} else if hasKadane && len(kadaneEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "kadane",
				Name:           "Kadane's Algorithm",
				Category:       "dynamic_programming",
				Evidence:       kadaneEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "dynamic_programming",
				Name:           "Dynamic Programming",
				Category:       "dynamic_programming",
				Evidence:       kadaneEvidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 13. Arrays / Sequence Detector
type ArraysDetector struct{}

func (d *ArraysDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var arrayEvidence []model.Evidence
		arrayOutputRelevant := false

		// 1. Inspect function arguments for array/list sequence parameters
		for _, arg := range fn.Args {
			argLower := strings.ToLower(arg)
			if strings.Contains(argLower, "num") || strings.Contains(argLower, "arr") ||
				strings.Contains(argLower, "list") || strings.Contains(argLower, "seq") ||
				strings.Contains(argLower, "val") || strings.Contains(argLower, "element") {
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, arg)
				if rel {
					arrayOutputRelevant = true
				}
				arrayEvidence = append(arrayEvidence, model.Evidence{
					Type:           "array_parameter",
					Line:           fn.LineNo,
					Description:    fmt.Sprintf("Sequence/Array parameter '%s' in function '%s'", arg, fn.Name),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		// 2. Inspect operations for list allocations, operations, and loops
		for _, op := range fn.Operations {
			switch op.Type {
			case "list_alloc":
				rel := isReachable
				if op.Var != "<return>" && op.Var != "" {
					rel = isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var)
				}
				if rel {
					arrayOutputRelevant = true
				}
				arrayEvidence = append(arrayEvidence, model.Evidence{
					Type:           "list_allocation",
					Line:           op.LineNo,
					Description:    fmt.Sprintf("List literal allocated '%s'", op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "list_append", "list_pop", "list_write", "list_read", "subscript_read":
				rel := isReachable && (op.Var == "" || IsVariableOutputRelevant(relevanceMap, fn.Name, op.Var))
				if rel {
					arrayOutputRelevant = true
				}
				arrayEvidence = append(arrayEvidence, model.Evidence{
					Type:           op.Type,
					Line:           op.LineNo,
					Description:    fmt.Sprintf("List operation '%s' on '%s'", op.Type, op.Var),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			case "for_loop":
				hasRelevant := false
				for _, iv := range op.IterVars {
					if IsVariableOutputRelevant(relevanceMap, fn.Name, iv) {
						hasRelevant = true
						break
					}
				}
				for _, tv := range op.TargetVars {
					if IsVariableOutputRelevant(relevanceMap, fn.Name, tv) {
						hasRelevant = true
						break
					}
				}
				rel := isReachable && hasRelevant
				if rel {
					arrayOutputRelevant = true
				}
				arrayEvidence = append(arrayEvidence, model.Evidence{
					Type:           "sequence_iteration",
					Line:           op.LineNo,
					Description:    "Iteration over array/sequence elements",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		// 3. Inspect return expressions for list literals
		for _, ret := range fn.Returns {
			trimmed := strings.TrimSpace(ret.Raw)
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				if isReachable {
					arrayOutputRelevant = true
				}
				arrayEvidence = append(arrayEvidence, model.Evidence{
					Type:           "list_return",
					Line:           ret.LineNo,
					Description:    fmt.Sprintf("Returned list expression: %s", trimmed),
					Reachable:      isReachable,
					OutputRelevant: isReachable,
				})
			}
		}

		if len(arrayEvidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "arrays",
				Name:           "Arrays",
				Category:       "arrays",
				Evidence:       arrayEvidence,
				Reachable:      isReachable,
				OutputRelevant: arrayOutputRelevant,
				Confidence:     1.0,
				Role:           model.RoleAuxiliary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 14. Brute Force / Nested Loop Detector
type BruteForceDetector struct{}

func (d *BruteForceDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		for _, op := range fn.Operations {
			if op.Type == "nested_loop" {
				rel := isReachable
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "nested_loops",
					Line:           op.LineNo,
					Description:    "Nested iterative loops (brute force pairwise scan)",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "brute_force",
				Name:           "Brute Force",
				Category:       "arrays",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 15. Bit Manipulation Detector
type BitManipulationDetector struct{}

func (d *BitManipulationDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false

		for _, op := range fn.Operations {
			if op.Type == "bitwise_xor" || op.Type == "bitwise_and" || op.Type == "bitwise_or" || op.Type == "bitwise_shift" || op.Type == "bitwise_not" {
				rel := isReachable
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           op.Type,
					Line:           op.LineNo,
					Description:    fmt.Sprintf("Bitwise operation (%s)", op.Type),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		if len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "bit_manipulation",
				Name:           "Bit Manipulation",
				Category:       "bit_manipulation",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 16. Trie Detector
type TrieDetector struct{}

func (d *TrieDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false
		isTrie := false

		for _, call := range fn.Calls {
			cLower := strings.ToLower(call.Name)
			if strings.Contains(cLower, "trie") || cLower == "insert" || cLower == "search" || cLower == "startswith" || strings.HasSuffix(cLower, ".insert") || strings.HasSuffix(cLower, ".search") || strings.HasSuffix(cLower, ".startswith") {
				isTrie = true
				evidence = append(evidence, model.Evidence{
					Type:           "trie_operation",
					Line:           call.LineNo,
					Description:    fmt.Sprintf("Trie method invocation '%s'", call.Name),
					Reachable:      isReachable,
					OutputRelevant: isReachable,
				})
			}
		}

		for varName, def := range fn.VarDefs {
			vLower := strings.ToLower(varName)
			if strings.Contains(vLower, "trie") || strings.Contains(vLower, "children") || strings.Contains(vLower, "is_end") {
				isTrie = true
				evidence = append(evidence, model.Evidence{
					Type:           "trie_structure",
					Line:           def.LineNo,
					Description:    fmt.Sprintf("Trie structure variable '%s'", varName),
					Reachable:      isReachable,
					OutputRelevant: isReachable,
				})
			}
		}

		if isTrie {
			outRel = isReachable
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "trie",
				Name:           "Trie (Prefix Tree)",
				Category:       "trie",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 17. Greedy Detector
type GreedyDetector struct{}

func (d *GreedyDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		outRel := false
		isGreedy := false

		// Check for greedy variable names and parameters
		for varName, def := range fn.VarDefs {
			vLower := strings.ToLower(varName)
			if strings.Contains(vLower, "reach") || strings.Contains(vLower, "greedy") || strings.Contains(vLower, "farthest") || strings.Contains(vLower, "best") || strings.Contains(vLower, "gas") || strings.Contains(vLower, "interval") {
				isGreedy = true
				evidence = append(evidence, model.Evidence{
					Type:           "greedy_variable",
					Line:           def.LineNo,
					Description:    fmt.Sprintf("Greedy tracking variable '%s'", varName),
					Reachable:      isReachable,
					OutputRelevant: isReachable,
				})
			}
		}

		for _, arg := range fn.Args {
			aLower := strings.ToLower(arg)
			if strings.Contains(aLower, "gas") || strings.Contains(aLower, "cost") || strings.Contains(aLower, "interval") {
				isGreedy = true
				evidence = append(evidence, model.Evidence{
					Type:           "greedy_variable",
					Line:           fn.LineNo,
					Description:    fmt.Sprintf("Greedy tracking parameter '%s'", arg),
					Reachable:      isReachable,
					OutputRelevant: isReachable,
				})
			}
		}

		// Check for max/min call inside function
		for _, call := range fn.Calls {
			cLower := strings.ToLower(call.Name)
			if cLower == "max" || cLower == "min" {
				// Combined with iteration over elements or greedy decision
				for varName := range fn.VarDefs {
					vLower := strings.ToLower(varName)
					if strings.Contains(vLower, "reach") || strings.Contains(vLower, "curr") || strings.Contains(vLower, "max") || strings.Contains(vLower, "min") || strings.Contains(vLower, "ans") {
						isGreedy = true
						evidence = append(evidence, model.Evidence{
							Type:           "greedy_choice",
							Line:           call.LineNo,
							Description:    fmt.Sprintf("Locally optimal choice via '%s()'", call.Name),
							Reachable:      isReachable,
							OutputRelevant: isReachable,
						})
						break
					}
				}
				if isGreedy {
					break
				}
			}
		}

		if isGreedy && len(evidence) > 0 {
			outRel = isReachable
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "greedy",
				Name:           "Greedy",
				Category:       "greedy",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 18. Topological Sort Detector
type TopologicalSortDetector struct{}

func (d *TopologicalSortDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		hasIndegree := false
		hasQueue := false
		hasGraph := false
		outRel := false

		fnLow := strings.ToLower(fn.Name)
		if strings.Contains(fnLow, "topo") || strings.Contains(fnLow, "kahn") {
			hasIndegree = true
		}

		for v := range fn.VarDefs {
			vLow := strings.ToLower(v)
			if strings.Contains(vLow, "indegree") || strings.Contains(vLow, "in_degree") ||
				strings.Contains(vLow, "deg") || strings.Contains(vLow, "incoming") ||
				strings.Contains(vLow, "prereq") {
				hasIndegree = true
				evidence = append(evidence, model.Evidence{
					Type:           "topological_indegree",
					Line:           fn.VarDefs[v].LineNo,
					Description:    fmt.Sprintf("Indegree vertex tracking array/map '%s'", v),
					Reachable:      isReachable,
					OutputRelevant: isReachable,
				})
			}
			if strings.Contains(vLow, "adj") || strings.Contains(vLow, "graph") {
				hasGraph = true
			}
			if strings.Contains(vLow, "topo") || strings.Contains(vLow, "order") {
				hasIndegree = true
			}
		}

		for _, arg := range fn.Args {
			argLow := strings.ToLower(arg)
			if strings.Contains(argLow, "prereq") || strings.Contains(argLow, "edge") || strings.Contains(argLow, "adj") {
				hasGraph = true
			}
		}

		for _, op := range fn.Operations {
			if op.Type == "queue_op" || op.Type == "list_pop" {
				hasQueue = true
			}
		}

		if (hasIndegree && (hasQueue || hasGraph)) || (hasGraph && hasQueue && len(evidence) > 0) {
			rel := isReachable
			if rel {
				outRel = true
			}
			if len(evidence) == 0 {
				evidence = append(evidence, model.Evidence{
					Type:           "topological_kahn_bfs",
					Line:           fn.LineNo,
					Description:    "Kahn algorithm topological dependency ordering",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "topological_sort",
				Name:           "Topological Sort",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "graphs",
				Name:           "Graphs",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RoleAuxiliary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 19. Dijkstra Detector
type DijkstraDetector struct{}

func (d *DijkstraDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		hasHeap := false
		hasDist := false
		outRel := false

		fnLow := strings.ToLower(fn.Name)
		if strings.Contains(fnLow, "dijkstra") || strings.Contains(fnLow, "shortest") {
			hasDist = true
		}

		for v := range fn.VarDefs {
			vLow := strings.ToLower(v)
			if strings.Contains(vLow, "dist") || strings.Contains(vLow, "cost") || strings.Contains(vLow, "distance") {
				hasDist = true
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, v)
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "dijkstra_distance_table",
					Line:           fn.VarDefs[v].LineNo,
					Description:    fmt.Sprintf("Distance/cost table '%s'", v),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
		}

		for _, op := range fn.Operations {
			if op.Type == "heap_call" || op.Type == "priority_queue" {
				hasHeap = true
			}
		}

		for _, call := range fn.Calls {
			if strings.Contains(call.Name, "heappop") || strings.Contains(call.Name, "heappush") {
				hasHeap = true
			}
		}

		if hasHeap && hasDist && len(evidence) > 0 {
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "dijkstra",
				Name:           "Dijkstra's Algorithm",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "graphs",
				Name:           "Graphs",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RoleAuxiliary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}

// 20. Union Find / DSU Detector
type UnionFindDetector struct{}

func (d *UnionFindDetector) Detect(
	functions []FunctionNode,
	reachableFuncs map[string]*FunctionNode,
	relevanceMap map[string]*FunctionRelevance,
) []model.DetectedConcept {
	var concepts []model.DetectedConcept

	for _, fn := range functions {
		isReachable := reachableFuncs[fn.Name] != nil
		var evidence []model.Evidence
		hasParent := false
		hasFind := false
		hasUnion := false
		outRel := false

		fnLow := strings.ToLower(fn.Name)
		if strings.Contains(fnLow, "union") || strings.Contains(fnLow, "dsu") || strings.Contains(fnLow, "find") {
			hasFind = true
		}

		for v := range fn.VarDefs {
			vLow := strings.ToLower(v)
			if strings.Contains(vLow, "parent") || strings.Contains(vLow, "root") || strings.Contains(vLow, "dsu") {
				hasParent = true
				rel := isReachable && IsVariableOutputRelevant(relevanceMap, fn.Name, v)
				if rel {
					outRel = true
				}
				evidence = append(evidence, model.Evidence{
					Type:           "union_find_parent_array",
					Line:           fn.VarDefs[v].LineNo,
					Description:    fmt.Sprintf("Disjoint set parent array/map '%s'", v),
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
			if strings.Contains(vLow, "find") {
				hasFind = true
			}
			if strings.Contains(vLow, "union") {
				hasUnion = true
			}
		}

		for _, call := range fn.Calls {
			cLow := strings.ToLower(call.Name)
			if strings.Contains(cLow, "find") {
				hasFind = true
			}
			if strings.Contains(cLow, "union") {
				hasUnion = true
			}
		}

		for _, other := range functions {
			if other.ParentName == fn.Name {
				oLow := strings.ToLower(other.Name)
				if strings.Contains(oLow, "find") {
					hasFind = true
				}
				if strings.Contains(oLow, "union") {
					hasUnion = true
				}
			}
		}

		if hasParent && (hasFind || hasUnion) {
			rel := isReachable
			if rel {
				outRel = true
			}
			if len(evidence) == 0 {
				evidence = append(evidence, model.Evidence{
					Type:           "disjoint_set_union",
					Line:           fn.LineNo,
					Description:    "Disjoint Set Union (DSU) operations",
					Reachable:      isReachable,
					OutputRelevant: rel,
				})
			}
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "union_find",
				Name:           "Union Find / Disjoint Set Union (DSU)",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RolePrimary,
				Status:         "DETECTED",
			})
			concepts = append(concepts, model.DetectedConcept{
				ConceptID:      "graphs",
				Name:           "Graphs",
				Category:       "graphs",
				Evidence:       evidence,
				Reachable:      isReachable,
				OutputRelevant: outRel,
				Confidence:     0.95,
				Role:           model.RoleAuxiliary,
				Status:         "DETECTED",
			})
		}
	}

	return concepts
}


