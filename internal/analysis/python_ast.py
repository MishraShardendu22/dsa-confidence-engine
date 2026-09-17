import ast
import sys
import json

class ASTExtractor(ast.NodeVisitor):
    def __init__(self):
        self.functions = []
        self.global_calls = []
        self.func_stack = []
        self.loop_depth = 0

    @property
    def current_func(self):
        return self.func_stack[-1] if self.func_stack else None

    def visit_FunctionDef(self, node):
        parent_name = self.current_func["name"] if self.current_func else ""
        func_info = {
            "name": node.name,
            "parent_name": parent_name,
            "lineno": node.lineno,
            "end_lineno": getattr(node, "end_lineno", node.lineno),
            "args": [a.arg for a in node.args.args],
            "calls": [],
            "returns": [],
            "var_defs": {},
            "var_mutations": [],
            "var_reads": [],
            "operations": [],
            "while_loops": [],
            "if_guards": []
        }
        self.functions.append(func_info)
        self.func_stack.append(func_info)
        self.generic_visit(node)
        self.func_stack.pop()

    def visit_AsyncFunctionDef(self, node):
        self.visit_FunctionDef(node)

    def _get_name(self, node):
        if isinstance(node, ast.Name):
            return node.id
        elif isinstance(node, ast.Attribute):
            val = self._get_name(node.value)
            return f"{val}.{node.attr}" if val else node.attr
        return ""

    def _collect_names(self, node):
        names = set()
        if node is None:
            return names
        for child in ast.walk(node):
            if isinstance(child, ast.Name):
                names.add(child.id)
        return names

    def visit_Call(self, node):
        call_name = self._get_name(node.func)
        args_names = []
        for arg in node.args:
            args_names.extend(list(self._collect_names(arg)))

        call_data = {
            "name": call_name,
            "lineno": node.lineno,
            "arg_vars": args_names
        }

        if self.current_func:
            self.current_func["calls"].append(call_data)

            if call_name.endswith(".sort"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "sorting_call",
                    "lineno": node.lineno,
                    "target": call_name,
                    "var": obj
                })
            elif call_name in ("sort", "sorted"):
                has_op = any(op.get("type") == "sorting_call" and op.get("lineno") == node.lineno for op in self.current_func["operations"])
                if not has_op:
                    self.current_func["operations"].append({
                        "type": "sorting_call",
                        "lineno": node.lineno,
                        "target": call_name,
                        "var": ""
                    })
            elif "heapq" in call_name or call_name in ("heappush", "heappop", "heapify"):
                self.current_func["operations"].append({
                    "type": "heap_op",
                    "lineno": node.lineno,
                    "op": call_name
                })
            elif call_name.endswith(".popleft") or "deque" in call_name:
                self.current_func["operations"].append({
                    "type": "queue_op",
                    "lineno": node.lineno,
                    "op": call_name
                })
            elif call_name.endswith(".get"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "dict_get",
                    "lineno": node.lineno,
                    "var": obj
                })
            elif call_name.endswith(".append"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "list_append",
                    "lineno": node.lineno,
                    "var": obj
                })
            elif call_name.endswith(".pop"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "list_pop",
                    "lineno": node.lineno,
                    "var": obj
                })
            elif call_name.endswith(".add"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "set_add",
                    "lineno": node.lineno,
                    "var": obj
                })
            elif call_name.endswith(".remove") or call_name.endswith(".discard"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "set_remove",
                    "lineno": node.lineno,
                    "var": obj
                })
            elif call_name == "set":
                self.current_func["operations"].append({
                    "type": "set_alloc",
                    "lineno": node.lineno,
                    "var": "<set>"
                })
            elif call_name in ("Counter", "collections.Counter") or call_name.endswith(".Counter"):
                self.current_func["operations"].append({
                    "type": "counter_call",
                    "lineno": node.lineno,
                    "var": call_name
                })
            elif call_name.endswith(".count"):
                obj = call_name.split(".")[0]
                self.current_func["operations"].append({
                    "type": "count_call",
                    "lineno": node.lineno,
                    "var": obj
                })
        else:
            self.global_calls.append(call_data)

        self.generic_visit(node)

    def visit_Return(self, node):
        if self.current_func:
            ret_vars = list(self._collect_names(node.value))
            raw_expr = ""
            try:
                raw_expr = ast.unparse(node.value) if node.value else ""
            except Exception:
                pass

            if isinstance(node.value, ast.List):
                self.current_func["operations"].append({
                    "type": "list_alloc",
                    "lineno": node.lineno,
                    "var": "<return>"
                })

            self.current_func["returns"].append({
                "lineno": node.lineno,
                "vars": ret_vars,
                "raw": raw_expr
            })
        self.generic_visit(node)

    def visit_Assign(self, node):
        if self.current_func:
            val_names = list(self._collect_names(node.value))
            for target in node.targets:
                if isinstance(target, ast.Name):
                    var_name = target.id
                    kind = "assign"
                    if isinstance(node.value, ast.Dict):
                        kind = "dict_alloc"
                        self.current_func["operations"].append({
                            "type": "dict_alloc",
                            "lineno": node.lineno,
                            "var": var_name
                        })
                    elif isinstance(node.value, ast.Call) and self._get_name(node.value.func) in ("dict", "defaultdict", "Counter"):
                        kind = "dict_alloc"
                        self.current_func["operations"].append({
                            "type": "dict_alloc",
                            "lineno": node.lineno,
                            "var": var_name
                        })
                    elif isinstance(node.value, ast.Set):
                        kind = "set_alloc"
                        self.current_func["operations"].append({
                            "type": "set_alloc",
                            "lineno": node.lineno,
                            "var": var_name
                        })
                    elif isinstance(node.value, ast.Call) and self._get_name(node.value.func) == "set":
                        kind = "set_alloc"
                        self.current_func["operations"].append({
                            "type": "set_alloc",
                            "lineno": node.lineno,
                            "var": var_name
                        })
                    elif isinstance(node.value, ast.List):
                        kind = "list_alloc"
                        self.current_func["operations"].append({
                            "type": "list_alloc",
                            "lineno": node.lineno,
                            "var": var_name
                        })
                    elif isinstance(node.value, ast.Call) and self._get_name(node.value.func) in ("sorted", "sort"):
                        self.current_func["operations"].append({
                            "type": "sorting_call",
                            "lineno": node.lineno,
                            "target": self._get_name(node.value.func),
                            "var": var_name
                        })

                    self.current_func["var_defs"][var_name] = {
                        "lineno": node.lineno,
                        "deps": val_names,
                        "kind": kind
                    }
                elif isinstance(target, ast.Subscript):
                    sub_var = self._get_name(target.value)
                    key_vars = list(self._collect_names(target.slice))
                    self.current_func["var_mutations"].append({
                        "var": sub_var,
                        "lineno": node.lineno,
                        "key_deps": key_vars,
                        "val_deps": val_names,
                        "kind": "subscript_assign"
                    })
                    self.current_func["operations"].append({
                        "type": "dict_write",
                        "lineno": node.lineno,
                        "var": sub_var
                    })

        self.generic_visit(node)

    def visit_DictComp(self, node):
        if self.current_func:
            self.current_func["operations"].append({
                "type": "dict_comp",
                "lineno": node.lineno,
                "var": "<dict_comp>"
            })
        self.generic_visit(node)

    def visit_AugAssign(self, node):
        if self.current_func:
            val_names = list(self._collect_names(node.value))
            if isinstance(node.target, ast.Subscript):
                sub_var = self._get_name(node.target.value)
                key_vars = list(self._collect_names(node.target.slice))
                self.current_func["var_mutations"].append({
                    "var": sub_var,
                    "lineno": node.lineno,
                    "key_deps": key_vars,
                    "val_deps": val_names,
                    "kind": "subscript_aug_assign"
                })
                self.current_func["operations"].append({
                    "type": "dict_write",
                    "lineno": node.lineno,
                    "var": sub_var
                })
            elif isinstance(node.target, ast.Name):
                var_name = node.target.id
                val_names.append(var_name)
                self.current_func["var_defs"][var_name] = {
                    "lineno": node.lineno,
                    "deps": val_names,
                    "kind": "aug_assign"
                }
                # Track pointer increments and decrements
                if isinstance(node.op, ast.Add):
                    self.current_func["operations"].append({
                        "type": "pointer_step",
                        "lineno": node.lineno,
                        "var": var_name,
                        "op": "inc"
                    })
                elif isinstance(node.op, ast.Sub):
                    self.current_func["operations"].append({
                        "type": "pointer_step",
                        "lineno": node.lineno,
                        "var": var_name,
                        "op": "dec"
                    })
        self.generic_visit(node)

    def visit_Subscript(self, node):
        if self.current_func and isinstance(node.ctx, ast.Load):
            sub_var = self._get_name(node.value)
            key_vars = list(self._collect_names(node.slice))
            self.current_func["var_reads"].append({
                "var": sub_var,
                "lineno": node.lineno,
                "key_vars": key_vars,
                "kind": "subscript_read"
            })
            self.current_func["operations"].append({
                "type": "dict_read",
                "lineno": node.lineno,
                "var": sub_var
            })
        self.generic_visit(node)

    def visit_Compare(self, node):
        if self.current_func:
            for op, comparator in zip(node.ops, node.comparators):
                if isinstance(op, (ast.In, ast.NotIn)):
                    right_var = self._get_name(comparator)
                    self.current_func["operations"].append({
                        "type": "membership_check",
                        "lineno": node.lineno,
                        "var": right_var
                    })
        self.generic_visit(node)

    def visit_While(self, node):
        if self.current_func:
            self.loop_depth += 1
            if self.loop_depth >= 2:
                self.current_func["operations"].append({
                    "type": "nested_loop",
                    "lineno": node.lineno
                })
            cond_vars = list(self._collect_names(node.test))
            cond_expr = ""
            try:
                cond_expr = ast.unparse(node.test)
            except Exception:
                pass

            self.current_func["while_loops"].append({
                "lineno": node.lineno,
                "cond_vars": cond_vars,
                "expr": cond_expr
            })
            self.generic_visit(node)
            self.loop_depth -= 1
        else:
            self.generic_visit(node)

    def visit_If(self, node):
        if self.current_func:
            cond_vars = list(self._collect_names(node.test))
            self.current_func["if_guards"].append({
                "lineno": node.lineno,
                "cond_vars": cond_vars
            })
        self.generic_visit(node)

    def visit_For(self, node):
        if self.current_func:
            self.loop_depth += 1
            if self.loop_depth >= 2:
                self.current_func["operations"].append({
                    "type": "nested_loop",
                    "lineno": node.lineno
                })
            target_names = list(self._collect_names(node.target))
            iter_names = list(self._collect_names(node.iter))
            self.current_func["operations"].append({
                "type": "for_loop",
                "lineno": node.lineno,
                "target_vars": target_names,
                "iter_vars": iter_names
            })
            for tvar in target_names:
                self.current_func["var_defs"][tvar] = {
                    "lineno": node.lineno,
                    "deps": iter_names,
                    "kind": "for_target"
                }
            self.generic_visit(node)
            self.loop_depth -= 1
        else:
            self.generic_visit(node)

def main():
    try:
        source = sys.stdin.read()
        tree = ast.parse(source)
        extractor = ASTExtractor()
        extractor.visit(tree)
        print(json.dumps({
            "status": "SUCCESS",
            "functions": extractor.functions,
            "global_calls": extractor.global_calls
        }))
    except Exception as e:
        print(json.dumps({
            "status": "ERROR",
            "error": str(e)
        }))

if __name__ == "__main__":
    main()
