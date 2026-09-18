package analysis

import "strings"

type FunctionNode struct {
	Name         string                `json:"name"`
	ParentName   string                `json:"parent_name"`
	LineNo       int                   `json:"lineno"`
	EndLineNo    int                   `json:"end_lineno"`
	Args         []string              `json:"args"`
	Calls        []CallNode            `json:"calls"`
	Returns      []ReturnNode          `json:"returns"`
	VarDefs      map[string]VarDefNode `json:"var_defs"`
	VarMutations []VarMutationNode     `json:"var_mutations"`
	VarReads     []VarReadNode         `json:"var_reads"`
	Operations   []OperationNode       `json:"operations"`
	WhileLoops   []WhileLoopNode       `json:"while_loops"`
	IfGuards     []IfGuardNode         `json:"if_guards"`
	Reachable    bool                  `json:"-"`
}

type CallNode struct {
	Name    string   `json:"name"`
	LineNo  int      `json:"lineno"`
	ArgVars []string `json:"arg_vars"`
}

type ReturnNode struct {
	LineNo int      `json:"lineno"`
	Vars   []string `json:"vars"`
	Raw    string   `json:"raw"`
}

type VarDefNode struct {
	LineNo int      `json:"lineno"`
	Deps   []string `json:"deps"`
	Kind   string   `json:"kind"`
}

type VarMutationNode struct {
	Var     string   `json:"var"`
	LineNo  int      `json:"lineno"`
	KeyDeps []string `json:"key_deps"`
	ValDeps []string `json:"val_deps"`
	Kind    string   `json:"kind"`
}

type VarReadNode struct {
	Var     string   `json:"var"`
	LineNo  int      `json:"lineno"`
	KeyVars []string `json:"key_vars"`
	Kind    string   `json:"kind"`
}

type OperationNode struct {
	Type       string   `json:"type"`
	LineNo     int      `json:"lineno"`
	Var        string   `json:"var,omitempty"`
	Op         string   `json:"op,omitempty"`
	Target     string   `json:"target,omitempty"`
	IterVars   []string `json:"iter_vars,omitempty"`
	TargetVars []string `json:"target_vars,omitempty"`
}

type WhileLoopNode struct {
	LineNo   int      `json:"lineno"`
	CondVars []string `json:"cond_vars"`
	Expr     string   `json:"expr"`
}

type IfGuardNode struct {
	LineNo    int      `json:"lineno"`
	CondVars  []string `json:"cond_vars"`
	HasReturn bool     `json:"has_return,omitempty"`
}

func ComputeReachableFunctions(functions []FunctionNode, contract AnalysisContract) map[string]*FunctionNode {
	funcNodes := make([]FunctionNode, len(functions))
	copy(funcNodes, functions)

	funcMap := make(map[string]*FunctionNode, len(funcNodes))
	for i := range funcNodes {
		funcNodes[i].Reachable = false
		funcMap[funcNodes[i].Name] = &funcNodes[i]
	}

	entrypoint := contract.Entrypoint
	if entrypoint == "" {
		entrypoint = "solve"
	}

	reachable := make(map[string]*FunctionNode)
	entry, exists := funcMap[entrypoint]
	if !exists {
		for _, alias := range contract.EntrypointAliases {
			if e, ok := funcMap[alias]; ok {
				entry = e
				exists = true
				break
			}
		}
	}

	if !exists {
		normProb := strings.ToLower(strings.ReplaceAll(contract.ProblemID, "_", ""))
		normEntry := strings.ToLower(strings.ReplaceAll(entrypoint, "_", ""))
		for i := range funcNodes {
			fnNorm := strings.ToLower(strings.ReplaceAll(funcNodes[i].Name, "_", ""))
			if fnNorm == normProb || fnNorm == normEntry || fnNorm == "solve" || fnNorm == "solution" {
				entry = &funcNodes[i]
				exists = true
				break
			}
		}
	}

	if !exists {
		var topLevel []*FunctionNode
		for i := range funcNodes {
			if funcNodes[i].ParentName == "" {
				topLevel = append(topLevel, &funcNodes[i])
			}
		}
		if len(topLevel) == 1 {
			entry = topLevel[0]
			exists = true
		}
	}

	if !exists {
		return reachable
	}

	queue := []*FunctionNode{entry}
	entry.Reachable = true
	reachable[entry.Name] = entry

	for head := 0; head < len(queue); head++ {
		curr := queue[head]

		// 1. Direct calls (including calls to reachable helpers)
		for _, call := range curr.Calls {
			if target, found := funcMap[call.Name]; found && !target.Reachable {
				target.Reachable = true
				reachable[target.Name] = target
				queue = append(queue, target)
			}
		}
	}

	return reachable
}
