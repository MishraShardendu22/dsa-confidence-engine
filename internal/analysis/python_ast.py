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
            "if_guards": [],
            "_dict_vars": set(),
            "_list_vars": set(),
            "_set_vars": set()
        }
        for a in node.args.args:
            arg_lower = a.arg.lower()
            if any(term in arg_lower for term in ("num", "arr", "list", "seq", "val", "element")):
                func_info["_list_vars"].add(a.arg)

        self.functions.append(func_info)
        self.func_stack.append(func_info)
        prev_loop_depth = self.loop_depth
        self.loop_depth = 0
        self.generic_visit(node)
        self.loop_depth = prev_loop_depth
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
                obj = call_name.rsplit(".", 1)[0]
                self.current_func["operations"].append({
                    "type": "dict_get",
                    "lineno": node.lineno,
                    "var": obj
                })
                self.current_func["_dict_vars"].add(obj)
            elif call_name.endswith(".append"):
                obj = call_name.rsplit(".", 1)[0]
                self.current_func["operations"].append({
                    "type": "list_append",
                    "lineno": node.lineno,
                    "var": obj
                })
                self.current_func["_list_vars"].add(obj)
            elif call_name.endswith(".pop"):
                obj = call_name.rsplit(".", 1)[0]
                op_type = "dict_pop" if obj in self.current_func.get("_dict_vars", set()) else "list_pop"
                self.current_func["operations"].append({
                    "type": op_type,
                    "lineno": node.lineno,
                    "var": obj
                })
            elif call_name.endswith(".add"):
                obj = call_name.rsplit(".", 1)[0]
                self.current_func["operations"].append({
                    "type": "set_add",
                    "lineno": node.lineno,
                    "var": obj
                })
                self.current_func["_set_vars"].add(obj)
            elif call_name.endswith(".remove") or call_name.endswith(".discard"):
                obj = call_name.rsplit(".", 1)[0]
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

    def _record_assign_target(self, target, val_node, val_names, lineno):
        if not self.current_func:
            return

        if isinstance(target, ast.Name):
            var_name = target.id
            kind = "assign"
            if isinstance(val_node, ast.Dict):
                kind = "dict_alloc"
                self.current_func["operations"].append({
                    "type": "dict_alloc",
                    "lineno": lineno,
                    "var": var_name
                })
                self.current_func["_dict_vars"].add(var_name)
            elif isinstance(val_node, ast.Call) and self._get_name(val_node.func) in ("dict", "defaultdict", "Counter"):
                kind = "dict_alloc"
                self.current_func["operations"].append({
                    "type": "dict_alloc",
                    "lineno": lineno,
                    "var": var_name
                })
                self.current_func["_dict_vars"].add(var_name)
            elif isinstance(val_node, ast.Set):
                kind = "set_alloc"
                self.current_func["operations"].append({
                    "type": "set_alloc",
                    "lineno": lineno,
                    "var": var_name
                })
                self.current_func["_set_vars"].add(var_name)
            elif isinstance(val_node, ast.Call) and self._get_name(val_node.func) == "set":
                kind = "set_alloc"
                self.current_func["operations"].append({
                    "type": "set_alloc",
                    "lineno": lineno,
                    "var": var_name
                })
                self.current_func["_set_vars"].add(var_name)
            elif isinstance(val_node, ast.List):
                kind = "list_alloc"
                self.current_func["operations"].append({
                    "type": "list_alloc",
                    "lineno": lineno,
                    "var": var_name
                })
                self.current_func["_list_vars"].add(var_name)
            elif isinstance(val_node, ast.BinOp) and isinstance(val_node.op, ast.Mult) and (isinstance(val_node.left, ast.List) or isinstance(val_node.right, ast.List)):
                kind = "list_alloc"
                self.current_func["operations"].append({
                    "type": "list_alloc",
                    "lineno": lineno,
                    "var": var_name
                })
                self.current_func["_list_vars"].add(var_name)
            elif isinstance(val_node, ast.Call) and self._get_name(val_node.func) in ("sorted", "sort"):
                self.current_func["operations"].append({
                    "type": "sorting_call",
                    "lineno": lineno,
                    "target": self._get_name(val_node.func),
                    "var": var_name
                })
                self.current_func["_list_vars"].add(var_name)
            elif isinstance(val_node, ast.BinOp) and isinstance(val_node.op, (ast.FloorDiv, ast.RShift)):
                self.current_func["operations"].append({
                    "type": "bisection_calc",
                    "lineno": lineno,
                    "var": var_name
                })

            self.current_func["var_defs"][var_name] = {
                "lineno": lineno,
                "deps": val_names,
                "kind": kind
            }
        elif isinstance(target, (ast.Tuple, ast.List)):
            has_paired = isinstance(val_node, (ast.Tuple, ast.List)) and len(val_node.elts) == len(target.elts)
            for idx, elt in enumerate(target.elts):
                elt_val = val_node.elts[idx] if has_paired else val_node
                elt_deps = list(self._collect_names(elt_val)) if has_paired else val_names
                self._record_assign_target(elt, elt_val, elt_deps, lineno)
        elif isinstance(target, ast.Subscript):
            sub_var = self._get_name(target.value)
            key_vars = list(self._collect_names(target.slice))
            self.current_func["var_mutations"].append({
                "var": sub_var,
                "lineno": lineno,
                "key_deps": key_vars,
                "val_deps": val_names,
                "kind": "subscript_assign"
            })
            op_type = "list_write" if sub_var in self.current_func.get("_list_vars", set()) else ("dict_write" if sub_var in self.current_func.get("_dict_vars", set()) else "subscript_write")
            self.current_func["operations"].append({
                "type": op_type,
                "lineno": lineno,
                "var": sub_var
            })

    def visit_Assign(self, node):
        if self.current_func:
            val_names = list(self._collect_names(node.value))
            for target in node.targets:
                self._record_assign_target(target, node.value, val_names, node.lineno)
        self.generic_visit(node)

    def visit_AnnAssign(self, node):
        if self.current_func and node.value is not None:
            val_names = list(self._collect_names(node.value))
            self._record_assign_target(node.target, node.value, val_names, node.lineno)
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
                op_type = "list_write" if sub_var in self.current_func.get("_list_vars", set()) else ("dict_write" if sub_var in self.current_func.get("_dict_vars", set()) else "subscript_write")
                self.current_func["operations"].append({
                    "type": op_type,
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
                elif isinstance(node.op, ast.BitXor):
                    self.current_func["operations"].append({
                        "type": "bitwise_xor",
                        "lineno": node.lineno,
                        "var": var_name
                    })
                elif isinstance(node.op, ast.BitAnd):
                    self.current_func["operations"].append({
                        "type": "bitwise_and",
                        "lineno": node.lineno,
                        "var": var_name
                    })
                elif isinstance(node.op, ast.BitOr):
                    self.current_func["operations"].append({
                        "type": "bitwise_or",
                        "lineno": node.lineno,
                        "var": var_name
                    })
                elif isinstance(node.op, (ast.LShift, ast.RShift)):
                    self.current_func["operations"].append({
                        "type": "bitwise_shift",
                        "lineno": node.lineno,
                        "var": var_name
                    })
        self.generic_visit(node)

    def visit_BinOp(self, node):
        if self.current_func:
            op_type = None
            if isinstance(node.op, ast.BitXor):
                op_type = "bitwise_xor"
            elif isinstance(node.op, ast.BitAnd):
                op_type = "bitwise_and"
            elif isinstance(node.op, ast.BitOr):
                op_type = "bitwise_or"
            elif isinstance(node.op, ast.LShift):
                op_type = "bitwise_shift"
            elif isinstance(node.op, ast.RShift):
                op_type = "bitwise_shift"

            if op_type:
                self.current_func["operations"].append({
                    "type": op_type,
                    "lineno": node.lineno,
                    "var": ""
                })
        self.generic_visit(node)

    def visit_UnaryOp(self, node):
        if self.current_func and isinstance(node.op, ast.Invert):
            self.current_func["operations"].append({
                "type": "bitwise_not",
                "lineno": node.lineno,
                "var": ""
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
            if sub_var in self.current_func.get("_dict_vars", set()):
                self.current_func["operations"].append({
                    "type": "dict_read",
                    "lineno": node.lineno,
                    "var": sub_var
                })
            elif sub_var in self.current_func.get("_list_vars", set()):
                self.current_func["operations"].append({
                    "type": "list_read",
                    "lineno": node.lineno,
                    "var": sub_var
                })
            else:
                self.current_func["operations"].append({
                    "type": "subscript_read",
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
            has_return = any(isinstance(stmt, ast.Return) for stmt in ast.walk(node))
            self.current_func["if_guards"].append({
                "lineno": node.lineno,
                "cond_vars": cond_vars,
                "has_return": has_return
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
        for fn in extractor.functions:
            fn.pop("_dict_vars", None)
            fn.pop("_list_vars", None)
            fn.pop("_set_vars", None)
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
