#!/usr/bin/env python3
"""
analyze_approaches.py - Autonomous Algorithmic Approach Discovery & Analysis Tool

Analyzes the DSA problem catalog (3,692 questions) to discover, categorize,
and enrich multi-approach strategies, paradigm families, and complexity profiles.
"""

import argparse
import json
import os
import re
import sys
from collections import Counter, defaultdict

# Approach catalog metadata mapping approach ID -> metadata
APPROACH_METADATA = {
    # 1. Array, Hash Table & Two Pointers
    "hashmap_lookup": {
        "name": "Single-Pass Hash Map Lookup",
        "paradigm": "Hashing & Arrays",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["hashmap", "arrays"]
    },
    "sorting_and_two_pointers": {
        "name": "Sorting + Inward Two Pointers",
        "paradigm": "Two Pointers & Sorting",
        "time_complexity": "O(N log N)",
        "space_complexity": "O(1)",
        "concepts": ["sorting", "two_pointers"]
    },
    "binary_search_complement": {
        "name": "Sorting + Binary Search Complement",
        "paradigm": "Binary Search",
        "time_complexity": "O(N log N)",
        "space_complexity": "O(1)",
        "concepts": ["sorting", "binary_search"]
    },
    "hashmap_frequency_lookup": {
        "name": "HashMap Frequency Tallying",
        "paradigm": "Hashing & Arrays",
        "time_complexity": "O(N)",
        "space_complexity": "O(K)",
        "concepts": ["hashmap", "frequency_count"]
    },
    "hashset_membership_check": {
        "name": "HashSet Fast Membership Validation",
        "paradigm": "Hashing & Arrays",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["hashset"]
    },

    # 2. Dynamic Programming
    "dp_tabulation_bottom_up": {
        "name": "Bottom-Up DP Tabulation",
        "paradigm": "Dynamic Programming",
        "time_complexity": "O(N * S)",
        "space_complexity": "O(N * S)",
        "concepts": ["dynamic_programming", "tabulation"]
    },
    "dp_memoization_top_down": {
        "name": "Top-Down DP with Memoization",
        "paradigm": "Dynamic Programming",
        "time_complexity": "O(N * S)",
        "space_complexity": "O(N * S)",
        "concepts": ["dynamic_programming", "memoization"]
    },
    "dp_space_optimized_rolling": {
        "name": "Space-Optimized DP Rolling Variables",
        "paradigm": "Dynamic Programming",
        "time_complexity": "O(N * S)",
        "space_complexity": "O(S)",
        "concepts": ["dynamic_programming", "tabulation"]
    },
    "binary_search_patience_sorting": {
        "name": "Patience Sorting / Binary Search (LIS)",
        "paradigm": "Binary Search & DP",
        "time_complexity": "O(N log N)",
        "space_complexity": "O(N)",
        "concepts": ["binary_search", "dynamic_programming"]
    },
    "kadane_algorithm": {
        "name": "Kadane's Algorithm Maximum Contiguous Subarray",
        "paradigm": "Dynamic Programming",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["kadane", "dynamic_programming"]
    },

    # 3. Graphs & Trees
    "dfs_recursive_traversal": {
        "name": "Depth-First Search (Recursive)",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O(V + E)",
        "space_complexity": "O(V)",
        "concepts": ["dfs", "recursion"]
    },
    "bfs_queue_level_order": {
        "name": "Breadth-First Search (Queue / Level-Order)",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O(V + E)",
        "space_complexity": "O(V)",
        "concepts": ["bfs", "queue"]
    },
    "union_find_disjoint_set": {
        "name": "Disjoint Set Union (Union-Find with Path Compression)",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O(E * alpha(V))",
        "space_complexity": "O(V)",
        "concepts": ["union_find"]
    },
    "kahn_topological_sort_indegree": {
        "name": "Kahn's Algorithm (BFS In-Degree Topological Sort)",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O(V + E)",
        "space_complexity": "O(V)",
        "concepts": ["topological_sort", "bfs"]
    },
    "dfs_cycle_detection_postorder": {
        "name": "DFS Postorder / 3-Color Cycle Detection",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O(V + E)",
        "space_complexity": "O(V)",
        "concepts": ["dfs", "topological_sort"]
    },
    "dijkstra_priority_queue": {
        "name": "Dijkstra's Algorithm (Min-Heap Shortest Path)",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O((V + E) log V)",
        "space_complexity": "O(V)",
        "concepts": ["dijkstra", "heap"]
    },
    "bfs_unweighted_shortest_path": {
        "name": "Unweighted Shortest Path BFS",
        "paradigm": "Graphs & Trees",
        "time_complexity": "O(V + E)",
        "space_complexity": "O(V)",
        "concepts": ["bfs", "queue"]
    },

    # 4. Binary Search
    "binary_search_iterative": {
        "name": "Iterative Binary Search",
        "paradigm": "Binary Search",
        "time_complexity": "O(log N)",
        "space_complexity": "O(1)",
        "concepts": ["binary_search"]
    },
    "binary_search_recursive": {
        "name": "Recursive Bisection Search",
        "paradigm": "Binary Search",
        "time_complexity": "O(log N)",
        "space_complexity": "O(log N)",
        "concepts": ["binary_search", "recursion"]
    },
    "binary_search_on_answer_predicate": {
        "name": "Binary Search on Monotonic Answer Space",
        "paradigm": "Binary Search",
        "time_complexity": "O(N log(High - Low))",
        "space_complexity": "O(1)",
        "concepts": ["binary_search"]
    },
    "binary_search_rotated_pivot": {
        "name": "Rotated Sorted Array Pivot Binary Search",
        "paradigm": "Binary Search",
        "time_complexity": "O(log N)",
        "space_complexity": "O(1)",
        "concepts": ["binary_search"]
    },

    # 5. Two Pointers & Sliding Window
    "sliding_window_dynamic_frequency_map": {
        "name": "Dynamic Sliding Window with Frequency Map",
        "paradigm": "Sliding Window",
        "time_complexity": "O(N)",
        "space_complexity": "O(K)",
        "concepts": ["sliding_window", "hashmap"]
    },
    "sliding_window_fixed_size": {
        "name": "Fixed-Size Sliding Window",
        "paradigm": "Sliding Window",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["sliding_window"]
    },
    "two_pointers_opposite_ends": {
        "name": "Opposite Ends Inward Pointers",
        "paradigm": "Two Pointers",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["two_pointers"]
    },
    "two_pointers_fast_slow": {
        "name": "Fast & Slow Pointers (Floyd's Cycle Finding)",
        "paradigm": "Two Pointers",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["two_pointers"]
    },

    # 6. Monotonic Stack & Stack
    "monotonic_stack_next_greater": {
        "name": "Monotonic Stack (Next Greater/Smaller Element)",
        "paradigm": "Stack & Monotonic",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["monotonic_stack", "stack"]
    },
    "two_pointers_boundary_sweep": {
        "name": "Two-Pointer Boundary Sweep (Trapping / Boundaries)",
        "paradigm": "Two Pointers",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["two_pointers"]
    },
    "stack_bracket_matching": {
        "name": "LIFO Stack Matching & Parsing",
        "paradigm": "Stack & Monotonic",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["stack"]
    },
    "stack_simulation": {
        "name": "Stack State Simulation",
        "paradigm": "Stack & Monotonic",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["stack"]
    },

    # 7. Heap & Top-K Selection
    "min_heap_k_elements": {
        "name": "Min-Heap of Size K (Top-K Streaming)",
        "paradigm": "Heap & Priority Queue",
        "time_complexity": "O(N log K)",
        "space_complexity": "O(K)",
        "concepts": ["heap"]
    },
    "max_heap_streaming": {
        "name": "Max-Heap Streaming Extraction",
        "paradigm": "Heap & Priority Queue",
        "time_complexity": "O(N + K log N)",
        "space_complexity": "O(N)",
        "concepts": ["heap"]
    },
    "quickselect_hoare_partition": {
        "name": "QuickSelect (Hoare In-Place Partitioning)",
        "paradigm": "Divide & Conquer",
        "time_complexity": "O(N) avg, O(N^2) worst",
        "space_complexity": "O(1)",
        "concepts": ["quickselect", "sorting"]
    },
    "bucket_sort_frequency": {
        "name": "Bucket Sort by Frequency Distribution",
        "paradigm": "Sorting & Hashing",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["sorting", "frequency_count"]
    },
    "two_heaps_median_tracking": {
        "name": "Two Heaps Continuous Median Tracking",
        "paradigm": "Heap & Priority Queue",
        "time_complexity": "O(log N) insert, O(1) find",
        "space_complexity": "O(N)",
        "concepts": ["heap"]
    },

    # 8. Bit Manipulation
    "bit_manipulation_xor_cancellation": {
        "name": "Bitwise XOR Invariant Cancellation",
        "paradigm": "Bit Manipulation",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["bit_manipulation"]
    },
    "bit_manipulation_mask_and_shift": {
        "name": "Bitwise Masking and Shifting",
        "paradigm": "Bit Manipulation",
        "time_complexity": "O(1) to O(W)",
        "space_complexity": "O(1)",
        "concepts": ["bit_manipulation"]
    },
    "bit_manipulation_brian_kernighan": {
        "name": "Brian Kernighan's Bit Trick (n & (n - 1))",
        "paradigm": "Bit Manipulation",
        "time_complexity": "O(set_bits)",
        "space_complexity": "O(1)",
        "concepts": ["bit_manipulation"]
    },

    # 9. Backtracking & Combinatorics
    "backtracking_dfs_state_restoration": {
        "name": "Backtracking DFS with State Restoration",
        "paradigm": "Backtracking",
        "time_complexity": "O(K^N)",
        "space_complexity": "O(N)",
        "concepts": ["backtracking", "recursion"]
    },
    "cascading_iterative_subsets": {
        "name": "Cascading Iterative Combination Generation",
        "paradigm": "Backtracking & Iteration",
        "time_complexity": "O(2^N)",
        "space_complexity": "O(2^N)",
        "concepts": ["backtracking", "arrays"]
    },
    "bitmask_subset_enumeration": {
        "name": "Bitmask Binary Subset Enumeration",
        "paradigm": "Bit Manipulation & Backtracking",
        "time_complexity": "O(N * 2^N)",
        "space_complexity": "O(1)",
        "concepts": ["bit_manipulation", "backtracking"]
    },

    # 10. Trie
    "trie_nested_hashmap": {
        "name": "Prefix Trie with Nested Hash Maps",
        "paradigm": "Trie & Trees",
        "time_complexity": "O(L) per operation",
        "space_complexity": "O(Total Chars)",
        "concepts": ["trie", "hashmap"]
    },
    "trie_fixed_array_children": {
        "name": "Prefix Trie with Fixed Alphabet Array (Size 26)",
        "paradigm": "Trie & Trees",
        "time_complexity": "O(L) per operation",
        "space_complexity": "O(26 * Total Chars)",
        "concepts": ["trie", "arrays"]
    },
    "hashset_prefix_lookup": {
        "name": "HashSet Substring Prefix Table",
        "paradigm": "Hashing",
        "time_complexity": "O(L^2)",
        "space_complexity": "O(L^2)",
        "concepts": ["hashset"]
    },

    # 11. Prefix Sum & Difference Array
    "prefix_sum_hashmap_lookup": {
        "name": "Prefix Sum Cumulative Balance + HashMap",
        "paradigm": "Prefix Sum & Hashing",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["prefix_sum", "hashmap"]
    },
    "prefix_sum_array_range_query": {
        "name": "Static 1D/2D Prefix Sum Array Queries",
        "paradigm": "Prefix Sum",
        "time_complexity": "O(1) query, O(N) prep",
        "space_complexity": "O(N)",
        "concepts": ["prefix_sum", "arrays"]
    },
    "difference_array_range_update": {
        "name": "Difference Array Sweep Line for Range Updates",
        "paradigm": "Prefix Sum & Arrays",
        "time_complexity": "O(1) update, O(N) reconstruction",
        "space_complexity": "O(N)",
        "concepts": ["difference_array", "prefix_sum"]
    },

    # 12. Greedy
    "greedy_sort_and_scan": {
        "name": "Greedy Sorting + Single Linear Scan",
        "paradigm": "Greedy",
        "time_complexity": "O(N log N)",
        "space_complexity": "O(1)",
        "concepts": ["greedy", "sorting"]
    },
    "greedy_local_optimal_choice": {
        "name": "Local Optimal Invariant Choice",
        "paradigm": "Greedy",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["greedy"]
    },

    # 13. Math & Number Theory
    "mathematical_formula_closed_form": {
        "name": "Closed-Form Mathematical Formula",
        "paradigm": "Math & Number Theory",
        "time_complexity": "O(1)",
        "space_complexity": "O(1)",
        "concepts": ["math"]
    },
    "iterative_simulation": {
        "name": "Iterative State-Step Simulation",
        "paradigm": "Simulation",
        "time_complexity": "O(Steps)",
        "space_complexity": "O(1)",
        "concepts": ["simulation", "arrays"]
    },
    "euclidean_gcd_algorithm": {
        "name": "Euclidean Greatest Common Divisor Algorithm",
        "paradigm": "Math & Number Theory",
        "time_complexity": "O(log(min(A, B)))",
        "space_complexity": "O(1)",
        "concepts": ["math"]
    },
    "sieve_of_eratosthenes": {
        "name": "Sieve of Eratosthenes Prime Generation",
        "paradigm": "Math & Number Theory",
        "time_complexity": "O(N log log N)",
        "space_complexity": "O(N)",
        "concepts": ["math"]
    },

    # 14. Linked List
    "dummy_head_pointer_iteration": {
        "name": "Dummy Sentinel Head Pointer Iteration",
        "paradigm": "Linked List",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["linked_list"]
    },
    "fast_slow_two_pointers": {
        "name": "Fast & Slow Runner Pointers",
        "paradigm": "Linked List & Two Pointers",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["linked_list", "two_pointers"]
    },
    "iterative_in_place_reversal": {
        "name": "Iterative In-Place Pointer Reversal",
        "paradigm": "Linked List",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["linked_list"]
    },

    # 15. Strings
    "string_builder_iteration": {
        "name": "Linear Scan and String Construction",
        "paradigm": "Strings",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["strings"]
    },
    "frequency_map_counting": {
        "name": "Character Frequency Counting",
        "paradigm": "Strings & Hashing",
        "time_complexity": "O(N)",
        "space_complexity": "O(Sigma)",
        "concepts": ["strings", "frequency_count"]
    },
    "two_pointers_palindrome_check": {
        "name": "Two Pointers Symmetry / Palindrome Check",
        "paradigm": "Strings & Two Pointers",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["strings", "two_pointers"]
    },

    # 16. Simulation & Sorting
    "iterative_state_simulation": {
        "name": "Explicit Rule State Machine Simulation",
        "paradigm": "Simulation",
        "time_complexity": "O(T)",
        "space_complexity": "O(State)",
        "concepts": ["simulation"]
    },
    "brute_force_enumeration": {
        "name": "Exhaustive Candidate Enumeration",
        "paradigm": "Brute Force",
        "time_complexity": "O(N^2)",
        "space_complexity": "O(1)",
        "concepts": ["brute_force"]
    },
    "custom_comparator_sort": {
        "name": "Sorting with Custom Comparator / Key Lambda",
        "paradigm": "Sorting",
        "time_complexity": "O(N log N)",
        "space_complexity": "O(N)",
        "concepts": ["sorting", "custom_sort"]
    },
    "two_pointers_after_sorting": {
        "name": "Presort Followed by Linear Pointer Scan",
        "paradigm": "Sorting & Two Pointers",
        "time_complexity": "O(N log N)",
        "space_complexity": "O(1)",
        "concepts": ["sorting", "two_pointers"]
    },

    # Fallback
    "iterative_optimal_solution": {
        "name": "Iterative Direct Solution",
        "paradigm": "Arrays & Iteration",
        "time_complexity": "O(N)",
        "space_complexity": "O(1)",
        "concepts": ["arrays"]
    },
    "recursive_alternative_solution": {
        "name": "Recursive Divide-and-Conquer Alternative",
        "paradigm": "Recursion",
        "time_complexity": "O(N)",
        "space_complexity": "O(N)",
        "concepts": ["recursion"]
    }
}


def deduce_problem_approaches(problem):
    """
    Autonomously analyzes a problem's topic tags, title, description, and difficulty
    to deduce the set of accepted algorithmic strategies and approaches.
    """
    existing = problem.get("accepted_strategies", [])
    if existing and len(existing) >= 2 and any(not s.endswith("_approach") for s in existing):
        return existing

    tags = set(t.replace("_", "-").lower() for t in problem.get("topic_tags", []))
    title = problem.get("title", "").lower()
    approaches = []

    # If handcrafted had 1 strategy, preserve it as primary
    if existing and len(existing) == 1 and not existing[0].endswith("_approach"):
        approaches.append(existing[0])

    # 1. Array & Hash-Table & Search
    if ("array" in tags and "hash-table" in tags) or "sum" in title:
        approaches.extend(["hashmap_lookup", "sorting_and_two_pointers"])
        if "binary-search" in tags:
            approaches.append("binary_search_complement")
    elif "hash-table" in tags:
        approaches.extend(["hashmap_frequency_lookup", "hashset_membership_check"])

    # 2. Dynamic Programming
    if "dynamic-programming" in tags or "dynamic_programming" in tags:
        approaches.extend(["dp_tabulation_bottom_up", "dp_memoization_top_down"])
        if "space-optimization" in tags or "rolling" in title:
            approaches.append("dp_space_optimized_rolling")
        if "longest-increasing-subsequence" in tags or "increasing subsequence" in title:
            approaches.append("binary_search_patience_sorting")
        if "maximum" in title and "subarray" in title:
            approaches.append("kadane_algorithm")

    # 3. Graphs and Trees
    if "depth-first-search" in tags or "breadth-first-search" in tags or "graph" in tags or "tree" in tags or "binary-tree" in tags:
        approaches.extend(["dfs_recursive_traversal", "bfs_queue_level_order"])
        if "union-find" in tags or "connected" in title or "islands" in title:
            approaches.append("union_find_disjoint_set")
        if "topological-sort" in tags or "course" in title:
            approaches.extend(["kahn_topological_sort_indegree", "dfs_cycle_detection_postorder"])
        if "shortest-path" in tags or "dijkstra" in tags:
            approaches.extend(["dijkstra_priority_queue", "bfs_unweighted_shortest_path"])

    # 4. Binary Search
    if "binary-search" in tags:
        approaches.extend(["binary_search_iterative", "binary_search_recursive"])
        if any(w in title for w in ["eating", "capacity", "ship", "split", "minimum", "smallest", "kth"]):
            approaches.append("binary_search_on_answer_predicate")
        if "rotated" in title:
            approaches.append("binary_search_rotated_pivot")

    # 5. Two Pointers & Sliding Window
    if "two-pointers" in tags or "sliding-window" in tags:
        if "sliding-window" in tags or any(w in title for w in ["substring", "window", "consecutive", "subarray"]):
            approaches.extend(["sliding_window_dynamic_frequency_map", "sliding_window_fixed_size"])
        if "two-pointers" in tags:
            approaches.extend(["two_pointers_opposite_ends", "two_pointers_fast_slow"])

    # 6. Monotonic Stack & Stack
    if "stack" in tags or "monotonic-stack" in tags:
        if "monotonic-stack" in tags or any(w in title for w in ["next", "greater", "daily", "temperature", "histogram", "rain", "water"]):
            approaches.extend(["monotonic_stack_next_greater", "two_pointers_boundary_sweep"])
        else:
            approaches.extend(["stack_bracket_matching", "stack_simulation"])

    # 7. Heap & Top-K
    if "heap-priority-queue" in tags:
        approaches.extend(["min_heap_k_elements", "max_heap_streaming"])
        if any(w in title for w in ["kth", "top", "frequent"]):
            approaches.extend(["quickselect_hoare_partition", "bucket_sort_frequency"])
        if "median" in title:
            approaches.append("two_heaps_median_tracking")

    # 8. Bit Manipulation
    if "bit-manipulation" in tags or "bitmask" in tags:
        approaches.extend(["bit_manipulation_xor_cancellation", "bit_manipulation_mask_and_shift", "bit_manipulation_brian_kernighan"])

    # 9. Backtracking & Combinatorics
    if "backtracking" in tags:
        approaches.extend(["backtracking_dfs_state_restoration", "cascading_iterative_subsets", "bitmask_subset_enumeration"])

    # 10. Trie
    if "trie" in tags:
        approaches.extend(["trie_nested_hashmap", "trie_fixed_array_children", "hashset_prefix_lookup"])

    # 11. Prefix Sum & Difference Array
    if "prefix-sum" in tags:
        approaches.extend(["prefix_sum_hashmap_lookup", "prefix_sum_array_range_query", "difference_array_range_update"])

    # 12. Greedy
    if "greedy" in tags:
        approaches.extend(["greedy_sort_and_scan", "greedy_local_optimal_choice"])

    # 13. Math & Number Theory
    if "math" in tags or "number-theory" in tags:
        approaches.extend(["mathematical_formula_closed_form", "iterative_simulation"])
        if "greatest-common-divisor" in tags or "gcd" in title:
            approaches.append("euclidean_gcd_algorithm")
        if any(w in title for w in ["prime", "primes"]) or "prime-number-sieve" in tags:
            approaches.append("sieve_of_eratosthenes")

    # 14. Linked List
    if "linked-list" in tags:
        approaches.extend(["dummy_head_pointer_iteration", "fast_slow_two_pointers", "iterative_in_place_reversal"])

    # 15. String
    if "string" in tags and not approaches:
        approaches.extend(["string_builder_iteration", "frequency_map_counting", "two_pointers_palindrome_check"])

    # 16. Simulation & Enumeration
    if ("simulation" in tags or "enumeration" in tags) and not approaches:
        approaches.extend(["iterative_state_simulation", "brute_force_enumeration"])

    # 17. Sorting
    if "sorting" in tags:
        approaches.extend(["custom_comparator_sort", "two_pointers_after_sorting"])

    # Fallback and minimum approach guarantee
    if len(approaches) < 2:
        if "arrays" in tags or "array" in tags:
            approaches.extend(["two_pass_array_sweep", "iterative_optimal_solution"])
        else:
            approaches.extend(["iterative_optimal_solution", "recursive_alternative_solution"])

    seen = set()
    result = []
    for a in approaches:
        if a not in seen:
            seen.add(a)
            result.append(a)
    return result


def analyze_dataset(dataset_path):
    with open(dataset_path, "r", encoding="utf-8") as f:
        problems = json.load(f)

    paradigm_distribution = Counter()
    approach_counts = Counter()
    approach_len_distribution = Counter()

    for p in problems:
        apps = deduce_problem_approaches(p)
        approach_len_distribution[len(apps)] += 1
        for a in apps:
            approach_counts[a] += 1
            meta = APPROACH_METADATA.get(a, {})
            paradigm = meta.get("paradigm", "Custom / Problem-Specific")
            paradigm_distribution[paradigm] += 1

    return problems, approach_counts, paradigm_distribution, approach_len_distribution


def print_summary(dataset_path):
    problems, approach_counts, paradigm_dist, len_dist = analyze_dataset(dataset_path)
    total = len(problems)
    at_least_2 = sum(count for length, count in len_dist.items() if length >= 2)
    at_least_3 = sum(count for length, count in len_dist.items() if length >= 3)
    avg_approaches = sum(l * c for l, c in len_dist.items()) / total

    print("=" * 80)
    print("      DSA CONFIDENCE ENGINE -- MULTI-APPROACH TAXONOMY ANALYSIS REPORT       ")
    print("=" * 80)
    print(f"Total Problems Analyzed     : {total:,}")
    print(f"Unique Approach Archetypes  : {len(approach_counts)}")
    print(f"Multi-Approach Coverage (>=2): {at_least_2:,} ({at_least_2/total*100:.1f}%)")
    print(f"Broad Multi-Approach (>=3)  : {at_least_3:,} ({at_least_3/total*100:.1f}%)")
    print(f"Average Approaches / Problem: {avg_approaches:.2f}")
    print("-" * 80)
    print("PARADIGM FAMILY DISTRIBUTION:")
    for paradigm, count in paradigm_dist.most_common():
        print(f"  {paradigm:35s}: {count:5,d} occurrences")
    print("-" * 80)
    print("TOP 20 FREQUENT APPROACH ARCHETYPES:")
    for app, count in approach_counts.most_common(20):
        meta = APPROACH_METADATA.get(app, {})
        name = meta.get("name", app)
        time_c = meta.get("time_complexity", "N/A")
        print(f"  {app:38s} | {count:4d} probs | {time_c:18s} | {name}")
    print("=" * 80)


def print_problem_detail(dataset_path, problem_id):
    with open(dataset_path, "r", encoding="utf-8") as f:
        problems = json.load(f)

    target = None
    for p in problems:
        if p["id"] == problem_id or p.get("titleSlug") == problem_id or p["id"].endswith("_" + problem_id):
            target = p
            break

    if not target:
        print(f"Problem '{problem_id}' not found.")
        return

    approaches = deduce_problem_approaches(target)
    print("=" * 80)
    print(f"Problem ID   : {target['id']}")
    print(f"Title        : {target['title']}")
    print(f"Difficulty   : {target.get('difficulty', 'Medium')}")
    print(f"Topic Tags   : {', '.join(target.get('topic_tags', []))}")
    print(f"Primary DSA  : {', '.join(target.get('primary_concepts', []))}")
    print("-" * 80)
    print(f"ACCEPTED ALGORITHMIC STRATEGIES ({len(approaches)} approaches):")
    for idx, app in enumerate(approaches, 1):
        meta = APPROACH_METADATA.get(app, {})
        name = meta.get("name", app)
        paradigm = meta.get("paradigm", "Custom")
        time_c = meta.get("time_complexity", "Variable")
        space_c = meta.get("space_complexity", "Variable")
        concepts = ", ".join(meta.get("concepts", []))
        print(f"  {idx}. [{app}]")
        print(f"     Name      : {name}")
        print(f"     Paradigm  : {paradigm}")
        print(f"     Complexity: Time {time_c}, Space {space_c}")
        print(f"     Concepts  : {concepts}")
    print("=" * 80)


def update_dataset_with_approaches(dataset_path):
    with open(dataset_path, "r", encoding="utf-8") as f:
        problems = json.load(f)

    updated_count = 0
    for p in problems:
        new_approaches = deduce_problem_approaches(p)
        if p.get("accepted_strategies") != new_approaches:
            p["accepted_strategies"] = new_approaches
            updated_count += 1

    with open(dataset_path, "w", encoding="utf-8") as f:
        json.dump(problems, f, indent=2)

    print(f"Successfully enriched {updated_count} / {len(problems)} problems in {dataset_path}.")


def main():
    parser = argparse.ArgumentParser(description="Analyze and discover multi-approach strategies across the DSA dataset.")
    parser.add_argument("--dataset", default="data/problems/dataset_all.json", help="Path to problem dataset JSON")
    parser.add_argument("--summary", action="store_true", help="Print overall summary report of approaches")
    parser.add_argument("--problem", help="Inspect approach breakdown for a specific problem ID")
    parser.add_argument("--update-dataset", action="store_true", help="Enrich the dataset JSON with deduced multi-approaches")
    args = parser.parse_args()

    if args.summary:
        print_summary(args.dataset)
    elif args.problem:
        print_problem_detail(args.dataset, args.problem)
    elif args.update_dataset:
        update_dataset_with_approaches(args.dataset)
    else:
        print_summary(args.dataset)


if __name__ == "__main__":
    main()
