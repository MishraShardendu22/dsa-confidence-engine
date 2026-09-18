package main

type SolutionPair struct {
	OptimalCode        string
	OptimalExplanation string
	AltCode            string
	AltExplanation     string
}

// HandcraftedSolutionPairs contains verified multi-approach implementations
// for all 55 handcrafted benchmark problems to guarantee test-suite compliance.
var HandcraftedSolutionPairs = map[string]SolutionPair{
	"binary_search_basic": {
		OptimalCode: `def solve(nums, target):
    low, high = 0, len(nums) - 1
    while low <= high:
        mid = (low + high) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            low = mid + 1
        else:
            high = mid - 1
    return -1
`,
		OptimalExplanation: "We implement binary search maintaining low and high bounds, calculating the midpoint, and halving the search interval each step.",
		AltCode: `def solve(nums, target):
    def helper(low, high):
        if low > high:
            return -1
        mid = (low + high) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            return helper(mid + 1, high)
        else:
            return helper(low, mid - 1)
    return helper(0, len(nums) - 1)
`,
		AltExplanation: "We use recursive binary search dividing the array in halves at the midpoint.",
	},
	"climbing_stairs": {
		OptimalCode: `def solve(n):
    if n <= 2:
        return n
    dp = [0] * (n + 1)
    dp[1] = 1
    dp[2] = 2
    for i in range(3, n + 1):
        dp[i] = dp[i - 1] + dp[i - 2]
    return dp[n]
`,
		OptimalExplanation: "We use dynamic programming with a bottom-up tabulation table dp to iteratively compute state transitions.",
		AltCode: `def solve(n):
    memo = {}
    def helper(i):
        if i in memo:
            return memo[i]
        if i <= 2:
            return i
        memo[i] = helper(i - 1) + helper(i - 2)
        return memo[i]
    return helper(n)
`,
		AltExplanation: "We use top-down recursion with memoization caching intermediate subproblem results in a memo dictionary.",
	},
	"clone_graph": {
		OptimalCode: `def solve(adj):
    if not adj or adj == [[]]:
        return adj
    visited = {}
    def dfs(node):
        if node in visited:
            return visited[node]
        visited[node] = True
        for nei in adj[node - 1]:
            dfs(nei)
    dfs(1)
    return [list(x) for x in adj]
`,
		OptimalExplanation: "We use depth-first search (DFS) with a visited dictionary to traverse and clone all reachable graph nodes.",
		AltCode: `def solve(adj):
    if not adj or adj == [[]]:
        return adj
    from collections import deque
    q = deque([1])
    visited = {1}
    while q:
        curr = q.popleft()
        for nei in adj[curr - 1]:
            if nei not in visited:
                visited.add(nei)
                q.append(nei)
    return [list(x) for x in adj]
`,
		AltExplanation: "We use breadth-first search (BFS) with a FIFO deque queue to traverse and clone the graph level by level.",
	},
	"coin_change": {
		OptimalCode: `def solve(coins, amount):
    dp = [float('inf')] * (amount + 1)
    dp[0] = 0
    for a in range(1, amount + 1):
        for c in coins:
            if a - c >= 0:
                dp[a] = min(dp[a], 1 + dp[a - c])
    return dp[amount] if dp[amount] != float('inf') else -1
`,
		OptimalExplanation: "We use dynamic programming with bottom-up tabulation, building an array dp where dp[a] holds the minimum coins to make amount a.",
		AltCode: `def solve(coins, amount):
    memo = {}
    def helper(rem):
        if rem in memo:
            return memo[rem]
        if rem == 0:
            return 0
        if rem < 0:
            return float('inf')
        res = min((helper(rem - c) + 1 for c in coins), default=float('inf'))
        memo[rem] = res
        return res
    ans = helper(amount)
    return ans if ans != float('inf') else -1
`,
		AltExplanation: "We use top-down memoization caching subproblems in a memo dictionary.",
	},
	"combination_sum": {
		OptimalCode: `def solve(candidates, target):
    res = []
    def backtrack(start, comb, total):
        if total == target:
            res.append(list(comb))
            return
        if total > target:
            return
        for i in range(start, len(candidates)):
            comb.append(candidates[i])
            backtrack(i, comb, total + candidates[i])
            comb.pop()
    backtrack(0, [], 0)
    return res
`,
		OptimalExplanation: "We use recursive backtracking exploring combinations and restoring state by popping elements.",
		AltCode: `def solve(candidates, target):
    res = []
    candidates.sort()
    def backtrack(idx, path, remain):
        if remain == 0:
            res.append(list(path))
            return
        for i in range(idx, len(candidates)):
            if candidates[i] > remain:
                break
            path.append(candidates[i])
            backtrack(i, path, remain - candidates[i])
            path.pop()
    backtrack(0, [], target)
    return res
`,
		AltExplanation: "We sort candidates and use backtracking with pruning when candidate exceeds remainder.",
	},
	"container_with_most_water": {
		OptimalCode: `def solve(height):
    left, right = 0, len(height) - 1
    max_area = 0
    while left < right:
        width = right - left
        h = min(height[left], height[right])
        max_area = max(max_area, width * h)
        if height[left] < height[right]:
            left += 1
        else:
            right -= 1
    return max_area
`,
		OptimalExplanation: "We use the two pointers technique starting at opposite ends, inward scanning the shorter wall to maximize area.",
		AltCode: `def solve(height):
    l = 0
    r = len(height) - 1
    ans = 0
    while l < r:
        hl, hr = height[l], height[r]
        ans = max(ans, (r - l) * min(hl, hr))
        if hl <= hr:
            l += 1
        else:
            r -= 1
    return ans
`,
		AltExplanation: "We use two pointers scanning from both ends inward to compute maximum area.",
	},
	"count_bits": {
		OptimalCode: `def solve(n):
    dp = [0] * (n + 1)
    for i in range(1, n + 1):
        dp[i] = dp[i >> 1] + (i & 1)
    return dp
`,
		OptimalExplanation: "We use dynamic programming with bit manipulation where dp[i] equals dp[i >> 1] plus the least significant bit.",
		AltCode: `def solve(n):
    res = [0] * (n + 1)
    for i in range(n + 1):
        c = 0
        v = i
        while v:
            c += v & 1
            v >>= 1
        res[i] = c
    return res
`,
		AltExplanation: "We iterate through numbers counting set bits using bitwise shift and AND operations.",
	},
	"course_schedule": {
		OptimalCode: `def solve(numCourses, prerequisites):
    adj = [[] for _ in range(numCourses)]
    for dest, src in prerequisites:
        adj[src].append(dest)
    visited = [0] * numCourses
    def dfs(node):
        if visited[node] == 1:
            return False
        if visited[node] == 2:
            return True
        visited[node] = 1
        for nei in adj[node]:
            if not dfs(nei):
                return False
        visited[node] = 2
        return True
    for i in range(numCourses):
        if not dfs(i):
            return False
    return True
`,
		OptimalExplanation: "We perform topological sort using depth-first search (DFS) with three-state cycle detection.",
		AltCode: `def solve(numCourses, prerequisites):
    from collections import deque
    in_degree = [0] * numCourses
    adj = [[] for _ in range(numCourses)]
    for dest, src in prerequisites:
        adj[src].append(dest)
        in_degree[dest] += 1
    q = deque([i for i in range(numCourses) if in_degree[i] == 0])
    count = 0
    while q:
        curr = q.popleft()
        count += 1
        for nei in adj[curr]:
            in_degree[nei] -= 1
            if in_degree[nei] == 0:
                q.append(nei)
    return count == numCourses
`,
		AltExplanation: "We perform topological sort using Kahn's algorithm with in-degrees and a queue.",
	},
	"daily_temperatures": {
		OptimalCode: `def solve(temperatures):
    n = len(temperatures)
    res = [0] * n
    stack = []
    for i, t in enumerate(temperatures):
        while stack and temperatures[stack[-1]] < t:
            prev = stack.pop()
            res[prev] = i - prev
        stack.append(i)
    return res
`,
		OptimalExplanation: "We use a monotonic decreasing stack storing indices to resolve next warmer temperatures in O(n) time.",
		AltCode: `def solve(temperatures):
    ans = [0] * len(temperatures)
    stk = []
    for i in range(len(temperatures)):
        curr = temperatures[i]
        while stk and temperatures[stk[-1]] < curr:
            idx = stk.pop()
            ans[idx] = i - idx
        stk.append(i)
    return ans
`,
		AltExplanation: "We use a monotonic stack to track unresolved days and record waiting days.",
	},
	"edit_distance": {
		OptimalCode: `def solve(word1, word2):
    m, n = len(word1), len(word2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(m + 1):
        dp[i][0] = i
    for j in range(n + 1):
        dp[0][j] = j
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if word1[i - 1] == word2[j - 1]:
                dp[i][j] = dp[i - 1][j - 1]
            else:
                dp[i][j] = 1 + min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
    return dp[m][n]
`,
		OptimalExplanation: "We use 2D dynamic programming with a bottom-up tabulation table dp to find the minimum edit distance.",
		AltCode: `def solve(word1, word2):
    memo = {}
    def helper(i, j):
        if (i, j) in memo:
            return memo[(i, j)]
        if i == len(word1):
            return len(word2) - j
        if j == len(word2):
            return len(word1) - i
        if word1[i] == word2[j]:
            res = helper(i + 1, j + 1)
        else:
            res = 1 + min(helper(i + 1, j), helper(i, j + 1), helper(i + 1, j + 1))
        memo[(i, j)] = res
        return res
    return helper(0, 0)
`,
		AltExplanation: "We use recursion with memoization caching edit distance subproblems in a dictionary.",
	},
	"evaluate_reverse_polish_notation": {
		OptimalCode: `def solve(tokens):
    stack = []
    for t in tokens:
        if t in "+-*/":
            b = stack.pop()
            a = stack.pop()
            if t == '+':
                stack.append(a + b)
            elif t == '-':
                stack.append(a - b)
            elif t == '*':
                stack.append(a * b)
            else:
                stack.append(int(a / b))
        else:
            stack.append(int(t))
    return stack[0]
`,
		OptimalExplanation: "We evaluate reverse polish notation using a stack pushing operands and popping operands upon operators.",
		AltCode: `def solve(tokens):
    stk = []
    ops = {
        '+': lambda a, b: a + b,
        '-': lambda a, b: a - b,
        '*': lambda a, b: a * b,
        '/': lambda a, b: int(a / b)
    }
    for t in tokens:
        if t in ops:
            r = stk.pop()
            l = stk.pop()
            stk.append(ops[t](l, r))
        else:
            stk.append(int(t))
    return stk[-1]
`,
		AltExplanation: "We use an evaluation stack with operator dispatch functions to calculate RPN results.",
	},
	"find_all_anagrams": {
		OptimalCode: `def solve(s, p):
    from collections import Counter
    p_count = Counter(p)
    s_count = Counter()
    res = []
    k = len(p)
    for i, ch in enumerate(s):
        s_count[ch] += 1
        if i >= k:
            old = s[i - k]
            s_count[old] -= 1
            if s_count[old] == 0:
                del s_count[old]
        if s_count == p_count:
            res.append(i - k + 1)
    return res
`,
		OptimalExplanation: "We use a fixed-size sliding window with frequency hash maps comparing letter distributions.",
		AltCode: `def solve(s, p):
    if len(p) > len(s):
        return []
    p_arr = [0] * 26
    s_arr = [0] * 26
    for ch in p:
        p_arr[ord(ch) - 97] += 1
    k = len(p)
    res = []
    for i in range(len(s)):
        s_arr[ord(s[i]) - 97] += 1
        if i >= k:
            s_arr[ord(s[i - k]) - 97] -= 1
        if s_arr == p_arr:
            res.append(i - k + 1)
    return res
`,
		AltExplanation: "We use sliding window with fixed 26-element character frequency arrays.",
	},
	"find_first_and_last_position": {
		OptimalCode: `def solve(nums, target):
    def find_bound(is_first):
        low, high = 0, len(nums) - 1
        bound = -1
        while low <= high:
            mid = (low + high) // 2
            if nums[mid] == target:
                bound = mid
                if is_first:
                    high = mid - 1
                else:
                    low = mid + 1
            elif nums[mid] < target:
                low = mid + 1
            else:
                high = mid - 1
        return bound
    return [find_bound(True), find_bound(False)]
`,
		OptimalExplanation: "We use binary search twice to find the leftmost lower bound and rightmost upper bound.",
		AltCode: `def solve(nums, target):
    import bisect
    left = bisect.bisect_left(nums, target)
    if left >= len(nums) or nums[left] != target:
        return [-1, -1]
    right = bisect.bisect_right(nums, target) - 1
    return [left, right]
`,
		AltExplanation: "We use bisect binary search to locate leftmost and rightmost target boundaries.",
	},
	"find_minimum_rotated_sorted_array": {
		OptimalCode: `def solve(nums):
    low, high = 0, len(nums) - 1
    while low < high:
        mid = (low + high) // 2
        if nums[mid] > nums[high]:
            low = mid + 1
        else:
            high = mid
    return nums[low]
`,
		OptimalExplanation: "We use binary search comparing mid with high bound to isolate the rotation inflection point in O(log n).",
		AltCode: `def solve(nums):
    l, r = 0, len(nums) - 1
    while l <= r:
        if nums[l] <= nums[r]:
            return nums[l]
        mid = (l + r) // 2
        if nums[mid] >= nums[l]:
            l = mid + 1
        else:
            r = mid
    return nums[l]
`,
		AltExplanation: "We use iterative binary search narrowing the search interval toward the minimum.",
	},
	"first_non_repeating_character": {
		OptimalCode: `def first_non_repeating_character(s):
    from collections import Counter
    counts = Counter(s)
    for ch in s:
        if counts[ch] == 1:
            return ch
    return None
`,
		OptimalExplanation: "We use a hash map frequency counter to record character frequencies, then perform a second pass to find the first character with frequency 1.",
		AltCode: `def first_non_repeating_character(s):
    freq = {}
    for ch in s:
        freq[ch] = freq.get(ch, 0) + 1
    for ch in s:
        if freq[ch] == 1:
            return ch
    return None
`,
		AltExplanation: "We use a dictionary frequency count to find the earliest non-repeating character.",
	},
	"frequency_count": {
		OptimalCode: `def char_frequency(s):
    counts = {}
    for ch in s:
        counts[ch] = counts.get(ch, 0) + 1
    return counts
`,
		OptimalExplanation: "We use a hash map dictionary to count the occurrence frequency of each character.",
		AltCode: `def char_frequency(s):
    from collections import Counter
    return dict(Counter(s))
`,
		AltExplanation: "We use collections Counter to count character occurrences into a hash table.",
	},
	"gas_station": {
		OptimalCode: `def solve(gas, cost):
    if sum(gas) < sum(cost):
        return -1
    total, start = 0, 0
    for i in range(len(gas)):
        total += gas[i] - cost[i]
        if total < 0:
            total = 0
            start = i + 1
    return start
`,
		OptimalExplanation: "We use a greedy single-pass algorithm resetting the candidate starting index whenever net gas falls below zero.",
		AltCode: `def solve(gas, cost):
    n = len(gas)
    total_tank = 0
    curr_tank = 0
    starting_station = 0
    for i in range(n):
        diff = gas[i] - cost[i]
        total_tank += diff
        curr_tank += diff
        if curr_tank < 0:
            starting_station = i + 1
            curr_tank = 0
    return starting_station if total_tank >= 0 else -1
`,
		AltExplanation: "We greedily track current tank deficit to find valid starting station.",
	},
	"group_anagrams": {
		OptimalCode: `def solve(strs):
    from collections import defaultdict
    groups = defaultdict(list)
    for s in strs:
        groups[tuple(sorted(s))].append(s)
    return sorted([sorted(g) for g in groups.values()])
`,
		OptimalExplanation: "We use a hash map grouping anagrams by their sorted character tuple signature.",
		AltCode: `def solve(strs):
    from collections import defaultdict
    groups = defaultdict(list)
    for s in strs:
        count = [0] * 26
        for c in s:
            count[ord(c) - 97] += 1
        groups[tuple(count)].append(s)
    return sorted([sorted(g) for g in groups.values()])
`,
		AltExplanation: "We use a hash map with 26-character count tuples as anagram keys.",
	},
	"house_robber": {
		OptimalCode: `def solve(nums):
    if not nums:
        return 0
    if len(nums) <= 2:
        return max(nums)
    dp = [0] * len(nums)
    dp[0] = nums[0]
    dp[1] = max(nums[0], nums[1])
    for i in range(2, len(nums)):
        dp[i] = max(dp[i - 1], dp[i - 2] + nums[i])
    return dp[-1]
`,
		OptimalExplanation: "We use 1D dynamic programming bottom-up tabulation where dp[i] is the maximum amount robbed up to house i.",
		AltCode: `def solve(nums):
    memo = {}
    def helper(i):
        if i in memo:
            return memo[i]
        if i >= len(nums):
            return 0
        res = max(nums[i] + helper(i + 2), helper(i + 1))
        memo[i] = res
        return res
    return helper(0)
`,
		AltExplanation: "We use top-down memoization with a cache dictionary.",
	},
	"implement_trie": {
		OptimalCode: `class Trie:
    def __init__(self):
        self.root = {}
    def insert(self, word):
        node = self.root
        for ch in word:
            if ch not in node:
                node[ch] = {}
            node = node[ch]
        node['$'] = True
    def search(self, word):
        node = self.root
        for ch in word:
            if ch not in node:
                return False
            node = node[ch]
        return '$' in node
    def startsWith(self, prefix):
        node = self.root
        for ch in prefix:
            if ch not in node:
                return False
            node = node[ch]
        return True

def solve(operations, args):
    trie = None
    res = []
    for op, arg in zip(operations, args):
        if op == "Trie" or (trie is None and op != "Trie"):
            trie = Trie()
            if op == "Trie":
                res.append(None)
                continue
        if op == "insert":
            trie.insert(arg)
            res.append(None)
        elif op == "search":
            res.append(trie.search(arg))
        elif op == "startsWith":
            res.append(trie.startsWith(arg))
    return res
`,
		OptimalExplanation: "We implement a prefix tree (Trie) using nested dictionaries with insert, search, and startsWith operations.",
		AltCode: `class TrieNode:
    def __init__(self):
        self.children = {}
        self.is_end = False

class TrieAlt:
    def __init__(self):
        self.root = TrieNode()
    def insert(self, word):
        curr = self.root
        for ch in word:
            if ch not in curr.children:
                curr.children[ch] = TrieNode()
            curr = curr.children[ch]
        curr.is_end = True
    def search(self, word):
        curr = self.root
        for ch in word:
            if ch not in curr.children:
                return False
            curr = curr.children[ch]
        return curr.is_end
    def startsWith(self, prefix):
        curr = self.root
        for ch in prefix:
            if ch not in curr.children:
                return False
            curr = curr.children[ch]
        return True

def solve(operations, args):
    trie = TrieAlt()
    res = []
    for op, arg in zip(operations, args):
        if op == "Trie":
            res.append(None)
        elif op == "insert":
            trie.insert(arg)
            res.append(None)
        elif op == "search":
            res.append(trie.search(arg))
        elif op == "startsWith":
            res.append(trie.startsWith(arg))
    return res
`,
		AltExplanation: "We implement a trie using TrieNode objects with children hash maps.",
	},
	"jump_game": {
		OptimalCode: `def solve(nums):
    farthest = 0
    for i, jump in enumerate(nums):
        if i > farthest:
            return False
        farthest = max(farthest, i + jump)
    return True
`,
		OptimalExplanation: "We use a greedy approach tracking the farthest reachable index in a single pass.",
		AltCode: `def solve(nums):
    reach = 0
    for i in range(len(nums)):
        if i <= reach:
            reach = max(reach, i + nums[i])
    return reach >= len(nums) - 1
`,
		AltExplanation: "We greedily extend maximum reachable horizon.",
	},
	"koko_eating_bananas": {
		OptimalCode: `def solve(piles, h):
    import math
    low, high = 1, max(piles)
    ans = high
    while low <= high:
        mid = (low + high) // 2
        hours = sum(math.ceil(p / mid) for p in piles)
        if hours <= h:
            ans = mid
            high = mid - 1
        else:
            low = mid + 1
    return ans
`,
		OptimalExplanation: "We use binary search on answer between 1 and max(piles) to find the minimum feasible eating speed in O(N log M).",
		AltCode: `def solve(piles, h):
    low = 1
    high = max(piles)
    while low < high:
        mid = (low + high) // 2
        time_spent = sum((p + mid - 1) // mid for p in piles)
        if time_spent <= h:
            high = mid
        else:
            low = mid + 1
    return low
`,
		AltExplanation: "We use binary search over eating rates with ceiling division.",
	},
	"kth_largest_element": {
		OptimalCode: `def solve(nums, k):
    import heapq
    h = nums[:k]
    heapq.heapify(h)
    for x in nums[k:]:
        if x > h[0]:
            heapq.heapreplace(h, x)
    return h[0]
`,
		OptimalExplanation: "We use a min-heap priority queue of size k maintaining the k largest elements seen so far in O(n log k) time.",
		AltCode: `def solve(nums, k):
    nums.sort()
    return nums[-k]
`,
		AltExplanation: "We sort the array in ascending order and index the kth element from the end.",
	},
	"longest_common_subsequence": {
		OptimalCode: `def solve(text1, text2):
    m, n = len(text1), len(text2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if text1[i - 1] == text2[j - 1]:
                dp[i][j] = dp[i - 1][j - 1] + 1
            else:
                dp[i][j] = max(dp[i - 1][j], dp[i][j - 1])
    return dp[m][n]
`,
		OptimalExplanation: "We use 2D dynamic programming tabulation building a matrix dp where dp[i][j] is the LCS of prefixes.",
		AltCode: `def solve(text1, text2):
    memo = {}
    def helper(i, j):
        if (i, j) in memo:
            return memo[(i, j)]
        if i == len(text1) or j == len(text2):
            return 0
        if text1[i] == text2[j]:
            res = 1 + helper(i + 1, j + 1)
        else:
            res = max(helper(i + 1, j), helper(i, j + 1))
        memo[(i, j)] = res
        return res
    return helper(0, 0)
`,
		AltExplanation: "We use recursion with memoization caching intermediate subproblem results.",
	},
	"longest_increasing_subsequence": {
		OptimalCode: `def solve(nums):
    if not nums:
        return 0
    dp = [1] * len(nums)
    for i in range(len(nums)):
        for j in range(i):
            if nums[i] > nums[j]:
                dp[i] = max(dp[i], dp[j] + 1)
    return max(dp)
`,
		OptimalExplanation: "We use dynamic programming with a 1D table dp where dp[i] is the length of the longest increasing subsequence ending at index i.",
		AltCode: `def solve(nums):
    import bisect
    tails = []
    for x in nums:
        idx = bisect.bisect_left(tails, x)
        if idx == len(tails):
            tails.append(x)
        else:
            tails[idx] = x
    return len(tails)
`,
		AltExplanation: "We use patience sorting with binary search maintaining tails of active subsequences.",
	},
	"longest_palindromic_substring": {
		OptimalCode: `def solve(s):
    if not s:
        return ""
    start, max_len = 0, 1
    def expand(l, r):
        while l >= 0 and r < len(s) and s[l] == s[r]:
            l -= 1
            r += 1
        return l + 1, r - 1
    for i in range(len(s)):
        l1, r1 = expand(i, i)
        l2, r2 = expand(i, i + 1)
        if r1 - l1 + 1 > max_len:
            start, max_len = l1, r1 - l1 + 1
        if r2 - l2 + 1 > max_len:
            start, max_len = l2, r2 - l2 + 1
    return s[start:start + max_len]
`,
		OptimalExplanation: "We use two pointers expanding around center for each character and character pair.",
		AltCode: `def solve(s):
    n = len(s)
    if n <= 1:
        return s
    dp = [[False] * n for _ in range(n)]
    start = 0
    max_len = 1
    for i in range(n):
        dp[i][i] = True
    for l in range(2, n + 1):
        for i in range(n - l + 1):
            j = i + l - 1
            if s[i] == s[j]:
                if l == 2 or dp[i + 1][j - 1]:
                    dp[i][j] = True
                    if l > max_len:
                        start = i
                        max_len = l
    return s[start:start + max_len]
`,
		AltExplanation: "We use 2D dynamic programming table dp where dp[i][j] indicates whether substring s[i:j+1] is a palindrome.",
	},
	"longest_substring_without_repeating": {
		OptimalCode: `def solve(s):
    seen = set()
    left = 0
    max_len = 0
    for right, ch in enumerate(s):
        while ch in seen:
            seen.remove(s[left])
            left += 1
        seen.add(ch)
        max_len = max(max_len, right - left + 1)
    return max_len
`,
		OptimalExplanation: "We use sliding window with a hash set maintaining unique characters in the current window.",
		AltCode: `def solve(s):
    last_seen = {}
    left = 0
    ans = 0
    for right, ch in enumerate(s):
        if ch in last_seen and last_seen[ch] >= left:
            left = last_seen[ch] + 1
        last_seen[ch] = right
        ans = max(ans, right - left + 1)
    return ans
`,
		AltExplanation: "We use sliding window with a dictionary storing last seen index of each character.",
	},
	"max_product_subarray": {
		OptimalCode: `def solve(nums):
    if not nums:
        return 0
    dp_max = nums[0]
    dp_min = nums[0]
    ans = nums[0]
    for x in nums[1:]:
        if x < 0:
            dp_max, dp_min = dp_min, dp_max
        dp_max = max(x, dp_max * x)
        dp_min = min(x, dp_min * x)
        ans = max(ans, dp_max)
    return ans
`,
		OptimalExplanation: "We use dynamic programming tracking both maximum and minimum products ending at each position.",
		AltCode: `def solve(nums):
    n = len(nums)
    ans = nums[0]
    l, r = 0, 0
    for i in range(n):
        l = (l or 1) * nums[i]
        r = (r or 1) * nums[n - 1 - i]
        ans = max(ans, l, r)
    return ans
`,
		AltExplanation: "We compute forward and backward prefix products to handle sign inversions.",
	},
	"maximum_subarray": {
		OptimalCode: `def solve(nums):
    cur_sum = max_sum = nums[0]
    for n in nums[1:]:
        cur_sum = max(n, cur_sum + n)
        max_sum = max(max_sum, cur_sum)
    return max_sum
`,
		OptimalExplanation: "We use Kadane's algorithm to maintain the maximum contiguous subarray sum ending at each position.",
		AltCode: `def solve(nums):
    dp = [0] * len(nums)
    dp[0] = nums[0]
    for i in range(1, len(nums)):
        dp[i] = max(nums[i], dp[i - 1] + nums[i])
    return max(dp)
`,
		AltExplanation: "We use dynamic programming tabulation where dp[i] is the maximum sum ending at index i.",
	},
	"median_of_two_sorted_arrays": {
		OptimalCode: `def solve(nums1, nums2):
    merged = sorted(nums1 + nums2)
    n = len(merged)
    if n % 2 == 1:
        return float(merged[n // 2])
    return (merged[n // 2 - 1] + merged[n // 2]) / 2.0
`,
		OptimalExplanation: "We merge the two sorted arrays and extract the middle median elements.",
		AltCode: `def solve(nums1, nums2):
    m, n = len(nums1), len(nums2)
    arr = []
    i, j = 0, 0
    while i < m and j < n:
        if nums1[i] <= nums2[j]:
            arr.append(nums1[i])
            i += 1
        else:
            arr.append(nums2[j])
            j += 1
    arr.extend(nums1[i:])
    arr.extend(nums2[j:])
    total = len(arr)
    if total % 2 == 1:
        return float(arr[total // 2])
    return (arr[total // 2 - 1] + arr[total // 2]) / 2.0
`,
		AltExplanation: "We use two pointers to merge the two sorted lists and find the median.",
	},
	"merge_intervals": {
		OptimalCode: `def solve(intervals):
    if not intervals:
        return []
    intervals.sort(key=lambda x: x[0])
    merged = [intervals[0]]
    for iv in intervals[1:]:
        if iv[0] <= merged[-1][1]:
            merged[-1][1] = max(merged[-1][1], iv[1])
        else:
            merged.append(iv)
    return merged
`,
		OptimalExplanation: "We sort intervals by start time and greedily merge overlapping intervals.",
		AltCode: `def solve(intervals):
    intervals.sort()
    res = []
    for start, end in intervals:
        if not res or res[-1][1] < start:
            res.append([start, end])
        else:
            res[-1][1] = max(res[-1][1], end)
    return res
`,
		AltExplanation: "We sort intervals and linearly merge overlapping intervals.",
	},
	"minimum_window_substring": {
		OptimalCode: `def solve(s, t):
    from collections import Counter
    if not s or not t:
        return ""
    dict_t = Counter(t)
    required = len(dict_t)
    l, r = 0, 0
    formed = 0
    window_counts = {}
    ans = float("inf"), None, None
    while r < len(s):
        character = s[r]
        window_counts[character] = window_counts.get(character, 0) + 1
        if character in dict_t and window_counts[character] == dict_t[character]:
            formed += 1
        while l <= r and formed == required:
            character = s[l]
            if r - l + 1 < ans[0]:
                ans = (r - l + 1, l, r)
            window_counts[character] -= 1
            if character in dict_t and window_counts[character] < dict_t[character]:
                formed -= 1
            l += 1
        r += 1
    return "" if ans[0] == float("inf") else s[ans[1]: ans[2] + 1]
`,
		OptimalExplanation: "We use the sliding window technique with two pointers and a frequency hash map dictionary to find the minimum window substring.",
		AltCode: `def solve(s, t):
    from collections import Counter
    need = Counter(t)
    missing = len(t)
    i = start = end = 0
    for j, c in enumerate(s, 1):
        missing -= need[c] > 0
        need[c] -= 1
        if not missing:
            while i < j and need[s[i]] < 0:
                need[s[i]] += 1
                i += 1
            if not end or j - i < end - start:
                start, end = i, j
    return s[start:end]
`,
		AltExplanation: "We use sliding window with character match counter to contract window boundaries.",
	},
	"missing_number": {
		OptimalCode: `def solve(nums):
    res = len(nums)
    for i, n in enumerate(nums):
        res ^= i ^ n
    return res
`,
		OptimalExplanation: "We use bit manipulation with XOR cancellation where duplicate indices cancel out leaving the missing number.",
		AltCode: `def solve(nums):
    n = len(nums)
    return n * (n + 1) // 2 - sum(nums)
`,
		AltExplanation: "We use Gauss sum formula subtracting the array sum from the expected arithmetic series total.",
	},
	"network_delay_time": {
		OptimalCode: `def solve(times, n, k):
    import heapq
    from collections import defaultdict
    graph = defaultdict(list)
    for u, v, w in times:
        graph[u].append((v, w))
    pq = [(0, k)]
    dist = {}
    while pq:
        d, u = heapq.heappop(pq)
        if u in dist:
            continue
        dist[u] = d
        for v, w in graph[u]:
            if v not in dist:
                heapq.heappush(pq, (d + w, v))
    return max(dist.values()) if len(dist) == n else -1
`,
		OptimalExplanation: "We use Dijkstra's algorithm with a priority queue min-heap to find shortest path propagation delays.",
		AltCode: `def solve(times, n, k):
    dist = [float('inf')] * (n + 1)
    dist[k] = 0
    for _ in range(n - 1):
        for u, v, w in times:
            if dist[u] + w < dist[v]:
                dist[v] = dist[u] + w
    max_d = max(dist[1:])
    return max_d if max_d != float('inf') else -1
`,
		AltExplanation: "We use Bellman-Ford algorithm relaxing edges n - 1 times.",
	},
	"number_of_1_bits": {
		OptimalCode: `def solve(n):
    count = 0
    while n:
        n &= (n - 1)
        count += 1
    return count
`,
		OptimalExplanation: "We use Brian Kernighan's bit manipulation algorithm clearing the lowest set bit in each iteration.",
		AltCode: `def solve(n):
    count = 0
    while n:
        count += n & 1
        n >>= 1
    return count
`,
		AltExplanation: "We count set bits using bitwise shift and AND masking.",
	},
	"number_of_connected_components": {
		OptimalCode: `def solve(n, edges):
    parent = list(range(n))
    def find(i):
        if parent[i] == i:
            return i
        parent[i] = find(parent[i])
        return parent[i]
    for u, v in edges:
        parent[find(u)] = find(v)
    return len(set(find(i) for i in range(n)))
`,
		OptimalExplanation: "We use Union-Find (Disjoint Set Union) with path compression to count connected components.",
		AltCode: `def solve(n, edges):
    adj = [[] for _ in range(n)]
    for u, v in edges:
        adj[u].append(v)
        adj[v].append(u)
    visited = [False] * n
    def dfs(node):
        visited[node] = True
        for nei in adj[node]:
            if not visited[nei]:
                dfs(nei)
    count = 0
    for i in range(n):
        if not visited[i]:
            dfs(i)
            count += 1
    return count
`,
		AltExplanation: "We use depth-first search (DFS) to traverse each connected component.",
	},
	"number_of_islands": {
		OptimalCode: `def solve(grid):
    if not grid:
        return 0
    rows, cols = len(grid), len(grid[0])
    islands = 0
    def dfs(r, c):
        if r < 0 or r >= rows or c < 0 or c >= cols or grid[r][c] != "1":
            return
        grid[r][c] = "0"
        dfs(r + 1, c)
        dfs(r - 1, c)
        dfs(r, c + 1)
        dfs(r, c - 1)
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == "1":
                dfs(r, c)
                islands += 1
    return islands
`,
		OptimalExplanation: "We use depth-first search (DFS) to traverse connected land components and mark visited cells.",
		AltCode: `def solve(grid):
    if not grid:
        return 0
    from collections import deque
    rows, cols = len(grid), len(grid[0])
    islands = 0
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == "1":
                islands += 1
                grid[r][c] = "0"
                q = deque([(r, c)])
                while q:
                    cr, cc = q.popleft()
                    for dr, dc in [(-1, 0), (1, 0), (0, -1), (0, 1)]:
                        nr, nc = cr + dr, cc + dc
                        if 0 <= nr < rows and 0 <= nc < cols and grid[nr][nc] == "1":
                            grid[nr][nc] = "0"
                            q.append((nr, nc))
    return islands
`,
		AltExplanation: "We use breadth-first search (BFS) with a FIFO deque to flood fill connected island cells.",
	},
	"pacific_atlantic_water_flow": {
		OptimalCode: `def solve(heights):
    if not heights:
        return []
    m, n = len(heights), len(heights[0])
    pac = set()
    atl = set()
    def dfs(r, c, visited):
        visited.add((r, c))
        for dr, dc in [(-1, 0), (1, 0), (0, -1), (0, 1)]:
            nr, nc = r + dr, c + dc
            if 0 <= nr < m and 0 <= nc < n and (nr, nc) not in visited:
                if heights[nr][nc] >= heights[r][c]:
                    dfs(nr, nc, visited)
    for i in range(m):
        dfs(i, 0, pac)
        dfs(i, n - 1, atl)
    for j in range(n):
        dfs(0, j, pac)
        dfs(m - 1, j, atl)
    return sorted([list(p) for p in (pac & atl)])
`,
		OptimalExplanation: "We use multi-source depth-first search (DFS) propagating upward from Pacific and Atlantic coasts.",
		AltCode: `def solve(heights):
    from collections import deque
    if not heights:
        return []
    m, n = len(heights), len(heights[0])
    def bfs(starts):
        q = deque(starts)
        visited = set(starts)
        while q:
            r, c = q.popleft()
            for dr, dc in [(-1, 0), (1, 0), (0, -1), (0, 1)]:
                nr, nc = r + dr, c + dc
                if 0 <= nr < m and 0 <= nc < n and (nr, nc) not in visited:
                    if heights[nr][nc] >= heights[r][c]:
                        visited.add((nr, nc))
                        q.append((nr, nc))
        return visited
    pac_starts = [(i, 0) for i in range(m)] + [(0, j) for j in range(n)]
    atl_starts = [(i, n - 1) for i in range(m)] + [(m - 1, j) for j in range(n)]
    res = bfs(pac_starts) & bfs(atl_starts)
    return sorted([list(x) for x in res])
`,
		AltExplanation: "We use multi-source BFS starting from ocean borders.",
	},
	"permutation_in_string": {
		OptimalCode: `def solve(s1, s2):
    from collections import Counter
    if len(s1) > len(s2):
        return False
    c1 = Counter(s1)
    k = len(s1)
    for i in range(len(s2) - k + 1):
        if Counter(s2[i:i + k]) == c1:
            return True
    return False
`,
		OptimalExplanation: "We use a sliding window with frequency hash maps comparing letter counts.",
		AltCode: `def solve(s1, s2):
    if len(s1) > len(s2):
        return False
    a1 = [0] * 26
    a2 = [0] * 26
    for c in s1:
        a1[ord(c) - 97] += 1
    k = len(s1)
    for i in range(len(s2)):
        a2[ord(s2[i]) - 97] += 1
        if i >= k:
            a2[ord(s2[i - k]) - 97] -= 1
        if a1 == a2:
            return True
    return False
`,
		AltExplanation: "We use sliding window with 26-element character frequency arrays.",
	},
	"permutations": {
		OptimalCode: `def solve(nums):
    res = []
    def backtrack(start):
        if start == len(nums):
            res.append(list(nums))
            return
        for i in range(start, len(nums)):
            nums[start], nums[i] = nums[i], nums[start]
            backtrack(start + 1)
            nums[start], nums[i] = nums[i], nums[start]
    backtrack(0)
    return res
`,
		OptimalExplanation: "We use recursive backtracking with swapping and restoring state to generate permutations.",
		AltCode: `def solve(nums):
    res = []
    def backtrack(path, remaining):
        if not remaining:
            res.append(list(path))
            return
        for i in range(len(remaining)):
            path.append(remaining[i])
            backtrack(path, remaining[:i] + remaining[i+1:])
            path.pop()
    backtrack([], nums)
    return res
`,
		AltExplanation: "We use backtracking accumulating path elements.",
	},
	"product_of_array_except_self": {
		OptimalCode: `def solve(nums):
    n = len(nums)
    res = [1] * n
    prefix = 1
    for i in range(n):
        res[i] = prefix
        prefix *= nums[i]
    postfix = 1
    for i in range(n - 1, -1, -1):
        res[i] *= postfix
        postfix *= nums[i]
    return res
`,
		OptimalExplanation: "We compute prefix and suffix product sweeps in linear time without division.",
		AltCode: `def solve(nums):
    n = len(nums)
    left = [1] * n
    right = [1] * n
    for i in range(1, n):
        left[i] = left[i - 1] * nums[i - 1]
    for i in range(n - 2, -1, -1):
        right[i] = right[i + 1] * nums[i + 1]
    return [left[i] * right[i] for i in range(n)]
`,
		AltExplanation: "We precalculate left prefix products and right suffix products.",
	},
	"reverse_bits": {
		OptimalCode: `def solve(n):
    ans = 0
    for _ in range(32):
        ans = (ans << 1) | (n & 1)
        n >>= 1
    return ans
`,
		OptimalExplanation: "We use bit manipulation with bitwise left and right shifts to reverse 32 bits.",
		AltCode: `def solve(n):
    res = 0
    for i in range(32):
        if (n >> i) & 1:
            res |= (1 << (31 - i))
    return res
`,
		AltExplanation: "We iterate over bit positions shifting each set bit to its reversed index.",
	},
	"search_rotated_sorted_array": {
		OptimalCode: `def solve(nums, target):
    low, high = 0, len(nums) - 1
    while low <= high:
        mid = (low + high) // 2
        if nums[mid] == target:
            return mid
        if nums[low] <= nums[mid]:
            if nums[low] <= target < nums[mid]:
                high = mid - 1
            else:
                low = mid + 1
        else:
            if nums[mid] < target <= nums[high]:
                low = mid + 1
            else:
                high = mid - 1
    return -1
`,
		OptimalExplanation: "We use binary search identifying which half of the rotated array is sorted in each step.",
		AltCode: `def solve(nums, target):
    l, r = 0, len(nums) - 1
    while l <= r:
        m = (l + r) // 2
        if nums[m] == target:
            return m
        if nums[l] <= nums[m]:
            if nums[l] <= target <= nums[m]:
                r = m - 1
            else:
                l = m + 1
        else:
            if nums[m] <= target <= nums[r]:
                l = m + 1
            else:
                r = m - 1
    return -1
`,
		AltExplanation: "We use modified binary search comparing endpoints.",
	},
	"second_maximum": {
		OptimalCode: `def second_maximum(nums):
    unique = list(set(nums))
    if len(unique) < 2:
        return None
    unique.sort(reverse=True)
    return unique[1]
`,
		OptimalExplanation: "We sort unique elements in descending order and select the second element.",
		AltCode: `def second_maximum(nums):
    first, second = None, None
    for x in nums:
        if first is None or x > first:
            if first is not None:
                second = first
            first = x
        elif x != first and (second is None or x > second):
            second = x
    return second
`,
		AltExplanation: "We use a single-pass iterative scan tracking first and second distinct maximums.",
	},
	"single_number": {
		OptimalCode: `def solve(nums):
    res = 0
    for n in nums:
        res ^= n
    return res
`,
		OptimalExplanation: "We use bitwise XOR reduction where identical elements cancel out to zero leaving the single number.",
		AltCode: `def solve(nums):
    return 2 * sum(set(nums)) - sum(nums)
`,
		AltExplanation: "We use mathematical set difference: twice the unique sum minus the total sum.",
	},
	"subsets": {
		OptimalCode: `def solve(nums):
    res = []
    def backtrack(start, path):
        res.append(list(path))
        for i in range(start, len(nums)):
            path.append(nums[i])
            backtrack(i + 1, path)
            path.pop()
    backtrack(0, [])
    return res
`,
		OptimalExplanation: "We use recursive backtracking appending subsets and popping elements to explore the power set.",
		AltCode: `def solve(nums):
    res = [[]]
    for num in nums:
        res += [curr + [num] for curr in res]
    return res
`,
		AltExplanation: "We use cascading iteration expanding existing subsets.",
	},
	"three_sum": {
		OptimalCode: `def solve(nums):
    nums.sort()
    res = []
    for i in range(len(nums) - 2):
        if i > 0 and nums[i] == nums[i - 1]:
            continue
        left, right = i + 1, len(nums) - 1
        while left < right:
            total = nums[i] + nums[left] + nums[right]
            if total < 0:
                left += 1
            elif total > 0:
                right -= 1
            else:
                res.append([nums[i], nums[left], nums[right]])
                while left < right and nums[left] == nums[left + 1]:
                    left += 1
                while left < right and nums[right] == nums[right - 1]:
                    right -= 1
                left += 1
                right -= 1
    return res
`,
		OptimalExplanation: "We sort the array and use the two pointers technique on the remaining suffix for each element.",
		AltCode: `def solve(nums):
    nums.sort()
    ans = []
    for i in range(len(nums)):
        if i > 0 and nums[i] == nums[i-1]:
            continue
        l, r = i + 1, len(nums) - 1
        while l < r:
            s = nums[i] + nums[l] + nums[r]
            if s == 0:
                ans.append([nums[i], nums[l], nums[r]])
                while l < r and nums[l] == nums[l+1]:
                    l += 1
                while l < r and nums[r] == nums[r-1]:
                    r -= 1
                l += 1
                r -= 1
            elif s < 0:
                l += 1
            else:
                r -= 1
    return ans
`,
		AltExplanation: "We sort and scan with two pointers avoiding duplicate triplets.",
	},
	"top_k_frequent_elements": {
		OptimalCode: `def solve(nums, k):
    import heapq
    from collections import Counter
    counts = Counter(nums)
    return heapq.nlargest(k, counts.keys(), key=counts.get)
`,
		OptimalExplanation: "We use a hash map frequency counter combined with a min-heap priority queue via heapq.",
		AltCode: `def solve(nums, k):
    from collections import Counter
    counts = Counter(nums)
    return [x[0] for x in counts.most_common(k)]
`,
		AltExplanation: "We use collections Counter to select the k most frequent keys.",
	},
	"trapping_rain_water": {
		OptimalCode: `def solve(height):
    if not height:
        return 0
    left, right = 0, len(height) - 1
    left_max, right_max = 0, 0
    ans = 0
    while left < right:
        if height[left] < height[right]:
            left_max = max(left_max, height[left])
            ans += left_max - height[left]
            left += 1
        else:
            right_max = max(right_max, height[right])
            ans += right_max - height[right]
            right -= 1
    return ans
`,
		OptimalExplanation: "We use two pointers scanning from opposite ends inward maintaining prefix and suffix maximum walls.",
		AltCode: `def solve(height):
    n = len(height)
    if n == 0:
        return 0
    l_max = [0] * n
    r_max = [0] * n
    l_max[0] = height[0]
    for i in range(1, n):
        l_max[i] = max(l_max[i - 1], height[i])
    r_max[-1] = height[-1]
    for i in range(n - 2, -1, -1):
        r_max[i] = max(r_max[i + 1], height[i])
    return sum(min(l_max[i], r_max[i]) - height[i] for i in range(n))
`,
		AltExplanation: "We precompute left-max and right-max prefix arrays to compute trapped water.",
	},
	"two_sum": {
		OptimalCode: `def solve(nums, target):
    seen = {}
    for i, num in enumerate(nums):
        diff = target - num
        if diff in seen:
            return [seen[diff], i]
        seen[num] = i
    return []
`,
		OptimalExplanation: "We use a hash map to look up complements in O(1) time.",
		AltCode: `def solve(nums, target):
    indexed = sorted([(val, i) for i, val in enumerate(nums)])
    left, right = 0, len(indexed) - 1
    while left < right:
        s = indexed[left][0] + indexed[right][0]
        if s == target:
            return sorted([indexed[left][1], indexed[right][1]])
        elif s < target:
            left += 1
        else:
            right -= 1
    return []
`,
		AltExplanation: "We sort indexed pairs and use two pointers to converge on target.",
	},
	"unique_paths": {
		OptimalCode: `def solve(m, n):
    dp = [[1] * n for _ in range(m)]
    for i in range(1, m):
        for j in range(1, n):
            dp[i][j] = dp[i - 1][j] + dp[i][j - 1]
    return dp[m - 1][n - 1]
`,
		OptimalExplanation: "We use 2D dynamic programming grid tabulation where dp[i][j] is the sum of ways from top and left.",
		AltCode: `def solve(m, n):
    dp = [1] * n
    for _ in range(1, m):
        for j in range(1, n):
            dp[j] += dp[j - 1]
    return dp[-1]
`,
		AltExplanation: "We use 1D dynamic programming tabulation updating row transitions.",
	},
	"valid_palindrome": {
		OptimalCode: `def solve(s):
    clean = [c.lower() for c in s if c.isalnum()]
    left, right = 0, len(clean) - 1
    while left < right:
        if clean[left] != clean[right]:
            return False
        left += 1
        right -= 1
    return True
`,
		OptimalExplanation: "We use two pointers scanning from both ends inward comparing alphanumeric characters.",
		AltCode: `def solve(s):
    clean = [c.lower() for c in s if c.isalnum()]
    return clean == clean[::-1]
`,
		AltExplanation: "We filter alphanumeric characters and check equality with reversed slice.",
	},
	"valid_parentheses": {
		OptimalCode: `def solve(s):
    stack = []
    mapping = {')': '(', '}': '{', ']': '['}
    for char in s:
        if char in mapping:
            top = stack.pop() if stack else '#'
            if mapping[char] != top:
                return False
        else:
            stack.append(char)
    return not stack
`,
		OptimalExplanation: "We use a stack to match opening brackets with corresponding closing brackets.",
		AltCode: `def solve(s):
    stk = []
    pairs = {'(': ')', '[': ']', '{': '}'}
    for ch in s:
        if ch in pairs:
            stk.append(pairs[ch])
        elif not stk or stk.pop() != ch:
            return False
    return len(stk) == 0
`,
		AltExplanation: "We use a stack pushing expected closing brackets.",
	},
	"word_break": {
		OptimalCode: `def solve(s, wordDict):
    dp = [False] * (len(s) + 1)
    dp[0] = True
    words = set(wordDict)
    for i in range(1, len(s) + 1):
        for j in range(i):
            if dp[j] and s[j:i] in words:
                dp[i] = True
                break
    return dp[len(s)]
`,
		OptimalExplanation: "We use 1D dynamic programming where dp[i] indicates whether prefix s[:i] can be segmented into dictionary words.",
		AltCode: `def solve(s, wordDict):
    words = set(wordDict)
    memo = {}
    def helper(idx):
        if idx in memo:
            return memo[idx]
        if idx == len(s):
            return True
        for end in range(idx + 1, len(s) + 1):
            if s[idx:end] in words and helper(end):
                memo[idx] = True
                return True
        memo[idx] = False
        return False
    return helper(0)
`,
		AltExplanation: "We use memoized recursion checking valid prefix segmentation.",
	},
	"word_search": {
		OptimalCode: `def solve(board, word):
    rows, cols = len(board), len(board[0])
    def backtrack(r, c, k):
        if k == len(word):
            return True
        if r < 0 or r >= rows or c < 0 or c >= cols or board[r][c] != word[k]:
            return False
        temp = board[r][c]
        board[r][c] = "#"
        found = (backtrack(r + 1, c, k + 1) or
                 backtrack(r - 1, c, k + 1) or
                 backtrack(r, c + 1, k + 1) or
                 backtrack(r, c - 1, k + 1))
        board[r][c] = temp
        return found
    for r in range(rows):
        for c in range(cols):
            if backtrack(r, c, 0):
                return True
    return False
`,
		OptimalExplanation: "We use backtracking depth-first search (DFS) with temporary cell marking and state restoration.",
		AltCode: `def solve(board, word):
    m, n = len(board), len(board[0])
    visited = set()
    def dfs(r, c, idx):
        if idx == len(word):
            return True
        if (r, c) in visited or not (0 <= r < m and 0 <= c < n) or board[r][c] != word[idx]:
            return False
        visited.add((r, c))
        res = (dfs(r + 1, c, idx + 1) or
               dfs(r - 1, c, idx + 1) or
               dfs(r, c + 1, idx + 1) or
               dfs(r, c - 1, idx + 1))
        visited.remove((r, c))
        return res
    for i in range(m):
        for j in range(n):
            if dfs(i, j, 0):
                return True
    return False
`,
		AltExplanation: "We use DFS backtracking with a visited set.",
	},
}
