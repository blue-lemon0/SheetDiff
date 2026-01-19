package main

import "sort"

// ========== 基本数据类型 ==========

// Row 数据行类型
//type Row map[string]string

// ========== 算法核心数据结构 ==========

// FieldNode 字段节点
type FieldNode struct {
	Field  string
	Values []string
	D      *RowSet // 独有行集合
	IsRoot bool
}

// FieldChain 字段链条
type FieldChain struct {
	Field string
	Nodes []*FieldNode
	Root  *FieldNode
}

// Tree 支配关系树
type Tree struct {
	Root     *FieldNode
	Children []*Tree
	Parent   *Tree
}

// PerfectRule 完美规则
type PerfectRule struct {
	Conditions  []RuleCondition
	Covered     int
	TotalCommon int
}

// RuleCondition 规则条件
type RuleCondition struct {
	Field  string
	Values []string
}

// ========== 主算法类 ==========

// RuleFinder 规则发现器
type RuleFinder struct {
	MaxValuesPerRule int
	MaxConditions    int
	MaxRowsToAnalyze int
}

// NewRuleFinder 创建规则发现器
func NewRuleFinder() *RuleFinder {
	return &RuleFinder{
		MaxValuesPerRule: 3,
		MaxConditions:    3,
		MaxRowsToAnalyze: 10000,
	}
}

// ========== 核心算法方法 ==========

// FindRules 发现规则（主要入口）
func (rf *RuleFinder) FindRules(rows []Row, commonSet *RowSet) []PerfectRule {
	// 1. 构建所有字段链条
	chains := rf.buildAllChains(rows, commonSet)

	// 2. 建立支配关系森林
	forest := rf.buildForest(chains)

	// 3. 寻找完美规则
	rules := rf.findPerfectRules(forest, commonSet.Len(), len(rows))

	return rules
}

// buildAllChains 构建所有字段链条
func (rf *RuleFinder) buildAllChains(rows []Row, commonSet *RowSet) []*FieldChain {
	// 获取所有字段名
	if len(rows) == 0 {
		return []*FieldChain{}
	}

	fields := make([]string, 0, len(rows[0]))
	for field := range rows[0] {
		fields = append(fields, field)
	}

	// 为每个字段构建链条（使用 pattern_analyzer.go 的强化分析）
	var chains []*FieldChain
	for _, field := range fields {
		// 使用 pattern_analyzer.go 的强化分析函数
		// 支持等于、前缀、范围等多种规律
		chain := analyzeFieldPatterns(field, rows, commonSet)
		if chain != nil && chain.Root != nil {
			chains = append(chains, chain)
		}
	}

	return chains
}

// buildForest 建立支配关系森林
func (rf *RuleFinder) buildForest(chains []*FieldChain) []*Tree {
	// 收集所有根节点
	var roots []*FieldNode
	for _, chain := range chains {
		if chain.Root != nil {
			roots = append(roots, chain.Root)
		}
	}

	return rf.buildTreesFromRoots(roots)
}

// buildTreesFromRoots 从根节点构建树
func (rf *RuleFinder) buildTreesFromRoots(roots []*FieldNode) []*Tree {
	if len(roots) == 0 {
		return []*Tree{}
	}

	// 1. 建立支配关系
	childrenMap := make(map[*FieldNode][]*FieldNode)

	for i := 0; i < len(roots); i++ {
		for j := 0; j < len(roots); j++ {
			if i == j {
				continue
			}

			// 检查支配关系
			if roots[i].D.SubsetOf(roots[j].D) {
				// roots[i] ⊆ roots[j]，roots[i]更好（更小的D）
				childrenMap[roots[i]] = append(childrenMap[roots[i]], roots[j])
			} else if roots[j].D.SubsetOf(roots[i].D) {
				childrenMap[roots[j]] = append(childrenMap[roots[j]], roots[i])
			}
		}
	}

	// 2. 找到顶级节点（不被任何其他节点支配的）
	var topLevelNodes []*FieldNode
	for _, node := range roots {
		isTopLevel := true
		for _, other := range roots {
			if node == other {
				continue
			}
			// 如果其他节点支配这个节点，那么它不是顶级
			if other.D.SubsetOf(node.D) && !node.D.SubsetOf(other.D) {
				isTopLevel = false
				break
			}
		}
		if isTopLevel {
			topLevelNodes = append(topLevelNodes, node)
		}
	}

	// 3. 构建树
	var forest []*Tree
	for _, rootNode := range topLevelNodes {
		tree := &Tree{Root: rootNode}
		if children, exists := childrenMap[rootNode]; exists {
			for _, childNode := range children {
				// 检查这个child是否确实被root支配
				if childNode.D.SubsetOf(rootNode.D) {
					tree.Children = append(tree.Children, &Tree{
						Root:   childNode,
						Parent: tree,
					})
				}
			}
		}
		forest = append(forest, tree)
	}

	return forest
}

// findPerfectRules 寻找完美规则
func (rf *RuleFinder) findPerfectRules(forest []*Tree, commonCount, totalRows int) []PerfectRule {
	// 收集所有树的根节点
	var allRoots []*FieldNode
	for _, tree := range forest {
		if tree.Root != nil {
			allRoots = append(allRoots, tree.Root)
		}
	}

	return rf.searchRuleCombinations(allRoots, commonCount, totalRows)
}

// searchRuleCombinations 搜索规则组合
func (rf *RuleFinder) searchRuleCombinations(roots []*FieldNode, commonCount, totalRows int) []PerfectRule {
	var rules []PerfectRule

	// 1. 检查单个完美字段（D为空）
	for _, root := range roots {
		if root.D.Len() == 0 {
			rules = append(rules, PerfectRule{
				Conditions:  []RuleCondition{{Field: root.Field, Values: root.Values}},
				Covered:     commonCount,
				TotalCommon: commonCount,
			})
			return rules // 有完美单字段，直接返回
		}
	}

	// 2. 尝试两个字段的组合
	for i := 0; i < len(roots); i++ {
		for j := i + 1; j < len(roots); j++ {
			// 检查两个D是否不相交
			if roots[i].D.Disjoint(roots[j].D) {
				rules = append(rules, PerfectRule{
					Conditions: []RuleCondition{
						{Field: roots[i].Field, Values: roots[i].Values},
						{Field: roots[j].Field, Values: roots[j].Values},
					},
					Covered:     commonCount,
					TotalCommon: commonCount,
				})
			}
		}
	}

	// 3. 尝试三个字段的组合
	for i := 0; i < len(roots); i++ {
		for j := i + 1; j < len(roots); j++ {
			for k := j + 1; k < len(roots); k++ {
				// 检查三个D是否两两不相交
				if roots[i].D.Disjoint(roots[j].D) &&
					roots[i].D.Disjoint(roots[k].D) &&
					roots[j].D.Disjoint(roots[k].D) {

					rules = append(rules, PerfectRule{
						Conditions: []RuleCondition{
							{Field: roots[i].Field, Values: roots[i].Values},
							{Field: roots[j].Field, Values: roots[j].Values},
							{Field: roots[k].Field, Values: roots[k].Values},
						},
						Covered:     commonCount,
						TotalCommon: commonCount,
					})
				}
			}
		}
	}

	// 4. 按组合后的D集合大小排序（越小越好）
	sort.Slice(rules, func(a, b int) bool {
		sizeA := 0
		for _, cond := range rules[a].Conditions {
			// 找到对应的FieldNode
			for _, root := range roots {
				if root.Field == cond.Field && equalStringSlices(root.Values, cond.Values) {
					sizeA += root.D.Len()
					break
				}
			}
		}
		
		sizeB := 0
		for _, cond := range rules[b].Conditions {
			// 找到对应的FieldNode
			for _, root := range roots {
				if root.Field == cond.Field && equalStringSlices(root.Values, cond.Values) {
					sizeB += root.D.Len()
					break
				}
			}
		}
		
		return sizeA < sizeB
	})

	return rules
}

// equalStringSlices 比较两个字符串切片是否相等
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
