package evaluator

type CanonicalSolution struct {
	Code        string
	Explanation string
}

var CanonicalBenchmarkSolutions = map[string]CanonicalSolution{
	"binary_search_basic": {
		Code: `def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target:
            return mid
        elif nums[mid] < target:
            lo = mid + 1
        else:
            hi = mid - 1
    return -1
`,
		Explanation: "I use binary search on the sorted array maintaining low and high bounds to bisect the search space.",
	},
	"climbing_stairs": {
		Code: `def solve(n):
    if n <= 2: return n
    dp = [0] * (n + 1)
    dp[1], dp[2] = 1, 2
    for i in range(3, n + 1):
        dp[i] = dp[i - 1] + dp[i - 2]
    return dp[n]
`,
		Explanation: "I use dynamic programming with bottom-up tabulation array to compute ways to climb stairs.",
	},
	"coin_change": {
		Code: `def solve(coins, amount):
    dp = [float('inf')] * (amount + 1)
    dp[0] = 0
    for c in coins:
        for a in range(c, amount + 1):
            dp[a] = min(dp[a], dp[a - c] + 1)
    return dp[amount] if dp[amount] != float('inf') else -1
`,
		Explanation: "I use bottom-up dynamic programming tabulation array to compute minimum coins needed.",
	},
	"longest_increasing_subsequence": {
		Code: `def solve(nums):
    if not nums: return 0
    dp = [1] * len(nums)
    for i in range(len(nums)):
        for j in range(i):
            if nums[j] < nums[i]:
                dp[i] = max(dp[i], dp[j] + 1)
    return max(dp)
`,
		Explanation: "I use dynamic programming tabulation tracking longest increasing subsequence ending at each index.",
	},
	"longest_common_subsequence": {
		Code: `def solve(text1, text2):
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
		Explanation: "I use a 2D dynamic programming table to compute longest common subsequence between two strings.",
	},
	"house_robber": {
		Code: `def solve(nums):
    if not nums: return 0
    if len(nums) == 1: return nums[0]
    dp = [0] * len(nums)
    dp[0] = nums[0]
    dp[1] = max(nums[0], nums[1])
    for i in range(2, len(nums)):
        dp[i] = max(dp[i - 1], dp[i - 2] + nums[i])
    return dp[-1]
`,
		Explanation: "I use 1D dynamic programming table storing max loot possible up to each house.",
	},
	"valid_parentheses": {
		Code: `def solve(s):
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
		Explanation: "I use a LIFO stack to match closing brackets with their corresponding open brackets.",
	},
	"three_sum": {
		Code: `def solve(nums):
    nums.sort()
    res = []
    for i in range(len(nums) - 2):
        if i > 0 and nums[i] == nums[i - 1]:
            continue
        l, r = i + 1, len(nums) - 1
        while l < r:
            s = nums[i] + nums[l] + nums[r]
            if s == 0:
                res.append([nums[i], nums[l], nums[r]])
                while l < r and nums[l] == nums[l + 1]: l += 1
                while l < r and nums[r] == nums[r - 1]: r -= 1
                l += 1
                r -= 1
            elif s < 0:
                l += 1
            else:
                r -= 1
    return res
`,
		Explanation: "I sort the array and use two pointers (left and right) to find matching triplets summing to zero.",
	},
	"single_number": {
		Code: `def solve(nums):
    res = 0
    for x in nums:
        res ^= x
    return res
`,
		Explanation: "I use bit manipulation with bitwise XOR cancellation since identical numbers cancel out to zero.",
	},
	"number_of_islands": {
		Code: `def solve(grid):
    if not grid: return 0
    rows, cols = len(grid), len(grid[0])
    count = 0
    def dfs(r, c):
        if r < 0 or r >= rows or c < 0 or c >= cols or grid[r][c] != '1':
            return
        grid[r][c] = '0'
        dfs(r + 1, c)
        dfs(r - 1, c)
        dfs(r, c + 1)
        dfs(r, c - 1)
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == '1':
                dfs(r, c)
                count += 1
    return count
`,
		Explanation: "I count connected components using depth-first search recursive flood fill on the grid.",
	},
	"jump_game": {
		Code: `def solve(nums):
    reach = 0
    for i, x in enumerate(nums):
        if i > reach: return False
        reach = max(reach, i + x)
    return True
`,
		Explanation: "I use a greedy algorithm tracking the farthest reachable index at each step.",
	},
	"group_anagrams": {
		Code: `from collections import defaultdict
def solve(strs):
    groups = defaultdict(list)
    for s in strs:
        key = tuple(sorted(s))
        groups[key].append(s)
    res = [sorted(group) for group in groups.values()]
    return sorted(res)
`,
		Explanation: "I group anagrams using a hash map dictionary with sorted character keys.",
	},
	"maximum_subarray": {
		Code: `def solve(nums):
    max_so_far = nums[0]
    curr_max = nums[0]
    for x in nums[1:]:
        curr_max = max(x, curr_max + x)
        max_so_far = max(max_so_far, curr_max)
    return max_so_far
`,
		Explanation: "I use Kadane's algorithm dynamic programming maintaining max contiguous subarray sum.",
	},
	"top_k_frequent_elements": {
		Code: `import heapq
from collections import Counter
def solve(nums, k):
    count = Counter(nums)
    return sorted(heapq.nlargest(k, count.keys(), key=count.get))
`,
		Explanation: "I count frequencies in a hash map and use a min-heap priority queue to select top k elements.",
	},
	"implement_trie": {
		Code: `class TrieNode:
    def __init__(self):
        self.children = {}
        self.is_end = False
class Trie:
    def __init__(self):
        self.root = TrieNode()
    def insert(self, word):
        curr = self.root
        for c in word:
            if c not in curr.children:
                curr.children[c] = TrieNode()
            curr = curr.children[c]
        curr.is_end = True
    def search(self, word):
        curr = self.root
        for c in word:
            if c not in curr.children:
                return False
            curr = curr.children[c]
        return curr.is_end
    def startsWith(self, prefix):
        curr = self.root
        for c in prefix:
            if c not in curr.children:
                return False
            curr = curr.children[c]
        return True
def solve(operations, args):
    trie = Trie()
    res = []
    for cmd, arg in zip(operations, args):
        if cmd == 'insert':
            trie.insert(arg)
            res.append(None)
        elif cmd == 'search':
            res.append(trie.search(arg))
        elif cmd == 'startsWith':
            res.append(trie.startsWith(arg))
    return res
`,
		Explanation: "I implement a prefix tree (Trie) using tree nodes with children dictionaries and an is_end terminal flag.",
	},
	"minimum_window_substring": {
		Code: `from collections import Counter
def solve(s, t):
    if not t or not s: return ""
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
    return "" if ans[0] == float("inf") else s[ans[1] : ans[2] + 1]
`,
		Explanation: "I use a sliding window with two pointers and a frequency hash map dictionary to find the minimum window.",
	},
	"two_sum": {
		Code: `def solve(nums, target):
    seen = {}
    for i, x in enumerate(nums):
        diff = target - x
        if diff in seen:
            return [seen[diff], i]
        seen[x] = i
    return []
`,
		Explanation: "I iterate through the array and store seen numbers in a hash map dictionary to find complements in O(1) lookup.",
	},
	"second_maximum": {
		Code: `def second_maximum(nums):
    unique = list(set(nums))
    if len(unique) < 2: return None
    unique.sort()
    return unique[-2]
`,
		Explanation: "I eliminate duplicates with a set and sort to find the second maximum element.",
	},
	"first_non_repeating_character": {
		Code: `from collections import Counter
def first_non_repeating_character(s):
    counts = Counter(s)
    for c in s:
        if counts[c] == 1:
            return c
    return None
`,
		Explanation: "I count character frequencies using a hash map and perform a second pass to find the first character with frequency one.",
	},
	"frequency_count": {
		Code: `from collections import Counter
def char_frequency(s):
    return dict(Counter(s))
`,
		Explanation: "I count character frequencies using a hash map dictionary tallying each character occurrence.",
	},
	"count_bits": {
		Code: `def solve(n):
    ans = [0] * (n + 1)
    for i in range(1, n + 1):
        ans[i] = ans[i >> 1] + (i & 1)
    return ans
`,
		Explanation: "I use dynamic programming with bit manipulation using the relation that ans[i] = ans[i >> 1] + (i & 1).",
	},
	"course_schedule": {
		Code: `from collections import deque
def solve(numCourses, prerequisites):
    adj = [[] for _ in range(numCourses)]
    indegree = [0] * numCourses
    for dest, src in prerequisites:
        adj[src].append(dest)
        indegree[dest] += 1
    q = deque([i for i in range(numCourses) if indegree[i] == 0])
    visited = 0
    while q:
        u = q.popleft()
        visited += 1
        for v in adj[u]:
            indegree[v] -= 1
            if indegree[v] == 0:
                q.append(v)
    return visited == numCourses
`,
		Explanation: "I detect cycles in the dependency graph using topological sort with Kahn's algorithm in-degree queue.",
	},
	"daily_temperatures": {
		Code: `def solve(temperatures):
    ans = [0] * len(temperatures)
    stack = []
    for i, t in enumerate(temperatures):
        while stack and temperatures[stack[-1]] < t:
            prev = stack.pop()
            ans[prev] = i - prev
        stack.append(i)
    return ans
`,
		Explanation: "I use a monotonic decreasing stack to find the next warmer day for each index.",
	},
	"edit_distance": {
		Code: `def solve(word1, word2):
    m, n = len(word1), len(word2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]
    for i in range(m + 1): dp[i][0] = i
    for j in range(n + 1): dp[0][j] = j
    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if word1[i - 1] == word2[j - 1]:
                dp[i][j] = dp[i - 1][j - 1]
            else:
                dp[i][j] = 1 + min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
    return dp[m][n]
`,
		Explanation: "I use a 2D dynamic programming table computing the Levenshtein edit distance between the two words.",
	},
	"clone_graph": {
		Code: `def solve(adj):
    if not adj: return []
    cloned = {}
    def dfs(node):
        if node in cloned:
            return
        cloned[node] = list(adj[node - 1])
        for neighbor in adj[node - 1]:
            dfs(neighbor)
    for i in range(1, len(adj) + 1):
        if i not in cloned:
            dfs(i)
    return [cloned[i] for i in range(1, len(adj) + 1)]
`,
		Explanation: "I clone the graph structure traversing nodes using depth-first search and mapping clones with a hash map.",
	},
	"container_with_most_water": {
		Code: `def solve(height):
    l, r = 0, len(height) - 1
    max_area = 0
    while l < r:
        area = min(height[l], height[r]) * (r - l)
        max_area = max(max_area, area)
        if height[l] < height[r]:
            l += 1
        else:
            r -= 1
    return max_area
`,
		Explanation: "I use two pointers at both ends moving the pointer with smaller height inward to maximize area.",
	},
	"combination_sum": {
		Code: `def solve(candidates, target):
    res = []
    def dfs(idx, cur, total):
        if total == target:
            res.append(list(cur))
            return
        if total > target or idx >= len(candidates):
            return
        cur.append(candidates[idx])
        dfs(idx, cur, total + candidates[idx])
        cur.pop()
        dfs(idx + 1, cur, total)
    dfs(0, [], 0)
    return res
`,
		Explanation: "I use backtracking recursive depth-first search exploring choices with path restoration.",
	},
	"evaluate_reverse_polish_notation": {
		Code: `def solve(tokens):
    stack = []
    for t in tokens:
        if t in "+-*/":
            b = stack.pop()
            a = stack.pop()
            if t == '+': stack.append(a + b)
            elif t == '-': stack.append(a - b)
            elif t == '*': stack.append(a * b)
            else: stack.append(int(a / b))
        else:
            stack.append(int(t))
    return stack[0]
`,
		Explanation: "I evaluate Reverse Polish Notation expressions using a LIFO stack to store operands and apply operators.",
	},
	"find_all_anagrams": {
		Code: `from collections import Counter
def solve(s, p):
    if len(p) > len(s): return []
    p_count = Counter(p)
    s_count = Counter(s[:len(p) - 1])
    res = []
    for i in range(len(p) - 1, len(s)):
        s_count[s[i]] += 1
        if s_count == p_count:
            res.append(i - len(p) + 1)
        s_count[s[i - len(p) + 1]] -= 1
        if s_count[s[i - len(p) + 1]] == 0:
            del s_count[s[i - len(p) + 1]]
    return res
`,
		Explanation: "I find anagram starting indices using a sliding window with frequency counting counter dictionaries.",
	},
	"find_first_and_last_position": {
		Code: `def solve(nums, target):
    def findBound(isFirst):
        lo, hi = 0, len(nums) - 1
        bound = -1
        while lo <= hi:
            mid = (lo + hi) // 2
            if nums[mid] == target:
                bound = mid
                if isFirst: hi = mid - 1
                else: lo = mid + 1
            elif nums[mid] < target: lo = mid + 1
            else: hi = mid - 1
        return bound
    return [findBound(True), findBound(False)]
`,
		Explanation: "I perform two passes of binary search with low and high bounds to find the first and last target positions.",
	},
	"find_minimum_rotated_sorted_array": {
		Code: `def solve(nums):
    lo, hi = 0, len(nums) - 1
    while lo < hi:
        mid = (lo + hi) // 2
        if nums[mid] > nums[hi]: lo = mid + 1
        else: hi = mid
    return nums[lo]
`,
		Explanation: "I use binary search on the rotated sorted array comparing the midpoint against the high pointer.",
	},
	"gas_station": {
		Code: `def solve(gas, cost):
    if sum(gas) < sum(cost): return -1
    total = start = 0
    for i in range(len(gas)):
        total += gas[i] - cost[i]
        if total < 0:
            total = 0
            start = i + 1
    return start
`,
		Explanation: "I use a greedy algorithm tracking running gas balance and updating the candidate starting station.",
	},
	"koko_eating_bananas": {
		Code: `import math
def solve(piles, h):
    lo, hi = 1, max(piles)
    while lo < hi:
        speed = (lo + hi) // 2
        hours = sum(math.ceil(p / speed) for p in piles)
        if hours <= h: hi = speed
        else: lo = speed + 1
    return lo
`,
		Explanation: "I use binary search on the answer speed space to find the minimum eating rate.",
	},
	"kth_largest_element": {
		Code: `import heapq
def solve(nums, k):
    return heapq.nlargest(k, nums)[-1]
`,
		Explanation: "I use a min-heap priority queue of size k to extract the kth largest element.",
	},
	"longest_palindromic_substring": {
		Code: `def solve(s):
    if not s: return ""
    start, max_len = 0, 1
    def expand(l, r):
        while l >= 0 and r < len(s) and s[l] == s[r]:
            l -= 1
            r += 1
        return l + 1, r - 1
    for i in range(len(s)):
        l1, r1 = expand(i, i)
        l2, r2 = expand(i, i + 1)
        if r1 - l1 + 1 > max_len: start, max_len = l1, r1 - l1 + 1
        if r2 - l2 + 1 > max_len: start, max_len = l2, r2 - l2 + 1
    return s[start : start + max_len]
`,
		Explanation: "I find the longest palindromic substring by expanding two pointers around each character center.",
	},
	"longest_substring_without_repeating": {
		Code: `def solve(s):
    seen = {}
    l = 0
    max_len = 0
    for r, c in enumerate(s):
        if c in seen and seen[c] >= l:
            l = seen[c] + 1
        seen[c] = r
        max_len = max(max_len, r - l + 1)
    return max_len
`,
		Explanation: "I use a dynamic sliding window with a hash map dictionary tracking the last seen index of each character.",
	},
	"max_product_subarray": {
		Code: `def solve(nums):
    res = max(nums)
    cur_min, cur_max = 1, 1
    for n in nums:
        tmp = cur_max * n
        cur_max = max(n * cur_max, n * cur_min, n)
        cur_min = min(tmp, n * cur_min, n)
        res = max(res, cur_max)
    return res
`,
		Explanation: "I use dynamic programming tracking both running minimum and maximum products at each element.",
	},
	"median_of_two_sorted_arrays": {
		Code: `def solve(nums1, nums2):
    A, B = nums1, nums2
    total = len(A) + len(B)
    half = total // 2
    if len(B) < len(A):
        A, B = B, A
    l, r = 0, len(A) - 1
    while l <= r:
        i = (l + r) // 2
        j = half - i - 2
        Aleft = A[i] if i >= 0 else float("-infinity")
        Aright = A[i + 1] if (i + 1) < len(A) else float("infinity")
        Bleft = B[j] if j >= 0 else float("-infinity")
        Bright = B[j + 1] if (j + 1) < len(B) else float("infinity")
        if Aleft <= Bright and Bleft <= Aright:
            if total % 2:
                return float(min(Aright, Bright))
            return (max(Aleft, Bleft) + min(Aright, Bright)) / 2.0
        elif Aleft > Bright:
            r = i - 1
        else:
            l = i + 1
    return float(A[0]) if A else float(B[0])
`,
		Explanation: "I use binary search to partition the two sorted arrays into left and right halves with bisection pointers.",
	},
	"merge_intervals": {
		Code: `def solve(intervals):
    intervals.sort(key=lambda x: x[0])
    merged = []
    for interval in intervals:
        if not merged or merged[-1][1] < interval[0]:
            merged.append(interval)
        else:
            merged[-1][1] = max(merged[-1][1], interval[1])
    return merged
`,
		Explanation: "I sort intervals by start time and use a greedy scan to merge overlapping intervals.",
	},
	"missing_number": {
		Code: `def solve(nums):
    res = len(nums)
    for i, x in enumerate(nums):
        res ^= i ^ x
    return res
`,
		Explanation: "I find the missing number using bit manipulation bitwise XOR cancellation against the index range.",
	},
	"network_delay_time": {
		Code: `import heapq
from collections import defaultdict
def solve(times, n, k):
    graph = defaultdict(list)
    for u, v, w in times: graph[u].append((v, w))
    pq = [(0, k)]
    dist = {}
    while pq:
        d, u = heapq.heappop(pq)
        if u in dist: continue
        dist[u] = d
        for v, w in graph[u]:
            if v not in dist: heapq.heappush(pq, (d + w, v))
    return max(dist.values()) if len(dist) == n else -1
`,
		Explanation: "I compute shortest path transmission time using Dijkstra's algorithm with a min-heap priority queue.",
	},
	"number_of_1_bits": {
		Code: `def solve(n):
    count = 0
    while n:
        n &= n - 1
        count += 1
    return count
`,
		Explanation: "I count set bits using bit manipulation with Brian Kernighan's n and n minus 1 bit clearing trick.",
	},
	"number_of_connected_components": {
		Code: `def solve(n, edges):
    parent = list(range(n))
    def find(i):
        if parent[i] == i: return i
        parent[i] = find(parent[i])
        return parent[i]
    count = n
    for u, v in edges:
        root_u, root_v = find(u), find(v)
        if root_u != root_v:
            parent[root_u] = root_v
            count -= 1
    return count
`,
		Explanation: "I count connected components using a disjoint set union (Union-Find) with path compression.",
	},
	"pacific_atlantic_water_flow": {
		Code: `def solve(heights):
    if not heights: return []
    m, n = len(heights), len(heights[0])
    pac, atl = set(), set()
    def dfs(r, c, visit, prev):
        if (r, c) in visit or r < 0 or c < 0 or r >= m or c >= n or heights[r][c] < prev: return
        visit.add((r, c))
        dfs(r + 1, c, visit, heights[r][c])
        dfs(r - 1, c, visit, heights[r][c])
        dfs(r, c + 1, visit, heights[r][c])
        dfs(r, c - 1, visit, heights[r][c])
    for c in range(n):
        dfs(0, c, pac, heights[0][c])
        dfs(m - 1, c, atl, heights[m - 1][c])
    for r in range(m):
        dfs(r, 0, pac, heights[r][0])
        dfs(r, n - 1, atl, heights[r][n - 1])
    return sorted([list(coord) for coord in pac.intersection(atl)])
`,
		Explanation: "I find cells reaching both oceans using multi-source depth-first search from boundary cells with visited sets.",
	},
	"permutation_in_string": {
		Code: `from collections import Counter
def solve(s1, s2):
    if len(s1) > len(s2): return False
    c1 = Counter(s1)
    c2 = Counter(s2[:len(s1) - 1])
    for i in range(len(s1) - 1, len(s2)):
        c2[s2[i]] += 1
        if c1 == c2: return True
        c2[s2[i - len(s1) + 1]] -= 1
        if c2[s2[i - len(s1) + 1]] == 0: del c2[s2[i - len(s1) + 1]]
    return False
`,
		Explanation: "I check for permutation substring using a fixed-size sliding window with frequency counting dictionaries.",
	},
	"permutations": {
		Code: `def solve(nums):
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
		Explanation: "I generate all permutations using recursive backtracking exploring available choices with path restoration.",
	},
	"product_of_array_except_self": {
		Code: `def solve(nums):
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
		Explanation: "I compute array products using two passes calculating running prefix and suffix products in linear time.",
	},
	"reverse_bits": {
		Code: `def solve(n):
    res = 0
    for _ in range(32):
        res = (res << 1) | (n & 1)
        n >>= 1
    return res
`,
		Explanation: "I reverse bits using bit manipulation with bitwise shifting and masking.",
	},
	"search_rotated_sorted_array": {
		Code: `def solve(nums, target):
    lo, hi = 0, len(nums) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        if nums[mid] == target: return mid
        if nums[lo] <= nums[mid]:
            if nums[lo] <= target < nums[mid]: hi = mid - 1
            else: lo = mid + 1
        else:
            if nums[mid] < target <= nums[hi]: lo = mid + 1
            else: hi = mid - 1
    return -1
`,
		Explanation: "I use binary search on rotated sorted array determining which half is sorted at each step.",
	},
	"subsets": {
		Code: `def solve(nums):
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
		Explanation: "I generate all subsets using backtracking recursion appending path snapshots to result.",
	},
	"trapping_rain_water": {
		Code: `def solve(height):
    if not height: return 0
    l, r = 0, len(height) - 1
    l_max, r_max = height[l], height[r]
    res = 0
    while l < r:
        if l_max < r_max:
            l += 1
            l_max = max(l_max, height[l])
            res += l_max - height[l]
        else:
            r -= 1
            r_max = max(r_max, height[r])
            res += r_max - height[r]
    return res
`,
		Explanation: "I calculate trapped rainwater using two pointers inward sweep maintaining left and right max heights.",
	},
	"unique_paths": {
		Code: `def solve(m, n):
    dp = [1] * n
    for _ in range(1, m):
        for j in range(1, n):
            dp[j] += dp[j - 1]
    return dp[-1]
`,
		Explanation: "I compute unique grid paths using dynamic programming with 1D space-optimized array accumulation.",
	},
	"valid_palindrome": {
		Code: `def solve(s):
    cleaned = [c.lower() for c in s if c.isalnum()]
    l, r = 0, len(cleaned) - 1
    while l < r:
        if cleaned[l] != cleaned[r]: return False
        l += 1
        r -= 1
    return True
`,
		Explanation: "I verify palindrome symmetry using two pointers comparing characters from both ends inward.",
	},
	"word_break": {
		Code: `def solve(s, wordDict):
    word_set = set(wordDict)
    dp = [False] * (len(s) + 1)
    dp[0] = True
    for i in range(1, len(s) + 1):
        for j in range(i):
            if dp[j] and s[j:i] in word_set:
                dp[i] = True
                break
    return dp[-1]
`,
		Explanation: "I determine word segmentation feasibility using 1D dynamic programming table with a hashset word dictionary.",
	},
	"word_search": {
		Code: `def solve(board, word):
    rows, cols = len(board), len(board[0])
    def dfs(r, c, k):
        if k == len(word): return True
        if r < 0 or r >= rows or c < 0 or c >= cols or board[r][c] != word[k]: return False
        tmp = board[r][c]
        board[r][c] = '#'
        res = dfs(r+1, c, k+1) or dfs(r-1, c, k+1) or dfs(r, c+1, k+1) or dfs(r, c-1, k+1)
        board[r][c] = tmp
        return res
    for r in range(rows):
        for c in range(cols):
            if dfs(r, c, 0): return True
    return False
`,
		Explanation: "I search for the word using recursive backtracking depth-first search marking visited grid cells with restoration.",
	},
}

func GetCanonicalBenchmarkSolution(problemID string) (CanonicalSolution, bool) {
	sol, found := CanonicalBenchmarkSolutions[problemID]
	return sol, found
}
